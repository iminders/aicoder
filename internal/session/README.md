# Session Management Design

## Overview

Long-running coding sessions generate extensive conversation history that can exceed LLM context windows and increase costs. This document outlines a comprehensive session management system to handle this challenge.

## Current State

The existing `Session` struct ([session.go](session.go)) provides basic functionality:
- `Messages []Message` - stores conversation turns
- `Snapshots []FileSnapshot` - tracks file changes for undo
- `Usage TokenUsage` - tracks token consumption
- `Save()` method - persists session to JSON file
- `ClearMessages()` - resets conversation (preserves system prompt)

## Design Goals

1. **Retention**: Keep recent and important conversations
2. **Compression**: Handle long documents/context efficiently
3. **Cost Control**: Prevent excessive token usage
4. **Persistence**: Maintain session history across restarts

## Proposed Architecture

```
SessionManager
├── SessionStore (persistence layer)
│   ├── sessions/           # Saved session files
│   └── sessions.json       # Session index
├── Session
│   ├── Metadata            # id, created, updated, tags, summary
│   ├── Messages []Message  # Conversation history
│   ├── MessagesIndex       # Quick lookup for messages
│   └── Strategy           # Compression/retention strategy
├── RetentionPolicy        # What to keep, what to discard
├── CompressionEngine      # Summarization and truncation
└── SessionSummarizer      # Generate session summaries
```

## 1. Session Metadata & Index

```go
type SessionMeta struct {
    ID          string    `json:"id"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    Title       string    `json:"title"`        // Auto-generated or user-defined
    Summary     string    `json:"summary"`       // First N messages summary
    Tags        []string  `json:"tags"`          // e.g., "bugfix", "refactor"
    MessageCount int      `json:"message_count"`
    TokenCount  int       `json:"token_count"`
    Model       string    `json:"model"`
    IsArchived  bool      `json:"is_archived"`
    IsPinned    bool      `json:"is_pinned"`    // Prevent auto-cleanup
}
```

**Session Index (`~/.aicoder/sessions/index.json`)**:
```go
type SessionIndex struct {
    Sessions    []SessionMeta `json:"sessions"`
    ActiveID    string        `json:"active_id"`
    LastUpdated time.Time     `json:"last_updated"`
}
```

## 2. Retention Policies

### 2.1 Sliding Window Strategy (Default)

Keep the most recent N messages that fit within token budget:

```go
type SlidingWindowConfig struct {
    MaxMessages     int   // e.g., 50
    MaxTokens       int   // e.g., 128000 (half of 200k context)
    PreserveSystem  bool  // Always keep system prompt
    PreserveSummary bool  // Keep summary of older messages
}
```

### 2.2 Importance-Based Retention

Mark messages as important to prevent auto-cleanup:

```go
type MessageFlags struct {
    IsImportant bool      `json:"is_important"` // Set by user via /pin
    IsDecision  bool      `json:"is_decision"`   // Code decisions, architecture choices
    HasContext  bool      `json:"has_context"`   // Contains long context (docs, code)
}

