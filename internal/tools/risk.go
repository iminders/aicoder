package tools

import (
	"path/filepath"
	"strings"
)

// ClassifyCommandRisk 分析命令字符串的风险等级
// 基于命令内容、参数和潜在影响进行评估
func ClassifyCommandRisk(cmd string) RiskLevel {
	cmdLower := strings.ToLower(strings.TrimSpace(cmd))

	// Critical: 极度危险的命令
	criticalPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=",
		":(){:|:&};:", // fork bomb
		"chmod -r 777 /",
		"chmod 777 /",
		"> /dev/sda",
		"format c:",
		"del /f /s /q c:\\",
	}

	for _, pattern := range criticalPatterns {
		if strings.Contains(cmdLower, strings.ToLower(pattern)) {
			return RiskHigh // 使用 RiskHigh 作为最高级别
		}
	}

	// High: 高风险命令
	highRiskKeywords := []string{
		"sudo",
		"rm -rf",
		"rm -r",
		"rmdir /s",
		"del /f",
		"format",
		"fdisk",
		"parted",
		"kill -9",
		"killall",
		"reboot",
		"shutdown",
		"halt",
		"poweroff",
		"init 0",
		"init 6",
		"systemctl stop",
		"systemctl disable",
		"service stop",
		"chown -r",
		"chmod -r",
	}

	for _, keyword := range highRiskKeywords {
		if strings.Contains(cmdLower, keyword) {
			return RiskHigh
		}
	}

	// Medium: 中等风险命令
	mediumRiskKeywords := []string{
		"rm ",
		"mv ",
		"cp -r",
		"wget",
		"curl",
		"git push",
		"git reset --hard",
		"npm install",
		"pip install",
		"apt install",
		"yum install",
		"brew install",
		"docker run",
		"docker rm",
		"make install",
		"chmod",
		"chown",
		"chgrp",
	}

	for _, keyword := range mediumRiskKeywords {
		if strings.Contains(cmdLower, keyword) {
			return RiskMedium
		}
	}

	// Low: 低风险命令（只读操作）
	lowRiskKeywords := []string{
		"ls",
		"cat",
		"echo",
		"pwd",
		"cd",
		"git status",
		"git log",
		"git diff",
		"git show",
		"ps",
		"top",
		"df",
		"du",
		"find",
		"grep",
		"awk",
		"sed",
		"head",
		"tail",
		"wc",
		"sort",
		"uniq",
		"which",
		"whereis",
		"man",
		"help",
		"env",
		"printenv",
		"date",
		"cal",
		"uptime",
		"whoami",
		"hostname",
	}

	for _, keyword := range lowRiskKeywords {
		if strings.HasPrefix(cmdLower, keyword) {
			return RiskLow
		}
	}

	// 默认: 未知命令视为中等风险
	return RiskMedium
}

// ClassifyToolRisk 基于工具名称和参数分析风险等级
// 这个函数可以根据工具的具体参数进行更精细的风险评估
func ClassifyToolRisk(toolName string, params map[string]interface{}) RiskLevel {
	switch toolName {
	// 文件系统工具
	case "read_file", "list_dir", "search_files":
		// 只读操作,低风险
		return RiskLow

	case "write_file", "edit_file":
		// 写操作,检查目标路径
		if path, ok := params["path"].(string); ok {
			if IsPathDangerous(path) {
				return RiskHigh
			}
		}
		return RiskMedium

	case "delete_file":
		// 删除操作,检查目标路径
		if path, ok := params["path"].(string); ok {
			if IsPathDangerous(path) {
				return RiskHigh
			}
			// 检查是否删除目录
			if strings.HasSuffix(path, "/") || !strings.Contains(path, ".") {
				return RiskHigh // 可能是目录
			}
		}
		return RiskHigh // 删除操作默认高风险

	// Shell 命令工具
	case "run_command":
		// 分析命令内容
		if cmd, ok := params["command"].(string); ok {
			return ClassifyCommandRisk(cmd)
		}
		return RiskMedium

	case "run_background":
		// 后台进程,分析命令内容
		if cmd, ok := params["command"].(string); ok {
			risk := ClassifyCommandRisk(cmd)
			// 后台进程风险提升一级
			if risk == RiskLow {
				return RiskMedium
			}
			return risk
		}
		return RiskMedium

	// 搜索工具
	case "grep_search", "ast_search", "web_search":
		// 搜索操作,低风险
		return RiskLow

	default:
		// 未知工具,默认中等风险
		return RiskMedium
	}
}

