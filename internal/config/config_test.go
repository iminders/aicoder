package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAPIKey(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "test-key-123")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	key := APIKey("anthropic")
	if key != "test-key-123" {
		t.Errorf("expected test-key-123, got %s", key)
	}

	if APIKey("unknown") != "" {
		t.Error("expected empty key for unknown provider")
	}
}

func TestMergeFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	_ = os.WriteFile(cfgPath, []byte(`{"model":"gpt-4o","maxTokens":4096}`), 0600)

	cfg := defaultConfig
	if err := mergeFile(&cfg, cfgPath); err != nil {
		t.Fatalf("mergeFile failed: %v", err)
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", cfg.Model)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected 4096, got %d", cfg.MaxTokens)
	}
}

func TestApplyEnv(t *testing.T) {
	os.Setenv("AICODER_MODEL", "claude-opus-4-5")
	os.Setenv("AICODER_PROVIDER", "openai")
	defer func() {
		os.Unsetenv("AICODER_MODEL")
		os.Unsetenv("AICODER_PROVIDER")
	}()

	cfg := defaultConfig
	applyEnv(&cfg)
	if cfg.Model != "claude-opus-4-5" {
		t.Errorf("expected claude-opus-4-5, got %s", cfg.Model)
	}
	if cfg.Provider != "openai" {
		t.Errorf("expected openai, got %s", cfg.Provider)
	}
}
