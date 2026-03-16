package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionUndo(t *testing.T) {
	sess := New("claude-sonnet-4-5")

	// Create a temp file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	original := []byte("original content")
	modified := []byte("modified content")
	_ = os.WriteFile(path, original, 0644)

	// Push a snapshot
	sess.PushSnapshot(FileSnapshot{
		FilePath: path,
		Before:   original,
		After:    modified,
	})

	// Write the modified version
	_ = os.WriteFile(path, modified, 0644)

	// Undo
	snap, err := sess.Undo()
	if err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if snap.FilePath != path {
		t.Errorf("expected path %s, got %s", path, snap.FilePath)
	}

	// Verify file was restored
	got, _ := os.ReadFile(path)
	if string(got) != string(original) {
		t.Errorf("expected %q, got %q", original, got)
	}

	// Double undo should fail
	_, err = sess.Undo()
	if err == nil {
		t.Error("expected error on empty undo stack")
	}
}

func TestTokenUsageCost(t *testing.T) {
	u := &TokenUsage{InputTokens: 1000, OutputTokens: 500}
	cost := u.CostEstimate("claude-sonnet-4-5")
	// 1000 * 3/1M + 500 * 15/1M = 0.003 + 0.0075 = 0.0105
	if cost < 0.010 || cost > 0.011 {
		t.Errorf("unexpected cost estimate: %f", cost)
	}
}

func TestClearMessages(t *testing.T) {
	sess := New("test-model")
	sess.AppendMessage(TextMessage("system", "sys"))
	sess.AppendMessage(TextMessage("user", "hello"))
	sess.AppendMessage(TextMessage("assistant", "hi"))
	sess.ClearMessages()
	if len(sess.Messages) != 1 {
		t.Errorf("expected 1 message (system) after clear, got %d", len(sess.Messages))
	}
	if sess.Messages[0].Role != "system" {
		t.Error("expected system message to be preserved")
	}
}
