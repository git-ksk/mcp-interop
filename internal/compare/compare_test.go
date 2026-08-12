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
	report := Artifacts(artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
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
			report := Artifacts(artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
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

func TestArtifactsReasonCodeChangeIsRegression(t *testing.T) {
	oldRun := testRun(t, "1.0.0")
	newRun := testRun(t, "2.0.0")
	oldRun.Stages[1].Status = interop.StatusFail
	newRun.Stages[1].Status = interop.StatusFail
	oldRun.Stages[1].ReasonCode = interop.ReasonDCRUnsupported
	newRun.Stages[1].ReasonCode = interop.ReasonDCRFailed

	report := Artifacts(artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
	if !report.HasRegression {
		t.Fatal("reason-code change must be a regression signal")
	}
	change := report.Runs[0].StageChanges[0]
	if !contains(change.RegressionKinds, RegressionReasonChanged) {
		t.Fatalf("reason change classification missing: %#v", change)
	}
}

func TestArtifactsMissingNewRunIsEvidenceLossRegression(t *testing.T) {
	oldRun := testRun(t, "1.0.0")
	newRun := testRun(t, "2.0.0")
	newRun.Client.ID = "cursor"
	newRun.Client.Product = "Cursor CLI"
	newRun.EvidenceProvenance.AdapterID = "cursor"

	report := Artifacts(artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
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
