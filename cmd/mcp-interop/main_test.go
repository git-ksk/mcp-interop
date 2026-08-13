package main

import (
	"bytes"
	"encoding/json"
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
		"--output",
		"result.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.endpoint != "https://example.com/mcp" || !options.json || !options.oauth || options.output != "result.json" {
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

func TestParseTestOptionsRejectsEmptyOutputPath(t *testing.T) {
	for _, args := range [][]string{
		{"https://example.com/mcp", "--output="},
		{"https://example.com/mcp", "--output", ""},
	} {
		if _, err := parseTestOptions(args); err == nil {
			t.Fatalf("expected empty output path to fail: %#v", args)
		}
	}
}

func TestLegacyTestJSONContractRemainsResultArray(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "codex-test", "https://example.com/mcp")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}

	data, err := json.Marshal([]interop.Result{result})
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 {
		t.Fatalf("legacy JSON top level must remain an array: %s", data)
	}
	for _, key := range []string{"client_id", "client_name", "client_version", "endpoint", "stages"} {
		if _, ok := decoded[0][key]; !ok {
			t.Fatalf("legacy JSON missing %q: %s", key, data)
		}
	}
	for _, forbidden := range []string{"schema_version", "artifact_type", "executed_at", "auth_mode", "evidence_provenance"} {
		if _, ok := decoded[0][forbidden]; ok {
			t.Fatalf("portable-artifact field %q leaked into legacy JSON: %s", forbidden, data)
		}
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

func TestWriteTestResultsIncludesReasonCode(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "codex-test", "https://example.com/mcp")
	result.Set(interop.StageReach, interop.StatusUnknown, "OAuth target discovered")
	result.SetWithReason(
		interop.StageAuth,
		interop.StatusFail,
		interop.ReasonDCRUnsupported,
		"Codex reports that Dynamic Client Registration is not supported for this OAuth target",
	)
	result.Set(interop.StageInit, interop.StatusSkip, "authentication did not complete")
	result.Set(interop.StageTools, interop.StatusSkip, "authentication did not complete")

	var output bytes.Buffer
	if err := writeTestResults(&output, []interop.Result{result}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"REASON", "DCR_UNSUPPORTED"} {
		if !strings.Contains(text, want) {
			t.Fatalf("reason output missing %q:\n%s", want, text)
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
