package antigravity

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteConfigUsesIsolatedGlobalConfig(t *testing.T) {
	home := t.TempDir()
	endpoint := "https://example.com/mcp?tenant=acme&value=%22quoted%22"
	if err := writeConfig(home, endpoint); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, testServerName) || !strings.Contains(text, `"serverUrl": "`+endpoint+`"`) {
		t.Fatalf("unexpected config: %s", text)
	}
	if strings.Contains(text, `"url":`) || strings.Contains(text, `"httpUrl":`) {
		t.Fatalf("legacy remote URL key present: %s", text)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("config permissions = %o, want 600", got)
		}
	}

	settingsPath := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	settings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(settings), `"modelProvider": "gemini"`) {
		t.Fatalf("isolated account-bypass setting missing: %s", settings)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("settings permissions = %o, want 600", got)
		}
	}
}

func TestReplaceEnvForHomeForcesNoAccountSessionMode(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/Users/example",
		"GEMINI_API_KEY=normal-user-secret",
		"OTHER=value",
	}
	got := replaceEnv(env, "HOME", "/tmp/isolated-home")

	values := map[string][]string{}
	for _, item := range got {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		values[strings.ToUpper(key)] = append(values[strings.ToUpper(key)], value)
	}
	if home := values["HOME"]; len(home) != 1 || home[0] != "/tmp/isolated-home" {
		t.Fatalf("HOME values = %#v", home)
	}
	if keys := values["GEMINI_API_KEY"]; len(keys) != 1 || keys[0] != isolatedGeminiAPIKey {
		t.Fatalf("GEMINI_API_KEY values = %#v", keys)
	}
	for _, item := range got {
		if strings.Contains(item, "normal-user-secret") {
			t.Fatalf("ambient Gemini API key leaked into isolated environment: %q", item)
		}
	}
}

func TestCountValidToolCacheFiles(t *testing.T) {
	home := t.TempDir()
	root := toolCacheRoot(home)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ping.json"), []byte(`{"name":"ping"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.json"), []byte(`not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte(`ignored`), 0o600); err != nil {
		t.Fatal(err)
	}

	count, err := countValidToolCacheFiles(home)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("cache count = %d, want 1", count)
	}
}

func TestCountValidToolCacheFilesMissingRoot(t *testing.T) {
	count, err := countValidToolCacheFiles(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cache count = %d, want 0", count)
	}
}
