package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/iminders/aicoder/internal/llm"
	"github.com/iminders/aicoder/internal/logger"
	"github.com/iminders/aicoder/internal/session"
)

const defaultBaseURL = "https://api.anthropic.com"
const apiVersion = "2023-06-01"

// Provider implements llm.Provider for Anthropic.
type Provider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func New(apiKey, baseURL, model string) *Provider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *Provider) Name() string         { return "anthropic" }
func (p *Provider) CurrentModel() string { return p.model }

// --- native message types ---

type msgContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type nativeMsg struct {
	Role    string       `json:"role"`
	Content []msgContent `json:"content"`
}

type requestBody struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system,omitempty"`
	Messages  []nativeMsg      `json:"messages"`
	Tools     []llm.ToolSchema `json:"tools,omitempty"`
	Stream    bool             `json:"stream"`
}

// convertMessages converts session.Message slice to native Anthropic messages.
func convertMessages(msgs []session.Message) []nativeMsg {
	var result []nativeMsg
	for _, m := range msgs {
		if m.Role == "system" {
			continue // system is a top-level field
		}
		nm := nativeMsg{Role: m.Role}
		for _, c := range m.Content {
			mc := msgContent{Type: c.Type, Text: c.Text, ID: c.ID, Name: c.Name}
			if c.Input != nil {
				if b, err := json.Marshal(c.Input); err == nil {
					mc.Input = b
				}
			}
			if c.ToolUseID != "" {
				mc.ToolUseID = c.ToolUseID
				mc.Content = c.Text
				mc.IsError = c.IsError
			}
			nm.Content = append(nm.Content, mc)
		}
		if len(nm.Content) > 0 {
			result = append(result, nm)
		}
	}
	return result
}

func extractSystem(msgs []session.Message) string {
	for _, m := range msgs {
		if m.Role == "system" {
			for _, c := range m.Content {
				if c.Type == "text" {
					return c.Text
				}
			}
		}
	}
	return ""
}

// Stream implements llm.Provider.
func (p *Provider) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamEvent, error) {
	msgs, ok := req.RawMsgs.([]session.Message)
	if !ok {
		return nil, fmt.Errorf("anthropic provider expects []session.Message")
	}

	body := requestBody{
		Model:     p.model,
		MaxTokens: req.MaxTokens,
		System:    extractSystem(msgs),
		Messages:  convertMessages(msgs),
		Tools:     req.Tools,
		Stream:    true,
	}
	if req.MaxTokens == 0 {
		body.MaxTokens = 8192
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)
	httpReq.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan llm.StreamEvent, 64)
	go p.readStream(resp.Body, ch)
	return ch, nil
}

// SSE event types from Anthropic
type sseEvent struct {
	Type         string           `json:"type"`
	Index        int              `json:"index"`
	Delta        *sseDelta        `json:"delta"`
	Usage        *sseUsage        `json:"usage"`
	ContentBlock *sseContentBlock `json:"content_block"`
}

type sseDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	PartialJSON string `json:"partial_json"`
	StopReason  string `json:"stop_reason"`
}

type sseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type sseContentBlock struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

func (p *Provider) readStream(body io.ReadCloser, ch chan<- llm.StreamEvent) {
	defer body.Close()
	defer close(ch)

	// Track partial tool_use blocks by index
	type toolAccum struct {
		id   string
		name string
		json strings.Builder
	}
	tools := map[int]*toolAccum{}

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var eventType string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var ev sseEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			logger.Debug("SSE parse error: %v | line: %s", err, data)
			continue
		}
		_ = eventType

		switch ev.Type {
		case "content_block_start":
			if ev.ContentBlock != nil && ev.ContentBlock.Type == "tool_use" {
				tools[ev.Index] = &toolAccum{
					id:   ev.ContentBlock.ID,
					name: ev.ContentBlock.Name,
				}
			}

		case "content_block_delta":
			if ev.Delta == nil {
				continue
			}
			switch ev.Delta.Type {
			case "text_delta":
				ch <- llm.StreamEvent{Type: "text_delta", Delta: ev.Delta.Text}
			case "input_json_delta":
				if t, ok := tools[ev.Index]; ok {
					t.json.WriteString(ev.Delta.PartialJSON)
				}
			}

		case "content_block_stop":
			if t, ok := tools[ev.Index]; ok {
				raw := json.RawMessage(t.json.String())
				if len(raw) == 0 {
					raw = json.RawMessage("{}")
				}
				ch <- llm.StreamEvent{
					Type: "tool_use_end",
					ToolUse: &llm.ToolUseBlock{
						ID:    t.id,
						Name:  t.name,
						Input: raw,
					},
				}
				delete(tools, ev.Index)
			}

		case "message_delta":
			if ev.Usage != nil {
				ch <- llm.StreamEvent{
					Type:   "usage",
					Input:  ev.Usage.InputTokens,
					Output: ev.Usage.OutputTokens,
				}
			}

		case "message_stop":
			ch <- llm.StreamEvent{Type: "done"}

		case "error":
			ch <- llm.StreamEvent{Type: "error", Err: fmt.Errorf("stream error: %s", data)}
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		ch <- llm.StreamEvent{Type: "error", Err: err}
	}
}
