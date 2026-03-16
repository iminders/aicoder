package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/iminders/aicoder/internal/tools"
)

type WebSearchTool struct{}

func (t *WebSearchTool) Name() string          { return "web_search" }
func (t *WebSearchTool) Risk() tools.RiskLevel { return tools.RiskLow }
func (t *WebSearchTool) Description() string {
	return "Search the web using a search API. Requires SEARCH_API_KEY and SEARCH_ENGINE_ID environment variables to be set. Returns top search results with titles, snippets, and URLs."
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

type googleSearchResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
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
	apiKey := os.Getenv("SEARCH_API_KEY")
	engineID := os.Getenv("SEARCH_ENGINE_ID")

	if apiKey == "" || engineID == "" {
		return &tools.Result{
			IsError: true,
			Content: "Web search is not configured. Please set SEARCH_API_KEY and SEARCH_ENGINE_ID environment variables.\n\n" +
				"To use Google Custom Search:\n" +
				"1. Get API key from: https://developers.google.com/custom-search/v1/overview\n" +
				"2. Create search engine at: https://programmablesearchengine.google.com/\n" +
				"3. Set environment variables:\n" +
				"   export SEARCH_API_KEY=\"your-api-key\"\n" +
				"   export SEARCH_ENGINE_ID=\"your-engine-id\"",
		}, nil
	}

	// 执行搜索
	results, err := t.googleSearch(ctx, in.Query, in.NumResults, in.Language, apiKey, engineID)
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

// googleSearch 使用 Google Custom Search API 进行搜索
func (t *WebSearchTool) googleSearch(ctx context.Context, query string, numResults int, language, apiKey, engineID string) ([]searchResult, error) {
	// 构建 API URL
	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Set("key", apiKey)
	params.Set("cx", engineID)
	params.Set("q", query)
	params.Set("num", fmt.Sprintf("%d", numResults))
	params.Set("lr", "lang_"+language)

	apiURL := baseURL + "?" + params.Encode()

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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
	var searchResp googleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 转换结果
	results := make([]searchResult, 0, len(searchResp.Items))
	for _, item := range searchResp.Items {
		results = append(results, searchResult{
			Title:   item.Title,
			Link:    item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}

func init() {
	tools.Global.Register(&WebSearchTool{})
}