// IsPathDangerous 检查路径是否指向危险位置
// 返回 true 表示路径危险,应该阻止或提高风险等级
func IsPathDangerous(path string) bool {
	// 规范化路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		// 无法解析路径,视为危险
		return true
	}

	// 危险路径前缀列表
	dangerousPaths := []string{
		"/etc/shadow",
		"/etc/passwd",
		"/etc/sudoers",
		"/etc/ssh",
		"/root",
		"/boot",
		"/sys",
		"/proc",
		"/dev",
		"/bin",
		"/sbin",
		"/usr/bin",
		"/usr/sbin",
		"/lib",
		"/lib64",
		"/var/log",
		"/var/run",
		"/.ssh",
		"/.gnupg",
		"/.aws",
		"/.kube",
	}

	// 检查是否匹配危险路径
	for _, dangerous := range dangerousPaths {
		// 处理用户主目录
		if strings.HasPrefix(dangerous, "/~") {
			// 跳过,由 checkSandbox 处理
			continue
		}

		// 检查是否在危险路径下
		if strings.HasPrefix(absPath, dangerous) {
			return true
		}
	}

	// 检查是否是根目录或系统关键目录
	if absPath == "/" || absPath == "/etc" || absPath == "/var" {
		return true
	}

	// 检查是否包含危险模式
	dangerousPatterns := []string{
		"*",      // 通配符
		"../",    // 路径遍历
		"..\\",   // Windows 路径遍历
		"/./",    // 当前目录引用
		"//",     // 双斜杠
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

// EvaluateOperationRisk 综合评估操作的风险等级
// 考虑工具类型、参数、操作范围等多个因素
func EvaluateOperationRisk(toolName string, params map[string]interface{}) RiskLevel {
	// 基础风险等级
	baseRisk := ClassifyToolRisk(toolName, params)

	// 根据操作范围调整风险
	if path, ok := params["path"].(string); ok {
		// 检查是否操作多个文件
		if strings.Contains(path, "*") || strings.Contains(path, "?") {
			// 批量操作,风险提升
			if baseRisk == RiskLow {
				baseRisk = RiskMedium
			} else if baseRisk == RiskMedium {
				baseRisk = RiskHigh
			}
		}

		// 检查是否递归操作
		if recursive, ok := params["recursive"].(bool); ok && recursive {
			// 递归操作,风险提升
			if baseRisk == RiskLow {
				baseRisk = RiskMedium
			} else if baseRisk == RiskMedium {
				baseRisk = RiskHigh
			}
		}
	}

	// 检查命令中的管道和重定向
	if cmd, ok := params["command"].(string); ok {
		if strings.Contains(cmd, "|") || strings.Contains(cmd, ">") || strings.Contains(cmd, ">>") {
			// 包含管道或重定向,风险可能更高
			if baseRisk == RiskLow {
				baseRisk = RiskMedium
			}
		}
	}

	return baseRisk
}

// GetRiskDescription 返回风险等级的详细描述
func GetRiskDescription(risk RiskLevel) string {
	switch risk {
	case RiskLow:
		return "低风险操作,通常是只读操作,不会修改系统状态"
	case RiskMedium:
		return "中等风险操作,可能修改文件或执行命令,需要用户确认"
	case RiskHigh:
		return "高风险操作,可能删除文件、修改系统配置或执行危险命令,强烈建议仔细确认"
	default:
		return "未知风险等级"
	}
}

// ShouldAutoApprove 判断操作是否应该自动批准
// 基于风险等级和配置决策
func ShouldAutoApprove(risk RiskLevel, autoApproveReads bool, autoApproveAll bool) bool {
	if autoApproveAll {
		return true
	}

	if risk == RiskLow && autoApproveReads {
		return true
	}

	return false
}
