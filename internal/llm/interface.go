package llm

import (
	"context"
	"encoding/json"
)

// ToolSchema describes one tool in the format sent to the LLM.
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// StreamEvent is emitted by Provider.Stream for each piece of the response.
type StreamEvent struct {
	Type    string // text_delta | tool_use_start | tool_use_delta | tool_use_end | thinking_delta | thinking_done | usage | done | error
	Delta   string
	ToolUse *ToolUseBlock
	Input   int // input token count (on usage event)
	Output  int // output token count (on usage event)
	Err     error
}

// ToolUseBlock carries a tool invocation from the model.
type ToolUseBlock struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// Provider is the abstraction over any LLM backend.
type Provider interface {
	// Stream sends messages and returns a channel of events.
	Stream(ctx context.Context, req *Request) (<-chan StreamEvent, error)
	Name() string
	CurrentModel() string
}

// Request bundles everything needed for one LLM call.
type Request struct {
	Model     string
	Messages  interface{} // provider-specific slice, built by each adapter
	RawMsgs   interface{} // pre-built native messages (used internally)
	Tools     []ToolSchema
	System    string
	MaxTokens int
}
