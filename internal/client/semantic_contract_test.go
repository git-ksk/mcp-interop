package client

import (
	"reflect"
	"testing"
)

func TestV1CandidateShippedAdapterIDsAndLifecycleStatesRemainStable(t *testing.T) {
	ids, err := ShippedLiveAdapterIDs()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"antigravity", "codex", "cursor"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("shipped adapter IDs changed: got=%v want=%v", ids, want)
	}
	if MaturityResearchOnly != "research_only" || MaturityBeta != "beta" || MaturityStable != "stable" {
		t.Fatalf("maturity state identity changed: %q %q %q", MaturityResearchOnly, MaturityBeta, MaturityStable)
	}
	if GraduationResearchOnly != "research_only" || GraduationEligibleBeta != "eligible_for_beta" {
		t.Fatalf("graduation state identity changed: %q %q", GraduationResearchOnly, GraduationEligibleBeta)
	}
	wantMaturity := map[string]Maturity{
		"codex":       MaturityStable,
		"cursor":      MaturityStable,
		"antigravity": MaturityStable,
	}
	for _, decision := range MaturityDecisions() {
		if decision.Maturity != wantMaturity[decision.ClientID] {
			t.Fatalf("unexpected shipped adapter maturity for %s: got %s want %s", decision.ClientID, decision.Maturity, wantMaturity[decision.ClientID])
		}
	}
}
