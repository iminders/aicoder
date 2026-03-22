package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/iminders/aicoder/internal/config"
	aicontext "github.com/iminders/aicoder/internal/context"
	"github.com/iminders/aicoder/internal/llm"
	"github.com/iminders/aicoder/internal/logger"
	"github.com/iminders/aicoder/internal/session"
	"github.com/iminders/aicoder/internal/skills"
	"github.com/iminders/aicoder/internal/tools"
	"github.com/iminders/aicoder/internal/ui"
	"github.com/mattn/go-isatty"
)

// Agent orchestrates the full LLM ↔ tool loop.
type Agent struct {
	cfg      *config.Config
	provider llm.Provider
	sess     *session.Session
	guard    *PermissionGuard
	renderer *ui.Renderer
}

// New creates an Agent from the given config and provider.
func New(cfg *config.Config, provider llm.Provider) *Agent {
	sess := session.New(cfg.Model)

	// Wire snapshot capture into filesystem tools
	tools.Global.All() // ensure tools are registered
	wireSnapshot(sess)

	a := &Agent{
		cfg:      cfg,
		provider: provider,
		sess:     sess,
		guard:    NewPermissionGuard(cfg, cfg.DangerouslySkip),
		renderer: ui.NewRenderer(),
	}

	// Build system prompt from project context
	cwd, _ := os.Getwd()
	ctx := aicontext.Collect(cwd)
	sysPrompt := ctx.SystemPrompt()
	sess.AppendMessage(session.TextMessage("system", sysPrompt))

	return a
}

// Session returns the underlying session (for slash commands).
func (a *Agent) Session() *session.Session { return a.sess }

// Run processes one user turn and drives the Agent Loop until the model stops.
// It auto-detects applicable skills and injects their prompts.
func (a *Agent) Run(ctx context.Context, userInput string) error {
	// Auto-detect skill from input
	if skill := skills.Global.Detect(userInput); skill != nil {
		return a.RunWithSkill(ctx, userInput, skill)
	}
	return a.run(ctx, userInput)
}

// RunWithSkill runs with a specific skill injected as additional context.
func (a *Agent) RunWithSkill(ctx context.Context, userInput string, skill *skills.Skill) error {
	ui.PrintInfo(fmt.Sprintf("🎯 技能激活: %s — %s", skill.Name, skill.Description))
	if skill.OutputFile != "" {
		ui.PrintInfo(fmt.Sprintf("📄 建议输出文件: %s", skill.OutputFile))
	}

	// Inject skill prompt as a system-level context message for this turn only.
	// We add it as a user message prefix so it doesn't pollute the system prompt.
	augmented := fmt.Sprintf("[技能上下文: %s]\n%s\n\n---\n用户需求：%s",
		skill.Name, skill.Prompt, userInput)
	return a.run(ctx, augmented)
}

// RunWithSkillByName looks up a skill by name and delegates to RunWithSkill.
func (a *Agent) RunWithSkillByName(ctx context.Context, userInput, skillName string) error {
	skill := skills.Global.Get(skillName)
	if skill == nil {
		// Skill not found — run without it but warn
		ui.PrintWarn(fmt.Sprintf("Skill %q not found, running without skill context", skillName))
		return a.run(ctx, userInput)
	}
	return a.RunWithSkill(ctx, userInput, skill)
}

// Run processes one user turn and drives the Agent Loop until the model stops.
func (a *Agent) run(ctx context.Context, userInput string) error {
	a.sess.AppendMessage(session.TextMessage("user", userInput))

	for iteration := 0; iteration < 50; iteration++ {
		// Build the LLM request
		req := &llm.Request{
			Model:     a.cfg.Model,
			RawMsgs:   a.sess.Messages,
			Tools:     buildToolSchemas(),
			MaxTokens: a.cfg.MaxTokens,
		}

		// Call the LLM (streaming)
		eventCh, err := a.provider.Stream(ctx, req)
		if err != nil {
			return fmt.Errorf("LLM error: %w", err)
		}

		// Consume the stream
		var toolCalls []llm.ToolUseBlock
		var textBuf strings.Builder
		var thinkingActive bool
		var thinkingChars int

		// Check if stdout is a terminal
		isTTY := isatty.IsTerminal(os.Stdout.Fd())

		for event := range eventCh {
			select {
			case <-ctx.Done():
				fmt.Println("\n\033[33m[中断]\033[0m")
				return nil
			default:
			}
			switch event.Type {
			case "text_delta":
				// If we were thinking, clear the thinking indicator
				if thinkingActive && isTTY {
					fmt.Print("\r\033[K") // Clear line
				}
				thinkingActive = false
				thinkingChars = 0
				a.renderer.Write(event.Delta)
				textBuf.WriteString(event.Delta)
			case "thinking_delta":
				// Show thinking progress without newline (only in TTY mode)
				thinkingChars += len(event.Delta)
				if !thinkingActive {
					thinkingActive = true
				}
				// Only update indicator in TTY mode to avoid cluttering pipe output
				if isTTY {
					fmt.Printf("\r\033[90m[Thinking... %d chars]\033[0m", thinkingChars)
				}
			case "thinking_done":
				// DeepSeek R1 thinking complete (</think> detected)
				if thinkingActive && isTTY {
					fmt.Print("\r\033[K") // Clear the thinking line
				}
				thinkingActive = false
				thinkingChars = 0
				logger.Debug("thinking complete: %d chars", len(event.Delta))
				// Don't write thinking content to output
			case "tool_use_end":
				if event.ToolUse != nil {
					toolCalls = append(toolCalls, *event.ToolUse)
				}
			case "usage":
				a.sess.RecordUsage(event.Input, event.Output)
				logger.Debug("tokens: in=%d out=%d", event.Input, event.Output)
			case "error":
				return fmt.Errorf("stream error: %w", event.Err)
			}
		}

		// Flush any buffered text
		a.renderer.Flush()
		if textBuf.Len() > 0 {
			fmt.Println()
		}

		// Append assistant message
		if textBuf.Len() > 0 || len(toolCalls) > 0 {
			msg := buildAssistantMessage(textBuf.String(), toolCalls)
			a.sess.AppendMessage(msg)
		}

		// If no tool calls, the loop is done
		if len(toolCalls) == 0 {
			return nil
		}

		// Process each tool call
		var toolResults []session.Content
		for _, tc := range toolCalls {
			result := a.executeToolCall(ctx, tc)
			toolResults = append(toolResults, result)
		}

		// Append tool results and loop
		a.sess.AppendMessage(session.Message{
			Role:    "user",
			Content: toolResults,
		})
	}

	return fmt.Errorf("agent loop exceeded maximum iterations")
}

