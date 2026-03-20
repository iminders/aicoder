package context

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SummarizerConfig 配置目录摘要生成器
type SummarizerConfig struct {
	MaxDepth      int      // 最大递归深度
	IgnoreDirs    []string // 忽略的目录名
	IgnoreFiles   []string // 忽略的文件名模式
	MaxFiles      int      // 每个目录最多显示的文件数
	MaxTotalFiles int      // 总共最多显示的文件数
}

// DefaultSummarizerConfig 返回默认配置
func DefaultSummarizerConfig() *SummarizerConfig {
	return &SummarizerConfig{
		MaxDepth: 3,
		IgnoreDirs: []string{
			".git",
			"node_modules",
			"vendor",
			".aicoder",
			"__pycache__",
			".pytest_cache",
			".venv",
			"venv",
			"dist",
			"build",
			"target",
			".idea",
			".vscode",
			".DS_Store",
		},
		IgnoreFiles: []string{
			".gitignore",
			".DS_Store",
			"*.pyc",
			"*.pyo",
			"*.so",
			"*.dylib",
			"*.dll",
			"*.exe",
			"*.o",
			"*.a",
		},
		MaxFiles:      20,
		MaxTotalFiles: 100,
	}
}

// DirectoryNode 表示目录树的一个节点
type DirectoryNode struct {
	Name     string
	IsDir    bool
	Children []*DirectoryNode
	Depth    int
}

// SummarizeDirectory 生成目录结构摘要
// 返回树形结构的字符串表示
func SummarizeDirectory(rootPath string, config *SummarizerConfig) string {
	if config == nil {
		config = DefaultSummarizerConfig()
	}

	// 构建目录树
	root := buildDirectoryTree(rootPath, config, 0)
	if root == nil {
		return "无法读取目录结构"
	}

	// 渲染为字符串
	var sb strings.Builder
	// sb.WriteString("项目结构(当前根目录为" + filepath.Base(rootPath) + "):\n")
	sb.WriteString("项目名称为：" + filepath.Base(rootPath) + "，项目结构:\n")
	totalFiles := 0
	renderTree(&sb, root, "", true, &totalFiles)

	return sb.String()
}

// buildDirectoryTree 递归构建目录树
func buildDirectoryTree(path string, config *SummarizerConfig, depth int) *DirectoryNode {
	// 检查深度限制
	if depth > config.MaxDepth {
		return nil
	}

	// 读取目录信息
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	// 创建节点
	node := &DirectoryNode{
		Name:     filepath.Base(path),
		IsDir:    info.IsDir(),
		Depth:    depth,
		Children: []*DirectoryNode{},
	}

	// 如果是文件,直接返回
	if !info.IsDir() {
		return node
	}

	// 读取目录内容
	entries, err := os.ReadDir(path)
	if err != nil {
		return node
	}

	// 分离目录和文件
	var dirs []fs.DirEntry
	var files []fs.DirEntry

	for _, entry := range entries {
		name := entry.Name()

		// 检查是否应该忽略
		if shouldIgnore(name, entry.IsDir(), config) {
			continue
		}

		if entry.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	// 排序
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name() < dirs[j].Name()
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// 递归处理子目录
	for _, dir := range dirs {
		childPath := filepath.Join(path, dir.Name())
		childNode := buildDirectoryTree(childPath, config, depth+1)
		if childNode != nil {
			node.Children = append(node.Children, childNode)
		}
	}

	// 添加文件 (限制数量)
	fileCount := 0
	for _, file := range files {
		if fileCount >= config.MaxFiles {
			// 添加省略标记
			node.Children = append(node.Children, &DirectoryNode{
				Name:  "...",
				IsDir: false,
				Depth: depth + 1,
			})
			break
		}

		node.Children = append(node.Children, &DirectoryNode{
			Name:  file.Name(),
			IsDir: false,
			Depth: depth + 1,
		})
		fileCount++
	}

	return node
}

// shouldIgnore 检查是否应该忽略该文件或目录
func shouldIgnore(name string, isDir bool, config *SummarizerConfig) bool {
	// 重要的配置文件不应该被忽略
	importantFiles := []string{
		".gitignore",
		".env.example",
		".dockerignore",
	}

	for _, important := range importantFiles {
		if name == important {
			return false
		}
	}

	// 检查隐藏文件 (以 . 开头)
	if strings.HasPrefix(name, ".") {
		return true
	}

	// 检查忽略目录
	if isDir {
		for _, ignore := range config.IgnoreDirs {
			if name == ignore {
				return true
			}
		}
	}

	// 检查忽略文件模式
	for _, pattern := range config.IgnoreFiles {
		matched, _ := filepath.Match(pattern, name)
		if matched {
			return true
		}
	}

	return false
}

// renderTree 渲染目录树为字符串
func renderTree(sb *strings.Builder, node *DirectoryNode, prefix string, isLast bool, totalFiles *int) {
	if node == nil {
		return
	}

	// 检查总文件数限制
	if totalFiles != nil && *totalFiles >= 100 {
		return
	}

	// 渲染当前节点
	if node.Depth > 0 {
		// 绘制树形结构
		if isLast {
			sb.WriteString(prefix + "└── ")
		} else {
			sb.WriteString(prefix + "├── ")
		}

		// 目录名称加粗或添加 /
		if node.IsDir {
			sb.WriteString(node.Name + "/\n")
		} else {
			sb.WriteString(node.Name + "\n")
			if totalFiles != nil {
				*totalFiles++
			}
		}
	}
	// else {
	// 根节点
	// sb.WriteString(node.Name + "/\n")
	// }

	// 递归渲染子节点
	for i, child := range node.Children {
		isChildLast := i == len(node.Children)-1

		// 计算新的前缀
		var newPrefix string
		if node.Depth == 0 {
			newPrefix = ""
		} else if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}

		renderTree(sb, child, newPrefix, isChildLast, totalFiles)
	}
}

