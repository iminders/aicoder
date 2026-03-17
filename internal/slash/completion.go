package slash

import "strings"

// CommandInfo holds metadata about a slash command.
type CommandInfo struct {
	Name        string
	Description string
	Usage       string
}

// AllCommands returns a list of all available slash commands.
func AllCommands() []CommandInfo {
	return []CommandInfo{
		{"/help", "显示帮助信息", "/help"},
		{"/clear", "清空会话上下文", "/clear"},
		{"/history", "查看对话历史摘要", "/history"},
		{"/undo", "撤销上一次文件修改", "/undo"},
		{"/diff", "查看本次会话所有文件变更", "/diff"},
		{"/commit", "Git 提交本次会话的变更", "/commit [message]"},
		{"/cost", "查看 Token 用量和费用估算", "/cost"},
		{"/model", "查看或切换 AI 模型", "/model [name]"},
		{"/config", "查看或修改配置", "/config [set key value]"},
		{"/init", "在当前目录初始化 AICODER.md", "/init"},
		{"/sessions", "列出历史会话", "/sessions"},
		{"/save", "保存当前会话到文件", "/save [filename]"},
		{"/tools", "列出可用工具", "/tools"},
		{"/exit", "退出程序", "/exit"},
		{"/quit", "退出程序 (同 /exit)", "/quit"},
		{"/q", "退出程序 (同 /exit)", "/q"},
	}
}

// Complete returns command suggestions for the given input prefix.
// Returns a list of matching commands and their descriptions.
func Complete(input string) []CommandInfo {
	if !strings.HasPrefix(input, "/") {
		return nil
	}

	input = strings.ToLower(input)
	var matches []CommandInfo

	for _, cmd := range AllCommands() {
		if strings.HasPrefix(cmd.Name, input) {
			matches = append(matches, cmd)
		}
	}

	return matches
}

// CompleteNames returns just the command names for completion.
func CompleteNames(input string) []string {
	matches := Complete(input)
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = m.Name
	}
	return names
}
