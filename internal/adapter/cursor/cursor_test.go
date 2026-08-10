package cursor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

type fakeRunner struct {
	results map[string]commandResult
	calls   []string
	homes   []string
}

func (f *fakeRunner) Run(_ context.Context, _ string, env []string, _ string, args ...string) commandResult {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	for _, item := range env {
		if strings.HasPrefix(item, "HOME=") {
			f.homes = append(f.homes, strings.TrimPrefix(item, "HOME="))
		}
	}
	if result, ok := f.results[key]; ok {
		return result
	}
	return commandResult{}
}

type fakeOAuthRunner struct {
	result           commandResult
	calls            []string
	homes            []string
	authorizationURL string
	handlerCalled    bool
}

func (f *fakeOAuthRunner) RunOAuth(ctx context.Context, _ string, env []string, _ string, authorize AuthorizationHandler, args ...string) commandResult {
	f.calls = append(f.calls, strings.Join(args, " "))
	for _, item := range env {
		if strings.HasPrefix(item, "HOME=") {
			f.homes = append(f.homes, strings.TrimPrefix(item, "HOME="))
		}
	}
	if authorize != nil && f.authorizationURL != "" {
		f.handlerCalled = true
		if err := authorize(ctx, f.authorizationURL); err != nil {
			return commandResult{err: err}
		}
	}
	return f.result
}

func TestAdapterPassesWhenListToolsSucceeds(t *testing.T) {
	runner := &fakeRunner{results: map[string]commandResult{
		"mcp enable " + testServerName: {},
		"mcp list":                     {stdout: testServerName + " connected\n"},
		"mcp list-tools " + testServerName: {
			stdout: "Available tools:\n- ping\n",
		},
	}}
	adapter := &Adapter{executable: "cursor-agent", version: "test", timeout: defaultTimeout, runner: runner}
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

	wantCalls := []string{
		"mcp enable " + testServerName,
		"mcp list",
		"mcp list-tools " + testServerName,
	}
	if strings.Join(runner.calls, "|") != strings.Join(wantCalls, "|") {
		t.Fatalf("calls = %#v, want %#v", runner.calls, wantCalls)
	}
	if len(runner.homes) != 3 || runner.homes[0] == "" {
		t.Fatalf("expected isolated HOME on every command, got %#v", runner.homes)
	}
	for _, home := range runner.homes[1:] {
		if home != runner.homes[0] {
			t.Fatalf("commands used different HOME values: %#v", runner.homes)
		}
	}
}

func TestAdapterStopsAtOAuthBoundaryWithoutOptIn(t *testing.T) {
	runner := &fakeRunner{results: map[string]commandResult{
		"mcp list": {stdout: testServerName + " authentication required\n"},
		"mcp list-tools " + testServerName: {
			stderr: "Authentication required. Run cursor-agent mcp login " + testServerName,
			err:    context.DeadlineExceeded,
		},
	}}
	adapter := &Adapter{executable: "cursor-agent", version: "test", timeout: defaultTimeout, runner: runner}
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
	assertStage(t, result, interop.StageAuth, interop.StatusSkip)
	assertStage(t, result, interop.StageInit, interop.StatusSkip)
	assertStage(t, result, interop.StageTools, interop.StatusSkip)
}

func TestAdapterCompletesOAuthThenAuthenticatedToolDiscovery(t *testing.T) {
	callCount := 0
	runner := &fakeRunner{results: map[string]commandResult{
		"mcp enable " + testServerName: {},
		"mcp list":                     {stdout: testServerName + " authentication required\n"},
		"mcp list-tools " + testServerName: {
			stderr: "Authentication required. Run cursor-agent mcp login " + testServerName,
			err:    errors.New("auth required"),
		},
	}}
	baseRunner := runner
	oauthRunner := &fakeOAuthRunner{authorizationURL: "http://127.0.0.1:9999/authorize?state=s&redirect_uri=http%3A%2F%2F127.0.0.1%3A54321%2Fcallback&code_challenge=c"}
	authorize := func(_ context.Context, _ string) error {
		callCount++
		baseRunner.results["mcp list"] = commandResult{stdout: testServerName + " connected\n"}
		baseRunner.results["mcp list-tools "+testServerName] = commandResult{stdout: "Available tools:\n- ping\n"}
		return nil
	}
	adapter := &Adapter{
		executable:   "cursor-agent",
		version:      "test",
		timeout:      defaultTimeout,
		oauthTimeout: defaultOAuthTimeout,
		authorize:    authorize,
		runner:       runner,
		oauthRunner:  oauthRunner,
	}
	session, err := interop.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Cleanup()

	result, err := adapter.Run(context.Background(), interop.Target{Endpoint: "https://example.com/mcp"}, session)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 1 || !oauthRunner.handlerCalled {
		t.Fatalf("authorization handler calls = %d, oauth handler=%v", callCount, oauthRunner.handlerCalled)
	}
	if len(oauthRunner.calls) != 1 || oauthRunner.calls[0] != "mcp login "+testServerName {
		t.Fatalf("OAuth calls = %#v", oauthRunner.calls)
	}
	assertStage(t, result, interop.StageReach, interop.StatusPass)
	assertStage(t, result, interop.StageAuth, interop.StatusPass)
	assertStage(t, result, interop.StageInit, interop.StatusPass)
	assertStage(t, result, interop.StageTools, interop.StatusPass)
	if len(oauthRunner.homes) != 1 || oauthRunner.homes[0] == "" || !strings.Contains(oauthRunner.homes[0], "cursor-home") {
		t.Fatalf("OAuth login did not use isolated HOME: %#v", oauthRunner.homes)
	}
}

