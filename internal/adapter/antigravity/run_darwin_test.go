//go:build darwin

package antigravity

import (
	"context"
	"os"
	"path/filepath"
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

func TestRunReturnsUnknownWhenNoToolCacheAppears(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fake-agy")
	content := `#!/bin/sh
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
