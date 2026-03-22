package context

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProjectContext holds auto-discovered project information.
type ProjectContext struct {
	RootDir       string
	AICoderMD     string // contents of .AICODER.md if found
	GitInfo       string
	ProjectInfo   string
	DirectoryTree string // directory structure summary
}

// Collect gathers project context starting from the given directory.
func Collect(startDir string) *ProjectContext {
	ctx := &ProjectContext{RootDir: startDir}
	ctx.AICoderMD = findAICoderMD(startDir)
	ctx.GitInfo = collectGit(startDir)
	ctx.ProjectInfo = detectProject(startDir)
	ctx.DirectoryTree = SummarizeDirectoryCompact(startDir)
	return ctx
}

// SystemPrompt builds the complete system prompt from collected context.
func (c *ProjectContext) SystemPrompt() string {
	var parts []string

	parts = append(parts, `你是 aicoder，一个专业的 AI 编程助手，运行在用户的终端环境中。
你可以读写文件、执行命令、搜索代码，帮助用户完成各种软件工程任务。
始终用简洁、准确的中文或英文（跟随用户语言）回复。
修改文件前先解释你的计划，执行命令前确认它是安全的。`)

	if c.AICoderMD != "" {
		parts = append(parts, "\n## 项目说明 (AICODER.md)\n"+c.AICoderMD)
	}
	if c.ProjectInfo != "" {
		parts = append(parts, "\n## 项目环境\n"+c.ProjectInfo)
	}
	if c.DirectoryTree != "" {
		parts = append(parts, "\n## "+c.DirectoryTree)
	}
	if c.GitInfo != "" {
		parts = append(parts, "\n## Git 状态\n"+c.GitInfo)
	}

	return strings.Join(parts, "\n")
}

func findAICoderMD(startDir string) string {
	dir := startDir
	for {
		path := filepath.Join(dir, ".AICODER.md")
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func collectGit(dir string) string {
	cmds := [][]string{
		{"git", "rev-parse", "--abbrev-ref", "HEAD"},
		{"git", "status", "--short"},
		{"git", "log", "--oneline", "-5"},
	}
	labels := []string{"Branch", "Status", "Recent commits"}
	var parts []string
	for i, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			parts = append(parts, fmt.Sprintf("**%s:**\n```\n%s```", labels[i], string(out)))
		}
	}
	return strings.Join(parts, "\n")
}

func detectProject(dir string) string {
	type check struct {
		file  string
		label string
	}
	checks := []check{
		{"go.mod", "Go"},
		{"package.json", "Node.js"},
		{"pyproject.toml", "Python"},
		{"Cargo.toml", "Rust"},
		{"pom.xml", "Java (Maven)"},
		{"build.gradle", "Java (Gradle)"},
		{"Gemfile", "Ruby"},
	}
	var found []string
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			data, _ := os.ReadFile(filepath.Join(dir, c.file))
			found = append(found, fmt.Sprintf("- %s (`%s` detected)\n  ```\n  %s\n  ```", c.label, c.file, truncate(string(data), 300)))
		}
	}
	if len(found) == 0 {
		return ""
	}
	return strings.Join(found, "\n")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