// Rules:
// - User can mark messages with /pin command
// - Decisions (code architecture, tool choices) auto-marked
// - Long content auto-truncated but summary preserved
```

### 2.3 Hierarchical Retention

```
┌─────────────────────────────────────────┐
│ Recent Messages (fully preserved)       │  ← Last 20 messages
├─────────────────────────────────────────┤
│ Middle Messages (summarized)            │  ← Messages 21-100
├─────────────────────────────────────────┤
│ Old Messages (condensed summary only)   │  ← Messages 100+
└─────────────────────────────────────────┘
```

## 3. Compression Strategies

### 3.1 Message Truncation

For individual long messages:

```go
func TruncateMessage(msg Message, maxTokens int) Message {
    // 1. Count tokens in message
    // 2. If exceeds maxTokens, keep beginning + summary of end
    // 3. Mark as truncated
}
```

### 3.2 Message Summarization

Convert a batch of messages to a summary:

```go
type MessageSummary struct {
    OriginalCount   int      `json:"original_count"`
    OriginalTokens  int      `json:"original_tokens"`
    SummaryText     string   `json:"summary_text"`
    KeyDecisions    []string `json:"key_decisions"`
    KeyFilesChanged []string `json:"key_files_changed"`
    TimeRange       string   `json:"time_range"`      // "10:30-11:45"
}
```

**Compression Trigger Points**:
- When `MaxTokens` exceeded after adding new message
- When message count exceeds `MaxMessages`
- User-triggered via `/compress` command

### 3.3 Context Compression for Long Documents

When processing large files or documents:

```go
type ContextCompression struct {
    OriginalSize int    `json:"original_size"`
    Compressed   bool  `json:"compressed"`
    Summary      string `json:"summary"`
    KeyExcerpts  []string `json:"key_excerpts"`  // Important sections
    FilePath     string   `json:"file_path,omitempty"`
}
```

**Rules**:
- Files > 10KB: Keep summary + first 500 chars + last 500 chars
- Files > 100KB: Summary only
- Keep all tool outputs (they may be referenced later)

## 4. Session Management Operations

### 4.1 Auto-Save

```go
type AutoSaveConfig struct {
    Enabled      bool          // Default: true
    Interval     time.Duration // Default: 30 seconds
    OnChange     bool          // Save after each message
}
```

### 4.2 Session List Commands

```
/sessions              # List all sessions with metadata
/sessions --recent 10  # Show 10 most recent
/sessions --pinned     # Show pinned sessions
/sessions --archived   # Show archived sessions
/sessions --search "debug"  # Search by title/tags
```

### 4.3 Session Operations

```
/session load <id>      # Load a previous session
/session save [name]   # Save current with name
/session archive        # Archive (hide from default list)
/session delete <id>   # Delete session
/session pin            # Pin current session
/session tag <tags>     # Add tags to session
```

### 4.4 Session Comparison

```
/diff <session_id>     # Compare current to previous session
```

## 5. Token Budget Management

### 5.1 Per-Session Budget

```go
type TokenBudget struct {
    MaxPerSession  int     // Default: 200000 (for 200k context models)
    WarningAt      float64 // Default: 0.8 (80% usage warning)
    HardLimit      bool    // Refuse to add more if true
}
```

### 5.2 Usage Tracking

```go
type SessionUsage struct {
    InputTokens   int
    OutputTokens  int
    ToolCallCount int
    EstimatedCost float64
}
```

**Warning Levels**:
- 80%: Show warning message
- 90%: Show strong warning, suggest `/compress`
- 100%: Auto-compress or refuse new input

## 6. Implementation Plan

### Phase 1: Enhanced Session Metadata ✅
- [x] Add `SessionMeta` struct
- [x] Create session index management
- [x] Add session listing command (`/sessions`)

### Phase 2: Retention Policies ✅
- [x] Implement `SlidingWindowPolicy`
- [x] Add `MessageFlags` for importance
- [x] Add `/pin` and `/unpin` commands

### Phase 3: Compression Engine ✅
- [x] Implement message truncation
- [x] Add summarization prompt integration
- [x] Create `ContextCompression` for large files
- [x] Add `/compress` command

### Phase 4: Auto-Management ✅
- [x] Implement auto-save
- [x] Add token budget warnings
- [x] Implement auto-compression trigger

### Phase 5: Advanced Features ✅
- [x] Session search functionality
- [x] Session comparison (`/compare`)
- [x] Session export/import

## 7. API Changes

### Session Manager

```go
type SessionManager struct {
    store       *SessionStore
    index       *SessionIndex
    cfg         *Config
}

func (sm *SessionManager) New(model string) *Session
func (sm *SessionManager) Load(id string) (*Session, error)
func (sm *SessionManager) Save(s *Session) error
func (sm *SessionManager) List(filter ListFilter) []SessionMeta
func (sm *SessionManager) Delete(id string) error
func (sm *SessionManager) Archive(id string) error
func (sm *SessionManager) Search(query string) []SessionMeta
```

### Session (Enhanced)

```go
type Session struct {
    // Existing fields...
    Meta       SessionMeta
    Index      *MessagesIndex  // For quick message lookup

    // New methods
    Compress(strategy CompressionStrategy) error
    TrimToTokenBudget(budget int) error
    AddPin(messageIndex int) error
    RemovePin(messageIndex int) error
    GenerateSummary() (string, error)
}
```

## 8. Configuration

```yaml
session:
  # Retention settings
  max_messages: 100
  max_tokens: 128000
  preserve_system: true
  preserve_summary: true

  # Compression
  auto_compress: true
  compress_threshold: 0.85
  truncate_long_messages: true
  long_message_threshold: 4000  # chars

  # Persistence
  auto_save: true
  auto_save_interval: 30s
  sessions_dir: ~/.aicoder/sessions

  # Budget
  token_budget: 200000
  warning_threshold: 0.8
  hard_limit: false
```

## 9. Slash Commands (New)

| Command | Description |
|---------|-------------|
| `/sessions` | List all sessions |
| `/session load <id>` | Load a session |
| `/session save [name]` | Save with name |
| `/session pin` | Pin current session |
| `/session tag <tags>` | Add tags |
| `/session archive` | Archive session |
| `/compress` | Manually trigger compression |
| `/budget` | Show token budget usage |
| `/pin <n>` | Pin message N |
| `/history` | Show history (existing) |
| `/clear` | Clear messages (existing) |

## 10. Backward Compatibility

The existing session format should be preserved:
- Existing JSON files remain readable
- New fields added with defaults
- Migration only when necessary

## 11. Considerations

### Trade-offs
- **Summarization quality**: AI summarization costs tokens but preserves context
- **Aggressive truncation**: May lose important details
- **Auto-compression**: May disrupt conversation flow if triggered inappropriately

### Edge Cases
- Single very long message (e.g., 50k token file content)
- Rapid-fire short messages
- Session with no meaningful content (just tool calls)
- Model-specific context window limits

### Future Enhancements
- Cross-session memory (learn from past sessions)
- Semantic search within sessions
- Session templates
- Collaborative session sharing
