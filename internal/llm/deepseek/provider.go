package deepseek

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

// Provider implements the LLM provider interface for DeepSeek API.
// DeepSeek R1 uses OpenAI-compatible API but has special handling for:
// 1. Reasoning tokens wrapped in <think>...</think> tags
// 2. Token usage may not be provided in streaming responses
type Provider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// New creates a new DeepSeek provider.
func New(apiKey, baseURL, model string) *Provider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	if model == "" {
		model = "deepseek-reasoner"
	}
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 3000 * time.Second},
	}
}

// Stream sends a streaming request to DeepSeek API.
func (p *Provider) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamEvent, error) {
	// Type assert messages from RawMsgs
	messages, ok := req.RawMsgs.([]session.Message)
	if !ok {
		return nil, fmt.Errorf("deepseek provider expects []session.Message")
	}

	apiMessages := convertMessages(messages)

	payload := map[string]interface{}{
		"model":       p.model,
		"messages":    apiMessages,
		"stream":      true,
		"max_tokens":  req.MaxTokens,
		"temperature": 1.0,
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		payload["tools"] = convertTools(req.Tools)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	logger.Debug("DeepSeek request to: %s", p.baseURL+"/v1/chat/completions")
	logger.Debug("DeepSeek model: %s", p.model)

	httpReq.Header.Set("Content-Type", "application/json")
	// Only set Authorization header if API key is provided (not needed for local deployments)
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		logger.Debug("DeepSeek: using API key authentication")
	} else {
		logger.Debug("DeepSeek: no API key (local deployment mode)")
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		logger.Debug("DeepSeek API error response: %s", string(body))
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan llm.StreamEvent, 10)

	go p.readStream(resp.Body, ch, messages)

	return ch, nil
}

