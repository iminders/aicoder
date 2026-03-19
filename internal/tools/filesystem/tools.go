package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/iminders/aicoder/internal/tools"
	"github.com/iminders/aicoder/pkg/diff"
)

// sandboxDenied are path prefixes that are always forbidden.
var sandboxDenied = []string{
	"/etc/shadow", "/etc/passwd", "/etc/sudoers",
	"/.ssh", "/.gnupg",
}

func checkSandbox(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	for _, denied := range sandboxDenied {
		full := denied
		if strings.HasPrefix(denied, "/~") {
			full = home + denied[1:]
		}
		if strings.HasPrefix(abs, full) {
			return fmt.Errorf("access denied: %s is in a protected path", path)
		}
	}
	return nil
}

// ─── WriteFileTool ────────────────────────────────────────────────────────────

// SnapshotFunc is set by the agent to capture file snapshots for /undo.
var SnapshotFunc func(toolName, callID, filePath string, before, after []byte)

type WriteFileTool struct{}

func (t *WriteFileTool) Name() string          { return "write_file" }
func (t *WriteFileTool) Risk() tools.RiskLevel { return tools.RiskMedium }
func (t *WriteFileTool) Description() string {
	return "Write content to a file, creating it if it doesn't exist (overwrites existing content)."
}
func (t *WriteFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path":    {"type": "string", "description": "File path to write"},
			"content": {"type": "string", "description": "Content to write"}
		},
		"required": ["path","content"]
	}`)
}

type writeInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *WriteFileTool) Execute(_ context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in writeInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if err := checkSandbox(in.Path); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}

	if err := os.MkdirAll(filepath.Dir(in.Path), 0755); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}

	// Snapshot before state
	var before []byte
	before, _ = os.ReadFile(in.Path)

	if err := os.WriteFile(in.Path, []byte(in.Content), 0644); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if SnapshotFunc != nil {
		SnapshotFunc("write_file", fmt.Sprintf("%d", time.Now().UnixNano()), in.Path, before, []byte(in.Content))
	}
	return &tools.Result{Content: fmt.Sprintf("Successfully wrote %d bytes to %s", len(in.Content), in.Path)}, nil
}

// ─── EditFileTool ────────────────────────────────────────────────────────────

type EditFileTool struct{}

func (t *EditFileTool) Name() string          { return "edit_file" }
func (t *EditFileTool) Risk() tools.RiskLevel { return tools.RiskMedium }
func (t *EditFileTool) Description() string {
	return "Edit a file by replacing an exact old_string with new_string. old_string must match exactly once."
}
func (t *EditFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path":       {"type": "string", "description": "File path to edit"},
			"old_string": {"type": "string", "description": "Exact string to replace"},
			"new_string": {"type": "string", "description": "Replacement string"}
		},
		"required": ["path","old_string","new_string"]
	}`)
}

type editInput struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func (t *EditFileTool) Execute(_ context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in editInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if err := checkSandbox(in.Path); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	before, err := os.ReadFile(in.Path)
	if err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	newContent, err := diff.ApplyEdit(string(before), in.OldString, in.NewString)
	if err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if err := os.WriteFile(in.Path, []byte(newContent), 0644); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if SnapshotFunc != nil {
		SnapshotFunc("edit_file", fmt.Sprintf("%d", time.Now().UnixNano()), in.Path, before, []byte(newContent))
	}
	patch := diff.ColorDiff(string(before), newContent, in.Path)
	return &tools.Result{Content: "File edited successfully.\n" + patch}, nil
}

// ─── ListDirTool ─────────────────────────────────────────────────────────────

type ListDirTool struct{}

