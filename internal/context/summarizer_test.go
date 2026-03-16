package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeDirectory(t *testing.T) {
	// 创建临时测试目录结构
	tmpDir := t.TempDir()

	// 创建测试文件和目录
	testStructure := map[string]string{
		"README.md":          "# Test Project",
		"main.go":            "package main",
		"go.mod":             "module test",
		"src/app.go":         "package src",
		"src/utils.go":       "package src",
		"tests/app_test.go":  "package tests",
		"docs/guide.md":      "# Guide",
		"node_modules/pkg/a": "should be ignored",
		".git/config":        "should be ignored",
	}

	for path, content := range testStructure {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 测试默认配置
	summary := SummarizeDirectory(tmpDir, nil)

	// 验证结果
	if !strings.Contains(summary, "项目结构:") {
		t.Error("Summary should contain '项目结构:'")
	}

	if !strings.Contains(summary, "README.md") {
		t.Error("Summary should contain README.md")
	}

	if !strings.Contains(summary, "main.go") {
		t.Error("Summary should contain main.go")
	}

	if !strings.Contains(summary, "src/") {
		t.Error("Summary should contain src directory")
	}

	// 验证忽略的目录不在结果中
	if strings.Contains(summary, "node_modules") {
		t.Error("Summary should not contain node_modules")
	}

	if strings.Contains(summary, ".git") {
		t.Error("Summary should not contain .git")
	}
}

func TestSummarizeDirectoryCompact(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建简单的测试结构
	testFiles := []string{
		"README.md",
		"main.go",
		"src/app.go",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	summary := SummarizeDirectoryCompact(tmpDir)

	if !strings.Contains(summary, "README.md") {
		t.Error("Compact summary should contain README.md")
	}
}

func TestGetImportantFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建重要文件
	importantFiles := []string{
		"README.md",
		"LICENSE",
		"package.json",
		"go.mod",
		"Makefile",
		"Dockerfile",
	}

	for _, file := range importantFiles {
		fullPath := filepath.Join(tmpDir, file)
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 创建普通文件
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)

	files := GetImportantFiles(tmpDir)

	// 验证重要文件都被找到
	expectedFiles := map[string]bool{
		"README.md":    false,
		"LICENSE":      false,
		"package.json": false,
		"go.mod":       false,
		"Makefile":     false,
		"Dockerfile":   false,
	}

	for _, file := range files {
		if _, exists := expectedFiles[file]; exists {
			expectedFiles[file] = true
		}
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Important file %s was not found", file)
		}
	}

	// 验证普通文件不在列表中
	for _, file := range files {
		if file == "test.txt" {
			t.Error("test.txt should not be in important files list")
		}
	}
}

func TestCountFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	testFiles := []string{
		"file1.go",
		"file2.go",
		"src/file3.go",
		"tests/file4.go",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 创建应该被忽略的文件
	os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "node_modules", "pkg.js"), []byte("test"), 0644)

	fileCount, dirCount := CountFiles(tmpDir, nil)

	if fileCount != 4 {
		t.Errorf("Expected 4 files, got %d", fileCount)
	}

	// dirCount 包括 tmpDir, src, tests (不包括 node_modules)
	if dirCount < 2 {
		t.Errorf("Expected at least 2 directories, got %d", dirCount)
	}
}

func TestShouldIgnore(t *testing.T) {
	config := DefaultSummarizerConfig()

	tests := []struct {
		name       string
		fileName   string
		isDir      bool
		wantIgnore bool
	}{
		{
			name:       "node_modules directory",
			fileName:   "node_modules",
			isDir:      true,
			wantIgnore: true,
		},
		{
			name:       ".git directory",
			fileName:   ".git",
			isDir:      true,
			wantIgnore: true,
		},
		{
			name:       "vendor directory",
			fileName:   "vendor",
			isDir:      true,
			wantIgnore: true,
		},
		{
			name:       "normal directory",
			fileName:   "src",
			isDir:      true,
			wantIgnore: false,
		},
		{
			name:       ".pyc file",
			fileName:   "test.pyc",
			isDir:      false,
			wantIgnore: true,
		},
		{
			name:       ".go file",
			fileName:   "main.go",
			isDir:      false,
			wantIgnore: false,
		},
		{
			name:       ".gitignore file",
			fileName:   ".gitignore",
			isDir:      false,
			wantIgnore: false,
		},
		{
			name:       "hidden file",
			fileName:   ".hidden",
			isDir:      false,
			wantIgnore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldIgnore(tt.fileName, tt.isDir, config)
			if got != tt.wantIgnore {
				t.Errorf("shouldIgnore(%q, %v) = %v, want %v", tt.fileName, tt.isDir, got, tt.wantIgnore)
			}
		})
	}
}

func TestBuildDirectoryTree(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试结构
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("test"), 0644)

	config := DefaultSummarizerConfig()
	tree := buildDirectoryTree(tmpDir, config, 0)

	if tree == nil {
		t.Fatal("buildDirectoryTree returned nil")
	}

	if !tree.IsDir {
		t.Error("Root should be a directory")
	}

	if len(tree.Children) == 0 {
		t.Error("Root should have children")
	}

	// 验证子节点
	hasFile := false
	hasDir := false

	for _, child := range tree.Children {
		if child.Name == "file1.txt" {
			hasFile = true
			if child.IsDir {
				t.Error("file1.txt should not be a directory")
			}
		}
		if child.Name == "subdir" {
			hasDir = true
			if !child.IsDir {
				t.Error("subdir should be a directory")
			}
		}
	}

	if !hasFile {
		t.Error("Tree should contain file1.txt")
	}

	if !hasDir {
		t.Error("Tree should contain subdir")
	}
}

func TestMaxDepthLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建深层嵌套结构
	os.MkdirAll(filepath.Join(tmpDir, "level1", "level2", "level3", "level4"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "level1", "level2", "level3", "level4", "deep.txt"), []byte("test"), 0644)

	config := &SummarizerConfig{
		MaxDepth:      2,
		IgnoreDirs:    []string{},
		IgnoreFiles:   []string{},
		MaxFiles:      20,
		MaxTotalFiles: 100,
	}

	summary := SummarizeDirectory(tmpDir, config)

	// 应该包含 level1 和 level2
	if !strings.Contains(summary, "level1") {
		t.Error("Summary should contain level1")
	}

	if !strings.Contains(summary, "level2") {
		t.Error("Summary should contain level2")
	}

	// 不应该包含 level4 (超过深度限制)
	if strings.Contains(summary, "level4") {
		t.Error("Summary should not contain level4 (exceeds max depth)")
	}
}

func TestMaxFilesLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建很多文件
	for i := 0; i < 30; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('A'+i))+".txt")
		os.WriteFile(filename, []byte("test"), 0644)
	}

	config := &SummarizerConfig{
		MaxDepth:      3,
		IgnoreDirs:    []string{},
		IgnoreFiles:   []string{},
		MaxFiles:      10,
		MaxTotalFiles: 100,
	}

	summary := SummarizeDirectory(tmpDir, config)

	// 应该包含省略标记
	if !strings.Contains(summary, "...") {
		t.Error("Summary should contain '...' when files exceed limit")
	}
}
