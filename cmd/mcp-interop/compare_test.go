package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	interopcompare "github.com/git-ksk/mcp-interop/internal/compare"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestParseCompareOptionsAcceptsFlagsAfterPaths(t *testing.T) {
	options, err := parseCompareOptions([]string{"old.json", "new.json", "--json", "--fail-on-regression"})
	if err != nil {
		t.Fatal(err)
	}
	if options.oldPath != "old.json" || options.newPath != "new.json" || !options.json || !options.failOnRegression {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestRunCompareFailOnRegressionExitContract(t *testing.T) {
	oldPath := filepath.Join(t.TempDir(), "old.json")
	newPath := filepath.Join(t.TempDir(), "new.json")
	oldRun := cliTestRun(t, "1.0.0")
	newRun := cliTestRun(t, "2.0.0")
	newRun.Stages[3].Status = interop.StatusUnknown
	if err := artifact.WriteFile(oldPath, artifact.NewArtifact([]artifact.Run{oldRun})); err != nil {
		t.Fatal(err)
	}
	if err := artifact.WriteFile(newPath, artifact.NewArtifact([]artifact.Run{newRun})); err != nil {
		t.Fatal(err)
	}
	if code := runCompare([]string{oldPath, newPath}); code != 0 {
		t.Fatalf("report-only compare exit = %d, want 0", code)
	}
	if code := runCompare([]string{oldPath, newPath, "--fail-on-regression"}); code != 1 {
		t.Fatalf("gated compare exit = %d, want 1", code)
	}
}

func TestRunCompareVersionOnlyChangePassesGate(t *testing.T) {
	oldPath := filepath.Join(t.TempDir(), "old.json")
	newPath := filepath.Join(t.TempDir(), "new.json")
	if err := artifact.WriteFile(oldPath, artifact.NewArtifact([]artifact.Run{cliTestRun(t, "1.0.0")})); err != nil {
		t.Fatal(err)
	}
	if err := artifact.WriteFile(newPath, artifact.NewArtifact([]artifact.Run{cliTestRun(t, "2.0.0")})); err != nil {
		t.Fatal(err)
	}
	if code := runCompare([]string{oldPath, newPath, "--fail-on-regression"}); code != 0 {
		t.Fatalf("version-only gated compare exit = %d, want 0", code)
	}
}

func TestWriteComparisonShowsMachineReasonInHumanOutput(t *testing.T) {
	oldRun := cliTestRun(t, "1.0.0")
	newRun := cliTestRun(t, "2.0.0")
	newRun.Stages[1].Status = interop.StatusFail
	newRun.Stages[1].ReasonCode = interop.ReasonDCRUnsupported
	report := interopcompare.Artifacts(artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))

	var output bytes.Buffer
	if err := writeComparison(&output, report); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"REGRESSION", "YES", "PASS_TO_FAIL", "DCR_UNSUPPORTED", "1.0.0 -> 2.0.0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("comparison output missing %q:\n%s", want, text)
		}
	}
}

func cliTestRun(t *testing.T, version string) artifact.Run {
	t.Helper()
	result := interop.NewResult("codex", "Codex CLI", version, "https://example.com/mcp")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}
	run, err := artifact.NewRun(
		result,
		time.Date(2026, 8, 12, 15, 0, 0, 0, time.UTC),
		"default",
		artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"},
		"v-test",
		"deadbeef",
	)
	if err != nil {
		t.Fatal(err)
	}
	return run
}
