package compare

import (
	"testing"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestArtifactsRecoveryToPassIsNotRegression(t *testing.T) {
	oldRun := testRun(t, "1.0.0")
	newRun := testRun(t, "2.0.0")
	oldRun.Stages[1].Status = interop.StatusFail
	oldRun.Stages[1].ReasonCode = interop.ReasonDCRUnsupported
	newRun.Stages[1].Status = interop.StatusPass
	newRun.Stages[1].ReasonCode = ""

	report := Artifacts(artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
	if report.HasRegression {
		t.Fatalf("FAIL-to-PASS recovery must not be a regression: %#v", report)
	}
	if len(report.Runs) != 1 || len(report.Runs[0].StageChanges) != 1 {
		t.Fatalf("expected the recovery to remain visible as one stage change: %#v", report)
	}
	change := report.Runs[0].StageChanges[0]
	if change.OldStatus != interop.StatusFail || change.NewStatus != interop.StatusPass {
		t.Fatalf("unexpected recovery change: %#v", change)
	}
	if change.Regression || len(change.RegressionKinds) != 0 {
		t.Fatalf("recovery change was classified as regression: %#v", change)
	}
}

func TestArtifactsReasonChangeOnContinuedFailureStillRegresses(t *testing.T) {
	oldRun := testRun(t, "1.0.0")
	newRun := testRun(t, "2.0.0")
	oldRun.Stages[1].Status = interop.StatusFail
	newRun.Stages[1].Status = interop.StatusFail
	oldRun.Stages[1].ReasonCode = interop.ReasonDCRUnsupported
	newRun.Stages[1].ReasonCode = interop.ReasonDCRFailed

	report := Artifacts(artifact.NewArtifact([]artifact.Run{oldRun}), artifact.NewArtifact([]artifact.Run{newRun}))
	if !report.HasRegression || !contains(report.Runs[0].RegressionKinds, RegressionReasonChanged) {
		t.Fatalf("continued failure reason change must remain a regression: %#v", report)
	}
}
