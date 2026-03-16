package search

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepSearch(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "util.go"), []byte("package main\n\nfunc Helper() string {\n\treturn \"world\"\n}\n"), 0644)

	tool := &GrepSearchTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": dir, "pattern": "func "})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if !strings.Contains(result.Content, "main") {
		t.Error("expected to find main function")
	}
	if !strings.Contains(result.Content, "Helper") {
		t.Error("expected to find Helper function")
	}
}

func TestGrepSearchCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Hello World\nThis is README\n"), 0644)

	tool := &GrepSearchTool{}
	input, _ := json.Marshal(map[string]interface{}{
		"path":             dir,
		"pattern":          "hello",
		"case_insensitive": true,
	})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if !strings.Contains(result.Content, "Hello") {
		t.Error("expected case-insensitive match")
	}
}

func TestGrepSearchWithGlob(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "data.json"), []byte("{\"key\":\"value\"}\n"), 0644)

	tool := &GrepSearchTool{}
	input, _ := json.Marshal(map[string]interface{}{
		"path":    dir,
		"pattern": "package",
		"include": "*.go",
	})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if !strings.Contains(result.Content, "main.go") {
		t.Error("expected main.go in results")
	}
	if strings.Contains(result.Content, "data.json") {
		t.Error("data.json should be excluded by glob filter")
	}
}

func TestGrepSearchNoMatch(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "empty.go"), []byte("package main\n"), 0644)

	tool := &GrepSearchTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": dir, "pattern": "XXXXXXNOTFOUND"})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if !strings.Contains(result.Content, "No matches") {
		t.Error("expected 'No matches' message")
	}
}
