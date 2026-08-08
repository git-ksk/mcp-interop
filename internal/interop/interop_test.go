package interop

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNewResultHasStableStageOrder(t *testing.T) {
	result := NewResult("cursor", "Cursor CLI", "2026.08", "https://example.com/mcp")
	got := make([]Stage, 0, len(result.Stages))
	for _, stage := range result.Stages {
		got = append(got, stage.Stage)
		if stage.Status != StatusUnknown {
			t.Fatalf("expected unknown default status, got %s", stage.Status)
		}
	}
	if !reflect.DeepEqual(got, OrderedStages) {
		t.Fatalf("unexpected stage order: %#v", got)
	}
}

func TestResultSetAndFailed(t *testing.T) {
	result := NewResult("codex", "Codex CLI", "1.0", "https://example.com/mcp")
	if !result.Set(StageReach, StatusPass, "connected") {
		t.Fatal("expected known stage to update")
	}
	if !result.Set(StageAuth, StatusFail, "login failed") {
		t.Fatal("expected known stage to update")
	}
	if !result.Failed() {
		t.Fatal("expected result to report failure")
	}
	if result.Set(Stage("other"), StatusPass, "") {
		t.Fatal("unexpected update of unknown stage")
	}
}

func TestRedactSecrets(t *testing.T) {
	input := `Authorization: Bearer abcdefghijklmnop https://example.com/callback?code=secret-code&access_token=secret-token {"client_secret":"super-secret"}`
	got := Redact(input)
	for _, secret := range []string{"abcdefghijklmnop", "secret-code", "secret-token", "super-secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("secret %q was not redacted: %s", secret, got)
		}
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker: %s", got)
	}
}

func TestSanitizeEndpointMasksCredentialQueries(t *testing.T) {
	input := "https://example.com/mcp?tenant=acme&api_key=very-secret&access_token=token-secret&X-Amz-Credential=aws-secret&X-Amz-Signature=signed-secret"
	got := SanitizeEndpoint(input)
	for _, secret := range []string{"very-secret", "token-secret", "aws-secret", "signed-secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("endpoint leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "tenant=acme") {
		t.Fatalf("expected non-sensitive routing parameter to remain: %s", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("expected endpoint redaction marker: %s", got)
	}
}

func TestSanitizeEndpointDoesNotOverRedactOrdinaryQueries(t *testing.T) {
	input := "https://example.com/mcp?author=alice&design=compact&tenant=acme"
	got := SanitizeEndpoint(input)
	for _, value := range []string{"author=alice", "design=compact", "tenant=acme"} {
		if !strings.Contains(got, value) {
			t.Fatalf("ordinary query %q was unexpectedly redacted: %s", value, got)
		}
	}
}

func TestSessionRejectsTraversal(t *testing.T) {
	session, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Cleanup()

	if _, err := session.Path("..", "outside"); err == nil {
		t.Fatal("expected path traversal to fail")
	}
	path, err := session.Path("config", "mcp.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, session.Root()) {
		t.Fatalf("expected path below session root: %s", path)
	}
}

type fakeAdapter struct {
	root string
}

func (a *fakeAdapter) Run(_ context.Context, target Target, session *Session) (Result, error) {
	a.root = session.Root()
	path, err := session.Path("credential.txt")
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(path, []byte("temporary"), 0o600); err != nil {
		return Result{}, err
	}

	result := NewResult("fake", "Fake Client", "1.0", target.Endpoint)
	result.Set(StageReach, StatusPass, "connected")
	result.Set(StageAuth, StatusFail, "Authorization: Bearer abcdefghijklmnop")
	return result, errors.New("oauth callback failed?access_token=secret-token")
}

func TestRunnerCleansSessionAndRedactsDiagnostics(t *testing.T) {
	adapter := &fakeAdapter{}
	result, err := NewRunner().Run(context.Background(), adapter, Target{Endpoint: "https://example.com/mcp?api_key=endpoint-secret&tenant=acme"})
	if err == nil {
		t.Fatal("expected fake adapter error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("runner leaked token in error: %v", err)
	}
	if strings.Contains(result.Endpoint, "endpoint-secret") {
		t.Fatalf("runner leaked endpoint credential: %s", result.Endpoint)
	}
	if !strings.Contains(result.Endpoint, "tenant=acme") {
		t.Fatalf("runner removed non-sensitive endpoint query: %s", result.Endpoint)
	}
	if adapter.root == "" {
		t.Fatal("fake adapter did not receive a session")
	}
	if _, statErr := os.Stat(adapter.root); !os.IsNotExist(statErr) {
		t.Fatalf("expected session root to be removed, stat error: %v", statErr)
	}
	auth, ok := result.Get(StageAuth)
	if !ok {
		t.Fatal("missing auth stage")
	}
	if strings.Contains(auth.Message, "abcdefghijklmnop") {
		t.Fatalf("runner leaked bearer token in result: %s", auth.Message)
	}
}

func TestTargetValidation(t *testing.T) {
	for _, endpoint := range []string{
		"",
		"ftp://example.com/mcp",
		"https://user:pass@example.com/mcp",
		"https://example.com/mcp#fragment",
	} {
		if err := (Target{Endpoint: endpoint}).Validate(); err == nil {
			t.Fatalf("expected endpoint %q to be rejected", endpoint)
		}
	}
	if err := (Target{Endpoint: "https://example.com/mcp"}).Validate(); err != nil {
		t.Fatalf("expected valid endpoint: %v", err)
	}
}
