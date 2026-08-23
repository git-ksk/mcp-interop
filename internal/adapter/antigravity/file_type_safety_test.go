package antigravity

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCountValidToolCacheFilesIgnoresSymlinkJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available in Windows CI")
	}
	home := t.TempDir()
	root := toolCacheRoot(home)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"name":"must-not-count"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "tool.json")); err != nil {
		t.Fatal(err)
	}

	count, err := countValidToolCacheFiles(home)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("symlink cache entry counted as evidence: %d", count)
	}
}

func TestOAuthTokenObservedDoesNotFollowSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available in Windows CI")
	}
	home := t.TempDir()
	path := oauthTokenPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "real-token-state.json")
	if err := os.WriteFile(outside, []byte(`{"token":"outside"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}

	observed, err := oauthTokenObserved(home)
	if err != nil {
		t.Fatal(err)
	}
	if observed {
		t.Fatal("symlink token marker was accepted as isolated OAuth evidence")
	}
}
