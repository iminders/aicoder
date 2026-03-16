package ui

import (
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// OutputModel handles streaming output display.
type OutputModel struct {
	content strings.Builder
	mu      sync.Mutex
	width   int
	height  int
}

// NewOutput creates a new output model.
func NewOutput() *OutputModel {
	return &OutputModel{
		width:  80,
		height: 24,
	}
}

// Write appends content to the output (thread-safe).
func (o *OutputModel) Write(data string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.content.WriteString(data)
}

// Clear clears the output.
func (o *OutputModel) Clear() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.content.Reset()
}

// Content returns the current output content.
func (o *OutputModel) Content() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.content.String()
}

// SetSize sets the output dimensions.
func (o *OutputModel) SetSize(width, height int) {
	o.width = width
	o.height = height
}

// View renders the output.
func (o *OutputModel) View() string {
	o.mu.Lock()
	defer o.mu.Unlock()

	content := o.content.String()
	if content == "" {
		return ""
	}

	// Split into lines and handle wrapping
	lines := strings.Split(content, "\n")
	var rendered []string

	for _, line := range lines {
		if len(line) <= o.width {
			rendered = append(rendered, line)
		} else {
			// Wrap long lines
			for len(line) > o.width {
				rendered = append(rendered, line[:o.width])
				line = line[o.width:]
			}
			if len(line) > 0 {
				rendered = append(rendered, line)
			}
		}
	}

	// Limit to height
	if len(rendered) > o.height {
		rendered = rendered[len(rendered)-o.height:]
	}

	return strings.Join(rendered, "\n")
}

// StreamWriter is an io.Writer that sends chunks to a bubbletea program.
type StreamWriter struct {
	program *tea.Program
}

// NewStreamWriter creates a new stream writer.
func NewStreamWriter(program *tea.Program) *StreamWriter {
	return &StreamWriter{
		program: program,
	}
}

// Write implements io.Writer.
func (w *StreamWriter) Write(p []byte) (n int, err error) {
	if w.program != nil {
		w.program.Send(streamChunkMsg(string(p)))
	}
	return len(p), nil
}
