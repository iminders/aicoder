package slash

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/iminders/aicoder/internal/config"
	"github.com/iminders/aicoder/internal/session"
	"github.com/iminders/aicoder/internal/skills"
	"github.com/iminders/aicoder/internal/ui"
	"github.com/iminders/aicoder/pkg/diff"
	"github.com/iminders/aicoder/pkg/version"
)

// Handler processes a slash command string. Returns true if the program should exit.
type Handler struct {
	sess    *session.Session
	cfg     *config.Config
	printer func(...interface{}) // For printing output (can be tea.Program.Println or fmt.Println)
}

func NewHandler(sess *session.Session, cfg *config.Config) *Handler {
	return &Handler{
		sess:    sess,
		cfg:     cfg,
		printer: func(args ...interface{}) { fmt.Println(args...) }, // Default to fmt.Println
	}
}

// SetPrinter sets the print function (use tea.Program.Println for TUI mode)
func (h *Handler) SetPrinter(printer func(...interface{})) {
	h.printer = printer
}

// println is a helper that uses the configured printer
func (h *Handler) println(args ...interface{}) {
	h.printer(args...)
}

// printf is a helper for formatted printing
func (h *Handler) printf(format string, args ...interface{}) {
	h.printer(fmt.Sprintf(format, args...))
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
	case "/sessions":
		h.cmdSessions()
	case "/save":
		h.cmdSave()
	case "/tools":
		h.cmdTools()
	case "/skill", "/skills":
		return h.cmdSkill(args)
	default:
		ui.PrintWarn(fmt.Sprintf("未知命令: %s  (输入 /help 查看所有命令)", cmd))
	}
	return true, false
}

