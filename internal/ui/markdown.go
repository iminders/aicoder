package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// MarkdownRenderer handles rendering markdown with syntax highlighting.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	width    int
}

// NewMarkdownRenderer creates a new markdown renderer.
func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	if width <= 0 {
		width = 80
	}

	// Create glamour renderer with dark style
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	return &MarkdownRenderer{
		renderer: renderer,
		width:    width,
	}, nil
}

// Render renders markdown text to a styled string.
func (m *MarkdownRenderer) Render(markdown string) (string, error) {
	if markdown == "" {
		return "", nil
	}

	out, err := m.renderer.Render(markdown)
	if err != nil {
		return markdown, err // Fallback to plain text
	}

	return strings.TrimSpace(out), nil
}

// RenderPlain renders markdown without formatting (useful for streaming).
func (m *MarkdownRenderer) RenderPlain(text string) string {
	return text
}

// SetWidth updates the rendering width.
func (m *MarkdownRenderer) SetWidth(width int) {
	m.width = width
	// Note: glamour doesn't support dynamic width changes,
	// so we'd need to recreate the renderer
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err == nil {
		m.renderer = renderer
	}
}

// RenderCodeBlock renders a code block with syntax highlighting.
func (m *MarkdownRenderer) RenderCodeBlock(code, lang string) (string, error) {
	markdown := "```" + lang + "\n" + code + "\n```"
	return m.Render(markdown)
}

// RenderInline renders inline markdown elements (bold, italic, code, links).
func (m *MarkdownRenderer) RenderInline(text string) (string, error) {
	// For inline rendering, we use glamour's rendering
	return m.Render(text)
}
