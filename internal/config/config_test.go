package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	wd := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	got, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.LLM.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("BaseURL = %q", got.LLM.BaseURL)
	}
	if got.LLM.Model != "deepseek-chat" {
		t.Fatalf("Model = %q", got.LLM.Model)
	}
	if got.LLM.APIKey != "" {
		t.Fatalf("APIKey = %q, want empty default", got.LLM.APIKey)
	}
	if got.Agent.MaxContextTokens != 12000 {
		t.Fatalf("Agent.MaxContextTokens = %d", got.Agent.MaxContextTokens)
	}
	if got.Agent.MaxFilePatchBytes != 20000 {
		t.Fatalf("Agent.MaxFilePatchBytes = %d", got.Agent.MaxFilePatchBytes)
	}
	if got.Agent.MaxFiles != 50 {
		t.Fatalf("Agent.MaxFiles = %d", got.Agent.MaxFiles)
	}
	if got.Agent.TimeoutSeconds != 180 {
		t.Fatalf("Agent.TimeoutSeconds = %d", got.Agent.TimeoutSeconds)
	}
}

func TestLoadTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aireview.toml")
	if err := os.WriteFile(path, []byte(`
language = "en-US"
review_focus = ["security"]

[llm]
base_url = "https://api.deepseek.com/v1/"
api_key = "test-key"
model = "deepseek-reasoner"

[agent]
max_context_tokens = 8000
max_file_patch_bytes = 16000
max_files = 25
timeout_seconds = 90

[mysql]
dsn = "root:123456@tcp(127.0.0.1:3306)/aireview?parseTime=true"
`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Language != "en-US" {
		t.Fatalf("Language = %q", got.Language)
	}
	if got.LLM.BaseURL != "https://api.deepseek.com/v1" {
		t.Fatalf("BaseURL = %q", got.LLM.BaseURL)
	}
	if got.LLM.APIKey != "test-key" {
		t.Fatalf("APIKey = %q", got.LLM.APIKey)
	}
	if got.LLM.Model != "deepseek-reasoner" {
		t.Fatalf("Model = %q", got.LLM.Model)
	}
	if got.MySQL.DSN != "root:123456@tcp(127.0.0.1:3306)/aireview?parseTime=true" {
		t.Fatalf("MySQL.DSN = %q", got.MySQL.DSN)
	}
	if got.Agent.MaxContextTokens != 8000 {
		t.Fatalf("Agent.MaxContextTokens = %d", got.Agent.MaxContextTokens)
	}
	if got.Agent.MaxFilePatchBytes != 16000 {
		t.Fatalf("Agent.MaxFilePatchBytes = %d", got.Agent.MaxFilePatchBytes)
	}
	if got.Agent.MaxFiles != 25 {
		t.Fatalf("Agent.MaxFiles = %d", got.Agent.MaxFiles)
	}
	if got.Agent.TimeoutSeconds != 90 {
		t.Fatalf("Agent.TimeoutSeconds = %d", got.Agent.TimeoutSeconds)
	}
	if len(got.ReviewFocus) != 1 || got.ReviewFocus[0] != "security" {
		t.Fatalf("ReviewFocus = %#v", got.ReviewFocus)
	}
}

func TestLoadDoesNotReadAPIKeyFromEnv(t *testing.T) {
	t.Setenv("LLM_API_KEY", "env-key")

	path := filepath.Join(t.TempDir(), "aireview.toml")
	if err := os.WriteFile(path, []byte(`
[llm]
api_key = "toml-key"
`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.LLM.APIKey != "toml-key" {
		t.Fatalf("APIKey = %q, want TOML value", got.LLM.APIKey)
	}
}
