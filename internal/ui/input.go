package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputModel represents a text input component.
type InputModel struct {
	prompt      string
	value       string
	placeholder string
	cursor      int
	width       int
	focused     bool
}

// NewInput creates a new input model.
func NewInput(prompt string) *InputModel {
	return &InputModel{
		prompt:  prompt,
		value:   "",
		cursor:  0,
		width:   80,
		focused: true,
	}
}

// SetPlaceholder sets the placeholder text.
func (i *InputModel) SetPlaceholder(text string) {
	i.placeholder = text
}

// SetWidth sets the input width.
func (i *InputModel) SetWidth(width int) {
	i.width = width
}

// Focus focuses the input.
func (i *InputModel) Focus() {
	i.focused = true
}

// Blur removes focus from the input.
func (i *InputModel) Blur() {
	i.focused = false
}

// Value returns the current input value.
func (i *InputModel) Value() string {
	return i.value
}

// SetValue sets the input value.
func (i *InputModel) SetValue(value string) {
	i.value = value
	i.cursor = len(value)
}

// Reset clears the input.
func (i *InputModel) Reset() {
	i.value = ""
	i.cursor = 0
}

// Update handles input events.
func (i *InputModel) Update(msg tea.Msg) (*InputModel, tea.Cmd) {
	if !i.focused {
		return i, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyBackspace:
			if i.cursor > 0 {
				i.value = i.value[:i.cursor-1] + i.value[i.cursor:]
				i.cursor--
			}
		case tea.KeyDelete:
			if i.cursor < len(i.value) {
				i.value = i.value[:i.cursor] + i.value[i.cursor+1:]
			}
		case tea.KeyLeft:
			if i.cursor > 0 {
				i.cursor--
			}
		case tea.KeyRight:
			if i.cursor < len(i.value) {
				i.cursor++
			}
		case tea.KeyHome:
			i.cursor = 0
		case tea.KeyEnd:
			i.cursor = len(i.value)
		case tea.KeyRunes:
			i.value = i.value[:i.cursor] + string(msg.Runes) + i.value[i.cursor:]
			i.cursor += len(msg.Runes)
		}
	}

	return i, nil
}

// View renders the input.
func (i *InputModel) View() string {
	theme := GetTheme()
	var b strings.Builder

	// Prompt
	if i.prompt != "" {
		b.WriteString(theme.PromptStyle.Render(i.prompt))
		b.WriteString(" ")
	}

	// Value or placeholder
	if i.value == "" && i.placeholder != "" {
		b.WriteString(theme.SubtitleStyle.Render(i.placeholder))
	} else {
		// Show value with cursor
		before := i.value[:i.cursor]
		after := ""
		if i.cursor < len(i.value) {
			after = i.value[i.cursor:]
		}

		b.WriteString(before)
		if i.focused {
			b.WriteString(lipgloss.NewStyle().Foreground(theme.Highlight).Render("_"))
		}
		b.WriteString(after)
	}

	return b.String()
}
