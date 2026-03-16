package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/iminders/aicoder/internal/tools"
)

// ForbiddenPatterns holds command substrings that are always blocked.
var ForbiddenPatterns = []string{
	"rm -rf /",
	"mkfs",
	"dd if=",
	":(){:|:&};:",
	"chmod -R 777 /",
}

func isForbidden(cmd string) bool {
	lower := strings.ToLower(cmd)
	for _, p := range ForbiddenPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// ─── RunCommandTool ───────────────────────────────────────────────────────────

type RunCommandTool struct{}

func (t *RunCommandTool) Name() string          { return "run_command" }
func (t *RunCommandTool) Risk() tools.RiskLevel { return tools.RiskMedium }
func (t *RunCommandTool) Description() string {
	return "Execute a shell command and return its stdout and stderr. Times out after 60 seconds."
}
func (t *RunCommandTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command":     {"type": "string",  "description": "Shell command to execute"},
			"working_dir": {"type": "string",  "description": "Working directory (optional, defaults to cwd)"},
			"timeout_sec": {"type": "integer", "description": "Timeout in seconds (default 60)"}
		},
		"required": ["command"]
	}`)
}

type runInput struct {
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir"`
	TimeoutSec int    `json:"timeout_sec"`
}

func (t *RunCommandTool) Execute(ctx context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in runInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if isForbidden(in.Command) {
		return &tools.Result{IsError: true, Content: fmt.Sprintf("command blocked by safety policy: %s", in.Command)}, nil
	}
	timeout := time.Duration(in.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", in.Command)
	if in.WorkingDir != "" {
		cmd.Dir = in.WorkingDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	var sb strings.Builder
	if stdout.Len() > 0 {
		sb.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		sb.WriteString("\n[stderr]\n")
		sb.WriteString(stderr.String())
	}
	result := strings.TrimSpace(sb.String())
	if result == "" {
		result = "(no output)"
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &tools.Result{IsError: true, Content: "command timed out after " + timeout.String()}, nil
		}
		return &tools.Result{IsError: true, Content: fmt.Sprintf("exit error: %v\n%s", err, result)}, nil
	}
	return &tools.Result{Content: result}, nil
}

// ─── RunBackgroundTool ────────────────────────────────────────────────────────

type RunBackgroundTool struct {
	procs map[string]*exec.Cmd
}

func NewRunBackgroundTool() *RunBackgroundTool {
	return &RunBackgroundTool{procs: map[string]*exec.Cmd{}}
}

func (t *RunBackgroundTool) Name() string          { return "run_background" }
func (t *RunBackgroundTool) Risk() tools.RiskLevel { return tools.RiskMedium }
func (t *RunBackgroundTool) Description() string {
	return "Start a long-running process in the background (e.g. a dev server). Returns a process ID label."
}
func (t *RunBackgroundTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command":     {"type": "string", "description": "Command to run in background"},
			"label":       {"type": "string", "description": "A short label to identify this process"},
			"working_dir": {"type": "string", "description": "Working directory (optional)"}
		},
		"required": ["command","label"]
	}`)
}

type bgInput struct {
	Command    string `json:"command"`
	Label      string `json:"label"`
	WorkingDir string `json:"working_dir"`
}

func (t *RunBackgroundTool) Execute(_ context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in bgInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if isForbidden(in.Command) {
		return &tools.Result{IsError: true, Content: "command blocked by safety policy"}, nil
	}
	cmd := exec.Command("sh", "-c", in.Command)
	if in.WorkingDir != "" {
		cmd.Dir = in.WorkingDir
	}
	if err := cmd.Start(); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	t.procs[in.Label] = cmd
	return &tools.Result{Content: fmt.Sprintf("Started background process '%s' (PID %d)", in.Label, cmd.Process.Pid)}, nil
}

func init() {
	tools.Global.Register(&RunCommandTool{})
	tools.Global.Register(NewRunBackgroundTool())
}
