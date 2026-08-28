package suite

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestCompatibilityEnvelopeKeepsExactObservedVersionsOnly(t *testing.T) {
	manifest := regressionTestManifest()
	first := compatibilityTestSet(t, manifest, "first", "1.2.0", time.Date(2026, 8, 20, 1, 0, 0, 0, time.UTC), interop.StatusPass)
	second := compatibilityTestSet(t, manifest, "second", "1.8.0", time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC), interop.StatusPass)

	envelope, err := BuildCompatibilityEnvelope(nil, []LoadedResultSet{first, second}, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.Points) != 2 {
		t.Fatalf("points=%d, want two exact observations: %#v", len(envelope.Points), envelope.Points)
	}
	for _, point := range envelope.Points {
		if point.State != CompatibilityTested {
			t.Fatalf("state=%s, want tested: %#v", point.State, point)
		}
	}
	query := compatibilityQuery("1.5.0")
	classification, err := ClassifyCompatibilityExact(envelope, query)
	if err != nil {
		t.Fatal(err)
	}
	if classification.State != CompatibilityUntested || classification.Point != nil {
		t.Fatalf("unobserved intermediate version was inferred: %#v", classification)
	}
	if strings.Join(classification.ObservedVersions, ",") != "1.2.0,1.8.0" {
		t.Fatalf("observed versions=%v", classification.ObservedVersions)
	}
}

