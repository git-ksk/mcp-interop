package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	diagnosepkg "github.com/git-ksk/mcp-interop/internal/diagnose"
)

func TestParseDiagnoseOptionsAcceptsFlagsAfterURL(t *testing.T) {
	options, err := parseDiagnoseOptions([]string{
		"https://example.com/mcp",
		"--profile", "chatgpt",
		"--client-id", "https://chatgpt.com/example/client.json",
		"--redirect-uri", "https://chatgpt.com/connector/oauth/example",
		"--json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.endpoint != "https://example.com/mcp" || options.profile != "chatgpt" || !options.json {
		t.Fatalf("unexpected options: %#v", options)
	}
	if options.clientID == "" || options.redirectURI == "" {
		t.Fatalf("expected client evidence options: %#v", options)
	}
}

func TestParseDiagnoseOptionsDefaultsToChatGPT(t *testing.T) {
	options, err := parseDiagnoseOptions([]string{"https://example.com/mcp"})
	if err != nil {
		t.Fatal(err)
	}
	if options.profile != "chatgpt" {
		t.Fatalf("profile=%q, want chatgpt", options.profile)
	}
}

func TestParseDiagnoseOptionsRejectsUnsupportedProfile(t *testing.T) {
	if _, err := parseDiagnoseOptions([]string{"https://example.com/mcp", "--profile", "other"}); err == nil {
		t.Fatal("expected unsupported profile to fail")
	}
}

func TestParseDiagnoseOptionsRequiresClientIDForRedirect(t *testing.T) {
	if _, err := parseDiagnoseOptions([]string{"https://example.com/mcp", "--redirect-uri", "https://chatgpt.com/connector/oauth/example"}); err == nil {
		t.Fatal("expected redirect without client-id to fail")
	}
}

func TestWriteDiagnoseReportClearlyLabelsPreflight(t *testing.T) {
	report := diagnosepkg.Report{
		Profile:  "chatgpt",
		Endpoint: "https://example.com/mcp",
		Checks: []diagnosepkg.Check{
			{ID: "client_registration", Status: diagnosepkg.StatusPass, Blocking: true, Message: "CIMD available"},
			{ID: "offline_access", Status: diagnosepkg.StatusWarn, Blocking: false, Message: "not advertised"},
		},
	}
	var output bytes.Buffer
	if err := writeDiagnoseReport(&output, report); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{
		"PROFILE",
		"chatgpt",
		"PREFLIGHT PASS",
		"client_registration",
		"WARN",
		"not a real ChatGPT client interoperability PASS",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("diagnose output missing %q:\n%s", want, text)
		}
	}
}

func TestWriteDiagnoseReportFailsOnBlockingFailure(t *testing.T) {
	report := diagnosepkg.Report{
		Profile:  "chatgpt",
		Endpoint: "https://example.com/mcp",
		Checks: []diagnosepkg.Check{
			{ID: "token_endpoint_auth", Status: diagnosepkg.StatusFail, Blocking: true, Message: "incompatible"},
		},
	}
	if report.Passed() {
		t.Fatal("expected report not to pass")
	}
	var output bytes.Buffer
	if err := writeDiagnoseReport(&output, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "PREFLIGHT FAIL") {
		t.Fatalf("unexpected output:\n%s", output.String())
	}
}

func TestParseDiagnoseOptionsAcceptsRuntimeEvidence(t *testing.T) {
	options, err := parseDiagnoseOptions([]string{
		"https://example.com/mcp",
		"--runtime-evidence", "evidence.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.runtimeEvidence != "evidence.json" {
		t.Fatalf("runtimeEvidence=%q", options.runtimeEvidence)
	}
}

func TestReadRuntimeEvidenceRejectsSecretFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.json")
	body := `{"client_id":"https://chatgpt.com/oauth/test/client.json","resource_matches":true,"client_assertion_present":false,"access_token":"secret"}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRuntimeEvidence(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown secret-bearing field to be rejected, got %v", err)
	}
}

func TestWriteDiagnoseReportSeparatesPreflightAndRuntimeFailure(t *testing.T) {
	report := diagnosepkg.Report{
		Profile:  "chatgpt",
		Endpoint: "https://example.com/mcp",
		Checks: []diagnosepkg.Check{
			{ID: "token_endpoint_auth", Status: diagnosepkg.StatusPass, Blocking: true, Message: "metadata compatible"},
		},
		RuntimeEvidence: &diagnosepkg.RuntimeEvidenceReport{
			Status:     diagnosepkg.StatusFail,
			ReasonCode: "TOKEN_AUTH_METHOD_MISMATCH",
			Checks: []diagnosepkg.RuntimeCheck{
				{ID: "token_auth_method", Status: diagnosepkg.StatusFail, Expected: "private_key_jwt", Observed: "none", ReasonCode: "TOKEN_AUTH_METHOD_MISMATCH", Message: "mismatch"},
			},
		},
	}
	if report.PreflightPassed() != true || report.Passed() != false {
		t.Fatalf("unexpected layered verdict: preflight=%v overall=%v", report.PreflightPassed(), report.Passed())
	}
	var output bytes.Buffer
	if err := writeDiagnoseReport(&output, report); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"PREFLIGHT PASS", "RUNTIME EVIDENCE", "TOKEN_AUTH_METHOD_MISMATCH", "private_key_jwt", "none"} {
		if !strings.Contains(text, want) {
			t.Fatalf("runtime diagnose output missing %q:\n%s", want, text)
		}
	}
}