func TestAdapterReportsCallbackPortConflict(t *testing.T) {
	runner := &fakeRunner{results: map[string]commandResult{
		"mcp list":                         {stdout: testServerName + " authentication required\n"},
		"mcp list-tools " + testServerName: {stderr: "Authentication required", err: errors.New("auth required")},
	}}
	oauthRunner := &fakeOAuthRunner{result: commandResult{stderr: "OAuth callback: listen tcp 127.0.0.1:49231: bind: address already in use", err: errors.New("exit 1")}}
	adapter := &Adapter{executable: "cursor-agent", version: "test", timeout: defaultTimeout, oauthTimeout: defaultOAuthTimeout, authorize: func(context.Context, string) error { return nil }, runner: runner, oauthRunner: oauthRunner}
	session, err := interop.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Cleanup()

	result, err := adapter.Run(context.Background(), interop.Target{Endpoint: "https://example.com/mcp"}, session)
	if err != nil {
		t.Fatal(err)
	}
	auth, _ := result.Get(interop.StageAuth)
	if auth.Status != interop.StatusFail || auth.ReasonCode != interop.ReasonOAuthCallbackPortConflict {
		t.Fatalf("auth = %#v", auth)
	}
}

func TestAdapterDoesNotClaimReachOnGenericListToolsFailure(t *testing.T) {
	runner := &fakeRunner{results: map[string]commandResult{
		"mcp list": {stdout: testServerName + " configured\n"},
		"mcp list-tools " + testServerName: {
			stderr: "connection failed",
			err:    context.Canceled,
		},
	}}
	adapter := &Adapter{executable: "cursor-agent", version: "test", timeout: defaultTimeout, runner: runner}
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
	assertStage(t, result, interop.StageTools, interop.StatusFail)
}

func TestWriteConfigUsesWorkspaceOnly(t *testing.T) {
	workspace := t.TempDir()
	endpoint := "https://example.com/mcp?tenant=acme&value=%22quoted%22"
	if err := writeConfig(workspace, endpoint); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(workspace, ".cursor", "mcp.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, testServerName) || !strings.Contains(text, endpoint) {
		t.Fatalf("unexpected config: %s", text)
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

func TestRequiresOAuth(t *testing.T) {
	for _, input := range []string{
		"Authentication required",
		"not logged in",
		"Run cursor-agent mcp login target",
	} {
		if !requiresOAuth(input) {
			t.Fatalf("expected OAuth requirement for %q", input)
		}
	}
	if requiresOAuth("connection refused") {
		t.Fatal("generic transport failure must not be classified as OAuth")
	}
}

func TestAuthorizationURLFromTextUsesDynamicRedirect(t *testing.T) {
	input := "Open http://127.0.0.1:8080/authorize?client_id=c&redirect_uri=http%3A%2F%2F127.0.0.1%3A51743%2Foauth%2Fcallback&state=s&code_challenge=abc to continue"
	got := authorizationURLFromText(input)
	if got == "" || !strings.Contains(got, "51743") {
		t.Fatalf("authorization URL = %q", got)
	}
	if authorizationURLFromText("docs: https://example.com/help") != "" {
		t.Fatal("non-authorization URL must not be selected")
	}
}

func TestClassifyOAuthFailure(t *testing.T) {
	cases := map[string]interop.ReasonCode{
		"EADDRINUSE callback server":                        interop.ReasonOAuthCallbackPortConflict,
		"Dynamic client registration not supported":        interop.ReasonDCRUnsupported,
		"Dynamic client registration failed with HTTP 500": interop.ReasonDCRFailed,
	}
	for input, want := range cases {
		if got := classifyOAuthFailure(input); got != want {
			t.Fatalf("classifyOAuthFailure(%q) = %q, want %q", input, got, want)
		}
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
