package filesystem

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileTool(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hello.txt")
	_ = os.WriteFile(p, []byte("line1\nline2\nline3\n"), 0644)

	tool := &ReadFileTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": p})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("unexpected error:", result.Content)
	}
	if !strings.Contains(result.Content, "line2") {
		t.Error("expected line2 in output")
	}
}

func TestReadFileRange(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.txt")
	lines := "a\nb\nc\nd\ne\n"
	_ = os.WriteFile(p, []byte(lines), 0644)

	tool := &ReadFileTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": p, "start_line": 2, "end_line": 3})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if !strings.Contains(result.Content, "b") || strings.Contains(result.Content, "d") {
		t.Error("expected only lines 2-3")
	}
}

func TestWriteFileTool(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "out.txt")

	tool := &WriteFileTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": p, "content": "hello world"})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "hello world" {
		t.Errorf("unexpected content: %s", got)
	}
}

func TestEditFileTool(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "edit.go")
	_ = os.WriteFile(p, []byte("func main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644)

	tool := &EditFileTool{}
	input, _ := json.Marshal(map[string]interface{}{
		"path":       p,
		"old_string": "fmt.Println(\"hello\")",
		"new_string": "fmt.Println(\"world\")",
	})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	got, _ := os.ReadFile(p)
	if !strings.Contains(string(got), "world") {
		t.Error("edit not applied")
	}
}

func TestSearchFilesTool(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main\nfunc Foo() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("package main\nfunc Bar() {}\n"), 0644)

	tool := &SearchFilesTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": dir, "pattern": "func "})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if !strings.Contains(result.Content, "Foo") || !strings.Contains(result.Content, "Bar") {
		t.Errorf("expected both functions in results: %s", result.Content)
	}
}

func TestListDirTool(t *testing.T) {
	dir := t.TempDir()
	_ = os.Mkdir(filepath.Join(dir, "subdir"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "subdir", "nested.txt"), []byte("y"), 0644)

	tool := &ListDirTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": dir})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if !strings.Contains(result.Content, "file.txt") {
		t.Error("expected file.txt in listing")
	}
	if !strings.Contains(result.Content, "subdir") {
		t.Error("expected subdir in listing")
	}
}

func TestSandboxBlock(t *testing.T) {
	tool := &ReadFileTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": "/etc/shadow"})
	result, _ := tool.Execute(context.Background(), input)
	if !result.IsError {
		t.Error("expected sandbox to block /etc/shadow")
	}
}

func TestDeleteFileTool(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "todelete.txt")
	_ = os.WriteFile(p, []byte("bye"), 0644)

	tool := &DeleteFileTool{}
	input, _ := json.Marshal(map[string]interface{}{"path": p})
	result, _ := tool.Execute(context.Background(), input)
	if result.IsError {
		t.Fatal(result.Content)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}