func (t *ListDirTool) Name() string          { return "list_dir" }
func (t *ListDirTool) Risk() tools.RiskLevel { return tools.RiskLow }
func (t *ListDirTool) Description() string {
	return "List directory contents as a tree. Respects .gitignore patterns."
}
func (t *ListDirTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path":  {"type": "string",  "description": "Directory path"},
			"depth": {"type": "integer", "description": "Max depth (default 3)"}
		},
		"required": ["path"]
	}`)
}

type listInput struct {
	Path  string `json:"path"`
	Depth int    `json:"depth"`
}

var defaultIgnore = []string{".git", "node_modules", "__pycache__", ".DS_Store", "vendor", "dist", "build", ".next"}

func (t *ListDirTool) Execute(_ context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in listInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if in.Depth <= 0 {
		in.Depth = 3
	}
	var sb strings.Builder
	walk(&sb, in.Path, "", 0, in.Depth)
	return &tools.Result{Content: sb.String()}, nil
}

func walk(sb *strings.Builder, path, prefix string, depth, maxDepth int) {
	if depth > maxDepth {
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		sb.WriteString(prefix + "[error: " + err.Error() + "]\n")
		return
	}
	for i, e := range entries {
		if isIgnored(e.Name()) {
			continue
		}
		connector := "├── "
		childPrefix := prefix + "│   "
		if i == len(entries)-1 {
			connector = "└── "
			childPrefix = prefix + "    "
		}
		sb.WriteString(prefix + connector + e.Name() + "\n")
		if e.IsDir() {
			walk(sb, filepath.Join(path, e.Name()), childPrefix, depth+1, maxDepth)
		}
	}
}

func isIgnored(name string) bool {
	for _, ig := range defaultIgnore {
		if name == ig {
			return true
		}
	}
	return false
}

// ─── SearchFilesTool ─────────────────────────────────────────────────────────

type SearchFilesTool struct{}

func (t *SearchFilesTool) Name() string          { return "search_files" }
func (t *SearchFilesTool) Risk() tools.RiskLevel { return tools.RiskLow }
func (t *SearchFilesTool) Description() string {
	return "Search for a regex pattern across files in a directory. Returns matching lines with file and line number."
}
func (t *SearchFilesTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path":    {"type": "string", "description": "Root directory to search"},
			"pattern": {"type": "string", "description": "Regular expression to search for"},
			"glob":    {"type": "string", "description": "File glob filter, e.g. '*.go' (optional)"}
		},
		"required": ["path","pattern"]
	}`)
}

type searchInput struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
	Glob    string `json:"glob"`
}

func (t *SearchFilesTool) Execute(_ context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in searchInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		return &tools.Result{IsError: true, Content: "invalid regex: " + err.Error()}, nil
	}

	var results []string
	maxResults := 100

	err = filepath.WalkDir(in.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && isIgnored(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if in.Glob != "" {
			matched, _ := filepath.Match(in.Glob, filepath.Base(p))
			if !matched {
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
				results = append(results, fmt.Sprintf("%s:%d: %s", p, i+1, strings.TrimSpace(line)))
				if len(results) >= maxResults {
					return fmt.Errorf("limit reached")
				}
			}
		}
		return nil
	})
	_ = err

	if len(results) == 0 {
		return &tools.Result{Content: "No matches found."}, nil
	}
	return &tools.Result{Content: strings.Join(results, "\n")}, nil
}

// ─── DeleteFileTool ──────────────────────────────────────────────────────────

type DeleteFileTool struct{}

func (t *DeleteFileTool) Name() string          { return "delete_file" }
func (t *DeleteFileTool) Risk() tools.RiskLevel { return tools.RiskHigh }
func (t *DeleteFileTool) Description() string {
	return "Delete a file from the filesystem. This is irreversible (unless /undo is used immediately)."
}
func (t *DeleteFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path to delete"}
		},
		"required": ["path"]
	}`)
}

type deleteInput struct {
	Path string `json:"path"`
}

func (t *DeleteFileTool) Execute(_ context.Context, raw json.RawMessage) (*tools.Result, error) {
	var in deleteInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if err := checkSandbox(in.Path); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	before, _ := os.ReadFile(in.Path)
	if err := os.Remove(in.Path); err != nil {
		return &tools.Result{IsError: true, Content: err.Error()}, nil
	}
	if SnapshotFunc != nil {
		SnapshotFunc("delete_file", fmt.Sprintf("%d", time.Now().UnixNano()), in.Path, before, nil)
	}
	return &tools.Result{Content: fmt.Sprintf("Deleted %s", in.Path)}, nil
}

// ─── Register all filesystem tools ───────────────────────────────────────────

func Register(r interface {
	Register(t interface {
		Name() string
		Risk() tools.RiskLevel
		Description() string
		Schema() json.RawMessage
		Execute(ctx context.Context, input json.RawMessage) (*tools.Result, error)
	})
}) {
}

func init() {
	tools.Global.Register(&ReadFileTool{})
	tools.Global.Register(&WriteFileTool{})
	tools.Global.Register(&EditFileTool{})
	tools.Global.Register(&ListDirTool{})
	tools.Global.Register(&SearchFilesTool{})
	tools.Global.Register(&DeleteFileTool{})
}
