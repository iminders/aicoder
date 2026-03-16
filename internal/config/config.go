package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// MCPServerConfig defines one MCP server entry.
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	URL     string            `json:"url"` // for SSE transport
}

// Config is the resolved, merged configuration.
type Config struct {
	// LLM
	Provider  string `json:"provider"` // anthropic | openai | ollama
	Model     string `json:"model"`
	MaxTokens int    `json:"maxTokens"`
	BaseURL   string `json:"baseUrl"` // custom endpoint

	// Security
	AutoApprove         bool     `json:"autoApprove"`
	AutoApproveReads    bool     `json:"autoApproveReads"`
	AutoApproveCommands []string `json:"autoApproveCommands"`
	ForbiddenCommands   []string `json:"forbiddenCommands"`
	DangerouslySkip     bool     `json:"-"` // CLI flag only

	// Files
	BackupOnWrite bool `json:"backupOnWrite"`

	// UI
	Theme    string `json:"theme"`
	Language string `json:"language"`

	// Network
	Proxy string `json:"proxy"`

	// MCP
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`

	// Runtime (not persisted)
	Verbose bool `json:"-"`
}

var defaultConfig = Config{
	Provider:         "anthropic",
	Model:            "claude-sonnet-4-5",
	MaxTokens:        8192,
	AutoApprove:      false,
	AutoApproveReads: true,
	ForbiddenCommands: []string{
		"rm -rf /", "sudo rm", "mkfs", "dd if=",
	},
	BackupOnWrite: true,
	Theme:         "dark",
	Language:      "zh-CN",
}

// Load merges configs from lowest to highest priority:
// defaults -> user (~/.aicoder/config.json) -> project (.aicoder/config.json) -> env vars.
func Load() (*Config, error) {
	cfg := defaultConfig // start with defaults

	// User-level config
	if home, err := os.UserHomeDir(); err == nil {
		_ = mergeFile(&cfg, filepath.Join(home, ".aicoder", "config.json"))
	}

	// Project-level config (walk up from cwd)
	if projCfg := findProjectConfig(); projCfg != "" {
		_ = mergeFile(&cfg, projCfg)
	}

	// Environment variable overrides
	applyEnv(&cfg)

	return &cfg, nil
}

func findProjectConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		p := filepath.Join(dir, ".aicoder", "config.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func mergeFile(dst *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Unmarshal into a temporary map so we only override keys that are present
	var overlay map[string]json.RawMessage
	if err := json.Unmarshal(data, &overlay); err != nil {
		return err
	}
	// Re-marshal and merge field by field via full unmarshal into dst
	return json.Unmarshal(data, dst)
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("AICODER_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("AICODER_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("AICODER_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("HTTPS_PROXY"); v != "" && cfg.Proxy == "" {
		cfg.Proxy = v
	}
}

// APIKey returns the API key for the given provider from environment variables.
func APIKey(provider string) string {
	switch provider {
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "google":
		return os.Getenv("GOOGLE_API_KEY")
	default:
		return os.Getenv("AICODER_API_KEY")
	}
}

// Save writes the config to the user-level config file.
func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".aicoder")
	_ = os.MkdirAll(dir, 0700)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0600)
}
