package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Message is a single LLM conversation turn.
type Message struct {
	Role    string    `json:"role"` // system | user | assistant
	Content []Content `json:"content"`
}

// Content is one block within a message.
type Content struct {
	Type      string      `json:"type"` // text | tool_use | tool_result | image
	Text      string      `json:"text,omitempty"`
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
}

// TextMessage builds a simple text message.
func TextMessage(role, text string) Message {
	return Message{Role: role, Content: []Content{{Type: "text", Text: text}}}
}

// TokenUsage tracks cumulative token consumption.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

func (u *TokenUsage) Add(in, out int) {
	u.InputTokens += in
	u.OutputTokens += out
}

// CostEstimate returns a rough USD cost estimate.
func (u *TokenUsage) CostEstimate(model string) float64 {
	// Approximate pricing per million tokens
	prices := map[string][2]float64{
		"claude-opus-4-5":   {15.0, 75.0},
		"claude-sonnet-4-5": {3.0, 15.0},
		"claude-haiku-4-5":  {0.25, 1.25},
		"gpt-4o":            {5.0, 15.0},
		"gpt-4o-mini":       {0.15, 0.60},
		"DeepSeek-R1":       {0.20, 3.00},
	}
	p, ok := prices[model]
	if !ok {
		p = [2]float64{3.0, 15.0}
	}
	return float64(u.InputTokens)/1e6*p[0] + float64(u.OutputTokens)/1e6*p[1]
}

// FileSnapshot records the before/after state of a file write.
type FileSnapshot struct {
	ToolCallID string
	ToolName   string
	FilePath   string
	Before     []byte // nil means file did not exist
	After      []byte
	Timestamp  time.Time
}

// Session holds all state for one aicoder conversation.
type Session struct {
	mu        sync.Mutex
	ID        string
	StartedAt time.Time
	Messages  []Message
	Snapshots []FileSnapshot
	Usage     TokenUsage
	Model     string
}

// New creates a fresh session.
func New(model string) *Session {
	return &Session{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		StartedAt: time.Now(),
		Model:     model,
	}
}

func (s *Session) AppendMessage(m Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, m)
}

func (s *Session) RecordUsage(in, out int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Usage.Add(in, out)
}

// PushSnapshot records a file operation for potential undo.
func (s *Session) PushSnapshot(snap FileSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Snapshots = append(s.Snapshots, snap)
}

// Undo reverts the most recent file snapshot.
func (s *Session) Undo() (FileSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Snapshots) == 0 {
		return FileSnapshot{}, fmt.Errorf("没有可撤销的操作")
	}
	snap := s.Snapshots[len(s.Snapshots)-1]
	s.Snapshots = s.Snapshots[:len(s.Snapshots)-1]

	if snap.Before == nil {
		if err := os.Remove(snap.FilePath); err != nil && !os.IsNotExist(err) {
			return snap, err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(snap.FilePath), 0755); err != nil {
			return snap, err
		}
		if err := os.WriteFile(snap.FilePath, snap.Before, 0644); err != nil {
			return snap, err
		}
	}
	return snap, nil
}

// ClearMessages resets conversation history (keeps system prompt if any).
func (s *Session) ClearMessages() {
	s.mu.Lock()
	defer s.mu.Unlock()
	var sys []Message
	for _, m := range s.Messages {
		if m.Role == "system" {
			sys = append(sys, m)
		}
	}
	s.Messages = sys
}

// AllFileChanges returns a map[path]->current_content for files touched this session.
func (s *Session) AllFileChanges() map[string][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := map[string][]byte{}
	for _, snap := range s.Snapshots {
		result[snap.FilePath] = snap.After
	}
	return result
}

// Save persists the session to ~/.aicoder/sessions/.
func (s *Session) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".aicoder", "sessions")
	_ = os.MkdirAll(dir, 0700)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, s.ID+".json")
	return os.WriteFile(path, data, 0600)
}
