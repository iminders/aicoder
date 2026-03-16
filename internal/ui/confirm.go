package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmModel represents a yes/no confirmation dialog.
type ConfirmModel struct {
	message  string
	selected int // 0 = Yes, 1 = No
	result   *bool
	done     bool
}

// NewConfirm creates a new confirmation dialog.
func NewConfirm(message string) *ConfirmModel {
	return &ConfirmModel{
		message:  message,
		selected: 0,
		done:     false,
	}
}

// Update handles confirmation events.
func (c *ConfirmModel) Update(msg tea.Msg) (*ConfirmModel, tea.Cmd) {
	if c.done {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			c.selected = 0
		case "right", "l":
			c.selected = 1
		case "y", "Y":
			c.selected = 0
			c.done = true
			result := true
			c.result = &result
			return c, tea.Quit
		case "n", "N":
			c.selected = 1
			c.done = true
			result := false
			c.result = &result
			return c, tea.Quit
		case "enter":
			c.done = true
			result := c.selected == 0
			c.result = &result
			return c, tea.Quit
		case "ctrl+c", "esc":
			c.done = true
			result := false
			c.result = &result
			return c, tea.Quit
		}
	}

	return c, nil
}

// View renders the confirmation dialog.
func (c *ConfirmModel) View() string {
	if c.done {
		return ""
	}

	theme := GetTheme()
	var b strings.Builder

	// Message
	b.WriteString(theme.WarningStyle.Render(c.message))
	b.WriteString("\n\n")

	// Options
	yesStyle := theme.SubtitleStyle
	noStyle := theme.SubtitleStyle

	if c.selected == 0 {
		yesStyle = theme.SuccessStyle.Bold(true)
	} else {
		noStyle = theme.ErrorStyle.Bold(true)
	}

	b.WriteString("  ")
	b.WriteString(yesStyle.Render("[Y] Yes"))
	b.WriteString("  ")
	b.WriteString(noStyle.Render("[N] No"))
	b.WriteString("\n\n")
	b.WriteString(theme.SubtitleStyle.Render("Use arrow keys or y/n to select, Enter to confirm"))

	return b.String()
}

// Result returns the confirmation result (nil if not done).
func (c *ConfirmModel) Result() *bool {
	return c.result
}

// IsDone returns whether the user has made a choice.
func (c *ConfirmModel) IsDone() bool {
	return c.done
}

// ShowConfirm displays a confirmation dialog and returns the result.
func ShowConfirm(message string) bool {
	model := NewConfirm(message)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return false
	}

	if m, ok := finalModel.(*ConfirmModel); ok {
		if m.result != nil {
			return *m.result
		}
	}

	return false
}
