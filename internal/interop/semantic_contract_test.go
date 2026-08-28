package interop

import "testing"

func TestV1CandidateCorePassRequiresEveryStableStage(t *testing.T) {
	result := NewResult("client", "Client", "1.0", "https://example.com/mcp")
	for _, stage := range OrderedStages {
		result.Set(stage, StatusPass, "direct evidence")
	}
	if !result.Passed() {
		t.Fatal("all four stable stages must produce PASS")
	}
	for _, status := range []Status{StatusFail, StatusSkip, StatusUnknown} {
		copyResult := result
		copyResult.Stages = append([]StageResult(nil), result.Stages...)
		copyResult.Set(StageTools, status, "not pass")
		if copyResult.Passed() {
			t.Fatalf("tools=%s incorrectly produced aggregate PASS", status)
		}
	}
}

func TestV1CandidateProtocolReadinessCannotBePromotedByFixtureOnlyEvidence(t *testing.T) {
	result := NewResult("client", "Client", "1.0", "https://example.com/mcp")
	fixture := ProtocolObservation{Era: ProtocolEraModern, Revision: "2026-07-28", Source: ProtocolEvidenceControlledFixture, Readiness: ProtocolReadinessToolInventory}
	if result.SetProtocolReadiness(StatusPass, fixture, "fixture only") {
		t.Fatal("fixture-only protocol evidence must not create deployment-specific init PASS")
	}
	real := ProtocolObservation{Era: ProtocolEraUnknown, Source: ProtocolEvidenceRealClientSurface, Readiness: ProtocolReadinessToolInventory}
	if !result.SetProtocolReadiness(StatusPass, real, "real client inventory") {
		t.Fatal("direct real-client readiness should project to init")
	}
	initStage, _ := result.Get(StageInit)
	if initStage.Status != StatusPass {
		t.Fatalf("init=%s want pass", initStage.Status)
	}
}
