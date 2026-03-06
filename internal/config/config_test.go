package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrefersXDGConfigHome(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	xdgConfigHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	xdgPath := writeConfig(t, filepath.Join(xdgConfigHome, "vigilo", "config.yaml"), "https://example.com/xdg")
	writeConfig(t, filepath.Join(os.Getenv("HOME"), ".config", "vigilo", "config.yaml"), "https://example.com/home")

	cfg, resolvedPath, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if resolvedPath != xdgPath {
		t.Fatalf("Load() resolved path = %q, want %q", resolvedPath, xdgPath)
	}

	if cfg.Slack.WebhookURL != "https://example.com/xdg" {
		t.Fatalf("Load() webhook = %q, want %q", cfg.Slack.WebhookURL, "https://example.com/xdg")
	}
}

func TestLoadFallsBackToHomeConfig(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "missing"))

	homePath := writeConfig(t, filepath.Join(homeDir, ".config", "vigilo", "config.yaml"), "https://example.com/home")

	cfg, resolvedPath, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if resolvedPath != homePath {
		t.Fatalf("Load() resolved path = %q, want %q", resolvedPath, homePath)
	}

	if cfg.Slack.WebhookURL != "https://example.com/home" {
		t.Fatalf("Load() webhook = %q, want %q", cfg.Slack.WebhookURL, "https://example.com/home")
	}
}

func writeConfig(t *testing.T, path string, webhookURL string) string {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	data := []byte("slack:\n  webhook_url: \"" + webhookURL + "\"\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
