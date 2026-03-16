package shell

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunCommandBasic(t *testing.T) {
	tool := &RunCommandTool{}
	input, _ := json.Marshal(map[string]interface{}{"command": "echo hello"})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if !strings.Contains(result.Content, "hello") {
		t.Errorf("expected 'hello', got: %s", result.Content)
	}
}

func TestRunCommandStderr(t *testing.T) {
	tool := &RunCommandTool{}
	input, _ := json.Marshal(map[string]interface{}{"command": "ls /nonexistent_path_xyz"})
	result, _ := tool.Execute(context.Background(), input)
	if !result.IsError {
		t.Error("expected error for nonexistent path")
	}
}

func TestRunCommandForbidden(t *testing.T) {
	tool := &RunCommandTool{}
	input, _ := json.Marshal(map[string]interface{}{"command": "rm -rf / --no-preserve-root"})
	result, _ := tool.Execute(context.Background(), input)
	if !result.IsError {
		t.Error("expected forbidden command to be blocked")
	}
	if !strings.Contains(result.Content, "blocked") {
		t.Error("expected 'blocked' message")
	}
}

func TestRunCommandTimeout(t *testing.T) {
	tool := &RunCommandTool{}
	input, _ := json.Marshal(map[string]interface{}{
		"command":     "sleep 10",
		"timeout_sec": 1,
	})
	result, _ := tool.Execute(context.Background(), input)
	if !result.IsError {
		t.Error("expected timeout error")
	}
	if !strings.Contains(result.Content, "timed out") {
		t.Errorf("expected 'timed out' message, got: %s", result.Content)
	}
}

func TestIsForbidden(t *testing.T) {
	cases := []struct {
		cmd      string
		expected bool
	}{
		{"echo hello", false},
		{"rm -rf /", true},
		{"ls -la", false},
		{"mkfs.ext4 /dev/sda", true},
		{"git status", false},
	}
	for _, c := range cases {
		if got := isForbidden(c.cmd); got != c.expected {
			t.Errorf("isForbidden(%q) = %v, want %v", c.cmd, got, c.expected)
		}
	}
}