// readStream reads SSE stream from DeepSeek API.
func (p *Provider) readStream(body io.ReadCloser, ch chan<- llm.StreamEvent, messages []session.Message) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 512*1024), 512*1024)

	var textBuffer strings.Builder
	var toolCalls []llm.ToolUseBlock
	var inThinking bool
	var thinkingBuffer strings.Builder
	var usageReceived bool

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			logger.Debug("parse chunk error: %v", err)
			continue
		}

		// Handle usage if provided
		if chunk.Usage != nil {
			usageReceived = true
			ch <- llm.StreamEvent{
				Type:   "usage",
				Input:  chunk.Usage.PromptTokens,
				Output: chunk.Usage.CompletionTokens,
			}
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]

		// Handle text content with thinking detection
		if choice.Delta.Content != "" {
			content := choice.Delta.Content

			// Check for thinking tags
			if strings.Contains(content, "<think>") {
				inThinking = true
				// Find where <think> starts and split content
				idx := strings.Index(content, "<think>")
				if idx > 0 {
					// Emit content before <think>
					beforeThink := content[:idx]
					textBuffer.WriteString(beforeThink)
					ch <- llm.StreamEvent{
						Type:  "text_delta",
						Delta: beforeThink,
					}
				}
				// Start collecting thinking content (skip the <think> tag itself)
				afterTag := content[idx+7:] // len("<think>") = 7

				// Check if </think> is also in this chunk
				if strings.Contains(afterTag, "</think>") {
					endIdx := strings.Index(afterTag, "</think>")
					thinkingContent := afterTag[:endIdx]
					thinkingBuffer.WriteString(thinkingContent)

					// Emit thinking_delta for the content
					if len(thinkingContent) > 0 {
						ch <- llm.StreamEvent{
							Type:  "thinking_delta",
							Delta: thinkingContent,
						}
					}

					// Immediately end thinking
					inThinking = false
					ch <- llm.StreamEvent{
						Type:  "thinking_done",
						Delta: thinkingBuffer.String(),
					}
					thinkingBuffer.Reset()

					// Process content after </think>
					afterThink := afterTag[endIdx+8:] // len("</think>") = 8
					if len(afterThink) > 0 {
						textBuffer.WriteString(afterThink)
						ch <- llm.StreamEvent{
							Type:  "text_delta",
							Delta: afterThink,
						}
					}
				} else {
					// No </think> yet, just accumulate
					thinkingBuffer.WriteString(afterTag)
					if len(afterTag) > 0 {
						ch <- llm.StreamEvent{
							Type:  "thinking_delta",
							Delta: afterTag,
						}
					}
				}
				continue
			}

			if inThinking {
				// Check if this chunk contains </think>
				if strings.Contains(content, "</think>") {
					idx := strings.Index(content, "</think>")
					// Add content before </think> to thinking buffer
					beforeEnd := content[:idx]
					if len(beforeEnd) > 0 {
						thinkingBuffer.WriteString(beforeEnd)
						ch <- llm.StreamEvent{
							Type:  "thinking_delta",
							Delta: beforeEnd,
						}
					}

					// End thinking immediately
					inThinking = false
					ch <- llm.StreamEvent{
						Type:  "thinking_done",
						Delta: thinkingBuffer.String(),
					}
					thinkingBuffer.Reset()

					// Process content after </think>
					afterThink := content[idx+8:] // len("</think>") = 8
					if len(afterThink) > 0 {
						textBuffer.WriteString(afterThink)
						ch <- llm.StreamEvent{
							Type:  "text_delta",
							Delta: afterThink,
						}
					}
				} else {
					// Still thinking, accumulate
					thinkingBuffer.WriteString(content)
					ch <- llm.StreamEvent{
						Type:  "thinking_delta",
						Delta: content,
					}
				}
			} else {
				// Normal content, not thinking
				textBuffer.WriteString(content)
				ch <- llm.StreamEvent{
					Type:  "text_delta",
					Delta: content,
				}
			}
		}

		// Handle tool calls
		if len(choice.Delta.ToolCalls) > 0 {
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Index >= len(toolCalls) {
					// Initialize new tool call
					inputJSON, _ := json.Marshal(map[string]string{"_args": ""})
					toolCalls = append(toolCalls, llm.ToolUseBlock{
						ID:    tc.ID,
						Name:  tc.Function.Name,
						Input: inputJSON,
					})
				}
				if tc.Function.Arguments != "" {
					// Accumulate arguments as JSON string
					var currentInput map[string]string
					json.Unmarshal(toolCalls[tc.Index].Input, &currentInput)
					if currentInput == nil {
						currentInput = map[string]string{}
					}
					currentInput["_args"] += tc.Function.Arguments
					toolCalls[tc.Index].Input, _ = json.Marshal(currentInput)
				}
			}
		}

		// Handle finish
		if choice.FinishReason == "tool_calls" {
			// Parse accumulated tool arguments
			for i := range toolCalls {
				var tempInput map[string]string
				json.Unmarshal(toolCalls[i].Input, &tempInput)
				if argsStr, ok := tempInput["_args"]; ok && argsStr != "" {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(argsStr), &args); err == nil {
						toolCalls[i].Input, _ = json.Marshal(args)
					}
				}
				ch <- llm.StreamEvent{
					Type:    "tool_use_end",
					ToolUse: &toolCalls[i],
				}
			}
		} else if choice.FinishReason == "stop" || choice.FinishReason == "length" {
			ch <- llm.StreamEvent{Type: "done"}
		}
	}

	// If no usage received, estimate tokens client-side
	if !usageReceived {
		inputTokens := estimateInputTokens(messages)
		outputTokens := estimateTokens(textBuffer.String())
		logger.Debug("DeepSeek: no usage from API, estimating: in=%d out=%d", inputTokens, outputTokens)
		ch <- llm.StreamEvent{
			Type:   "usage",
			Input:  inputTokens,
			Output: outputTokens,
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- llm.StreamEvent{
			Type: "error",
			Err:  fmt.Errorf("stream read error: %w", err),
		}
	}
}