// SummarizeDirectoryCompact 生成紧凑的目录摘要
// 只显示重要文件和目录结构
func SummarizeDirectoryCompact(rootPath string) string {
	config := &SummarizerConfig{
		MaxDepth: 2,
		IgnoreDirs: []string{
			".git", "node_modules", "vendor", ".aicoder",
			"__pycache__", ".pytest_cache", ".venv", "venv",
			"dist", "build", "target",
		},
		IgnoreFiles: []string{
			"*.pyc", "*.pyo", "*.so", "*.dylib", "*.dll",
			"*.exe", "*.o", "*.a",
		},
		MaxFiles:      10,
		MaxTotalFiles: 50,
	}

	return SummarizeDirectory(rootPath, config)
}

// GetImportantFiles 获取项目中的重要文件列表
// 返回配置文件、README、文档等
func GetImportantFiles(rootPath string) []string {
	importantPatterns := []string{
		"README*",
		"LICENSE*",
		"CHANGELOG*",
		"CONTRIBUTING*",
		"Makefile",
		"Dockerfile",
		"docker-compose.yml",
		"package.json",
		"go.mod",
		"requirements.txt",
		"pyproject.toml",
		"Cargo.toml",
		"pom.xml",
		"build.gradle",
		"Gemfile",
		".gitignore",
		".env.example",
		".AICODER.md",
	}

	var files []string

	for _, pattern := range importantPatterns {
		matches, err := filepath.Glob(filepath.Join(rootPath, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			// 转换为相对路径
			rel, err := filepath.Rel(rootPath, match)
			if err != nil {
				rel = filepath.Base(match)
			}
			files = append(files, rel)
		}
	}

	// 排序
	sort.Strings(files)

	return files
}

// CountFiles 统计目录中的文件数量
func CountFiles(rootPath string, config *SummarizerConfig) (int, int) {
	if config == nil {
		config = DefaultSummarizerConfig()
	}

	var fileCount, dirCount int

	filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// 检查是否应该忽略
		if shouldIgnore(d.Name(), d.IsDir(), config) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			dirCount++
		} else {
			fileCount++
		}

		return nil
	})

	return fileCount, dirCount
}
