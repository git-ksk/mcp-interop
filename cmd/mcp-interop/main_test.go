package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestParseTestOptionsAcceptsFlagsAfterURL(t *testing.T) {
	options, err := parseTestOptions([]string{
		"https://example.com/mcp",
		"--client",
		"codex",
		"--oauth",
		"--json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.endpoint != "https://example.com/mcp" || !options.json || !options.oauth {
		t.Fatalf("unexpected options: %#v", options)
	}
	if !reflect.DeepEqual(options.clients, []string{"codex"}) {
		t.Fatalf("unexpected clients: %#v", options.clients)
	}
}

func TestParseTestOptionsDefaultsOAuthOff(t *testing.T) {
	options, err := parseTestOptions([]string{"https://example.com/mcp"})
	if err != nil {
		t.Fatal(err)
	}
	if options.oauth {
		t.Fatal("OAuth must remain explicit opt-in")
	}
}

func TestParseTestOptionsDeduplicatesClients(t *testing.T) {
	options, err := parseTestOptions([]string{
		"--client=codex, CODEX,",
		"https://example.com/mcp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(options.clients, []string{"codex"}) {
		t.Fatalf("unexpected clients: %#v", options.clients)
	}
}

func TestParseTestOptionsRejectsEmptyClientList(t *testing.T) {
	if _, err := parseTestOptions([]string{"https://example.com/mcp", "--client", ","}); err == nil {
		t.Fatal("expected empty client list to fail")
	}
}

func TestWriteTestResultsIncludesCrossClientSummary(t *testing.T) {
	codex := interop.NewResult("codex", "Codex CLI", "codex-test", "https://example.com/mcp")
	for _, stage := range interop.OrderedStages {
		codex.Set(stage, interop.StatusPass, "ok")
	}

	cursor := interop.NewResult("cursor", "Cursor CLI", "cursor-test", "https://example.com/mcp")
	cursor.Set(interop.StageReach, interop.StatusPass, "reached")
	cursor.Set(interop.StageAuth, interop.StatusSkip, "auth required")
	cursor.Set(interop.StageInit, interop.StatusSkip, "auth incomplete")
	cursor.Set(interop.StageTools, interop.StatusSkip, "auth incomplete")

	var output bytes.Buffer
	if err := writeTestResults(&output, []interop.Result{codex, cursor}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{
		"SUMMARY",
		"CLIENT",
		"REACH",
		"AUTH",
		"INIT",
		"TOOLS",
		"Codex CLI",
		"Cursor CLI",
		"PASS",
		"SKIP",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary output missing %q:\n%s", want, text)
		}
	}
}

func TestWriteTestResultsSingleClientDoesNotPrintSummary(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "codex-test", "https://example.com/mcp")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}

	var output bytes.Buffer
	if err := writeTestResults(&output, []interop.Result{result}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "SUMMARY") {
		t.Fatalf("single-client output unexpectedly contains summary:\n%s", output.String())
	}
}
