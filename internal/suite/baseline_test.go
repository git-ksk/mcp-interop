package suite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestCreateBaselineSnapshotsEvidenceWithoutProtectedPath(t *testing.T) {
	source := regressionTestSet(t, regressionTestManifest(), "source", "1.2.3", interop.StatusPass, "")
	createdAt := time.Date(2026, 8, 28, 7, 30, 0, 0, time.UTC)
	output := filepath.Join(t.TempDir(), "baseline")

	baseline, err := CreateBaseline(source, output, nil, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Descriptor.CreatedAt != createdAt {
		t.Fatalf("created_at=%s, want %s", baseline.Descriptor.CreatedAt, createdAt)
	}
	if baseline.Descriptor.ManifestFingerprint != source.Index.ManifestFingerprint {
		t.Fatalf("manifest fingerprint changed: %#v", baseline.Descriptor)
	}
	if baseline.Descriptor.ResultSetDigest == "" || baseline.Descriptor.Supersedes != "" {
		t.Fatalf("unexpected descriptor: %#v", baseline.Descriptor)
	}
	if baseline.ResultSet.IndexPath == source.IndexPath {
		t.Fatal("baseline must snapshot the result set instead of referencing the source path")
	}

	var allJSON strings.Builder
	err = filepath.WalkDir(output, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		allJSON.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(allJSON.String(), "protected-value") {
		t.Fatal("protected endpoint path leaked into baseline bundle")
	}
}

func TestCreateBaselineRefusesExistingDestination(t *testing.T) {
	source := regressionTestSet(t, regressionTestManifest(), "source", "1.0.0", interop.StatusPass, "")
	output := filepath.Join(t.TempDir(), "baseline")
	if err := os.Mkdir(output, 0o700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(output, "keep.txt")
	if err := os.WriteFile(keep, []byte("keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := CreateBaseline(source, output, nil, time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected overwrite refusal, got %v", err)
	}
	data, readErr := os.ReadFile(keep)
	if readErr != nil || string(data) != "keep\n" {
		t.Fatalf("existing destination changed: data=%q err=%v", data, readErr)
	}
}

func TestReadBaselineDetectsSnapshotMutation(t *testing.T) {
	source := regressionTestSet(t, regressionTestManifest(), "source", "1.0.0", interop.StatusPass, "")
	output := filepath.Join(t.TempDir(), "baseline")
	baseline, err := CreateBaseline(source, output, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	entry := baseline.ResultSet.Index.Runs[0]
	artifactPath := filepath.Join(output, BaselineResultsDir, filepath.FromSlash(entry.Artifact))
	value, err := artifact.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	value.Runs[0].Client.Version = "1.0.1"
	if err := artifact.WriteFile(artifactPath, value); err != nil {
		t.Fatal(err)
	}

	_, err = ReadBaseline(output)
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch after mutation, got %v", err)
	}
}

func TestCreateBaselineRecordsExplicitSupersedesFingerprint(t *testing.T) {
	manifest := regressionTestManifest()
	firstSource := regressionTestSet(t, manifest, "first-source", "1.0.0", interop.StatusPass, "")
	first, err := CreateBaseline(
		firstSource,
		filepath.Join(t.TempDir(), "baseline-1"),
		nil,
		time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	secondSource := regressionTestSet(t, manifest, "second-source", "2.0.0", interop.StatusPass, "")
	second, err := CreateBaseline(
		secondSource,
		filepath.Join(t.TempDir(), "baseline-2"),
		&first,
		time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	firstFingerprint, err := BaselineFingerprint(first.Descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if second.Descriptor.Supersedes != firstFingerprint {
		t.Fatalf("supersedes=%q, want %q", second.Descriptor.Supersedes, firstFingerprint)
	}
}

func TestCreateBaselineRejectsSupersedingManifestMismatch(t *testing.T) {
	first := regressionTestSet(t, regressionTestManifest(), "first", "1.0.0", interop.StatusPass, "")
	previous, err := CreateBaseline(
		first,
		filepath.Join(t.TempDir(), "baseline-1"),
		nil,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	otherManifest := regressionTestManifest()
	otherManifest.Targets[0].DeploymentID = "production-b"
	otherManifest.Targets[0].ID = "production-b"
	otherManifest.Targets[0].Endpoint.Variable = "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_B"
	other := regressionTestSet(t, otherManifest, "other", "2.0.0", interop.StatusPass, "")
	output := filepath.Join(t.TempDir(), "baseline-2")

	_, err = CreateBaseline(other, output, &previous, time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "manifest fingerprint differs") {
		t.Fatalf("expected manifest mismatch, got %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("mismatched supersede created output: %v", statErr)
	}
}

func TestCompareBaselineResultSetsRejectsDeploymentChange(t *testing.T) {
	manifest := regressionTestManifest()
	baselineSource := regressionTestSet(t, manifest, "baseline-source", "1.0.0", interop.StatusPass, "")
	baseline, err := CreateBaseline(
		baselineSource,
		filepath.Join(t.TempDir(), "baseline"),
		nil,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	attempt := regressionTestSet(t, manifest, "attempt", "2.0.0", interop.StatusPass, "")
	entry := attempt.Index.Runs[0]
	oldRun := attempt.Artifacts[resultEntryKey(entry)].Runs[0]
	result := interop.NewResult(oldRun.Client.ID, oldRun.Client.Product, oldRun.Client.Version, "https://other.example/mcp")
	for _, stage := range oldRun.Stages {
		result.SetWithReason(stage.Stage, stage.Status, stage.ReasonCode, "test")
	}
	changedRun, err := artifact.NewRunV2ProtectedPath(
		result,
		"https://other.example/mcp/protected-value",
		entry.DeploymentID,
		oldRun.ExecutedAt,
		oldRun.AuthMode,
		oldRun.EvidenceProvenance,
		oldRun.Runtime.MCPInteropVersion,
		oldRun.Runtime.MCPInteropCommit,
	)
	if err != nil {
		t.Fatal(err)
	}
	attempt.Artifacts[resultEntryKey(entry)] = artifact.NewArtifactV2([]artifact.Run{changedRun})

	_, err = CompareBaselineResultSets(baseline, []LoadedResultSet{attempt})
	if err == nil || !strings.Contains(err.Error(), "deployment fingerprint differs") {
		t.Fatalf("expected deployment mismatch, got %v", err)
	}
}

func TestCompareBaselineResultSetsRejectsPlatformChange(t *testing.T) {
	manifest := regressionTestManifest()
	baselineSource := regressionTestSet(t, manifest, "baseline-source", "1.0.0", interop.StatusPass, "")
	baseline, err := CreateBaseline(
		baselineSource,
		filepath.Join(t.TempDir(), "baseline"),
		nil,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	attempt := regressionTestSet(t, manifest, "attempt", "2.0.0", interop.StatusPass, "")
	entry := attempt.Index.Runs[0]
	value := attempt.Artifacts[resultEntryKey(entry)]
	value.Runs[0].Platform.Arch = "different-arch"
	attempt.Artifacts[resultEntryKey(entry)] = value

	_, err = CompareBaselineResultSets(baseline, []LoadedResultSet{attempt})
	if err == nil || !strings.Contains(err.Error(), "platform differs") {
		t.Fatalf("expected platform mismatch, got %v", err)
	}
}

func TestCreateBaselineRejectsRunnerObservation(t *testing.T) {
	source := regressionTestSet(t, regressionTestManifest(), "source", "1.0.0", interop.StatusPass, "")
	entry := source.Index.Runs[0]
	value := source.Artifacts[resultEntryKey(entry)]
	value.Runs[0].EvidenceProvenance = artifact.EvidenceProvenance{Kind: artifact.ProvenanceRunnerObservation}
	value.Runs[0].Client.Version = ""
	for i := range value.Runs[0].Stages {
		value.Runs[0].Stages[i].Status = interop.StatusUnknown
		value.Runs[0].Stages[i].ReasonCode = ""
	}
	entry.Outcome = OutcomeNonPass
	entry.ExitCode = 1
	source.Index.Runs[0] = entry
	source.Artifacts[resultEntryKey(entry)] = value

	_, err := CreateBaseline(source, filepath.Join(t.TempDir(), "baseline"), nil, time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "not real-client adapter evidence") {
		t.Fatalf("expected runner observation rejection, got %v", err)
	}
}
