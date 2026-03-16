package slash

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourorg/aicoder/internal/config"
	"github.com/yourorg/aicoder/internal/session"
	"github.com/yourorg/aicoder/internal/ui"
	"github.com/yourorg/aicoder/pkg/diff"
	"github.com/yourorg/aicoder/pkg/version"
)

// Handler processes a slash command string. Returns true if the program should exit.
type Handler struct {
	sess *session.Session
	cfg  *config.Config
}

func NewHandler(sess *session.Session, cfg *config.Config) *Handler {
	return &Handler{sess: sess, cfg: cfg}
}

// Handle dispatches a slash command. Returns (handled, shouldExit).
func (h *Handler) Handle(input string) (handled bool, shouldExit bool) {
	parts := strings.Fields(input)
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "/") {
		return false, false
	}
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/help":
		h.cmdHelp()
	case "/exit", "/quit", "/q":
		return true, true
	case "/clear":
		h.cmdClear()
	case "/history":
		h.cmdHistory()
	case "/undo":
		h.cmdUndo()
	case "/diff":
		h.cmdDiff()
	case "/commit":
		h.cmdCommit(args)
	case "/cost":
		h.cmdCost()
	case "/model":
		h.cmdModel(args)
	case "/config":
		h.cmdConfig(args)
	case "/init":
		h.cmdInit()
	default:
		ui.PrintWarn(fmt.Sprintf("未知命令: %s  (输入 /help 查看所有命令)", cmd))
	}
	return true, false
}

func (h *Handler) cmdHelp() {
	help := `
┌─────────────────────────────────────────────────────────────────┐
│                    aicoder 命令帮助                              │
├──────────────┬──────────────────────────────────────────────────┤
│ /help        │ 显示此帮助信息                                    │
│ /clear       │ 清空当前会话上下文                                │
│ /history     │ 查看对话历史摘要                                  │
│ /undo        │ 撤销上一次文件修改                                │
│ /diff        │ 查看本次会话所有文件变更                          │
│ /commit [msg]│ Git 提交本次会话的变更                            │
│ /cost        │ 查看 Token 用量和费用估算                         │
│ /model [m]   │ 查看或切换 AI 模型                               │
│ /config      │ 查看当前配置                                      │
│ /init        │ 在当前目录初始化 AICODER.md                       │
│ /exit        │ 退出程序                                          │
└──────────────┴──────────────────────────────────────────────────┘`
	fmt.Println(help)
}

func (h *Handler) cmdClear() {
	h.sess.ClearMessages()
	ui.PrintSuccess("会话上下文已清空（系统提示词保留）")
}

func (h *Handler) cmdHistory() {
	msgs := h.sess.Messages
	if len(msgs) == 0 {
		ui.PrintInfo("暂无对话历史")
		return
	}
	fmt.Printf("\033[1m对话历史 (%d 条消息):\033[0m\n", len(msgs))
	ui.PrintDivider()
	for i, m := range msgs {
		if m.Role == "system" {
			continue
		}
		role := m.Role
		icon := "👤"
		color := "\033[0m"
		if role == "assistant" {
			icon = "🤖"
			color = "\033[36m"
		}
		var preview string
		for _, c := range m.Content {
			if c.Type == "text" && len(c.Text) > 0 {
				preview = c.Text
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				break
			}
			if c.Type == "tool_use" {
				preview = fmt.Sprintf("[工具调用: %s]", c.Name)
				break
			}
		}
		fmt.Printf("%s[%d] %s %s\033[0m\n", color, i, icon, preview)
	}
	ui.PrintDivider()
}

func (h *Handler) cmdUndo() {
	snap, err := h.sess.Undo()
	if err != nil {
		ui.PrintError(err.Error())
		return
	}
	if snap.Before == nil {
		ui.PrintSuccess(fmt.Sprintf("撤销：已删除 %s（文件原本不存在）", snap.FilePath))
	} else {
		ui.PrintSuccess(fmt.Sprintf("撤销：%s 已恢复到修改前状态", snap.FilePath))
	}
}

func (h *Handler) cmdDiff() {
	changes := h.sess.AllFileChanges()
	if len(changes) == 0 {
		ui.PrintInfo("本次会话暂无文件变更")
		return
	}
	fmt.Printf("\033[1m本次会话文件变更 (%d 个文件):\033[0m\n", len(changes))
	ui.PrintDivider()
	for path, after := range changes {
		before, _ := os.ReadFile(path)
		d := diff.ColorDiff(string(before), string(after), path)
		if d != "" {
			fmt.Print(d)
		}
	}
}

