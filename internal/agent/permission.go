package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iminders/aicoder/internal/config"
	"github.com/iminders/aicoder/internal/tools"
)

// PermAction is the outcome of a permission check.
type PermAction int

const (
	PermAllow        PermAction = iota
	PermNeedsConfirm            // ask the user
	PermDeny                    // hard block
)

// PermResult is returned by PermissionGuard.Check.
type PermResult struct {
	Action  PermAction
	Reason  string
	Preview string // human-readable preview of what will happen
}

// PermissionGuard enforces access control around tool calls.
type PermissionGuard struct {
	cfg         *config.Config
	alwaysAllow map[string]bool // session-scoped "always allow" keys
	skipAll     bool            // --dangerously-skip-permissions
}

func NewPermissionGuard(cfg *config.Config, skipAll bool) *PermissionGuard {
	return &PermissionGuard{
		cfg:         cfg,
		alwaysAllow: map[string]bool{},
		skipAll:     skipAll,
	}
}

// Check evaluates whether tool execution should proceed.
func (g *PermissionGuard) Check(t tools.Tool, input json.RawMessage) PermResult {
	if g.skipAll {
		return PermResult{Action: PermAllow, Reason: "dangerously-skip-permissions enabled"}
	}

	// 1. Hard deny: forbidden command patterns
	if t.Name() == "run_command" || t.Name() == "run_background" {
		var inp struct{ Command string `json:"command"` }
		_ = json.Unmarshal(input, &inp)
		for _, forbidden := range g.cfg.ForbiddenCommands {
			if strings.Contains(strings.ToLower(inp.Command), strings.ToLower(forbidden)) {
				return PermResult{Action: PermDeny, Reason: fmt.Sprintf("命令含有禁止关键词 '%s'", forbidden)}
			}
		}
	}

	// 2. Global auto-approve
	if g.cfg.AutoApprove {
		return PermResult{Action: PermAllow}
	}

	// 3. Session-scoped always-allow
	key := sessionKey(t, input)
	if g.alwaysAllow[t.Name()] || g.alwaysAllow[key] {
		return PermResult{Action: PermAllow, Reason: "本次会话已授权"}
	}

	// 4. Read-only tools auto-approve
	if t.Risk() == tools.RiskLow && g.cfg.AutoApproveReads {
		return PermResult{Action: PermAllow}
	}

	// 5. Auto-approve listed commands
	if t.Name() == "run_command" {
		var inp struct{ Command string `json:"command"` }
		_ = json.Unmarshal(input, &inp)
		for _, approved := range g.cfg.AutoApproveCommands {
			if strings.HasPrefix(strings.TrimSpace(inp.Command), approved) {
				return PermResult{Action: PermAllow}
			}
		}
	}

	// 6. Needs user confirmation
	preview := buildPreview(t, input)
	return PermResult{
		Action:  PermNeedsConfirm,
		Preview: preview,
	}
}

// SetAlwaysAllow marks a tool (or specific call) as always-allowed for this session.
func (g *PermissionGuard) SetAlwaysAllow(toolName string) {
	g.alwaysAllow[toolName] = true
}

func sessionKey(t tools.Tool, input json.RawMessage) string {
	return t.Name() + ":" + string(input)
}

func buildPreview(t tools.Tool, input json.RawMessage) string {
	var m map[string]interface{}
	_ = json.Unmarshal(input, &m)

	var lines []string
	lines = append(lines, fmt.Sprintf("工具：%s", t.Name()))
	for k, v := range m {
		s := fmt.Sprintf("%v", v)
		if len(s) > 120 {
			s = s[:120] + "..."
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", k, s))
	}
	lines = append(lines, fmt.Sprintf("风险等级：%s", t.Risk().String()))
	return strings.Join(lines, "\n")
}
