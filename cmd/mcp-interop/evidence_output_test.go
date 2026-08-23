package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWritePrivateJSONFilePreservesExistingFileOnEncodeFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.json")
	const original = "keep-me\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writePrivateJSONFile(path, map[string]any{"unsupported": make(chan int)})
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

func TestRunEvidenceMergeReplacesExistingOutputWithPrivateFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	output := filepath.Join(dir, "merged.json")
	if err := os.WriteFile(input, []byte(`{"schema_version":3,"tool_challenge":{"expected":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("old public content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := runEvidenceMerge([]string{input, "-o", output}); code != 0 {
		t.Fatalf("merge exit=%d", code)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "old public content") || !strings.Contains(string(data), `"schema_version": 3`) {
		t.Fatalf("unexpected merged output: %s", data)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(output)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("output mode=%#o, want 0600", info.Mode().Perm())
		}
	}
}

func TestWritePrivateJSONFileReplacesSymlinkWithoutTouchingTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available in Windows CI")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	output := filepath.Join(dir, "evidence.json")
	const targetContent = "must-not-change\n"
	if err := os.WriteFile(target, []byte(targetContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, output); err != nil {
		t.Fatal(err)
	}

	if err := writePrivateJSONFile(output, map[string]string{"status": "ok"}); err != nil {
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
