package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iminders/aicoder/internal/tools"
)

type WebSearchTool struct{}

func (t *WebSearchTool) Name() string          { return "web_search" }
func (t *WebSearchTool) Risk() tools.RiskLevel { return tools.RiskLow }
func (t *WebSearchTool) Description() string {
	return "Search the web using Tavily API. Requires TAVILY_API_KEY environment variable to be set. Returns top search results with titles, snippets, and URLs."
}

func (t *WebSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query":       {"type": "string",  "description": "Search query"},
			"num_results": {"type": "integer", "description": "Number of results to return (default: 5, max: 10)"},
			"language":    {"type": "string",  "description": "Language code (e.g., 'en', 'zh-CN', default: 'en')"}
		},
		"required": ["query"]
	}`)
}

type webSearchInput struct {
	Query      string `json:"query"`
	NumResults int    `json:"num_results"`
	Language   string `json:"language"`
}

type searchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

type tavilySearchResponse struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

func (t *WebSearchTool) Execute(ctx context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in webSearchInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}

	// 设置默认值
	if in.NumResults == 0 {
		in.NumResults = 5
	}
	if in.NumResults > 10 {
		in.NumResults = 10
	}
	if in.Language == "" {
		in.Language = "en"
	}

	// 检查环境变量
	apiKey := os.Getenv("TAVILY_API_KEY")

	if apiKey == "" {
		return &tools.Result{
			IsError: true,
			Content: "Web search is not configured. Please set TAVILY_API_KEY environment variable.\n\n" +
				"To use Tavily Search:\n" +
				"1. Get API key from: https://tavily.com/\n" +
				"2. Set environment variable:\n" +
				"   export TAVILY_API_KEY=\"tvly-your-api-key\"",
		}, nil
	}

	// 执行搜索
	results, err := t.tavilySearch(ctx, in.Query, in.NumResults, apiKey)
	if err != nil {
		return &tools.Result{IsError: true, Content: fmt.Sprintf("Search failed: %v", err)}, nil
	}

	if len(results) == 0 {
		return &tools.Result{Content: "No search results found."}, nil
	}

	// 格式化结果
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d results for: %s\n\n", len(results), in.Query))

	for i, result := range results {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		output.WriteString(fmt.Sprintf("   URL: %s\n", result.Link))
		output.WriteString(fmt.Sprintf("   %s\n\n", result.Snippet))
	}

	return &tools.Result{
		Content: output.String(),
		Metadata: map[string]any{
			"num_results": len(results),
			"query":       in.Query,
		},
	}, nil
}

// tavilySearch 使用 Tavily Search API 进行搜索
func (t *WebSearchTool) tavilySearch(ctx context.Context, query string, numResults int, apiKey string) ([]searchResult, error) {
	// 构建请求体
	requestBody := map[string]interface{}{
		"api_key":        apiKey,
		"query":          query,
		"max_results":    numResults,
		"search_depth":   "basic",
		"include_answer": false,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.tavily.com/search", strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 设置超时
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var searchResp tavilySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 转换结果
	results := make([]searchResult, 0, len(searchResp.Results))
	for _, item := range searchResp.Results {
		results = append(results, searchResult{
			Title:   item.Title,
			Link:    item.URL,
			Snippet: item.Content,
		})
	}

	return results, nil
}

func init() {
	tools.Global.Register(&WebSearchTool{})
}
