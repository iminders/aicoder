package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppState represents the current state of the application.
type AppState int

const (
	StateIdle AppState = iota
	StateThinking
	StateStreaming
	StateWaitingConfirm
)

// Model is the bubbletea model for the TUI.
type Model struct {
	// State
	state    AppState
	width    int
	height   int
	quitting bool

	// UI components
	spinner  *Spinner
	markdown *MarkdownRenderer
	theme    *Theme

	// Content
	messages      []Message
	currentOutput strings.Builder
	inputBuffer   string
	statusText    string

	// Metadata
	modelName string
	directory string
	tokens    int

	// Callbacks
	onSubmit func(string) error
	onCancel func()
}

// Message represents a chat message.
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
	Time    time.Time
}

// NewModel creates a new TUI model.
func NewModel(modelName, directory string) *Model {
	md, _ := NewMarkdownRenderer(80)
	return &Model{
		state:     StateIdle,
		spinner:   NewSpinner(),
		markdown:  md,
		theme:     GetTheme(),
		messages:  []Message{},
		modelName: modelName,
		directory: directory,
		tokens:    0,
	}
}

// Init initializes the model (bubbletea lifecycle).
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model (bubbletea lifecycle).
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.markdown != nil {
			m.markdown.SetWidth(msg.Width - 4)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tickMsg:
		if m.state == StateThinking || m.state == StateStreaming {
			m.spinner.Tick()
			return m, tick(m.spinner.Interval())
		}
		return m, nil

	case streamChunkMsg:
		m.currentOutput.WriteString(string(msg))
		return m, nil

	case stateChangeMsg:
		m.state = AppState(msg)
		if m.state == StateThinking || m.state == StateStreaming {
			return m, tick(m.spinner.Interval())
		}
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.onCancel != nil {
			m.onCancel()
		}
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEnter:
		if m.state == StateIdle && m.inputBuffer != "" {
			input := m.inputBuffer
			m.inputBuffer = ""
			m.messages = append(m.messages, Message{
				Role:    "user",
				Content: input,
				Time:    time.Now(),
			})
			if m.onSubmit != nil {
				go m.onSubmit(input)
			}
			m.state = StateThinking
			return m, tick(m.spinner.Interval())
		}
		return m, nil

	case tea.KeyBackspace:
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
		return m, nil

	case tea.KeyRunes:
		if m.state == StateIdle {
			m.inputBuffer += string(msg.Runes)
		}
		return m, nil
	}

	return m, nil
}

// View renders the UI (bubbletea lifecycle).
func (m *Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Status bar
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n\n")

	// Messages
	for _, msg := range m.messages {
		b.WriteString(m.renderMessage(msg))
		b.WriteString("\n")
	}

	// Current streaming output
	if m.currentOutput.Len() > 0 {
		b.WriteString(m.theme.InfoStyle.Render("Assistant: "))
		b.WriteString(m.currentOutput.String())
		b.WriteString("\n")
	}

	// Spinner for thinking/streaming state
	if m.state == StateThinking {
		b.WriteString(m.spinner.View())
		b.WriteString(" Thinking...\n")
	} else if m.state == StateStreaming {
		b.WriteString(m.spinner.View())
		b.WriteString(" Streaming...\n")
	}

	// Input prompt
	if m.state == StateIdle {
		b.WriteString("\n")
		b.WriteString(m.theme.PromptStyle.Render("> "))
		b.WriteString(m.inputBuffer)
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render("_"))
	}

	return b.String()
}

// renderStatusBar renders the top status bar.
func (m *Model) renderStatusBar() string {
	left := fmt.Sprintf(" aicoder | %s ", m.modelName)
	right := fmt.Sprintf(" %s | %d tokens ", m.directory, m.tokens)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return m.theme.StatusBarStyle.Render(left + strings.Repeat(" ", gap) + right)
}

// renderMessage renders a single message.
func (m *Model) renderMessage(msg Message) string {
	var b strings.Builder

	// Role prefix
	switch msg.Role {
	case "user":
		b.WriteString(m.theme.PromptStyle.Render("You: "))
	case "assistant":
		b.WriteString(m.theme.InfoStyle.Render("Assistant: "))
	case "system":
		b.WriteString(m.theme.SubtitleStyle.Render("System: "))
	}

	// Content (render as markdown if possible)
	if m.markdown != nil {
		rendered, err := m.markdown.Render(msg.Content)
		if err == nil {
			b.WriteString(rendered)
		} else {
			b.WriteString(msg.Content)
		}
	} else {
		b.WriteString(msg.Content)
	}

	return b.String()
}

// SetOnSubmit sets the callback for when user submits input.
func (m *Model) SetOnSubmit(fn func(string) error) {
	m.onSubmit = fn
}

// SetOnCancel sets the callback for when user cancels.
func (m *Model) SetOnCancel(fn func()) {
	m.onCancel = fn
}

// AddMessage adds a message to the history.
func (m *Model) AddMessage(role, content string) {
	m.messages = append(m.messages, Message{
		Role:    role,
		Content: content,
		Time:    time.Now(),
	})
}

// SetState changes the application state.
func (m *Model) SetState(state AppState) {
	m.state = state
}

// UpdateTokens updates the token count.
func (m *Model) UpdateTokens(tokens int) {
	m.tokens = tokens
}

// Custom messages for bubbletea
type tickMsg time.Time
type streamChunkMsg string
type stateChangeMsg AppState

// tick returns a command that sends a tick message after the given duration.
func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// StreamChunk returns a command that sends a stream chunk message.
func StreamChunk(chunk string) tea.Cmd {
	return func() tea.Msg {
		return streamChunkMsg(chunk)
	}
}

// ChangeState returns a command that changes the application state.
func ChangeState(state AppState) tea.Cmd {
	return func() tea.Msg {
		return stateChangeMsg(state)
	}
}
