package doctorcfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCodexFromProjectAndAuth(t *testing.T) {
	tmp := t.TempDir()

	codexHome := filepath.Join(tmp, "homecodex")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatalf("mkdir codex home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(strings.TrimSpace(`
model = "gpt-user"
model_provider = "openai"

[model_providers.openai]
base_url = "https://user.example/v1"
env_key = "OPENAI_API_KEY"
`)), 0o644); err != nil {
		t.Fatalf("write user config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "auth.json"), []byte(`{"tokens":{"access_token":"tok-from-auth"}}`), 0o644); err != nil {
		t.Fatalf("write auth json: %v", err)
	}

	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir root .codex: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".codex", "config.toml"), []byte(strings.TrimSpace(`
model = "gpt-root"
`)), 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	wd := filepath.Join(repo, "service", "api")
	if err := os.MkdirAll(filepath.Join(wd, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir nested .codex: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wd, ".codex", "config.toml"), []byte(strings.TrimSpace(`
model = "gpt-nested"
`)), 0o644); err != nil {
		t.Fatalf("write nested config: %v", err)
	}

	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("OPENAI_API_KEY", "")

	result := ResolveCodex(CodexOptions{CWD: wd})
	if result.BaseURL != "https://user.example/v1" {
		t.Fatalf("expected base URL from user config, got %q", result.BaseURL)
	}
	if result.Model != "gpt-nested" {
		t.Fatalf("expected model from nested project config, got %q", result.Model)
	}
	if result.APIKey != "tok-from-auth" {
		t.Fatalf("expected API key from auth token, got %q", result.APIKey)
	}
	if !strings.Contains(result.CheckDetails, "profile=default") {
		t.Fatalf("expected profile detail, got %q", result.CheckDetails)
	}
}

func TestResolveCodexProfileOverrideAndFallbackKeys(t *testing.T) {
	tmp := t.TempDir()
	codexHome := filepath.Join(tmp, "homecodex")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatalf("mkdir codex home: %v", err)
	}

	cfg := strings.TrimSpace(`
model = "gpt-default"
model_provider = "openai"
openai_api_base = "https://api.compat.example/v1"
openai_api_key_env_var = "COMPAT_KEY"

[profiles.alt]
model = "gpt-alt"
model_provider = "openai"
`)
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("COMPAT_KEY", "compat-secret")

	result := ResolveCodex(CodexOptions{CWD: tmp, ProfileOverride: "alt"})
	if result.BaseURL != "https://api.compat.example/v1" {
		t.Fatalf("expected compatibility base URL, got %q", result.BaseURL)
	}
	if result.Model != "gpt-alt" {
		t.Fatalf("expected profile override model, got %q", result.Model)
	}
	if result.APIKey != "compat-secret" {
		t.Fatalf("expected env key API key, got %q", result.APIKey)
	}
	if !result.CheckOK {
		t.Fatalf("expected check OK, got message %q", result.CheckMessage)
	}
}

func TestResolveCodexMissingFields(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CODEX_HOME", filepath.Join(tmp, "does-not-exist"))

	result := ResolveCodex(CodexOptions{CWD: tmp})
	if result.CheckOK {
		t.Fatal("expected codex check to fail when config is missing")
	}
	if !strings.Contains(result.CheckMessage, "missing") {
		t.Fatalf("expected missing fields in message, got %q", result.CheckMessage)
	}
}