func TestCompatibilityAutoUpdatedUnseenVersionIsUntestedNotRegressed(t *testing.T) {
	manifest := regressionTestManifest()
	baselineSource := compatibilityTestSet(t, manifest, "baseline-source", "1.0.0", time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC), interop.StatusPass)
	baseline, err := CreateBaseline(baselineSource, t.TempDir()+"/accepted", nil, time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := BuildCompatibilityEnvelope(&baseline, nil, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	classification, err := ClassifyCompatibilityExact(envelope, compatibilityQuery("2.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if classification.State != CompatibilityUntested {
		t.Fatalf("auto-updated unseen version state=%s, want untested", classification.State)
	}
}

func TestCompatibilityStaleUsesAgeAndLaterObservedVersionWithoutRanges(t *testing.T) {
	manifest := regressionTestManifest()
	old := compatibilityTestSet(t, manifest, "old", "1.0.0", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	newer := compatibilityTestSet(t, manifest, "newer", "next-build", time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	policy := CompatibilityStalePolicy{MaxAgeSeconds: int64((14 * 24 * time.Hour) / time.Second), StaleOnClientVersionChange: true}
	envelope, err := BuildCompatibilityEnvelope(nil, []LoadedResultSet{old, newer}, policy, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	oldClass, err := ClassifyCompatibilityExact(envelope, compatibilityQuery("1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if oldClass.State != CompatibilityStale || oldClass.Point == nil {
		t.Fatalf("old point=%#v", oldClass)
	}
	if !containsString(oldClass.Point.StaleReasons, CompatibilityStaleByAge) || !containsString(oldClass.Point.StaleReasons, CompatibilityStaleByVersionChange) {
		t.Fatalf("stale reasons=%v", oldClass.Point.StaleReasons)
	}
	newClass, err := ClassifyCompatibilityExact(envelope, compatibilityQuery("next-build"))
	if err != nil {
		t.Fatal(err)
	}
	if newClass.State != CompatibilityTested {
		t.Fatalf("latest point state=%s", newClass.State)
	}
}

func TestCompatibilityKnownBrokenAndUnknownRemainDistinct(t *testing.T) {
	manifest := regressionTestManifest()
	broken := compatibilityTestSet(t, manifest, "broken", "1.0.0", time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), interop.StatusFail)
	unknown := compatibilityTestSet(t, manifest, "unknown", "2.0.0", time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC), interop.StatusUnknown)
	envelope, err := BuildCompatibilityEnvelope(nil, []LoadedResultSet{broken, unknown}, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	brokenClass, _ := ClassifyCompatibilityExact(envelope, compatibilityQuery("1.0.0"))
	unknownClass, _ := ClassifyCompatibilityExact(envelope, compatibilityQuery("2.0.0"))
	if brokenClass.State != CompatibilityKnownBroken {
		t.Fatalf("FAIL state=%s, want known_broken", brokenClass.State)
	}
	if unknownClass.State != CompatibilityUnknown {
		t.Fatalf("UNKNOWN state=%s, want unknown", unknownClass.State)
	}
}

func TestCompatibilityRegressedRequiresComparableBaselineRegression(t *testing.T) {
	manifest := regressionTestManifest()
	baselineSource := compatibilityTestSet(t, manifest, "baseline-source", "1.0.0", time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	baseline, err := CreateBaseline(baselineSource, t.TempDir()+"/accepted", nil, time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	regressed := compatibilityTestSet(t, manifest, "regressed", "2.0.0", time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC), interop.StatusFail)
	envelope, err := BuildCompatibilityEnvelope(&baseline, []LoadedResultSet{regressed}, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	classification, err := ClassifyCompatibilityExact(envelope, compatibilityQuery("2.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if classification.State != CompatibilityRegressed || classification.Point == nil {
		t.Fatalf("regressed classification=%#v", classification)
	}
	if len(classification.Point.Observations) != 1 || !classification.Point.Observations[0].Regression {
		t.Fatalf("regression evidence missing: %#v", classification.Point)
	}
}

func TestCompatibilityMixedRetryEvidenceStaysUnknown(t *testing.T) {
	manifest := regressionTestManifest()
	failed := compatibilityTestSet(t, manifest, "failed", "2.0.0", time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC), interop.StatusFail)
	passed := compatibilityTestSet(t, manifest, "passed", "2.0.0", time.Date(2026, 8, 27, 0, 5, 0, 0, time.UTC), interop.StatusPass)
	envelope, err := BuildCompatibilityEnvelope(nil, []LoadedResultSet{failed, passed}, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	classification, err := ClassifyCompatibilityExact(envelope, compatibilityQuery("2.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if classification.State != CompatibilityUnknown || classification.Point == nil || !classification.Point.Unstable || len(classification.Point.Observations) != 2 {
		t.Fatalf("mixed retry evidence was collapsed: %#v", classification)
	}
}

func TestCompatibilityExecutionErrorBecomesGapWithoutInventedVersion(t *testing.T) {
	manifest := regressionTestManifest()
	set := compatibilityTestSet(t, manifest, "error", "1.0.0", time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	entry := set.Index.Runs[0]
	set.Index.Runs[0].Outcome = OutcomeError
	set.Index.Runs[0].ExitCode = 1
	set.Index.Runs[0].Artifact = ""
	delete(set.Artifacts, resultEntryKey(entry))

	envelope, err := BuildCompatibilityEnvelope(nil, []LoadedResultSet{set}, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.Points) != 0 || len(envelope.EvidenceGaps) != 1 {
		t.Fatalf("execution error fabricated point: %#v", envelope)
	}
	if envelope.EvidenceGaps[0].Kind != CompatibilityGapExecutionError || envelope.EvidenceGaps[0].ClientVersion != "" {
		t.Fatalf("unexpected evidence gap: %#v", envelope.EvidenceGaps[0])
	}
}

func TestCompatibilityRejectsDeploymentMismatch(t *testing.T) {
	manifest := regressionTestManifest()
	first := compatibilityTestSet(t, manifest, "first", "1.0.0", time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	second := compatibilityTestSet(t, manifest, "second", "2.0.0", time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	entry := second.Index.Runs[0]
	value := second.Artifacts[resultEntryKey(entry)]
	run := value.Runs[0]
	result := interop.NewResult(run.Client.ID, run.Client.Product, run.Client.Version, "https://other.example/mcp")
	for _, stage := range run.Stages {
		result.SetWithReason(stage.Stage, stage.Status, stage.ReasonCode, "test")
	}
	changed, err := artifact.NewRunV2ProtectedPath(result, "https://other.example/mcp/protected", entry.DeploymentID, run.ExecutedAt, run.AuthMode, run.EvidenceProvenance, run.Runtime.MCPInteropVersion, run.Runtime.MCPInteropCommit)
	if err != nil {
		t.Fatal(err)
	}
	second.Artifacts[resultEntryKey(entry)] = artifact.NewArtifactV2([]artifact.Run{changed})

	_, err = BuildCompatibilityEnvelope(nil, []LoadedResultSet{first, second}, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "deployment fingerprint differs") {
		t.Fatalf("expected deployment mismatch, got %v", err)
	}
}

func compatibilityTestSet(t *testing.T, manifest Manifest, name, version string, executedAt time.Time, toolsStatus interop.Status) LoadedResultSet {
	t.Helper()
	set := regressionTestSet(t, manifest, name, version, toolsStatus, "")
	entry := set.Index.Runs[0]
	value := set.Artifacts[resultEntryKey(entry)]
	value.Runs[0].ExecutedAt = executedAt.UTC()
	if toolsStatus == interop.StatusPass {
		set.Index.Runs[0].Outcome = OutcomePass
		set.Index.Runs[0].ExitCode = 0
	} else {
		set.Index.Runs[0].Outcome = OutcomeNonPass
		set.Index.Runs[0].ExitCode = 1
	}
	set.Artifacts[resultEntryKey(entry)] = value
	return set
}

func compatibilityQuery(version string) CompatibilityQuery {
	return CompatibilityQuery{
		TargetID:      "production-a",
		DeploymentID:  "production-a",
		ClientID:      "codex",
		ClientVersion: version,
		Platform:      artifact.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH},
		AuthMode:      AuthNone,
	}
}

func TestCompatibilityVersionOnlyChangeAgainstBaselineIsTested(t *testing.T) {
	manifest := regressionTestManifest()
	baselineSource := compatibilityTestSet(t, manifest, "baseline-version", "1.0.0", time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	baseline, err := CreateBaseline(baselineSource, t.TempDir()+"/accepted", nil, time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	updated := compatibilityTestSet(t, manifest, "updated", "2.0.0", time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	envelope, err := BuildCompatibilityEnvelope(&baseline, []LoadedResultSet{updated}, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	classification, err := ClassifyCompatibilityExact(envelope, compatibilityQuery("2.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if classification.State != CompatibilityTested || classification.Point == nil {
		t.Fatalf("version-only change classification=%#v", classification)
	}
	for _, observation := range classification.Point.Observations {
		if observation.Regression || len(observation.RegressionKinds) != 0 {
			t.Fatalf("version-only change carried regression: %#v", observation)
		}
	}
}

func TestCompatibilityEnvelopeDoesNotSerializeProtectedEndpointPath(t *testing.T) {
	manifest := regressionTestManifest()
	set := compatibilityTestSet(t, manifest, "secret-safe", "1.0.0", time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC), interop.StatusPass)
	envelope, err := BuildCompatibilityEnvelope(nil, []LoadedResultSet{set}, CompatibilityStalePolicy{}, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"protected-value", "https://example.com/mcp"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("compatibility envelope leaked endpoint material %q: %s", forbidden, data)
		}
	}
}
