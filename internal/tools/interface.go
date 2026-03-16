package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// RiskLevel indicates how dangerous a tool invocation is.
type RiskLevel int

const (
	RiskLow    RiskLevel = iota // read-only operations
	RiskMedium                  // file writes, safe shell commands
	RiskHigh                    // deletes, system-level commands
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "低"
	case RiskMedium:
		return "中"
	case RiskHigh:
		return "高"
	}
	return "未知"
}

// Result is the structured output of a tool execution.
type Result struct {
	Content  string
	IsError  bool
	Metadata map[string]any
}

// Tool is the interface every built-in and MCP tool must implement.
type Tool interface {
	Name() string
	Description() string
	// Schema returns a JSON Schema object for the tool's input.
	Schema() json.RawMessage
	// Execute runs the tool and returns a result.
	Execute(ctx context.Context, input json.RawMessage) (*Result, error)
	// Risk returns the risk level of this tool.
	Risk() RiskLevel
}

// Registry holds all registered tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// Global is the default tool registry.
var Global = &Registry{tools: map[string]Tool{}}

func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %q", name)
	}
	return t, nil
}

func (r *Registry) All() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}
