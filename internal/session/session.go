package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SessionMeta holds metadata about a session for listing and management.
type SessionMeta struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Model        string    `json:"model"`
	IsArchived   bool      `json:"is_archived"`
	IsPinned     bool      `json:"is_pinned"`
}

// MessageFlags marks messages with additional metadata.
type MessageFlags struct {
	IsImportant bool `json:"is_important"`
	IsPinned    bool `json:"is_pinned"`
}

// SessionIndex tracks all sessions for the user.
type SessionIndex struct {
	Sessions   []SessionMeta `json:"sessions"`
	ActiveID   string        `json:"active_id"`
	LastUpdate time.Time     `json:"last_update"`
}

// Message is a single LLM conversation turn.
type Message struct {
	Role    string       `json:"role"` // system | user | assistant
	Content []Content    `json:"content"`
	Flags   MessageFlags `json:"flags,omitempty"`
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
		"DeepSeek-R1":       {0.24, 0.42},
		"MiniMax-M2.5":      {0.24, 0.42},
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

	// PendingSkill / PendingPrompt are set by /skill <name> <prompt>
	// and consumed by the interactive loop to trigger RunWithSkill.
	PendingSkillName string
	PendingPrompt    string
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

func (s *Session) ToMeta() SessionMeta {
	return SessionMeta{
		ID:           s.ID,
		Title:        s.Title(),
		Summary:      s.Summary(),
		Tags:         []string{},
		CreatedAt:    s.StartedAt,
		UpdatedAt:    time.Now(),
		MessageCount: len(s.Messages),
		InputTokens:  s.Usage.InputTokens,
		OutputTokens: s.Usage.OutputTokens,
		Model:        s.Model,
		IsArchived:   false,
		IsPinned:     false,
	}
}

func (s *Session) Title() string {
	for _, m := range s.Messages {
		if m.Role == "user" && len(m.Content) > 0 {
			for _, c := range m.Content {
				if c.Type == "text" && len(c.Text) > 0 {
					title := c.Text
					if len(title) > 50 {
						title = title[:50] + "..."
					}
					return title
				}
			}
		}
	}
	return "Untitled Session"
}

func (s *Session) Summary() string {
	var userMsgs []string
	count := 0
	for _, m := range s.Messages {
		if m.Role == "user" && count < 3 {
			for _, c := range m.Content {
				if c.Type == "text" && len(c.Text) > 0 {
					text := c.Text
					if len(text) > 100 {
						text = text[:100] + "..."
					}
					userMsgs = append(userMsgs, text)
					count++
					break
				}
			}
		}
	}
	if len(userMsgs) == 0 {
		return "Empty session"
	}
	result := ""
	for i, msg := range userMsgs {
		if i > 0 {
			result += " | "
		}
		result += msg
	}
	return result
}

func (s *Session) PinMessage(idx int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx < 0 || idx >= len(s.Messages) {
		return fmt.Errorf("消息索引无效: %d", idx)
	}
	s.Messages[idx].Flags.IsPinned = true
	s.Messages[idx].Flags.IsImportant = true
	return nil
}

func (s *Session) UnpinMessage(idx int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx < 0 || idx >= len(s.Messages) {
		return fmt.Errorf("消息索引无效: %d", idx)
	}
	s.Messages[idx].Flags.IsPinned = false
	s.Messages[idx].Flags.IsImportant = false
	return nil
}

func (s *Session) MessageCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Messages)
}

func (s *Session) PinnedMessages() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	var pinned []Message
	for _, m := range s.Messages {
		if m.Flags.IsPinned {
			pinned = append(pinned, m)
		}
	}
	return pinned
}

func (s *Session) TrimToCount(maxMsgs int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Messages) <= maxMsgs {
		return
	}
	var sys []Message
	var pinned []Message
	var other []Message

	for _, m := range s.Messages {
		if m.Role == "system" {
			sys = append(sys, m)
		} else if m.Flags.IsPinned {
			pinned = append(pinned, m)
		} else {
			other = append(other, m)
		}
	}

	keepCount := maxMsgs - len(sys) - len(pinned)
	if keepCount < 0 {
		keepCount = 0
	}
	if keepCount > len(other) {
		keepCount = len(other)
	}

	s.Messages = append(sys, pinned...)
	s.Messages = append(s.Messages, other[len(other)-keepCount:]...)
}

