package suite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	interopcompare "github.com/git-ksk/mcp-interop/internal/compare"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestCompareResultSetsSurfacesPassToFailAndReasonRegression(t *testing.T) {
	manifest := regressionTestManifest()
	baseline := regressionTestSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	attempt := regressionTestSet(t, manifest, "attempt", "2.0.0", interop.StatusFail, interop.ReasonDCRUnsupported)

	report, err := CompareResultSets(baseline, []LoadedResultSet{attempt})
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionRegression || !report.HasRegression || report.HasUnstable {
		t.Fatalf("unexpected decision: %#v", report)
	}
	if len(report.Runs) != 1 || len(report.Runs[0].Attempts) != 1 {
		t.Fatalf("unexpected report shape: %#v", report.Runs)
	}
	current := report.Runs[0].Attempts[0]
	if !current.ClientVersionChanged || current.Evidence.ClientVersion != "2.0.0" {
		t.Fatalf("client version change not retained: %#v", current)
	}
	for _, want := range []string{interopcompare.RegressionPassToFail, interopcompare.RegressionReasonChanged} {
		if !containsString(current.RegressionKinds, want) {
			t.Fatalf("regression kind %q missing: %#v", want, current.RegressionKinds)
		}
	}
	if len(current.StageChanges) == 0 || current.StageChanges[0].NewReasonCode != interop.ReasonDCRUnsupported {
		t.Fatalf("reason-code stage change missing: %#v", current.StageChanges)
	}
}

func TestCompareResultSetsRetryCannotHideFirstFailure(t *testing.T) {
	manifest := regressionTestManifest()
	baseline := regressionTestSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	failed := regressionTestSet(t, manifest, "attempt-1", "2.0.0", interop.StatusUnknown, "")
	passed := regressionTestSet(t, manifest, "attempt-2", "2.0.0", interop.StatusPass, "")

	report, err := CompareResultSets(baseline, []LoadedResultSet{failed, passed})
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionRegressionAndUnstable || !report.HasRegression || !report.HasUnstable {
		t.Fatalf("retry failure was hidden: %#v", report)
	}
	if report.AttemptCount != 2 || len(report.Runs[0].Attempts) != 2 {
		t.Fatalf("attempt evidence was collapsed: %#v", report.Runs[0].Attempts)
	}
	first, second := report.Runs[0].Attempts[0], report.Runs[0].Attempts[1]
	if first.Evidence == nil || first.Evidence.Outcome != OutcomeNonPass || !first.Regression {
		t.Fatalf("first failed attempt not retained: %#v", first)
	}
	if second.Evidence == nil || second.Evidence.Outcome != OutcomePass || second.Regression {
		t.Fatalf("second recovered attempt unexpected: %#v", second)
	}
	if !containsString(first.RegressionKinds, interopcompare.RegressionPassToUnknown) {
		t.Fatalf("first failure regression kind missing: %#v", first.RegressionKinds)
	}
}

func TestCompareResultSetsStableKnownNonPassIsNotRetryInstability(t *testing.T) {
	manifest := regressionTestManifest()
	baseline := regressionTestSet(t, manifest, "baseline", "1.0.0", interop.StatusFail, interop.ReasonDCRUnsupported)
	attempt1 := regressionTestSet(t, manifest, "attempt-1", "2.0.0", interop.StatusFail, interop.ReasonDCRUnsupported)
	attempt2 := regressionTestSet(t, manifest, "attempt-2", "2.0.0", interop.StatusFail, interop.ReasonDCRUnsupported)

	report, err := CompareResultSets(baseline, []LoadedResultSet{attempt1, attempt2})
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionClean || report.HasRegression || report.HasUnstable {
		t.Fatalf("stable known non-pass should be clean regression evidence: %#v", report)
	}
}

func TestCompareResultSetsRejectsDifferentRetryManifests(t *testing.T) {
	manifest := regressionTestManifest()
	baseline := regressionTestSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	attempt1 := regressionTestSet(t, manifest, "attempt-1", "2.0.0", interop.StatusPass, "")
	other := regressionTestManifest()
	other.Targets[0].Clients = append(other.Targets[0].Clients, ClientSelection{ID: "cursor", Auth: AuthNone})
	attempt2 := regressionTestSet(t, other, "attempt-2", "2.0.0", interop.StatusPass, "")

	_, err := CompareResultSets(baseline, []LoadedResultSet{attempt1, attempt2})
	if err == nil || !strings.Contains(err.Error(), "manifest fingerprint differs") {
		t.Fatalf("expected retry manifest mismatch, got %v", err)
	}
}

