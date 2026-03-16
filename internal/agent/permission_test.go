package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/iminders/aicoder/internal/config"
	"github.com/iminders/aicoder/internal/tools"
)

// mockTool is a test double for tools.Tool.
type mockTool struct {
	name string
	risk tools.RiskLevel
}

func (m *mockTool) Name() string            { return m.name }
func (m *mockTool) Description() string     { return "mock" }
func (m *mockTool) Schema() json.RawMessage { return json.RawMessage(`{}`) }
func (m *mockTool) Risk() tools.RiskLevel   { return m.risk }
func (m *mockTool) Execute(_ context.Context, _ json.RawMessage) (*tools.Result, error) {
	return &tools.Result{Content: "ok"}, nil
}

func baseConfig() *config.Config {
	return &config.Config{
		AutoApprove:         false,
		AutoApproveReads:    true,
		AutoApproveCommands: []string{"go test", "npm test"},
		ForbiddenCommands:   []string{"rm -rf /", "mkfs"},
	}
}

func TestPermLowRisk(t *testing.T) {
	g := NewPermissionGuard(baseConfig(), false)
	tool := &mockTool{name: "read_file", risk: tools.RiskLow}
	res := g.Check(tool, json.RawMessage(`{}`))
	if res.Action != PermAllow {
		t.Errorf("expected Allow for low-risk read, got %v", res.Action)
	}
}

func TestPermMediumRisk(t *testing.T) {
	g := NewPermissionGuard(baseConfig(), false)
	tool := &mockTool{name: "write_file", risk: tools.RiskMedium}
	res := g.Check(tool, json.RawMessage(`{}`))
	if res.Action != PermNeedsConfirm {
		t.Errorf("expected NeedsConfirm for medium-risk, got %v", res.Action)
	}
}

func TestPermAutoApproveAll(t *testing.T) {
	cfg := baseConfig()
	cfg.AutoApprove = true
	g := NewPermissionGuard(cfg, false)
	tool := &mockTool{name: "delete_file", risk: tools.RiskHigh}
	res := g.Check(tool, json.RawMessage(`{}`))
	if res.Action != PermAllow {
		t.Errorf("expected Allow with autoApprove=true")
	}
}

func TestPermForbiddenCommand(t *testing.T) {
	g := NewPermissionGuard(baseConfig(), false)
	tool := &mockTool{name: "run_command", risk: tools.RiskMedium}
	input, _ := json.Marshal(map[string]string{"command": "rm -rf / --no-preserve-root"})
	res := g.Check(tool, input)
	if res.Action != PermDeny {
		t.Errorf("expected Deny for forbidden command, got %v", res.Action)
	}
}

func TestPermApprovedCommand(t *testing.T) {
	g := NewPermissionGuard(baseConfig(), false)
	tool := &mockTool{name: "run_command", risk: tools.RiskMedium}
	input, _ := json.Marshal(map[string]string{"command": "go test ./..."})
	res := g.Check(tool, input)
	if res.Action != PermAllow {
		t.Errorf("expected Allow for auto-approved command, got %v", res.Action)
	}
}

func TestPermDangerouslySkip(t *testing.T) {
	g := NewPermissionGuard(baseConfig(), true)
	tool := &mockTool{name: "delete_file", risk: tools.RiskHigh}
	res := g.Check(tool, json.RawMessage(`{}`))
	if res.Action != PermAllow {
		t.Errorf("expected Allow with dangerouslySkip=true")
	}
}

func TestPermAlwaysAllow(t *testing.T) {
	g := NewPermissionGuard(baseConfig(), false)
	tool := &mockTool{name: "write_file", risk: tools.RiskMedium}
	g.SetAlwaysAllow("write_file")
	res := g.Check(tool, json.RawMessage(`{}`))
	if res.Action != PermAllow {
		t.Errorf("expected Allow after SetAlwaysAllow")
	}
}