func (s *Session) EstimatedTokens() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return estimateTokens(s.Messages)
}

func estimateTokens(msgs []Message) int {
	text := ""
	for _, m := range msgs {
		text += m.Role + ": "
		for _, c := range m.Content {
			if c.Text != "" {
				text += c.Text + " "
			}
		}
		text += "\n"
	}
	return len(text) / 4
}

type ListFilter struct {
	Limit           int
	Offset          int
	Search          string
	Tags            []string
	IncludeArchived bool
	OnlyPinned      bool
}

type SessionManager struct {
	mu        sync.Mutex
	indexPath string
	index     *SessionIndex
}

func NewSessionManager() (*SessionManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".aicoder", "sessions")
	_ = os.MkdirAll(dir, 0700)

	sm := &SessionManager{
		indexPath: filepath.Join(dir, "index.json"),
		index:     &SessionIndex{Sessions: []SessionMeta{}},
	}

	sm.loadIndex()
	return sm, nil
}

func (sm *SessionManager) loadIndex() {
	data, err := os.ReadFile(sm.indexPath)
	if err != nil {
		return
	}
	json.Unmarshal(data, sm.index)
	if sm.index.Sessions == nil {
		sm.index.Sessions = []SessionMeta{}
	}
}

func (sm *SessionManager) SaveIndex() error {
	sm.index.LastUpdate = time.Now()
	data, err := json.MarshalIndent(sm.index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.indexPath, data, 0600)
}

func (sm *SessionManager) UpdateSessionMeta(meta SessionMeta) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	found := false
	for i, s := range sm.index.Sessions {
		if s.ID == meta.ID {
			sm.index.Sessions[i] = meta
			found = true
			break
		}
	}
	if !found {
		sm.index.Sessions = append(sm.index.Sessions, meta)
	}
	return sm.SaveIndex()
}

func (sm *SessionManager) ListSessions(filter ListFilter) []SessionMeta {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var result []SessionMeta
	for _, s := range sm.index.Sessions {
		if s.IsArchived && !filter.IncludeArchived {
			continue
		}
		if !filter.OnlyPinned && s.IsPinned {
		}
		if filter.OnlyPinned && !s.IsPinned {
			continue
		}
		if filter.Search != "" {
			found := false
			lower := filter.Search
			for _, tag := range s.Tags {
				if strings.Contains(strings.ToLower(tag), lower) {
					found = true
					break
				}
			}
			if !found && !strings.Contains(strings.ToLower(s.Title), lower) {
				continue
			}
		}
		if len(filter.Tags) > 0 {
			hasTag := false
			for _, ft := range filter.Tags {
				for _, st := range s.Tags {
					if ft == st {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		result = append(result, s)
	}

	sortByUpdated(result)

	start := filter.Offset
	end := len(result)
	if filter.Limit > 0 && filter.Limit < end-start {
		end = start + filter.Limit
	}
	if start >= len(result) {
		return []SessionMeta{}
	}
	if end > len(result) {
		end = len(result)
	}
	return result[start:end]
}

func sortByUpdated(sessions []SessionMeta) {
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].UpdatedAt.After(sessions[i].UpdatedAt) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}
}

func (sm *SessionManager) GetSession(id string) (*Session, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".aicoder", "sessions", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("会话不存在: %s", id)
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (sm *SessionManager) DeleteSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".aicoder", "sessions", id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	for i, s := range sm.index.Sessions {
		if s.ID == id {
			sm.index.Sessions = append(sm.index.Sessions[:i], sm.index.Sessions[i+1:]...)
			break
		}
	}
	return sm.SaveIndex()
}

func (sm *SessionManager) ArchiveSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, s := range sm.index.Sessions {
		if s.ID == id {
			sm.index.Sessions[i].IsArchived = true
			break
		}
	}
	return sm.SaveIndex()
}

func (sm *SessionManager) PinSession(id string, pinned bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, s := range sm.index.Sessions {
		if s.ID == id {
			sm.index.Sessions[i].IsPinned = pinned
			break
		}
	}
	return sm.SaveIndex()
}

type MessageSummary struct {
	OriginalCount  int      `json:"original_count"`
	OriginalTokens int      `json:"original_tokens"`
	SummaryText    string   `json:"summary_text"`
	KeyDecisions   []string `json:"key_decisions"`
	FilesModified  []string `json:"files_modified"`
	TimeRange      string   `json:"time_range"`
}

type ContextCompression struct {
	OriginalSize int      `json:"original_size"`
	Compressed   bool     `json:"compressed"`
	Summary      string   `json:"summary"`
	KeyExcerpts  []string `json:"key_excerpts"`
	FilePath     string   `json:"file_path,omitempty"`
}

type CompressionResult struct {
	BeforeCount  int
	AfterCount   int
	BeforeTokens int
	AfterTokens  int
	Summaries    []MessageSummary
}

func (s *Session) Compress(maxMsgs int, maxTokens int) CompressionResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := CompressionResult{
		BeforeCount: len(s.Messages),
	}

	result.BeforeTokens = estimateTokens(s.Messages)

	var sys []Message
	var pinned []Message
	var other []Message

	for _, m := range s.Messages {
		if m.Role == "system" {
			sys = append(sys, m)
		} else if m.Flags.IsPinned {
			pinned = append(pinned, m)
		} else {
			other = append(other, m)
		}
	}

	keepCount := maxMsgs - len(sys) - len(pinned)
	if keepCount < 0 {
		keepCount = 0
	}

	if keepCount >= len(other) {
		result.AfterCount = len(s.Messages)
		result.AfterTokens = result.BeforeTokens
		return result
	}

	discarded := other[:len(other)-keepCount]
	remaining := other[len(other)-keepCount:]

	result.Summaries = summarizeMessageBatch(discarded)

	s.Messages = append(sys, pinned...)
	s.Messages = append(s.Messages, remaining...)

	result.AfterCount = len(s.Messages)
	result.AfterTokens = estimateTokens(s.Messages)
	return result
}

