package openai

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/iminders/aicoder/internal/llm"
	"github.com/iminders/aicoder/internal/session"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		minToken int
		maxToken int
	}{
		{
			name:     "empty string",
			text:     "",
			minToken: 0,
			maxToken: 0,
		},
		{
			name:     "english text",
			text:     "Hello, world!",
			minToken: 2,
			maxToken: 5,
		},
		{
			name:     "chinese text",
			text:     "你好，世界！",
			minToken: 3,
			maxToken: 6,
		},
		{
			name:     "mixed text",
			text:     "Hello 你好 world 世界",
			minToken: 4,
			maxToken: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := estimateTokens(tt.text)
			if tokens < tt.minToken || tokens > tt.maxToken {
				t.Errorf("estimateTokens(%q) = %d, want between %d and %d",
					tt.text, tokens, tt.minToken, tt.maxToken)
			}
		})
	}
}

func TestEstimateInputTokens(t *testing.T) {
	msgs := []session.Message{
		{
			Role: "user",
			Content: []session.Content{
				{Type: "text", Text: "Hello, how are you?"},
			},
		},
		{
			Role: "assistant",
			Content: []session.Content{
				{Type: "text", Text: "I'm doing well, thank you!"},
			},
		},
		{
			Role: "user",
			Content: []session.Content{
				{
					Type:  "tool_use",
					ID:    "tool1",
					Name:  "get_weather",
					Input: json.RawMessage(`{"city":"Beijing"}`),
				},
			},
		},
	}

	tokens := estimateInputTokens(msgs)
	// Should be > 0 and reasonable
	if tokens < 10 || tokens > 100 {
		t.Errorf("estimateInputTokens() = %d, want between 10 and 100", tokens)
	}
}

// TestReadStreamWithoutUsage tests that token estimation works when usage is null
func TestReadStreamWithoutUsage(t *testing.T) {
	// Simulate SSE stream without usage field
	sseData := `data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":""}]}
data: {"choices":[{"delta":{"content":" world"},"finish_reason":""}]}
data: {"choices":[{"delta":{"content":"!"},"finish_reason":"stop"}]}
data: [DONE]
`

	body := io.NopCloser(strings.NewReader(sseData))
	ch := make(chan llm.StreamEvent, 64)

	msgs := []session.Message{
		{
			Role: "user",
			Content: []session.Content{
				{Type: "text", Text: "Say hello"},
			},
		},
	}

	go readStream(body, ch, msgs)

	var events []llm.StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	// Check that we received text deltas
	textCount := 0
	usageCount := 0
	for _, e := range events {
		if e.Type == "text_delta" {
			textCount++
		}
		if e.Type == "usage" {
			usageCount++
		}
	}

	if textCount != 3 {
		t.Errorf("Expected 3 text_delta events, got %d", textCount)
	}

	// Should have exactly 1 usage event (estimated)
	if usageCount != 1 {
		t.Errorf("Expected 1 usage event (estimated), got %d", usageCount)
	}

	// Check that usage event has reasonable values
	for _, e := range events {
		if e.Type == "usage" {
			if e.Input <= 0 {
				t.Errorf("Expected positive input tokens, got %d", e.Input)
			}
			if e.Output <= 0 {
				t.Errorf("Expected positive output tokens, got %d", e.Output)
			}
		}
	}
}

// TestReadStreamWithUsage tests that provided usage is used when available
func TestReadStreamWithUsage(t *testing.T) {
	// Simulate SSE stream with usage field
	sseData := `data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":""}]}
data: {"choices":[{"delta":{"content":" world"},"finish_reason":""}]}
data: {"choices":[{"delta":{"content":"!"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":3}}
data: [DONE]
`

	body := io.NopCloser(strings.NewReader(sseData))
	ch := make(chan llm.StreamEvent, 64)

	msgs := []session.Message{
		{
			Role: "user",
			Content: []session.Content{
				{Type: "text", Text: "Say hello"},
			},
		},
	}

	go readStream(body, ch, msgs)

	var events []llm.StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	// Should have exactly 1 usage event (from LLM)
	usageCount := 0
	var usageEvent llm.StreamEvent
	for _, e := range events {
		if e.Type == "usage" {
			usageCount++
			usageEvent = e
		}
	}

	if usageCount != 1 {
		t.Errorf("Expected 1 usage event, got %d", usageCount)
	}

	// Check that usage values match what was provided
	if usageEvent.Input != 10 {
		t.Errorf("Expected input tokens = 10, got %d", usageEvent.Input)
	}
	if usageEvent.Output != 3 {
		t.Errorf("Expected output tokens = 3, got %d", usageEvent.Output)
	}
}

