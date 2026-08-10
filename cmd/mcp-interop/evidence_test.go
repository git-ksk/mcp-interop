package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	diagnosepkg "github.com/git-ksk/mcp-interop/internal/diagnose"
)

func TestParseEvidenceSingleOptionsAcceptsJSONBeforeOrAfterPath(t *testing.T) {
	for _, args := range [][]string{{"--json", "evidence.json"}, {"evidence.json", "--json"}} {
		options, err := parseEvidenceSingleOptions(args)
		if err != nil {
			t.Fatal(err)
		}
		if options.path != "evidence.json" || !options.json {
			t.Fatalf("options=%#v", options)
		}
	}
}

func TestParseEvidenceMergeOptionsAcceptsOutputAfterInputs(t *testing.T) {
	options, err := parseEvidenceMergeOptions([]string{"a.json", "b.json", "-o", "merged.json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(options.inputs) != 2 || options.output != "merged.json" {
		t.Fatalf("options=%#v", options)
	}
}

func TestRunEvidenceMergeWritesCanonicalV3WithPrivateNewFileMode(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")
	output := filepath.Join(dir, "merged.json")
	if err := os.WriteFile(first, []byte(`{"schema_version":2,"tool_auth":{"oauth2_security_scheme_present":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(`{"schema_version":3,"tool_challenge":{"expected":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := runEvidenceMerge([]string{first, second, "-o", output}); code != 0 {
		t.Fatalf("merge exit=%d", code)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var evidence diagnosepkg.ChatGPTRuntimeEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.SchemaVersion != 3 || evidence.ToolAuth != nil || evidence.ToolMetadata == nil || evidence.ToolChallenge == nil {
		t.Fatalf("evidence=%#v", evidence)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(output)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("output mode=%#o, want 0600", info.Mode().Perm())
		}
	}
}

func TestRunEvidenceMergeRejectsConflictingObservations(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")
	if err := os.WriteFile(first, []byte(`{"schema_version":3,"resource_request":{"bearer_present":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(`{"schema_version":3,"resource_request":{"bearer_present":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := runEvidenceMerge([]string{first, second}); code != 1 {
		t.Fatalf("merge exit=%d, want 1", code)
	}
}

func TestRunEvidenceMergeRejectsUnknownSecretBearingField(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "secret.json")
	if err := os.WriteFile(input, []byte(`{"schema_version":3,"tool_challenge":{"expected":true},"access_token":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := runEvidenceMerge([]string{input}); code != 1 {
		t.Fatalf("merge exit=%d, want 1", code)
	}
}

func TestWriteEvidenceSummaryDoesNotExposeObservedValues(t *testing.T) {
	summary := diagnosepkg.RuntimeEvidenceInputSummary{
		SchemaVersion: 3,
		Sections: []diagnosepkg.RuntimeEvidenceSectionSummary{
			{Section: "registration", Supplied: 2},
			{Section: "resource_request", Supplied: 4},
		},
		TotalSupplied: 6,
	}
	var output bytes.Buffer
	if err := writeEvidenceSummary(&output, summary); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"SCHEMA", "v3", "TOTAL_SUPPLIED", "6", "registration", "resource_request"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"client.json", "Bearer", "true", "false"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary exposed value %q: %s", forbidden, text)
		}
	}
}