func summarizeMessageBatch(msgs []Message) []MessageSummary {
	if len(msgs) == 0 {
		return nil
	}

	var summaries []MessageSummary
	var batch []Message
	var batchTokens int
	const batchTokenLimit = 4000

	appendSummary := func() {
		if len(batch) == 0 {
			return
		}
		summary := MessageSummary{
			OriginalCount:  len(batch),
			OriginalTokens: batchTokens,
			SummaryText:    generateTextSummary(batch),
			KeyDecisions:   extractKeyDecisions(batch),
			FilesModified:  extractFilesModified(batch),
		}
		summaries = append(summaries, summary)
		batch = nil
		batchTokens = 0
	}

	for _, m := range msgs {
		tokens := estimateTokens([]Message{m})
		if batchTokens+tokens > batchTokenLimit && len(batch) > 0 {
			appendSummary()
		}
		batch = append(batch, m)
		batchTokens += tokens
	}
	appendSummary()

	return summaries
}

func generateTextSummary(msgs []Message) string {
	var userTexts []string
	var assistantTexts []string

	for _, m := range msgs {
		if m.Role == "user" {
			for _, c := range m.Content {
				if c.Type == "text" && len(c.Text) > 0 {
					text := c.Text
					if len(text) > 150 {
						text = text[:150] + "..."
					}
					userTexts = append(userTexts, text)
					break
				}
			}
		} else if m.Role == "assistant" {
			for _, c := range m.Content {
				if c.Type == "text" && len(c.Text) > 0 {
					text := c.Text
					if len(text) > 150 {
						text = text[:150] + "..."
					}
					assistantTexts = append(assistantTexts, text)
					break
				}
			}
		}
	}

	summary := ""
	if len(userTexts) > 0 {
		summary += "用户请求: " + userTexts[0]
	}
	if len(assistantTexts) > 0 {
		if summary != "" {
			summary += " | "
		}
		summary += "助手回复: " + assistantTexts[0]
	}
	if len(userTexts) > 1 {
		summary += fmt.Sprintf(" (+%d 条消息)", len(userTexts)-1)
	}
	return summary
}