func (h *Handler) cmdCommit(args []string) {
	// First check if we're in a git repo
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		ui.PrintError("当前目录不是 Git 仓库")
		return
	}
	// Stage all changes
	if out, err := exec.Command("git", "add", "-A").CombinedOutput(); err != nil {
		ui.PrintError("git add 失败: " + string(out))
		return
	}
	msg := strings.Join(args, " ")
	if msg == "" {
		msg = fmt.Sprintf("aicoder: session changes %s", time.Now().Format("2006-01-02 15:04"))
	}
	out, err := exec.Command("git", "commit", "-m", msg).CombinedOutput()
	if err != nil {
		ui.PrintError("git commit 失败: " + string(out))
		return
	}
	ui.PrintSuccess("已提交: " + msg)
	fmt.Println(string(out))
}

func (h *Handler) cmdCost() {
	usage := h.sess.Usage
	model := h.sess.Model
	est := usage.CostEstimate(model)
	ui.PrintDivider()
	fmt.Printf("  \033[1m模型:\033[0m      %s\n", model)
	fmt.Printf("  \033[1m输入 tokens:\033[0m %d\n", usage.InputTokens)
	fmt.Printf("  \033[1m输出 tokens:\033[0m %d\n", usage.OutputTokens)
	fmt.Printf("  \033[1m费用估算:\033[0m   $%.4f USD\n", est)
	ui.PrintDivider()
}

func (h *Handler) cmdModel(args []string) {
	if len(args) == 0 {
		fmt.Printf("当前模型: \033[1m%s\033[0m\n", h.sess.Model)
		fmt.Println("可用模型示例:")
		models := []string{
			"claude-opus-4-5", "claude-sonnet-4-5", "claude-haiku-4-5-20251001",
			"gpt-4o", "gpt-4o-mini",
		}
		for _, m := range models {
			fmt.Printf("  - %s\n", m)
		}
		fmt.Println("用法: /model <model-name>")
		return
	}
	newModel := args[0]
	h.sess.Model = newModel
	h.cfg.Model = newModel
	ui.PrintSuccess(fmt.Sprintf("模型已切换为: %s", newModel))
}

func (h *Handler) cmdConfig(args []string) {
	_ = args
	fmt.Printf("\033[1m当前配置:\033[0m\n")
	ui.PrintDivider()
	fmt.Printf("  provider:          %s\n", h.cfg.Provider)
	fmt.Printf("  model:             %s\n", h.cfg.Model)
	fmt.Printf("  maxTokens:         %d\n", h.cfg.MaxTokens)
	fmt.Printf("  autoApprove:       %v\n", h.cfg.AutoApprove)
	fmt.Printf("  autoApproveReads:  %v\n", h.cfg.AutoApproveReads)
	fmt.Printf("  backupOnWrite:     %v\n", h.cfg.BackupOnWrite)
	fmt.Printf("  theme:             %s\n", h.cfg.Theme)
	fmt.Printf("  language:          %s\n", h.cfg.Language)
	if h.cfg.Proxy != "" {
		fmt.Printf("  proxy:             %s\n", h.cfg.Proxy)
	}
	ui.PrintDivider()
}

func (h *Handler) cmdInit() {
	path := filepath.Join("AICODER.md")
	if _, err := os.Stat(path); err == nil {
		ui.PrintWarn("AICODER.md 已存在，跳过初始化")
		return
	}
	template := fmt.Sprintf(`# 项目说明
<!-- 在此描述项目的整体背景和目标 -->

# 代码规范
<!-- 列出代码风格、命名规范、测试要求等 -->

# 常用命令
<!-- 列出常用的构建、测试、运行命令 -->

# 注意事项
<!-- 描述需要特别注意的事项，例如禁止修改的文件、特殊依赖等 -->

_由 aicoder v%s 生成于 %s_
`, version.Version, time.Now().Format("2006-01-02"))

	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		ui.PrintError("创建 AICODER.md 失败: " + err.Error())
		return
	}
	ui.PrintSuccess("已创建 AICODER.md，请编辑它来描述您的项目")
}
