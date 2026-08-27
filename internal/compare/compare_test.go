package compare

import (
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestArtifactsVersionOnlyChangeIsNotRegression(t *testing.T) {
	oldRun := testRun(t, "1.0.0")
	newRun := testRun(t, "2.0.0")
	report := mustArtifacts(t, artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
	if report.HasRegression {
		t.Fatalf("version-only change marked regression: %#v", report)
	}
	if len(report.Runs) != 1 || len(report.Runs[0].StageChanges) != 0 {
		t.Fatalf("unexpected version-only comparison: %#v", report)
	}
	if report.Runs[0].OldClientVersion != "1.0.0" || report.Runs[0].NewClientVersion != "2.0.0" {
		t.Fatalf("client versions not preserved: %#v", report.Runs[0])
	}
}

func TestArtifactsPassToNonPassRegressions(t *testing.T) {
	for _, tc := range []struct {
		name string
		to   interop.Status
		kind string
	}{
		{name: "fail", to: interop.StatusFail, kind: RegressionPassToFail},
		{name: "unknown", to: interop.StatusUnknown, kind: RegressionPassToUnknown},
		{name: "skip", to: interop.StatusSkip, kind: RegressionPassToSkip},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldRun := testRun(t, "1.0.0")
			newRun := testRun(t, "2.0.0")
			newRun.Stages[1].Status = tc.to
			report := mustArtifacts(t, artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
			if !report.HasRegression || len(report.Runs[0].StageChanges) != 1 {
				t.Fatalf("expected regression: %#v", report)
			}
			change := report.Runs[0].StageChanges[0]
			if change.Stage != interop.StageAuth || !contains(change.RegressionKinds, tc.kind) {
				t.Fatalf("unexpected change: %#v", change)
			}
		})
	}
}

func TestArtifactsReasonCodeChangesAreRegressions(t *testing.T) {
	for _, tc := range []struct {
		name string
		old  interop.ReasonCode
		new  interop.ReasonCode
	}{
		{name: "changed", old: interop.ReasonDCRUnsupported, new: interop.ReasonDCRFailed},
		{name: "added", old: "", new: interop.ReasonDCRFailed},
		{name: "removed", old: interop.ReasonDCRUnsupported, new: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldRun := testRun(t, "1.0.0")
			newRun := testRun(t, "2.0.0")
			oldRun.Stages[1].Status = interop.StatusFail
			newRun.Stages[1].Status = interop.StatusFail
			oldRun.Stages[1].ReasonCode = tc.old
			newRun.Stages[1].ReasonCode = tc.new

			report := mustArtifacts(t, artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
			if !report.HasRegression {
				t.Fatal("reason-code change must be a regression signal")
			}
			change := report.Runs[0].StageChanges[0]
			if !contains(change.RegressionKinds, RegressionReasonChanged) {
				t.Fatalf("reason change classification missing: %#v", change)
			}
		})
	}
}

func TestArtifactsMissingNewRunIsEvidenceLossRegression(t *testing.T) {
	oldRun := testRun(t, "1.0.0")
	newRun := testRun(t, "2.0.0")
	newRun.Client.ID = "cursor"
	newRun.Client.Product = "Cursor CLI"
	newRun.EvidenceProvenance.AdapterID = "cursor"

	report := mustArtifacts(t, artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
	if !report.HasRegression {
		t.Fatal("missing baseline run in new artifact must regress")
	}
	var sawMissing, sawNewOnly bool
	for _, run := range report.Runs {
		switch run.State {
		case RunMissingNew:
			sawMissing = run.Regression && contains(run.RegressionKinds, RegressionRunMissing)
		case RunNewOnly:
			sawNewOnly = !run.Regression
		}
	}
	if !sawMissing || !sawNewOnly {
		t.Fatalf("unexpected missing/new-only report: %#v", report)
	}
}

func testRun(t *testing.T, version string) artifact.Run {
	t.Helper()
	result := interop.NewResult("codex", "Codex CLI", version, "https://example.com/mcp?token=never-persist")
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

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mustArtifacts(t *testing.T, oldArtifact, newArtifact artifact.Artifact) Report {
	t.Helper()
	report, err := Artifacts(oldArtifact, newArtifact)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func TestArtifactsV2PairsByDeploymentIDAcrossProtectedPathChange(t *testing.T) {
	oldRun := testRunV2(t, "1.0.0", "https://example.com/mcp/old-secret", "production-a")
	newRun := testRunV2(t, "2.0.0", "https://example.com/mcp/new-secret", "production-a")

	report := mustArtifacts(t, artifact.NewArtifactV2([]artifact.Run{oldRun}), artifact.NewArtifactV2([]artifact.Run{newRun}))
	if report.HasRegression || len(report.Runs) != 1 || report.Runs[0].State != RunCompared {
		t.Fatalf("v2 deployment identity did not pair across protected-path change: %#v", report)
	}
}

func TestArtifactsRejectCrossSchemaComparison(t *testing.T) {
	v1 := artifact.NewArtifact([]artifact.Run{testRun(t, "1.0.0")})
	v2 := artifact.NewArtifactV2([]artifact.Run{testRunV2(t, "2.0.0", "https://example.com/mcp/secret", "production-a")})
	if _, err := Artifacts(v1, v2); err == nil {
		t.Fatal("cross-schema comparison must be explicit, not silently paired")
	}
}

func testRunV2(t *testing.T, version, endpoint, deploymentID string) artifact.Run {
	t.Helper()
	result := interop.NewResult("codex", "Codex CLI", version, endpoint)
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}
	run, err := artifact.NewRunV2ProtectedPath(
		result,
		endpoint,
		deploymentID,
		time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC),
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

func TestArtifactsV2UsesComparisonReportSchemaV2(t *testing.T) {
	run := testRunV2(t, "1.0.0", "https://example.com/mcp/secret", "production-a")
	report := mustArtifacts(t, artifact.NewArtifactV2([]artifact.Run{run}), artifact.NewArtifactV2([]artifact.Run{run}))
	if report.SchemaVersion != ReportSchemaVersionV2 {
		t.Fatalf("v2 comparison report schema=%d, want %d", report.SchemaVersion, ReportSchemaVersionV2)
	}
}
