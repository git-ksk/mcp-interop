package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/toolinventory"
)

func cliToolEvidence(t *testing.T, name string, inventory, expected []string) string {
	t.Helper()
	result := interop.NewResult("codex", "Codex CLI", "0.152.1", "https://example.test/very-private-endpoint?access=secret")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "direct evidence")
	}
	if !result.SetObservedToolNames(inventory) {
		t.Fatal("failed to record real client tools")
	}
	check := interop.CheckToolExpectation(result, expected, nil)
	run, err := artifact.NewRunV2ProtectedPath(result, result.Endpoint, "test-a", time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), "default", artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"}, "dev", "deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := toolinventory.New(run, check)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := toolinventory.WriteFile(path, evidence); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestToolsCompareCLI(t *testing.T) {
	base := cliToolEvidence(t, "base.json", []string{"ping", "read_tool"}, []string{"ping", "read_tool"})
	current := cliToolEvidence(t, "current.json", []string{"ping", "third_tool"}, []string{"ping", "read_tool"})
	var out, stderr bytes.Buffer
	if rc := runTools([]string{"compare", base, current, "--json", "--fail-on-drift"}, &out, &stderr); rc != 1 {
		t.Fatalf("unnoticed missing expected tool: rc=%d stdout=%s stderr=%s", rc, out.String(), stderr.String())
	}
	var diff toolinventory.Diff
	if err := json.Unmarshal(out.Bytes(), &diff); err != nil {
		t.Fatal(err)
	}
	if diff.Decision != toolinventory.DecisionRegression || len(diff.NewlyMissing) != 1 || diff.NewlyMissing[0] != "read_tool" || diff.CurrentCount == nil || *diff.CurrentCount != 2 {
		t.Fatalf("incorrect diff: %#v", diff)
	}
	for _, secret := range []string{"third_tool", "very-private-endpoint", "access=secret"} {
		if strings.Contains(out.String(), secret) {
			t.Fatalf("unrequested tool name or URL leaked: %q", secret)
		}
	}
	out.Reset()
	stderr.Reset()
	if rc := runTools([]string{"compare", base, current}, &out, &stderr); rc != 0 || !strings.Contains(out.String(), "NEWLY_MISSING") {
		t.Fatalf("ungated compare: rc=%d out=%s stderr=%s", rc, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	if rc := runTools([]string{"compare", base, base, "--fail-on-drift"}, &out, &stderr); rc != 0 {
		t.Fatalf("clean diff gated: %d %s", rc, stderr.String())
	}
}

func TestToolsCompareRejectsInvalidInputs(t *testing.T) {
	base := cliToolEvidence(t, "base.json", []string{"ping"}, []string{"ping"})
	for _, args := range [][]string{
		{}, {"unknown", base, base}, {"compare", base}, {"compare", base, base, "--invalid"},
		{"compare", base, "missing-path"}, {"compare", base, base, base},
	} {
		var out, stderr bytes.Buffer
		if rc := runTools(args, &out, &stderr); rc != 2 {
			t.Errorf("invalid args accepted: %#v rc=%d", args, rc)
		}
	}
}

func TestParseTestToolEvidenceSafetyConstraints(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "core.json")
	toolOut := filepath.Join(root, "tool.json")
	base := []string{"https://example.test/private-path", "--client", "codex", "--deployment-id", "fixture", "--output", output, "--expect-tool", "ping"}
	valid := append(append([]string{}, base...), "--tool-evidence", toolOut)
	options, err := parseTestOptions(valid)
	if err != nil || options.toolEvidence != toolOut {
		t.Fatalf("valid option failed: %#v %v", options, err)
	}
	for _, extra := range [][]string{
		{"--tool-evidence"}, {"--tool-evidence="}, {"--tool-evidence", "-"},
		{"--tool-evidence", output}, {"--tool-evidence", filepath.Join(root, "subdir", "missing.json")},
		{"--tool-evidence", toolOut, "--tool-evidence", toolOut},
	} {
		if _, err := parseTestOptions(append(append([]string{}, base...), extra...)); err == nil {
			t.Errorf("accepted unsafe evidence args: %#v", extra)
		}
	}
	for _, tc := range []struct {
		name string
		opts []string
	}{
		{"missing declaration", []string{"https://example.test", "--client", "codex", "--deployment-id", "fixture", "--output", output, "--tool-evidence", toolOut}},
		{"missing core output", []string{"https://example.test", "--client", "codex", "--tool-evidence", toolOut, "--expect-tool", "ping"}},
		{"multiple clients", []string{"https://example.test", "--client", "codex,cursor", "--deployment-id", "fixture", "--output", output, "--tool-evidence", toolOut, "--expect-tool", "ping"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseTestOptions(tc.opts); err == nil {
				t.Fatal("unsafe evidence invocation accepted")
			}
		})
	}
	if err := os.WriteFile(toolOut, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := parseTestOptions(valid); err == nil {
		t.Fatal("refused to block existing output")
	}
}
