package ui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Spinner represents a loading spinner animation.
type Spinner struct {
	frames   []string
	current  int
	interval time.Duration
	style    lipgloss.Style
}

// NewSpinner creates a new spinner with default settings.
func NewSpinner() *Spinner {
	return &Spinner{
		frames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
		current:  0,
		interval: 80 * time.Millisecond,
		style:    GetTheme().SpinnerStyle,
	}
}

// NewDotsSpinner creates a spinner with dots animation.
func NewDotsSpinner() *Spinner {
	return &Spinner{
		frames: []string{
			"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷",
		},
		current:  0,
		interval: 80 * time.Millisecond,
		style:    GetTheme().SpinnerStyle,
	}
}

// NewLineSpinner creates a spinner with line animation.
func NewLineSpinner() *Spinner {
	return &Spinner{
		frames: []string{
			"-", "\\", "|", "/",
		},
		current:  0,
		interval: 100 * time.Millisecond,
		style:    GetTheme().SpinnerStyle,
	}
}

// Tick advances the spinner to the next frame.
func (s *Spinner) Tick() {
	s.current = (s.current + 1) % len(s.frames)
}

// View returns the current frame as a styled string.
func (s *Spinner) View() string {
	return s.style.Render(s.frames[s.current])
}

// Interval returns the animation interval.
func (s *Spinner) Interval() time.Duration {
	return s.interval
}

// SetStyle sets the spinner style.
func (s *Spinner) SetStyle(style lipgloss.Style) {
	s.style = style
}
