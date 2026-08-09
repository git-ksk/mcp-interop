package main

import (
	"bytes"
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
