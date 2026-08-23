package artifact

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileReplacesExistingPublicFilePrivately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.json")
	if err := os.WriteFile(path, []byte("old public content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := NewArtifact([]Run{validRun(t)})
	if err := WriteFile(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Runs) != 1 || got.Runs[0].Endpoint != want.Runs[0].Endpoint {
		t.Fatalf("unexpected artifact after replacement: %#v", got)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("artifact mode=%#o, want 0600", info.Mode().Perm())
		}
	}
}

func TestWriteFileReplacesSymlinkWithoutTouchingTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available in Windows CI")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	path := filepath.Join(dir, "artifact.json")
	const targetContent = "must-not-change\n"
	if err := os.WriteFile(target, []byte(targetContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(path, NewArtifact([]Run{validRun(t)})); err != nil {
		t.Fatal(err)
	}
	gotTarget, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotTarget) != targetContent {
		t.Fatalf("symlink target changed: %q", gotTarget)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("artifact output remained a symlink")
	}
	if _, err := ReadFile(path); err != nil {
		t.Fatalf("replacement is not a valid artifact: %v", err)
	}
}