func TestReadResultSetRejectsSymlinkArtifact(t *testing.T) {
	manifest := regressionTestManifest()
	root := t.TempDir()
	setDir := filepath.Join(root, "set")
	if err := os.MkdirAll(filepath.Join(setDir, "artifacts"), 0o700); err != nil {
		t.Fatal(err)
	}
	entry := ResultEntry{
		TargetID:     "production-a",
		DeploymentID: "production-a",
		ClientID:     "codex",
		AuthMode:     AuthNone,
		Outcome:      OutcomePass,
		ExitCode:     0,
		Artifact:     "artifacts/production-a--codex--none.json",
	}
	index, err := NewResultIndex(manifest, []ResultEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteResultIndex(filepath.Join(setDir, "index.json"), index); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "outside.json")
	if err := os.WriteFile(target, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(setDir, filepath.FromSlash(entry.Artifact))); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadResultSet(filepath.Join(setDir, "index.json")); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected symlink artifact rejection, got %v", err)
	}
}

func regressionTestManifest() Manifest {
	return Manifest{
		SchemaVersion:    SchemaVersionV1,
		ExecutionContext: ExecutionTrusted,
		Targets: []Target{{
			ID:           "production-a",
			Endpoint:     EndpointReference{Source: EndpointEnvironment, Variable: "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"},
			DeploymentID: "production-a",
			Clients:      []ClientSelection{{ID: "codex", Auth: AuthNone}},
		}},
	}
}

