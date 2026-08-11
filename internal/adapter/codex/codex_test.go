package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestRPCClientIgnoresNotificationsAndUnrelatedResponses(t *testing.T) {
	input := strings.Join([]string{
		`{"method":"configWarning","params":{}}`,
		`{"id":99,"result":{"ignored":true}}`,
		`{"id":1,"result":{"value":"ok"}}`,
		"",
	}, "\n")
	var output bytes.Buffer
	client := newRPCClient(strings.NewReader(input), &output)

	var result struct {
		Value string `json:"value"`
	}
	if err := client.call("test/method", map[string]any{"x": 1}, &result); err != nil {
		t.Fatal(err)
	}
	if result.Value != "ok" {
		t.Fatalf("unexpected result: %#v", result)
	}

	var request struct {
		ID     int64  `json:"id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &request); err != nil {
		t.Fatalf("invalid request JSON: %v", err)
	}
	if request.ID != 1 || request.Method != "test/method" {
		t.Fatalf("unexpected request: %#v", request)
	}
}

func TestRPCClientRetainsErrorDetailsWithoutExposingThem(t *testing.T) {
	input := `{"id":1,"error":{"code":-32601,"message":"secret remote text","data":{"detail":"secret data"}}}` + "\n"
	client := newRPCClient(strings.NewReader(input), &bytes.Buffer{})
	err := client.call("missing", map[string]any{}, nil)
	if err == nil || err.Error() != "codex app-server JSON-RPC error -32601" {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("raw app-server error leaked through Error(): %v", err)
	}

	var rpcErr *rpcCallError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected typed rpcCallError, got %T", err)
	}
	if rpcErr.Message != "secret remote text" || !strings.Contains(string(rpcErr.Data), "secret data") {
		t.Fatalf("classification details were not retained: %#v", rpcErr)
	}
}

func TestClassifyOAuthStartFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want interop.ReasonCode
	}{
		{
			name: "Monokura-like explicit unsupported",
			err: &rpcCallError{
				Code:    -32000,
				Message: "Registration failed: Dynamic registration failed: Registration failed: Dynamic client registration not supported",
			},
			want: interop.ReasonDCRUnsupported,
		},
		{
			name: "dynamic registration failure",
			err: &rpcCallError{
				Code:    -32000,
				Message: "OAuth login failed",
				Data:    json.RawMessage(`{"error":"dynamic client registration failed with status 500"}`),
			},
			want: interop.ReasonDCRFailed,
		},
		{
			name: "generic OAuth failure is not overclassified",
			err: &rpcCallError{
				Code:    -32000,
				Message: "OAuth login failed",
			},
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyOAuthStartFailure(test.err); got != test.want {
				t.Fatalf("reason = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLoginOAuthReportsDCRUnsupported(t *testing.T) {
	input := `{"id":1,"error":{"code":-32000,"message":"Registration failed: Dynamic registration failed: Registration failed: Dynamic client registration not supported"}}` + "\n"
	rpc := newRPCClient(strings.NewReader(input), &bytes.Buffer{})
	adapter := New("codex", "codex-cli 0.133.0")
	result := interop.NewResult(clientID, clientName, "codex-cli 0.133.0", "https://example.com/mcp")

	_, ok, err := adapter.loginOAuth(context.Background(), rpc, &result)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("OAuth login unexpectedly succeeded")
	}
	auth, found := result.Get(interop.StageAuth)
	if !found {
		t.Fatal("missing auth stage")
	}
	if auth.Status != interop.StatusFail || auth.ReasonCode != interop.ReasonDCRUnsupported {
		t.Fatalf("unexpected auth result: %#v", auth)
	}
	if strings.Contains(auth.Message, "Registration failed") {
		t.Fatalf("raw client error leaked into result message: %q", auth.Message)
	}
}

func TestInterpretStatusWithToolsProvesLiveInterop(t *testing.T) {
	result := interop.NewResult(clientID, clientName, "codex-cli test", "https://example.com/mcp")
	interpretStatus(&result, serverStatus{
		Name:       testServerName,
		AuthStatus: "unsupported",
		Tools: map[string]json.RawMessage{
			"ping": json.RawMessage(`{"name":"ping"}`),
		},
	})

	assertStage(t, result, interop.StageReach, interop.StatusPass)
	assertStage(t, result, interop.StageAuth, interop.StatusPass)
	assertStage(t, result, interop.StageInit, interop.StatusPass)
	assertStage(t, result, interop.StageTools, interop.StatusPass)
}

func TestInterpretStatusUnknownAuthWithToolsProvesNoUnresolvedAuthGate(t *testing.T) {
	result := interop.NewResult(clientID, clientName, "codex-cli test", "https://example.com/mcp")
	interpretStatus(&result, serverStatus{
		Name:       testServerName,
		AuthStatus: "newFutureState",
		Tools: map[string]json.RawMessage{
			"ping": json.RawMessage(`{"name":"ping"}`),
		},
	})

	assertStage(t, result, interop.StageReach, interop.StatusPass)
	assertStage(t, result, interop.StageAuth, interop.StatusPass)
	assertStage(t, result, interop.StageInit, interop.StatusPass)
	assertStage(t, result, interop.StageTools, interop.StatusPass)
}

func TestInterpretStatusUnknownAuthWithoutToolsRemainsUnknown(t *testing.T) {
	result := interop.NewResult(clientID, clientName, "codex-cli test", "https://example.com/mcp")
	interpretStatus(&result, serverStatus{
		Name:       testServerName,
		AuthStatus: "newFutureState",
		Tools:      map[string]json.RawMessage{},
	})

	assertStage(t, result, interop.StageReach, interop.StatusUnknown)
	assertStage(t, result, interop.StageAuth, interop.StatusUnknown)
	assertStage(t, result, interop.StageInit, interop.StatusUnknown)
	assertStage(t, result, interop.StageTools, interop.StatusUnknown)
}

func TestInterpretStatusDoesNotTreatEmptyInventoryAsSuccess(t *testing.T) {
	result := interop.NewResult(clientID, clientName, "codex-cli test", "https://example.com/mcp")
	interpretStatus(&result, serverStatus{
		Name:       testServerName,
		AuthStatus: "unsupported",
		Tools:      map[string]json.RawMessage{},
	})

	assertStage(t, result, interop.StageReach, interop.StatusUnknown)
	assertStage(t, result, interop.StageAuth, interop.StatusUnknown)
	assertStage(t, result, interop.StageInit, interop.StatusUnknown)
	assertStage(t, result, interop.StageTools, interop.StatusUnknown)
}

func TestInterpretStatusStopsAtRequiredOAuth(t *testing.T) {
	result := interop.NewResult(clientID, clientName, "codex-cli test", "https://example.com/mcp")
	interpretStatus(&result, serverStatus{
		Name:       testServerName,
		AuthStatus: "notLoggedIn",
		Tools:      map[string]json.RawMessage{},
	})

	assertStage(t, result, interop.StageReach, interop.StatusUnknown)
	assertStage(t, result, interop.StageAuth, interop.StatusSkip)
	assertStage(t, result, interop.StageInit, interop.StatusSkip)
	assertStage(t, result, interop.StageTools, interop.StatusSkip)
}

func TestInterpretStatusPreservesObservedAuthenticatedState(t *testing.T) {
	for _, authStatus := range []string{"oAuth", "bearerToken"} {
		t.Run(authStatus, func(t *testing.T) {
			result := interop.NewResult(clientID, clientName, "codex-cli test", "https://example.com/mcp")
			interpretStatus(&result, serverStatus{
				Name:       testServerName,
				AuthStatus: authStatus,
				Tools:      map[string]json.RawMessage{},
			})
			assertStage(t, result, interop.StageAuth, interop.StatusPass)
			assertStage(t, result, interop.StageTools, interop.StatusUnknown)
		})
	}
}

func TestWriteConfigUsesOnlyIsolatedFile(t *testing.T) {
	dir := t.TempDir()
	endpoint := "https://example.com/mcp?tenant=acme&value=%22quoted%22"
	if err := writeConfig(dir, endpoint); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "config.toml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, `mcp_oauth_credentials_store = "file"`) {
		t.Fatalf("Codex OAuth storage is not forced to isolated file mode: %s", text)
	}
	if !strings.Contains(text, "[mcp_servers."+testServerName+"]") || !strings.Contains(text, endpoint) {
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

func TestOptionsEnableAuthorizationWithoutChangingDefault(t *testing.T) {
	plain := New("codex", "test")
	if plain.authorize != nil || plain.oauthTimeout != defaultOAuthTimeout {
		t.Fatalf("unexpected default adapter options: %#v", plain)
	}

	called := false
	handler := func(context.Context, string) error {
		called = true
		return errors.New("expected test error")
	}
	configured := New("codex", "test", WithAuthorizationHandler(handler), WithOAuthTimeout(3*time.Second))
	if configured.authorize == nil || configured.oauthTimeout != 3*time.Second {
		t.Fatalf("OAuth options were not applied: %#v", configured)
	}
	if err := configured.authorize(context.Background(), "https://example.com/authorize"); err == nil || !called {
		t.Fatal("configured authorization handler did not run")
	}
}

func TestFindStatusMatchesOnlyTarget(t *testing.T) {
	statuses := []serverStatus{{Name: "other"}, {Name: testServerName, AuthStatus: "oAuth"}}
	status, ok := findStatus(statuses, testServerName)
	if !ok || status.AuthStatus != "oAuth" {
		t.Fatalf("unexpected match: %#v, %v", status, ok)
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
