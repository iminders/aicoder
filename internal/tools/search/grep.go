package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iminders/aicoder/internal/tools"
)

type GrepSearchTool struct{}

func (t *GrepSearchTool) Name() string          { return "grep_search" }
func (t *GrepSearchTool) Risk() tools.RiskLevel { return tools.RiskLow }
func (t *GrepSearchTool) Description() string {
	return "Search for a regex pattern across a directory tree. Returns file:line: match entries."
}
func (t *GrepSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path":            {"type": "string",  "description": "Root directory to search"},
			"pattern":         {"type": "string",  "description": "Regex pattern"},
			"case_insensitive":{"type": "boolean", "description": "Case-insensitive match"},
			"include":         {"type": "string",  "description": "Glob filter e.g. '*.go'"},
			"context_lines":   {"type": "integer", "description": "Lines of context around each match (default 0)"}
		},
		"required": ["path","pattern"]
	}`)
}

type grepInput struct {
	Path            string `json:"path"`
	Pattern         string `json:"pattern"`
	CaseInsensitive bool   `json:"case_insensitive"`
	Include         string `json:"include"`
	ContextLines    int    `json:"context_lines"`
}

func (t *GrepSearchTool) Execute(_ context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in grepInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	pat := in.Pattern
	if in.CaseInsensitive {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return &tools.Result{IsError: true, Content: "invalid regex: " + err.Error()}, nil
	}

	var results []string
	const maxResults = 200

	_ = filepath.WalkDir(in.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if in.Include != "" {
			if matched, _ := filepath.Match(in.Include, filepath.Base(p)); !matched {
				return nil
			}
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				if in.ContextLines > 0 {
					start := i - in.ContextLines
					if start < 0 {
						start = 0
					}
					end := i + in.ContextLines + 1
					if end > len(lines) {
						end = len(lines)
					}
					for j := start; j < end; j++ {
						prefix := "  "
						if j == i {
							prefix = "> "
						}
						results = append(results, fmt.Sprintf("%s:%d:%s%s", p, j+1, prefix, lines[j]))
					}
					results = append(results, "---")
				} else {
					results = append(results, fmt.Sprintf("%s:%d: %s", p, i+1, strings.TrimSpace(line)))
				}
				if len(results) >= maxResults {
					return fmt.Errorf("limit")
				}
			}
		}
		return nil
	})

	if len(results) == 0 {
		return &tools.Result{Content: "No matches found."}, nil
	}
	return &tools.Result{Content: strings.Join(results, "\n")}, nil
}

func init() {
	tools.Global.Register(&GrepSearchTool{})
}
