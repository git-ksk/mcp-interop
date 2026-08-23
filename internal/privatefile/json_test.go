package privatefile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteJSONPreservesExistingFileOnEncodeFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.json")
	const original = "keep-me\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := WriteJSON(path, map[string]any{"unsupported": make(chan int)})
	if err == nil {
		t.Fatal("expected JSON encoding failure")
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != original {
		t.Fatalf("existing output changed after encode failure: %q", got)
	}
}

func TestWriteJSONReplacesExistingFileWithPrivateMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteJSON(path, map[string]string{"status": "ok"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "ok" {
		t.Fatalf("unexpected JSON: %#v", got)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("output mode=%#o, want 0600", info.Mode().Perm())
		}
	}
}

func TestWriteJSONReplacesSymlinkWithoutTouchingTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available in Windows CI")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	output := filepath.Join(dir, "output.json")
	const targetContent = "must-not-change\n"
	if err := os.WriteFile(target, []byte(targetContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, output); err != nil {
		t.Fatal(err)
	}

	if err := WriteJSON(output, map[string]string{"status": "ok"}); err != nil {
		t.Fatal(err)
	}
	gotTarget, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotTarget) != targetContent {
		t.Fatalf("symlink target changed: %q", gotTarget)
	}
	info, err := os.Lstat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("output remained a symlink")
	}
}
