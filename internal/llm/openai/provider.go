// Package openai implements the OpenAI-compatible LLM provider.
//
// Token Usage Handling:
// This provider supports both LLM-provided usage information and client-side estimation.
// When the LLM provides usage data in the streaming response, it is used directly.
// When usage is null (common with local LLM deployments), the provider estimates
// token counts using heuristics:
//   - English text: ~4 characters per token
//   - CJK text (Chinese/Japanese/Korean): ~1.5 characters per token
//   - Mixed text: automatically detected based on character composition
//
// This ensures consistent token tracking across different LLM backends.
package openai

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

const defaultBaseURL = "https://api.openai.com"

// Provider implements llm.Provider for OpenAI-compatible APIs.
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
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *Provider) Name() string         { return "openai" }
func (p *Provider) CurrentModel() string { return p.model }

// --- native types ---

type oaiMsg struct {
	Role       string        `json:"role"`
	Content    interface{}   `json:"content"` // string or []oaiContent
	ToolCallID string        `json:"tool_call_id,omitempty"`
	ToolCalls  []oaiToolCall `json:"tool_calls,omitempty"`
}

type oaiContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type oaiToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function oaiFunctionCall `json:"function"`
}

type oaiFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type oaiTool struct {
	Type     string          `json:"type"`
	Function oaiToolFunction `json:"function"`
}

type oaiToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type requestBody struct {
	Model     string    `json:"model"`
	Messages  []oaiMsg  `json:"messages"`
	Tools     []oaiTool `json:"tools,omitempty"`
	Stream    bool      `json:"stream"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

func convertMessages(msgs []session.Message) []oaiMsg {
	var result []oaiMsg
	for _, m := range msgs {
		om := oaiMsg{Role: m.Role}
		var texts []string
		var toolCalls []oaiToolCall
		var toolResultID, toolResultText string
		isToolResult := false

		for _, c := range m.Content {
			switch c.Type {
			case "text":
				texts = append(texts, c.Text)
			case "tool_use":
				inp, _ := json.Marshal(c.Input)
				toolCalls = append(toolCalls, oaiToolCall{
					ID:       c.ID,
					Type:     "function",
					Function: oaiFunctionCall{Name: c.Name, Arguments: string(inp)},
				})
			case "tool_result":
				isToolResult = true
				toolResultID = c.ToolUseID
				toolResultText = c.Text
			}
		}

		if isToolResult {
			om.Role = "tool"
			om.ToolCallID = toolResultID
			om.Content = toolResultText
			result = append(result, om)
			continue
		}
		if len(toolCalls) > 0 {
			om.ToolCalls = toolCalls
		}
		if len(texts) > 0 {
			om.Content = strings.Join(texts, "\n")
		} else if len(toolCalls) == 0 {
			continue
		}
		result = append(result, om)
	}
	return result
}

func convertTools(tools []llm.ToolSchema) []oaiTool {
	var result []oaiTool
	for _, t := range tools {
		result = append(result, oaiTool{
			Type: "function",
			Function: oaiToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return result
}

func (p *Provider) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamEvent, error) {
	msgs, ok := req.RawMsgs.([]session.Message)
	if !ok {
		return nil, fmt.Errorf("openai provider expects []session.Message")
	}

	// Inject system message if present
	native := convertMessages(msgs)
	for _, m := range msgs {
		if m.Role == "system" {
			for _, c := range m.Content {
				sysMsg := oaiMsg{Role: "system", Content: c.Text}
				native = append([]oaiMsg{sysMsg}, native...)
			}
			break
		}
	}

	body := requestBody{
		Model:     p.model,
		Messages:  native,
		Tools:     convertTools(req.Tools),
		Stream:    true,
		MaxTokens: req.MaxTokens,
	}
	if len(body.Tools) == 0 {
		body.Tools = nil
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(b))
	}

	ch := make(chan llm.StreamEvent, 64)
	go readStream(resp.Body, ch, msgs)
	return ch, nil
}

type sseChunk struct {
	Choices []struct {
		Delta struct {
			Content   string        `json:"content"`
			ToolCalls []oaiToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// estimateTokens estimates token count from text content
// Uses a simple heuristic: ~4 chars per token for English, ~1.5 chars per token for CJK
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	charCount := 0
	cjkCount := 0

	for _, r := range text {
		charCount++
		// Check if character is CJK (Chinese, Japanese, Korean)
		if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
			(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
			(r >= 0x2A700 && r <= 0x2B73F) || // CJK Extension C
			(r >= 0x2B740 && r <= 0x2B81F) || // CJK Extension D
			(r >= 0x2B820 && r <= 0x2CEAF) || // CJK Extension E
			(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
			(r >= 0x2F800 && r <= 0x2FA1F) { // CJK Compatibility Ideographs Supplement
			cjkCount++
		}
	}

	// Estimate: CJK chars ~1.5 chars/token, others ~4 chars/token
	if cjkCount > charCount/2 {
		// Mostly CJK text
		return int(float64(charCount) / 1.5)
	}
	// Mostly non-CJK text
	return charCount / 4
}

// estimateInputTokens estimates input tokens from messages
func estimateInputTokens(msgs []session.Message) int {
	total := 0
	for _, msg := range msgs {
		for _, c := range msg.Content {
			switch c.Type {
			case "text":
				total += estimateTokens(c.Text)
			case "tool_use":
				// Estimate tool use as JSON string
				if c.Input != nil {
					// Convert input to JSON string for estimation
					if inputBytes, err := json.Marshal(c.Input); err == nil {
						total += estimateTokens(string(inputBytes))
					}
				}
				total += estimateTokens(c.Name) + 10 // overhead for tool structure
			case "tool_result":
				total += estimateTokens(c.Text) + 5 // overhead for result structure
			}
		}
	}
	// Add overhead for message structure (role, etc.)
	total += len(msgs) * 4
	return total
}

func readStream(body io.ReadCloser, ch chan<- llm.StreamEvent, msgs []session.Message) {
	defer body.Close()
	defer close(ch)

	type toolAccum struct {
		id   string
		name string
		args strings.Builder
	}
	tools := map[int]*toolAccum{}

	var outputText strings.Builder
	var receivedUsage bool

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 512*1024), 512*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk sseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			logger.Debug("OpenAI SSE parse error: %v", err)
			continue
		}

		if chunk.Usage != nil {
			receivedUsage = true
			ch <- llm.StreamEvent{
				Type:   "usage",
				Input:  chunk.Usage.PromptTokens,
				Output: chunk.Usage.CompletionTokens,
			}
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				ch <- llm.StreamEvent{Type: "text_delta", Delta: choice.Delta.Content}
				outputText.WriteString(choice.Delta.Content)
			}
			for _, tc := range choice.Delta.ToolCalls {
				idx := 0 // OpenAI uses index field
				if _, ok := tools[idx]; !ok {
					tools[idx] = &toolAccum{}
				}
				t := tools[idx]
				if tc.ID != "" {
					t.id = tc.ID
				}
				if tc.Function.Name != "" {
					t.name = tc.Function.Name
				}
				t.args.WriteString(tc.Function.Arguments)
			}
			if choice.FinishReason == "tool_calls" {
				for _, t := range tools {
					raw := json.RawMessage(t.args.String())
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
					// Add tool call to output for token estimation
					outputText.WriteString(t.name)
					outputText.WriteString(t.args.String())
				}
				tools = map[int]*toolAccum{}
			}
			if choice.FinishReason == "stop" {
				ch <- llm.StreamEvent{Type: "done"}
			}
		}
	}

	// If no usage was received, estimate tokens
	if !receivedUsage {
		inputTokens := estimateInputTokens(msgs)
		outputTokens := estimateTokens(outputText.String())

		logger.Debug("Estimated tokens (usage not provided by LLM): in=%d out=%d", inputTokens, outputTokens)

		ch <- llm.StreamEvent{
			Type:   "usage",
			Input:  inputTokens,
			Output: outputTokens,
		}
	}
}
