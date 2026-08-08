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
