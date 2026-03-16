package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/iminders/aicoder/internal/tools"
)

type ReadFileTool struct{}

func (t *ReadFileTool) Name() string          { return "read_file" }
func (t *ReadFileTool) Risk() tools.RiskLevel { return tools.RiskLow }
func (t *ReadFileTool) Description() string {
	return "Read the contents of a file. Optionally specify start_line and end_line (1-based) to read a range."
}
func (t *ReadFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path":       {"type": "string",  "description": "File path to read"},
			"start_line": {"type": "integer", "description": "First line to read (1-based, optional)"},
			"end_line":   {"type": "integer", "description": "Last line to read (inclusive, optional)"}
		},
		"required": ["path"]
	}`)
}

type readInput struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

func (t *ReadFileTool) Execute(_ context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in readInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: "invalid input: " + err.Error()}, nil
	}
	if err := checkSandbox(in.Path); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	data, err := os.ReadFile(in.Path)
	if err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	content := string(data)
	if in.StartLine > 0 || in.EndLine > 0 {
		lines := strings.Split(content, "\n")
		s := in.StartLine - 1
		if s < 0 {
			s = 0
		}
		e := in.EndLine
		if e <= 0 || e > len(lines) {
			e = len(lines)
		}
		if s >= len(lines) {
			return &tools.Result{IsError: true, Content: fmt.Sprintf("start_line %d exceeds file length %d", in.StartLine, len(lines))}, nil
		}
		// Add line numbers
		var sb strings.Builder
		for i := s; i < e; i++ {
			sb.WriteString(fmt.Sprintf("%4d\t%s\n", i+1, lines[i]))
		}
		content = sb.String()
	}
	return &tools.Result{Content: content}, nil
}