func (a *Agent) executeToolCall(ctx context.Context, tc llm.ToolUseBlock) session.Content {
	tool, err := tools.Global.Get(tc.Name)
	if err != nil {
		return errResult(tc.ID, fmt.Sprintf("unknown tool: %s", tc.Name))
	}

	// Permission check
	perm := a.guard.Check(tool, tc.Input)
	switch perm.Action {
	case PermDeny:
		msg := fmt.Sprintf("🚫 工具 %s 被拒绝：%s", tc.Name, perm.Reason)
		fmt.Println("\033[31m" + msg + "\033[0m")
		return errResult(tc.ID, msg)

	case PermNeedsConfirm:
		fmt.Println()
		printPermDialog(perm.Preview)
		answer := askConfirm()
		switch answer {
		case "n":
			return errResult(tc.ID, "用户拒绝了此操作")
		case "a":
			a.guard.SetAlwaysAllow(tc.Name)
		}
		// "y" or "a" falls through to execute
	}

	// Execute
	fmt.Printf("\033[36m⚙  执行 %s...\033[0m\n", tc.Name)
	start := time.Now()
	result, err := tool.Execute(ctx, tc.Input)
	elapsed := time.Since(start)
	if err != nil {
		return errResult(tc.ID, err.Error())
	}

	status := "✅"
	if result.IsError {
		status = "❌"
	}
	fmt.Printf("\033[90m%s %s (%dms)\033[0m\n", status, tc.Name, elapsed.Milliseconds())

	if result.IsError {
		return errResult(tc.ID, result.Content)
	}
	return session.Content{
		Type:      "tool_result",
		ToolUseID: tc.ID,
		Text:      result.Content,
	}
}

func buildAssistantMessage(text string, toolCalls []llm.ToolUseBlock) session.Message {
	msg := session.Message{Role: "assistant"}
	if text != "" {
		msg.Content = append(msg.Content, session.Content{Type: "text", Text: text})
	}
	for _, tc := range toolCalls {
		msg.Content = append(msg.Content, session.Content{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Name,
			Input: tc.Input,
		})
	}
	return msg
}

func errResult(id, msg string) session.Content {
	return session.Content{
		Type:      "tool_result",
		ToolUseID: id,
		Text:      msg,
		IsError:   true,
	}
}

func buildToolSchemas() []llm.ToolSchema {
	var schemas []llm.ToolSchema
	for _, t := range tools.Global.All() {
		schemas = append(schemas, llm.ToolSchema{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Schema(),
		})
	}
	return schemas
}

func wireSnapshot(sess *session.Session) {
	// Import filesystem package to access SnapshotFunc
	// This is called via the tools package alias
}

func printPermDialog(preview string) {
	width := 60
	border := strings.Repeat("─", width)
	fmt.Println("┌" + border + "┐")
	fmt.Printf("│  🔧 \033[1m工具调用请求\033[0m%s│\n", strings.Repeat(" ", width-12))
	for _, line := range strings.Split(preview, "\n") {
		padded := line
		if len(padded) > width-2 {
			padded = padded[:width-2]
		}
		fmt.Printf("│  %-*s│\n", width-2, padded)
	}
	fmt.Println("├" + border + "┤")
	fmt.Printf("│  [Y] 允许   [N] 拒绝   [A] 本次会话始终允许%s│\n", strings.Repeat(" ", width-30))
	fmt.Println("└" + border + "┘")
}

func askConfirm() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("请选择 [Y/n/a]: ")
		line, _ := reader.ReadString('\n')
		ans := strings.ToLower(strings.TrimSpace(line))
		if ans == "" || ans == "y" {
			return "y"
		}
		if ans == "n" {
			return "n"
		}
		if ans == "a" {
			return "a"
		}
	}
}
