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
	"github.com/git-ksk/mcp-interop/internal/suite"
)

func TestRunSuiteCompareRetryCannotHideRegression(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	baseline := writeCLISuiteCompareSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	failed := writeCLISuiteCompareSet(t, manifest, "attempt-1", "2.0.0", interop.StatusUnknown, "")
	passed := writeCLISuiteCompareSet(t, manifest, "attempt-2", "2.0.0", interop.StatusPass, "")

	var stdout, stderr bytes.Buffer
	code := runSuiteCompare([]string{baseline, failed, passed, "--json", "--fail-on-regression"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report suite.RegressionReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Decision != suite.DecisionRegressionAndUnstable || !report.HasRegression || !report.HasUnstable || report.AttemptCount != 2 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if len(report.Runs) != 1 || len(report.Runs[0].Attempts) != 2 || !report.Runs[0].Attempts[0].Regression {
		t.Fatalf("first attempt was not retained: %#v", report.Runs)
	}
}

func TestRunSuiteCompareHumanAndJSONExpressSameCleanDecision(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	baseline := writeCLISuiteCompareSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	current := writeCLISuiteCompareSet(t, manifest, "attempt", "2.0.0", interop.StatusPass, "")

	var human, humanErr bytes.Buffer
	if code := runSuiteCompare([]string{baseline, current, "--fail-on-regression"}, &human, &humanErr); code != 0 {
		t.Fatalf("human exit=%d stderr=%s", code, humanErr.String())
	}
	for _, want := range []string{"DECISION", "CLEAN", "REGRESSION", "NO", "UNSTABLE", "NO", "1.0.0", "2.0.0"} {
		if !strings.Contains(human.String(), want) {
			t.Fatalf("human report missing %q:\n%s", want, human.String())
		}
	}

	var machine, machineErr bytes.Buffer
	if code := runSuiteCompare([]string{baseline, current, "--json", "--fail-on-regression"}, &machine, &machineErr); code != 0 {
		t.Fatalf("json exit=%d stderr=%s", code, machineErr.String())
	}
	var report suite.RegressionReport
	if err := json.Unmarshal(machine.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Decision != suite.DecisionClean || report.HasRegression || report.HasUnstable {
		t.Fatalf("JSON decision disagrees with human report: %#v", report)
	}
}

func TestRunSuiteCompareInvalidInputReturnsTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runSuiteCompare([]string{"missing-baseline", "missing-attempt"}, &stdout, &stderr); code != 2 {
		t.Fatalf("invalid-input exit=%d stderr=%s", code, stderr.String())
	}
}

func TestParseSuiteCompareOptionsRequiresAttemptAndRejectsUnknownFlag(t *testing.T) {
	if _, err := parseSuiteCompareOptions([]string{"baseline"}); err == nil {
		t.Fatal("expected missing attempt error")
	}
	if _, err := parseSuiteCompareOptions([]string{"baseline", "attempt", "--wat"}); err == nil {
		t.Fatal("expected unknown flag error")
	}
}

func cliSuiteCompareManifest() suite.Manifest {
	return suite.Manifest{
		SchemaVersion:    suite.SchemaVersionV1,
		ExecutionContext: suite.ExecutionTrusted,
		Targets: []suite.Target{{
			ID:           "production-a",
			Endpoint:     suite.EndpointReference{Source: suite.EndpointEnvironment, Variable: "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"},
			DeploymentID: "production-a",
			Clients:      []suite.ClientSelection{{ID: "codex", Auth: suite.AuthNone}},
		}},
	}
}

func writeCLISuiteCompareSet(t *testing.T, manifest suite.Manifest, name, version string, toolsStatus interop.Status, reason interop.ReasonCode) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(filepath.Join(root, "artifacts"), 0o700); err != nil {
		t.Fatal(err)
	}
	endpoint := "https://example.com/mcp/protected-value"
	result := interop.NewResult("codex", "Codex CLI", version, endpoint)
	for _, stage := range interop.OrderedStages {
		status := interop.StatusPass
		stageReason := interop.ReasonCode("")
		if stage == interop.StageTools {
			status = toolsStatus
			stageReason = reason
		}
		result.SetWithReason(stage, status, stageReason, "test")
	}
	run, err := artifact.NewRunV2ProtectedPath(
		result,
		endpoint,
		"production-a",
		time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC),
		"default",
		artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"},
		"test",
		"deadbeef",
	)
	if err != nil {
		t.Fatal(err)
	}
	reference := "artifacts/production-a--codex--none.json"
	if err := artifact.WriteFile(filepath.Join(root, filepath.FromSlash(reference)), artifact.NewArtifactV2([]artifact.Run{run})); err != nil {
		t.Fatal(err)
	}
	outcome := suite.OutcomePass
	exitCode := 0
	if toolsStatus != interop.StatusPass {
		outcome = suite.OutcomeNonPass
		exitCode = 1
	}
	index, err := suite.NewResultIndex(manifest, []suite.ResultEntry{{
		TargetID:     "production-a",
		DeploymentID: "production-a",
		ClientID:     "codex",
		AuthMode:     suite.AuthNone,
		Outcome:      outcome,
		ExitCode:     exitCode,
		Artifact:     reference,
	}})
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(root, "index.json")
	if err := suite.WriteResultIndex(indexPath, index); err != nil {
		t.Fatal(err)
	}
	return root
}