func (h *Handler) cmdHelp() {
	help := `
┌──────────────────────────────────────────────────────────────────┐
│                     aicoder 命令帮助                              │
├───────────────┬──────────────────────────────────────────────────┤
│ /help         │ 显示此帮助信息                                    │
│ /clear        │ 清空当前会话上下文                                │
│ /history      │ 查看对话历史摘要                                  │
│ /undo         │ 撤销上一次文件修改                                │
│ /diff         │ 查看本次会话所有文件变更                          │
│ /commit [msg] │ Git 提交本次会话的变更                            │
│ /cost         │ 查看 Token 用量和费用估算                         │
│ /model [m]    │ 查看或切换 AI 模型                               │
│ /config       │ 查看当前配置                                      │
│ /init         │ 在当前目录初始化 .AICODER.md                       │
│ /sessions     │ 列出历史会话                                      │
│ /save         │ 手动保存当前会话                                  │
│ /tools        │ 列出所有可用工具                                  │
│ /skill list          │ 列出所有内置 Skill                             │
│ /skill <名称>        │ 显示 Skill 详情                               │
│ /skill <名称> <提示> │ 以指定 Skill 模式运行                         │
│ /skill new <名称>    │ 创建自定义 Skill 模板                         │
│ /exit         │ 退出程序                                          │
└───────────────┴──────────────────────────────────────────────────┘`
	h.printer(help)
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
	h.printf("\033[1m对话历史 (%d 条消息):\033[0m\n", len(msgs))
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
		h.printf("%s[%d] %s %s\033[0m\n", color, i, icon, preview)
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
	h.printf("\033[1m本次会话文件变更 (%d 个文件):\033[0m\n", len(changes))
	ui.PrintDivider()
	for path, after := range changes {
		before, _ := os.ReadFile(path)
		d := diff.ColorDiff(string(before), string(after), path)
		if d != "" {
			h.printer(d)
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
	h.println(string(out))
}

func (h *Handler) cmdCost() {
	usage := h.sess.Usage
	model := h.sess.Model
	est := usage.CostEstimate(model)
	ui.PrintDivider()
	h.printf("  \033[1m模型:\033[0m      %s\n", model)
	h.printf("  \033[1m输入 tokens:\033[0m %d\n", usage.InputTokens)
	h.printf("  \033[1m输出 tokens:\033[0m %d\n", usage.OutputTokens)
	h.printf("  \033[1m费用估算:\033[0m   $%.4f USD\n", est)
	ui.PrintDivider()
}

func (h *Handler) cmdModel(args []string) {
	if len(args) == 0 {
		h.printf("当前模型: \033[1m%s\033[0m\n", h.sess.Model)
		h.println("可用模型示例:")
		models := []string{
			"claude-opus-4-5", "claude-sonnet-4-5", "claude-haiku-4-5-20251001",
			"gpt-4o", "gpt-4o-mini",
		}
		for _, m := range models {
			h.printf("  - %s\n", m)
		}
		h.println("用法: /model <model-name>")
		return
	}
	newModel := args[0]
	h.sess.Model = newModel
	h.cfg.Model = newModel
	ui.PrintSuccess(fmt.Sprintf("模型已切换为: %s", newModel))
}

func (h *Handler) cmdConfig(args []string) {
	_ = args
	h.printf("\033[1m当前配置:\033[0m\n")
	ui.PrintDivider()
	h.printf("  provider:          %s\n", h.cfg.Provider)
	h.printf("  model:             %s\n", h.cfg.Model)
	h.printf("  maxTokens:         %d\n", h.cfg.MaxTokens)
	h.printf("  autoApprove:       %v\n", h.cfg.AutoApprove)
	h.printf("  autoApproveReads:  %v\n", h.cfg.AutoApproveReads)
	h.printf("  backupOnWrite:     %v\n", h.cfg.BackupOnWrite)
	h.printf("  theme:             %s\n", h.cfg.Theme)
	h.printf("  language:          %s\n", h.cfg.Language)
	if h.cfg.Proxy != "" {
		h.printf("  proxy:             %s\n", h.cfg.Proxy)
	}
	ui.PrintDivider()
}

func (h *Handler) cmdInit() {
	path := filepath.Join(".AICODER.md")
	if _, err := os.Stat(path); err == nil {
		ui.PrintWarn(".AICODER.md 已存在，跳过初始化")
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

# 工具使用规范
<!-- 说明如何使用 aicoder 的工具，例如文件操作、命令执行等 -->
允许使用web_search工具进行联网搜索
允许git clone 到third_party目录, 但禁止直接修改第三方代码

_由 aicoder v%s 生成于 %s_
`, version.Version, time.Now().Format("2006-01-02"))

	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		ui.PrintError("创建 .AICODER.md 失败: " + err.Error())
		return
	}
	ui.PrintSuccess("已创建 .AICODER.md，请编辑它来描述您的项目")
}
func (h *Handler) cmdSessions() {
	home, err := os.UserHomeDir()
	if err != nil {
		ui.PrintError("无法确定 home 目录: " + err.Error())
		return
	}
	dir := filepath.Join(home, ".aicoder", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		ui.PrintInfo("暂无历史会话记录 (会话在退出时自动保存)")
		return
	}

	h.printf("\033[1m历史会话 (%d 个):\033[0m\n", len(entries))
	ui.PrintDivider()

	// Show most recent 20, newest first
	start := 0
	if len(entries) > 20 {
		start = len(entries) - 20
	}
	for i := len(entries) - 1; i >= start; i-- {
		e := entries[i]
		info, _ := e.Info()
		size := ""
		if info != nil {
			size = fmt.Sprintf("%.1fKB", float64(info.Size())/1024)
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		// Try to parse timestamp from name (Unix nanoseconds)
		modTime := ""
		if info != nil {
			modTime = info.ModTime().Format("2006-01-02 15:04")
		}
		h.printf("  \033[36m%s\033[0m  %s  %s\n", name[:min(len(name), 20)], modTime, size)
	}
	ui.PrintDivider()
	h.println("  提示：会话文件保存在", dir)
}

func (h *Handler) cmdSave() {
	if err := h.sess.Save(); err != nil {
		ui.PrintError("保存会话失败: " + err.Error())
		return
	}
	ui.PrintSuccess(fmt.Sprintf("会话已保存 (ID: %s)", h.sess.ID))
}

func (h *Handler) cmdTools() {
	// Import tools package to list all registered tools
	// We use a type assertion via the session's known tool names
	h.printf("\033[1m已注册工具:\033[0m\n")
	ui.PrintDivider()

	// Tool metadata is stored in the global registry; we query it via the session
	// Since we can't import tools here (circular), we print a static summary
	rows := []struct{ name, risk, desc string }{
		{"read_file",       "低", "读取文件内容（支持行范围）"},
		{"write_file",      "中", "写入或创建文件"},
		{"edit_file",       "中", "精确替换文件中的字符串"},
		{"list_dir",        "低", "列出目录结构（树形）"},
		{"search_files",    "低", "正则搜索文件内容"},
		{"delete_file",     "高", "删除文件（不可逆）"},
		{"run_command",     "中", "执行 Shell 命令"},
		{"run_background",  "中", "后台启动长时进程"},
		{"grep_search",     "低", "全目录正则搜索"},
		{"web_search",      "低", "联网搜索"},
	}
	for _, r := range rows {
		riskColor := "\033[32m"
		if r.risk == "中" { riskColor = "\033[33m" }
		if r.risk == "高" { riskColor = "\033[31m" }
		h.printf("  %-18s %s[%s]\033[0m  %s\n", r.name, riskColor, r.risk, r.desc)
	}
	ui.PrintDivider()
	h.println("  MCP 工具以 <server>__<tool> 格式列出（连接后可见）")
}

func min(a, b int) int {
	if a < b { return a }
	return b
}



// cmdSkill handles: /skill list | /skill <name> | /skill <name> <prompt> | /skill new <name>
// It returns (handled bool, shouldExit bool) so the caller can optionally
// hand off to the agent with a skill override.
func (h *Handler) cmdSkill(args []string) (bool, bool) {

	if len(args) == 0 || args[0] == "list" {
		h.printSkillList()
		return true, false
	}

	if args[0] == "new" {
		if len(args) < 2 {
			ui.PrintWarn("用法: /skill new <名称>")
			return true, false
		}
		h.createUserSkill(args[1])
		return true, false
	}

	// /skill <name> [optional prompt...]
	skillName := args[0]
	sk := skills.Global.Get(skillName)
	if sk == nil {
		ui.PrintError(fmt.Sprintf("未找到 Skill %q，输入 /skill list 查看所有可用 Skill", skillName))
		return true, false
	}

	if len(args) == 1 {
		// Show skill details
		h.printSkillDetail(sk)
		return true, false
	}

	// /skill <name> <user prompt> — signal caller to run agent with this skill
	// We store the pending skill+prompt in session metadata and return a special
	// sentinel so the interactive loop can handle it.
	prompt := strings.Join(args[1:], " ")
	fmt.Printf("\033[90m[Skill %q 已激活，正在处理: %s]\033[0m\n", sk.Name, prompt)
	// Inject skill directly — store on session for the loop to pick up
	h.sess.PendingSkillName = sk.Name
	h.sess.PendingPrompt = prompt
	return true, false
}

func (h *Handler) printSkillList() {
	all := skills.Global.All()
	fmt.Printf("\033[1m内置 Skill (%d 个):\033[0m\n", len(all))
	ui.PrintDivider()
	for _, s := range all {
		tag := "\033[34m[内置]\033[0m"
		if strings.HasPrefix(s.Name, "user:") {
			tag = "\033[32m[自定义]\033[0m"
		}
		outFile := ""
		if s.OutputFile != "" {
			outFile = fmt.Sprintf("  \033[90m→ %s\033[0m", s.OutputFile)
		}
		fmt.Printf("  %-12s %s  %s%s\n", s.Name, tag, s.Description, outFile)
	}
	ui.PrintDivider()
	fmt.Println("  用法: /skill <名称> <你的需求描述>")
	fmt.Println("  示例: /skill prd 电商平台用户评价系统")
	fmt.Println("  自动触发: 直接描述需求，aicoder 会自动匹配合适的 Skill")
}

func (h *Handler) printSkillDetail(sk *skills.Skill) {
	fmt.Printf("\n\033[1m🎯 Skill: %s\033[0m\n", sk.Name)
	ui.PrintDivider()
	fmt.Printf("  \033[1m描述:\033[0m    %s\n", sk.Description)
	if len(sk.Aliases) > 0 {
		fmt.Printf("  \033[1m别名:\033[0m    %s\n", strings.Join(sk.Aliases, ", "))
	}
	if len(sk.Triggers) > 0 {
		fmt.Printf("  \033[1m触发词:\033[0m  %s\n", strings.Join(sk.Triggers[:min(3, len(sk.Triggers))], " | "))
	}
	if sk.OutputFile != "" {
		fmt.Printf("  \033[1m输出文件:\033[0m %s\n", sk.OutputFile)
	}
	ui.PrintDivider()
	// Show first 10 lines of the prompt as preview
	lines := strings.Split(sk.Prompt, "\n")
	preview := lines
	truncated := false
	if len(lines) > 12 {
		preview = lines[:12]
		truncated = true
	}
	fmt.Println("\033[90m" + strings.Join(preview, "\n") + "\033[0m")
	if truncated {
		fmt.Printf("\033[90m... (共 %d 行) ...\033[0m\n", len(lines))
	}
	ui.PrintDivider()
	fmt.Printf("  运行: /skill %s <你的需求描述>\n\n", sk.Name)
}

func (h *Handler) createUserSkill(name string) {
	home, err := os.UserHomeDir()
	if err != nil {
		ui.PrintError("无法获取 home 目录: " + err.Error())
		return
	}
	dir := filepath.Join(home, ".aicoder", "skills")
	if err := os.MkdirAll(dir, 0700); err != nil {
		ui.PrintError("创建目录失败: " + err.Error())
		return
	}
	path := filepath.Join(dir, name+".md")
	if _, err := os.Stat(path); err == nil {
		ui.PrintWarn(fmt.Sprintf("Skill %q 已存在: %s", name, path))
		return
	}

	template := fmt.Sprintf(`---
name: %s
aliases: []
description: 在这里填写 Skill 的一句话描述
triggers:
  - 触发关键词1
  - 触发关键词2
output_file: output.md
---

# Skill: %s

在这里描述这个 Skill 的职责和使用场景。

## 输出结构

1. **章节一** — 说明
2. **章节二** — 说明

## 写作规范

- 规范1
- 规范2

## 操作方式

1. 先用工具收集信息
2. 按结构生成文档
3. 保存到输出文件

_由 aicoder v%s 生成于 %s_
`, name, name, version.Version, time.Now().Format("2006-01-02"))

	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		ui.PrintError("创建 Skill 失败: " + err.Error())
		return
	}
	ui.PrintSuccess(fmt.Sprintf("已创建自定义 Skill 模板: %s", path))
	fmt.Println("  请编辑该文件，然后重启 aicoder 或输入 /skill list 刷新")
}