func regressionTestSet(t *testing.T, manifest Manifest, name, version string, toolsStatus interop.Status, reason interop.ReasonCode) LoadedResultSet {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(filepath.Join(root, "artifacts"), 0o700); err != nil {
		t.Fatal(err)
	}
	entries := make([]ResultEntry, 0, RunCount(manifest))
	for _, target := range manifest.Targets {
		for _, selection := range target.Clients {
			endpoint := "https://example.com/mcp/protected-value"
			result := interop.NewResult(selection.ID, selection.ID+" test", version, endpoint)
			for _, stage := range interop.OrderedStages {
				status := interop.StatusPass
				stageReason := interop.ReasonCode("")
				if stage == interop.StageTools {
					status = toolsStatus
					stageReason = reason
				}
				result.SetWithReason(stage, status, stageReason, "test")
			}
			authMode := "default"
			if selection.Auth == AuthOAuth {
				authMode = "oauth"
			}
			run, err := artifact.NewRunV2ProtectedPath(
				result,
				endpoint,
				target.DeploymentID,
				time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC),
				authMode,
				artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: selection.ID},
				"test",
				"deadbeef",
			)
			if err != nil {
				t.Fatal(err)
			}
			reference := "artifacts/" + target.ID + "--" + selection.ID + "--" + string(selection.Auth) + ".json"
			if err := artifact.WriteFile(filepath.Join(root, filepath.FromSlash(reference)), artifact.NewArtifactV2([]artifact.Run{run})); err != nil {
				t.Fatal(err)
			}
			outcome := OutcomePass
			exitCode := 0
			if toolsStatus != interop.StatusPass {
				outcome = OutcomeNonPass
				exitCode = 1
			}
			entries = append(entries, ResultEntry{
				TargetID:     target.ID,
				DeploymentID: target.DeploymentID,
				ClientID:     selection.ID,
				AuthMode:     selection.Auth,
				Outcome:      outcome,
				ExitCode:     exitCode,
				Artifact:     reference,
			})
		}
	}
	index, err := NewResultIndex(manifest, entries)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(root, "index.json")
	if err := WriteResultIndex(indexPath, index); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadResultSet(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestCompareResultSetsRejectsDifferentBaselineManifest(t *testing.T) {
	baselineManifest := regressionTestManifest()
	baseline := regressionTestSet(t, baselineManifest, "baseline", "1.0.0", interop.StatusPass, "")
	currentManifest := regressionTestManifest()
	currentManifest.Targets[0].Clients = append(currentManifest.Targets[0].Clients, ClientSelection{ID: "cursor", Auth: AuthNone})
	attempt := regressionTestSet(t, currentManifest, "attempt", "2.0.0", interop.StatusPass, "")

	_, err := CompareResultSets(baseline, []LoadedResultSet{attempt})
	if err == nil || !strings.Contains(err.Error(), "baseline manifest fingerprint differs") {
		t.Fatalf("expected baseline/current manifest mismatch, got %v", err)
	}
}

func TestReadResultSetRejectsArtifactDirectorySymlinkEscape(t *testing.T) {
	manifest := regressionTestManifest()
	root := t.TempDir()
	setDir := filepath.Join(root, "set")
	outsideDir := filepath.Join(root, "outside")
	if err := os.MkdirAll(setDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDir, 0o700); err != nil {
		t.Fatal(err)
	}
	entry := ResultEntry{
		TargetID:     "production-a",
		DeploymentID: "production-a",
		ClientID:     "codex",
		AuthMode:     AuthNone,
		Outcome:      OutcomePass,
		ExitCode:     0,
		Artifact:     "artifacts/production-a--codex--none.json",
	}
	index, err := NewResultIndex(manifest, []ResultEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteResultIndex(filepath.Join(setDir, "index.json"), index); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(setDir, "artifacts")); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(outsideDir, "production-a--codex--none.json")
	result := interop.NewResult("codex", "Codex CLI", "1.0.0", "https://example.com/mcp/protected")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}
	run, err := artifact.NewRunV2ProtectedPath(
		result,
		"https://example.com/mcp/protected",
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
	if err := artifact.WriteFile(artifactPath, artifact.NewArtifactV2([]artifact.Run{run})); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadResultSet(filepath.Join(setDir, "index.json")); err == nil || !strings.Contains(err.Error(), "resolved artifact escapes result set") {
		t.Fatalf("expected artifact-directory symlink escape rejection, got %v", err)
	}
}

func TestCompareResultSetsRetainsMissingAttemptEvidence(t *testing.T) {
	manifest := regressionTestManifest()
	manifest.Targets[0].Clients = append(manifest.Targets[0].Clients, ClientSelection{ID: "cursor", Auth: AuthNone})
	baseline := regressionTestSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	attempt := regressionTestSet(t, manifest, "attempt", "2.0.0", interop.StatusPass, "")
	attempt.Index.Runs = attempt.Index.Runs[:1]
	if err := ValidateResultIndex(attempt.Index); err != nil {
		t.Fatal(err)
	}

	report, err := CompareResultSets(baseline, []LoadedResultSet{attempt})
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionRegressionAndUnstable || !report.HasRegression || !report.HasUnstable {
		t.Fatalf("missing evidence did not affect decision: %#v", report)
	}
	var missing *AttemptComparison
	for i := range report.Runs {
		if report.Runs[i].ClientID == "cursor" {
			missing = &report.Runs[i].Attempts[0]
			break
		}
	}
	if missing == nil || missing.State != AttemptMissingEvidence || !missing.Regression || !containsString(missing.RegressionKinds, interopcompare.RegressionRunMissing) {
		t.Fatalf("missing attempt evidence not retained: %#v", missing)
	}
}

func TestCompareResultSetsRetainsExecutionError(t *testing.T) {
	manifest := regressionTestManifest()
	baseline := regressionTestSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	attempt := regressionTestSet(t, manifest, "attempt", "2.0.0", interop.StatusPass, "")
	entry := attempt.Index.Runs[0]
	delete(attempt.Artifacts, resultEntryKey(entry))
	attempt.Index.Runs[0].Outcome = OutcomeError
	attempt.Index.Runs[0].ExitCode = 1
	attempt.Index.Runs[0].Artifact = ""
	if err := ValidateResultIndex(attempt.Index); err != nil {
		t.Fatal(err)
	}

	report, err := CompareResultSets(baseline, []LoadedResultSet{attempt})
	if err != nil {
		t.Fatal(err)
	}
	current := report.Runs[0].Attempts[0]
	if current.State != AttemptExecutionError || current.Evidence == nil || current.Evidence.Outcome != OutcomeError || !current.Regression {
		t.Fatalf("execution error was not retained: %#v", current)
	}
	if report.Decision != DecisionRegressionAndUnstable {
		t.Fatalf("unexpected execution-error decision: %s", report.Decision)
	}
}

func TestCompareResultSetsDeclaresProtocolEvidenceUnavailableInV2(t *testing.T) {
	manifest := regressionTestManifest()
	baseline := regressionTestSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	attempt := regressionTestSet(t, manifest, "attempt", "2.0.0", interop.StatusPass, "")
	report, err := CompareResultSets(baseline, []LoadedResultSet{attempt})
	if err != nil {
		t.Fatal(err)
	}
	if report.ProtocolEvidenceStatus != ProtocolEvidenceNotSerialized {
		t.Fatalf("protocol evidence status = %q", report.ProtocolEvidenceStatus)
	}
}
