package search

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestASTSearchTool_Go_Functions(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testCode := `package main

import "fmt"

// PublicFunc is an exported function
func PublicFunc() {
	fmt.Println("public")
}

// privateFunc is an unexported function
func privateFunc() {
	fmt.Println("private")
}

type MyStruct struct {
	Name string
}

// Method is a method on MyStruct
func (m *MyStruct) Method() {
	fmt.Println(m.Name)
}

func (m *MyStruct) privateMethod() {
	fmt.Println("private method")
}
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &ASTSearchTool{}

	tests := []struct {
		name         string
		queryType    string
		namePattern  string
		exportedOnly bool
		wantContains []string
		wantNotContain []string
	}{
		{
			name:         "find all functions",
			queryType:    "function",
			namePattern:  "",
			exportedOnly: false,
			wantContains: []string{"PublicFunc", "privateFunc"},
		},
		{
			name:         "find exported functions only",
			queryType:    "function",
			namePattern:  "",
			exportedOnly: true,
			wantContains: []string{"PublicFunc"},
			wantNotContain: []string{"privateFunc"},
		},
		{
			name:         "find methods",
			queryType:    "method",
			namePattern:  "",
			exportedOnly: false,
			wantContains: []string{"Method", "privateMethod"},
		},
		{
			name:         "find exported methods only",
			queryType:    "method",
			namePattern:  "",
			exportedOnly: true,
			wantContains: []string{"Method"},
			wantNotContain: []string{"privateMethod"},
		},
		{
			name:         "find struct",
			queryType:    "struct",
			namePattern:  "MyStruct",
			exportedOnly: false,
			wantContains: []string{"struct MyStruct"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := astInput{
				Path:         tmpDir,
				QueryType:    tt.queryType,
				Name:         tt.namePattern,
				ExportedOnly: tt.exportedOnly,
			}

			inputJSON, _ := json.Marshal(input)
			result, err := tool.Execute(context.Background(), inputJSON)

			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if result.IsError {
				t.Fatalf("Result is error: %s", result.Content)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result.Content, want) {
					t.Errorf("Result should contain %q, got:\n%s", want, result.Content)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result.Content, notWant) {
					t.Errorf("Result should not contain %q, got:\n%s", notWant, result.Content)
				}
			}
		})
	}
}

func TestASTSearchTool_Go_Types(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "types.go")

	testCode := `package main

type MyInterface interface {
	DoSomething()
}

type MyStruct struct {
	Field string
}

type MyInt int

type MyFunc func(string) error
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &ASTSearchTool{}

	tests := []struct {
		name         string
		queryType    string
		wantContains []string
	}{
		{
			name:         "find all types",
			queryType:    "type",
			wantContains: []string{"MyInterface", "MyStruct", "MyInt", "MyFunc"},
		},
		{
			name:         "find interfaces",
			queryType:    "interface",
			wantContains: []string{"MyInterface"},
		},
		{
			name:         "find structs",
			queryType:    "struct",
			wantContains: []string{"MyStruct"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := astInput{
				Path:      tmpDir,
				QueryType: tt.queryType,
			}

			inputJSON, _ := json.Marshal(input)
			result, err := tool.Execute(context.Background(), inputJSON)

			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if result.IsError {
				t.Fatalf("Result is error: %s", result.Content)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result.Content, want) {
					t.Errorf("Result should contain %q, got:\n%s", want, result.Content)
				}
			}
		})
	}
}

func TestASTSearchTool_InvalidQueryType(t *testing.T) {
	tool := &ASTSearchTool{}

	input := astInput{
		Path:      ".",
		QueryType: "invalid",
	}

	inputJSON, _ := json.Marshal(input)
	result, err := tool.Execute(context.Background(), inputJSON)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error for invalid query type")
	}

	if !strings.Contains(result.Content, "Invalid query_type") {
		t.Errorf("Expected error message about invalid query type, got: %s", result.Content)
	}
}

func TestASTSearchTool_NamePattern(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testCode := `package main

func GetUser() {}
func GetPost() {}
func SetUser() {}
func DeleteUser() {}
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &ASTSearchTool{}

	tests := []struct {
		name         string
		namePattern  string
		wantContains []string
		wantNotContain []string
	}{
		{
			name:         "find Get* functions",
			namePattern:  "Get",
			wantContains: []string{"GetUser", "GetPost"},
			wantNotContain: []string{"SetUser", "DeleteUser"},
		},
		{
			name:         "find *User functions",
			namePattern:  "User",
			wantContains: []string{"GetUser", "SetUser", "DeleteUser"},
			wantNotContain: []string{"GetPost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := astInput{
				Path:      tmpDir,
				QueryType: "function",
				Name:      tt.namePattern,
			}

			inputJSON, _ := json.Marshal(input)
			result, err := tool.Execute(context.Background(), inputJSON)

			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if result.IsError {
				t.Fatalf("Result is error: %s", result.Content)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result.Content, want) {
					t.Errorf("Result should contain %q, got:\n%s", want, result.Content)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result.Content, notWant) {
					t.Errorf("Result should not contain %q, got:\n%s", notWant, result.Content)
				}
			}
		})
	}
}

func TestASTSearchTool_Imports(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testCode := `package main

import (
	"fmt"
	"os"
	"github.com/example/pkg"
)

func main() {}
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &ASTSearchTool{}

	input := astInput{
		Path:      tmpDir,
		QueryType: "import",
	}

	inputJSON, _ := json.Marshal(input)
	result, err := tool.Execute(context.Background(), inputJSON)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("Result is error: %s", result.Content)
	}

	wantImports := []string{"fmt", "os", "github.com/example/pkg"}
	for _, want := range wantImports {
		if !strings.Contains(result.Content, want) {
			t.Errorf("Result should contain import %q, got:\n%s", want, result.Content)
		}
	}
}