func extractKeyDecisions(msgs []Message) []string {
	var decisions []string
	keywords := []string{"决定", "选择", "采用", "使用", "创建", "修改", "重构", "优化"}

	for _, m := range msgs {
		for _, c := range m.Content {
			if c.Type != "text" || c.Text == "" {
				continue
			}
			text := c.Text
			for _, kw := range keywords {
				idx := strings.Index(text, kw)
				if idx > 0 && idx < 100 {
					end := idx + 50
					if end > len(text) {
						end = len(text)
					}
					decision := strings.TrimSpace(text[idx:end])
					if len(decision) > 10 {
						decisions = append(decisions, decision)
					}
					break
				}
			}
		}
	}

	if len(decisions) > 5 {
		decisions = decisions[:5]
	}
	return decisions
}

func extractFilesModified(msgs []Message) []string {
	files := make(map[string]bool)
	for _, m := range msgs {
		for _, c := range m.Content {
			if c.Type == "tool_use" && c.Name == "write_file" || c.Name == "edit_file" {
				if input, ok := c.Input.(map[string]interface{}); ok {
					if path, ok := input["path"].(string); ok {
						files[path] = true
					}
				}
			}
		}
	}

	result := make([]string, 0, len(files))
	for f := range files {
		result = append(result, f)
	}
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

func (s *Session) ShouldAutoCompress(threshold float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens := estimateTokens(s.Messages)
	maxTokens := 128000
	ratio := float64(tokens) / float64(maxTokens)
	return ratio >= threshold
}

func CompressContent(content string, maxTokens int) ContextCompression {
	result := ContextCompression{
		OriginalSize: len(content),
	}

	originalTokens := len(content) / 4

	if originalTokens <= maxTokens {
		result.Compressed = false
		result.Summary = content
		return result
	}

	result.Compressed = true

	charLimit := maxTokens * 4
	if len(content) <= charLimit*2 {
		result.Summary = content[:charLimit]
		result.KeyExcerpts = []string{content[len(content)-charLimit:]}
	} else {
		result.Summary = content[:charLimit] + "\n...[内容已压缩]...\n" + content[len(content)-charLimit:]
		result.KeyExcerpts = []string{
			content[:500],
			content[len(content)-500:],
		}
	}

	return result
}

func (s *Session) AddTag(tag string) {
	s.mu.Lock()
	defer s.mu.Unlock()
}

func (s *Session) RemoveTag(tag string) {
	s.mu.Lock()
	defer s.mu.Unlock()
}

func (sm *SessionManager) AddTagToSession(id, tag string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, sess := range sm.index.Sessions {
		if sess.ID == id {
			found := false
			for _, t := range sm.index.Sessions[i].Tags {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				sm.index.Sessions[i].Tags = append(sm.index.Sessions[i].Tags, tag)
			}
			break
		}
	}
	return sm.SaveIndex()
}

func (sm *SessionManager) RemoveTagFromSession(id, tag string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, sess := range sm.index.Sessions {
		if sess.ID == id {
			newTags := make([]string, 0)
			for _, t := range sm.index.Sessions[i].Tags {
				if t != tag {
					newTags = append(newTags, t)
				}
			}
			sm.index.Sessions[i].Tags = newTags
			break
		}
	}
	return sm.SaveIndex()
}

func (sm *SessionManager) CompareSessions(id1, id2 string) (string, error) {
	sess1, err := sm.GetSession(id1)
	if err != nil {
		return "", err
	}
	sess2, err := sm.GetSession(id2)
	if err != nil {
		return "", err
	}

	compare := fmt.Sprintf("会话对比: %s vs %s\n", sess1.Title(), sess2.Title())
	compare += fmt.Sprintf("消息数: %d vs %d\n", len(sess1.Messages), len(sess2.Messages))
	compare += fmt.Sprintf("Tokens: %d vs %d\n",
		sess1.Usage.InputTokens+sess1.Usage.OutputTokens,
		sess2.Usage.InputTokens+sess2.Usage.OutputTokens)
	compare += fmt.Sprintf("创建时间: %s vs %s\n",
		sess1.StartedAt.Format("2006-01-02 15:04"),
		sess2.StartedAt.Format("2006-01-02 15:04"))

	return compare, nil
}

func (sm *SessionManager) ExportSession(id string) ([]byte, error) {
	sess, err := sm.GetSession(id)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(sess, "", "  ")
}

func (sm *SessionManager) ImportSession(data []byte) (*Session, error) {
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	sess.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	sess.StartedAt = time.Now()

	if err := sess.Save(); err != nil {
		return nil, err
	}
	sm.UpdateSessionMeta(sess.ToMeta())

	return &sess, nil
}
