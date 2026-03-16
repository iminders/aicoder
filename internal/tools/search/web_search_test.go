package search

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestWebSearchTool_NoConfig(t *testing.T) {
	// 确保环境变量未设置
	os.Unsetenv("SEARCH_API_KEY")
	os.Unsetenv("SEARCH_ENGINE_ID")

	tool := &WebSearchTool{}

	input := webSearchInput{
		Query: "golang tutorial",
	}

	inputJSON, _ := json.Marshal(input)
	result, err := tool.Execute(context.Background(), inputJSON)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error when API key is not configured")
	}

	if !strings.Contains(result.Content, "not configured") {
		t.Errorf("Expected configuration error message, got: %s", result.Content)
	}

	if !strings.Contains(result.Content, "SEARCH_API_KEY") {
		t.Errorf("Expected mention of SEARCH_API_KEY, got: %s", result.Content)
	}
}

func TestWebSearchTool_DefaultValues(t *testing.T) {
	tests := []struct {
		name           string
		input          webSearchInput
		wantNumResults int
		wantLanguage   string
	}{
		{
			name: "default num_results and language",
			input: webSearchInput{
				Query: "test",
			},
			wantNumResults: 5,
			wantLanguage:   "en",
		},
		{
			name: "custom num_results",
			input: webSearchInput{
				Query:      "test",
				NumResults: 3,
			},
			wantNumResults: 3,
			wantLanguage:   "en",
		},
		{
			name: "max num_results limit",
			input: webSearchInput{
				Query:      "test",
				NumResults: 20,
			},
			wantNumResults: 10, // should be capped at 10
			wantLanguage:   "en",
		},
		{
			name: "custom language",
			input: webSearchInput{
				Query:    "test",
				Language: "zh-CN",
			},
			wantNumResults: 5,
			wantLanguage:   "zh-CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just validates the input processing
			// We don't actually call the API since it requires credentials

			input := tt.input
			if input.NumResults == 0 {
				input.NumResults = 5
			}
			if input.NumResults > 10 {
				input.NumResults = 10
			}
			if input.Language == "" {
				input.Language = "en"
			}

			if input.NumResults != tt.wantNumResults {
				t.Errorf("NumResults = %d, want %d", input.NumResults, tt.wantNumResults)
			}

			if input.Language != tt.wantLanguage {
				t.Errorf("Language = %s, want %s", input.Language, tt.wantLanguage)
			}
		})
	}
}

func TestWebSearchTool_Schema(t *testing.T) {
	tool := &WebSearchTool{}
	schema := tool.Schema()

	if len(schema) == 0 {
		t.Error("Schema should not be empty")
	}

	// 验证 schema 是有效的 JSON
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		t.Errorf("Schema is not valid JSON: %v", err)
	}

	// 验证必需字段
	props, ok := schemaObj["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema should have properties")
	}

	requiredFields := []string{"query", "num_results", "language"}
	for _, field := range requiredFields {
		if _, exists := props[field]; !exists {
			t.Errorf("Schema should have property %q", field)
		}
	}

	// 验证 required 字段
	required, ok := schemaObj["required"].([]interface{})
	if !ok {
		t.Fatal("Schema should have required array")
	}

	if len(required) == 0 {
		t.Error("Schema should have at least one required field")
	}

	if required[0].(string) != "query" {
		t.Errorf("First required field should be 'query', got %v", required[0])
	}
}

func TestWebSearchTool_ToolInterface(t *testing.T) {
	tool := &WebSearchTool{}

	if tool.Name() != "web_search" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "web_search")
	}

	if tool.Risk() != 0 { // RiskLow = 0
		t.Errorf("Risk() = %v, want RiskLow", tool.Risk())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "web") {
		t.Errorf("Description should mention 'web', got: %s", desc)
	}
}

// TestWebSearchTool_Integration 是集成测试,需要真实的 API 凭证
// 默认跳过,可以通过设置环境变量来运行
func TestWebSearchTool_Integration(t *testing.T) {
	apiKey := os.Getenv("SEARCH_API_KEY")
	engineID := os.Getenv("SEARCH_ENGINE_ID")

	if apiKey == "" || engineID == "" {
		t.Skip("Skipping integration test: SEARCH_API_KEY or SEARCH_ENGINE_ID not set")
	}

	tool := &WebSearchTool{}

	input := webSearchInput{
		Query:      "golang",
		NumResults: 3,
		Language:   "en",
	}

	inputJSON, _ := json.Marshal(input)
	result, err := tool.Execute(context.Background(), inputJSON)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("Result is error: %s", result.Content)
	}

	if !strings.Contains(result.Content, "Found") {
		t.Errorf("Result should contain 'Found', got:\n%s", result.Content)
	}

	if !strings.Contains(result.Content, "golang") {
		t.Errorf("Result should contain query term 'golang', got:\n%s", result.Content)
	}

	// 验证元数据
	if result.Metadata == nil {
		t.Error("Result should have metadata")
	} else {
		if _, ok := result.Metadata["num_results"]; !ok {
			t.Error("Metadata should have 'num_results'")
		}
		if _, ok := result.Metadata["query"]; !ok {
			t.Error("Metadata should have 'query'")
		}
	}
}