// estimateTokens estimates token count for text.
// Uses similar logic to OpenAI provider.
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	cjkCount := 0
	totalRunes := 0

	for _, r := range text {
		totalRunes++
		// Check if character is CJK
		if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
			(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
			(r >= 0x2A700 && r <= 0x2B73F) || // CJK Extension C
			(r >= 0x2B740 && r <= 0x2B81F) || // CJK Extension D
			(r >= 0x2B820 && r <= 0x2CEAF) || // CJK Extension E
			(r >= 0x3000 && r <= 0x303F) || // CJK Symbols
			(r >= 0xFF00 && r <= 0xFFEF) { // Fullwidth Forms
			cjkCount++
		}
	}

	cjkRatio := float64(cjkCount) / float64(totalRunes)

	if cjkRatio > 0.5 {
		// Mostly CJK: ~1.5 chars per token
		return int(float64(totalRunes) / 1.5)
	} else {
		// Mostly English: ~4 chars per token
		return len(text) / 4
	}
}

// estimateInputTokens estimates input token count.
func estimateInputTokens(messages []session.Message) int {
	total := 0
	for _, msg := range messages {
		// Add 4 tokens for message structure
		total += 4

		for _, content := range msg.Content {
			if content.Type == "text" {
				total += estimateTokens(content.Text)
			} else if content.Type == "tool_use" {
				// Estimate tool_use as JSON
				data, _ := json.Marshal(content)
				total += estimateTokens(string(data))
			} else if content.Type == "tool_result" {
				total += estimateTokens(content.Text)
			}
		}
	}
	return total
}

// convertMessages converts session messages to DeepSeek API format (OpenAI-compatible).
func convertMessages(messages []session.Message) []map[string]interface{} {
	var result []map[string]interface{}

	for _, msg := range messages {
		// Handle different message types
		if msg.Role == "user" {
			// Check if this is a tool result message
			hasToolResult := false
			for _, c := range msg.Content {
				if c.Type == "tool_result" {
					hasToolResult = true
					break
				}
			}

			if hasToolResult {
				// Convert tool results to OpenAI format (role: "tool")
				for _, c := range msg.Content {
					if c.Type == "tool_result" {
						result = append(result, map[string]interface{}{
							"role":         "tool",
							"tool_call_id": c.ToolUseID,
							"content":      c.Text,
						})
					}
				}
			} else {
				// Regular user message
				var textParts []string
				for _, c := range msg.Content {
					if c.Type == "text" {
						textParts = append(textParts, c.Text)
					}
				}
				if len(textParts) > 0 {
					result = append(result, map[string]interface{}{
						"role":    "user",
						"content": strings.Join(textParts, "\n"),
					})
				}
			}
		} else if msg.Role == "assistant" {
			// Check if this message has tool calls
			var toolCalls []map[string]interface{}
			var textContent string

			for _, c := range msg.Content {
				if c.Type == "text" {
					textContent = c.Text
				} else if c.Type == "tool_use" {
					// Convert to OpenAI tool_calls format
					argsJSON, _ := json.Marshal(c.Input)
					toolCalls = append(toolCalls, map[string]interface{}{
						"id":   c.ID,
						"type": "function",
						"function": map[string]interface{}{
							"name":      c.Name,
							"arguments": string(argsJSON),
						},
					})
				}
			}

			m := map[string]interface{}{
				"role": "assistant",
			}

			if textContent != "" {
				m["content"] = textContent
			} else {
				m["content"] = "" // OpenAI requires content field
			}

			if len(toolCalls) > 0 {
				m["tool_calls"] = toolCalls
			}

			result = append(result, m)
		} else {
			// System or other roles - simple text
			var textParts []string
			for _, c := range msg.Content {
				if c.Type == "text" {
					textParts = append(textParts, c.Text)
				}
			}
			if len(textParts) > 0 {
				result = append(result, map[string]interface{}{
					"role":    msg.Role,
					"content": strings.Join(textParts, "\n"),
				})
			}
		}
	}

	return result
}

// convertTools converts session tools to DeepSeek API format.
func convertTools(tools []llm.ToolSchema) []map[string]interface{} {
	var result []map[string]interface{}

	for _, tool := range tools {
		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			},
		})
	}

	return result
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "deepseek"
}

// CurrentModel returns the current model name.
func (p *Provider) CurrentModel() string {
	return p.model
}
