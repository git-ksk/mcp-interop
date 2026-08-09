//go:build darwin

package antigravity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestRunPassesWhenPTYChildMaterializesToolCache(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fake-agy")
	content := `#!/bin/sh
set -eu
root="$HOME/.gemini/antigravity-cli/mcp/mcp-interop-target"
mkdir -p "$root"
printf '%s\n' '{"name":"ping"}' > "$root/ping.json"
sleep 10
`
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}

	adapter := New(script, "agy-test")
	adapter.timeout = 3 * time.Second
	session, err := interop.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Cleanup()

	result, err := adapter.Run(context.Background(), interop.Target{Endpoint: "https://example.com/mcp"}, session)
	if err != nil {
		t.Fatal(err)
	}
	assertStage(t, result, interop.StageReach, interop.StatusPass)
	assertStage(t, result, interop.StageAuth, interop.StatusPass)
	assertStage(t, result, interop.StageInit, interop.StatusPass)
	assertStage(t, result, interop.StageTools, interop.StatusPass)
}

func TestRunReapsDescendantsBeforeSessionCleanup(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fake-agy")
	content := `#!/bin/sh
set -eu
root="$HOME/.gemini/antigravity-cli/mcp/mcp-interop-target"
mkdir -p "$root"
echo $$ > "$HOME/fake-agy.pid"
(
  echo $$ > "$HOME/fake-writer.pid"
  while :; do
    printf '%s\n' "still-writing" > "$root/live-state.tmp"
    sleep 0.05
  done
) &
printf '%s\n' '{"name":"ping"}' > "$root/ping.json"
wait
`
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}

	adapter := New(script, "agy-test")
	adapter.timeout = 3 * time.Second
	session, err := interop.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	root := session.Root()

	result, err := adapter.Run(context.Background(), interop.Target{Endpoint: "https://example.com/mcp"}, session)
	if err != nil {
		_ = session.Cleanup()
		t.Fatal(err)
	}
	assertStage(t, result, interop.StageTools, interop.StatusPass)

	for _, name := range []string{"fake-agy.pid", "fake-writer.pid"} {
		pid := readPIDFile(t, filepath.Join(root, "antigravity-home", name))
		assertProcessGone(t, pid)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("automatic session cleanup after adapter return: %v", err)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary session root still exists after cleanup: %v", err)
	}
}

func TestRunReturnsUnknownWhenNoToolCacheAppears(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fake-agy")
	content := `#!/bin/sh
set -eu
echo $$ > "$HOME/fake-agy.pid"
sleep 10
`
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}

	adapter := New(script, "agy-test")
	adapter.timeout = 300 * time.Millisecond
	session, err := interop.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Cleanup()

	result, err := adapter.Run(context.Background(), interop.Target{Endpoint: "https://example.com/mcp"}, session)
	if err != nil {
		t.Fatal(err)
	}
	assertStage(t, result, interop.StageReach, interop.StatusUnknown)
	assertStage(t, result, interop.StageAuth, interop.StatusUnknown)
	assertStage(t, result, interop.StageInit, interop.StatusUnknown)
	assertStage(t, result, interop.StageTools, interop.StatusUnknown)

	pid := readPIDFile(t, filepath.Join(session.Root(), "antigravity-home", "fake-agy.pid"))
	assertProcessGone(t, pid)
}

func readPIDFile(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parse PID from %s: %v", path, err)
	}
	return pid
}

func assertProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("process %d still exists after adapter return (kill(0)=%v)", pid, err)
	}
}

func assertStage(t *testing.T, result interop.Result, stage interop.Stage, want interop.Status) {
	t.Helper()
	got, ok := result.Get(stage)
	if !ok {
		t.Fatalf("missing stage %s", stage)
	}
	if got.Status != want {
		t.Fatalf("stage %s status = %s, want %s (message: %s)", stage, got.Status, want, got.Message)
	}
}
