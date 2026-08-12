//go:build darwin

package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestRunOAuthObservesIsolatedTokenAndAuthenticatedTools(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fake-agy-oauth")
	content := `#!/bin/sh
set -eu
token_dir="$HOME/.gemini/antigravity"
cache_dir="$HOME/.gemini/antigravity-cli/mcp/mcp-interop-target"
mkdir -p "$token_dir" "$cache_dir"
printf '%s' 'DO-NOT-READ-FAKE-TOKEN' > "$token_dir/mcp_oauth_tokens.json"
printf '%s\n' '{"name":"ping"}' > "$cache_dir/ping.json"
`
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}

	var terminal bytes.Buffer
	adapter := New(script, "agy-test", WithOAuthIO(strings.NewReader("fixture-code\r"), &terminal), WithOAuthTimeout(10*time.Second))
	session, err := interop.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Cleanup()

	result, err := adapter.Run(context.Background(), interop.Target{Endpoint: "https://example.com/mcp"}, session)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed() {
		t.Fatalf("OAuth result did not pass: %#v", result)
	}
	auth, _ := result.Get(interop.StageAuth)
	if !strings.Contains(auth.Message, "isolated HOME") {
		t.Fatalf("auth message does not record isolation evidence: %q", auth.Message)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("DO-NOT-READ-FAKE-TOKEN")) || bytes.Contains(encoded, []byte("fixture-code")) {
		t.Fatalf("OAuth secret material leaked into result: %s", encoded)
	}
}

func TestRunOAuthKeepsAuthPassWhenTokenAppearsButToolsDoNot(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fake-agy-oauth-token-only")
	content := `#!/bin/sh
set -eu
token_dir="$HOME/.gemini/antigravity"
mkdir -p "$token_dir"
printf '%s' 'FAKE-TOKEN' > "$token_dir/mcp_oauth_tokens.json"
`
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}

	adapter := New(script, "agy-test", WithOAuthIO(nil, nil), WithOAuthTimeout(10*time.Second))
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
	assertStage(t, result, interop.StageInit, interop.StatusUnknown)
	assertStage(t, result, interop.StageTools, interop.StatusUnknown)
}

func TestDriveOAuthNavigationWritesExpectedSequence(t *testing.T) {
	var output bytes.Buffer
	driveOAuthNavigation(context.Background(), &output, strings.NewReader("fixture-code\r"), 0, 0)

	if got, want := output.String(), "/mcp\r\rfixture-code\r"; got != want {
		t.Fatalf("OAuth navigation bytes = %q, want %q", got, want)
	}
}

func TestOAuthTokenObservationUsesOnlyFileMetadata(t *testing.T) {
	home := t.TempDir()
	path := oauthTokenPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("opaque-secret"), 0o000); err != nil {
		t.Fatal(err)
	}
	seen, err := oauthTokenObserved(home)
	if err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("expected non-empty isolated OAuth token file to be observed")
	}
}
