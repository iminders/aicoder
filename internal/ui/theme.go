package ui

import "github.com/charmbracelet/lipgloss"

// Theme defines color tokens and styles for the TUI.
type Theme struct {
	// Base colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Info      lipgloss.Color
	Muted     lipgloss.Color

	// UI element colors
	Border    lipgloss.Color
	Highlight lipgloss.Color
	Dim       lipgloss.Color

	// Styles
	TitleStyle     lipgloss.Style
	SubtitleStyle  lipgloss.Style
	ErrorStyle     lipgloss.Style
	SuccessStyle   lipgloss.Style
	WarningStyle   lipgloss.Style
	InfoStyle      lipgloss.Style
	PromptStyle    lipgloss.Style
	DividerStyle   lipgloss.Style
	StatusBarStyle lipgloss.Style
	SpinnerStyle   lipgloss.Style
	CodeBlockStyle lipgloss.Style
	KeywordStyle   lipgloss.Style
}

// DefaultTheme returns the default theme with carefully chosen colors.
func DefaultTheme() *Theme {
	t := &Theme{
		// Base colors (using ANSI codes for compatibility)
		Primary:   lipgloss.Color("12"),  // Bright Blue
		Secondary: lipgloss.Color("14"),  // Bright Cyan
		Success:   lipgloss.Color("10"),  // Bright Green
		Warning:   lipgloss.Color("11"),  // Bright Yellow
		Error:     lipgloss.Color("9"),   // Bright Red
		Info:      lipgloss.Color("14"),  // Bright Cyan
		Muted:     lipgloss.Color("8"),   // Gray
		Border:    lipgloss.Color("240"), // Dark Gray
		Highlight: lipgloss.Color("13"),  // Bright Magenta
		Dim:       lipgloss.Color("243"), // Medium Gray
	}

	// Build styles
	t.TitleStyle = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1)

	t.SubtitleStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Italic(true)

	t.ErrorStyle = lipgloss.NewStyle().
		Foreground(t.Error).
		Bold(true)

	t.SuccessStyle = lipgloss.NewStyle().
		Foreground(t.Success).
		Bold(true)

	t.WarningStyle = lipgloss.NewStyle().
		Foreground(t.Warning).
		Bold(true)

	t.InfoStyle = lipgloss.NewStyle().
		Foreground(t.Info)

	t.PromptStyle = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	t.DividerStyle = lipgloss.NewStyle().
		Foreground(t.Border).
		Faint(true)

	t.StatusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(t.Border).
		Padding(0, 1)

	t.SpinnerStyle = lipgloss.NewStyle().
		Foreground(t.Secondary)

	t.CodeBlockStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("223")).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		MarginLeft(2)

	t.KeywordStyle = lipgloss.NewStyle().
		Foreground(t.Highlight).
		Bold(true)

	return t
}

// DarkTheme returns a dark theme variant.
func DarkTheme() *Theme {
	t := DefaultTheme()
	// Adjust colors for dark background
	t.Primary = lipgloss.Color("75")    // Light Blue
	t.Secondary = lipgloss.Color("117") // Light Cyan
	t.Success = lipgloss.Color("120")   // Light Green
	return t
}

// LightTheme returns a light theme variant.
func LightTheme() *Theme {
	t := DefaultTheme()
	// Adjust colors for light background
	t.Primary = lipgloss.Color("27")   // Dark Blue
	t.Secondary = lipgloss.Color("31") // Dark Cyan
	t.Success = lipgloss.Color("34")   // Dark Green
	t.Muted = lipgloss.Color("240")    // Dark Gray
	return t
}

// Global theme instance
var currentTheme = DefaultTheme()

// GetTheme returns the current theme.
func GetTheme() *Theme {
	return currentTheme
}

// SetTheme sets the current theme.
func SetTheme(t *Theme) {
	currentTheme = t
}
