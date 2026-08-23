package antigravity

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCountValidToolCacheFilesRejectsOversizedJSONFile(t *testing.T) {
	home := t.TempDir()
	root := toolCacheRoot(home)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "oversized.json")
	payload := append([]byte(`{"schema":"`), bytes.Repeat([]byte("x"), maxToolCacheFileBytes)...)
	payload = append(payload, []byte(`"}`)...)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := countValidToolCacheFiles(home)
	if err == nil || !strings.Contains(err.Error(), "cache file exceeds") {
		t.Fatalf("expected cache size-limit error, got %v", err)
	}
}

func TestValidBoundedToolCacheJSONAcceptsSmallValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool.json")
	if err := os.WriteFile(path, []byte(`{"name":"ping"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	valid, err := validBoundedToolCacheJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected small valid JSON cache entry")
	}
}
