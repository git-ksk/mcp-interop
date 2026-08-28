package client

import (
	"reflect"
	"testing"
)

func TestCurrentGraduationDecisionsKeepAllResearchCandidatesBlocked(t *testing.T) {
	decisions, err := CurrentGraduationDecisions()
	if err != nil {
		t.Fatal(err)
	}
	wantIssues := map[string]int{
		"copilot": 48,
		"vscode":  6,
		"chatgpt": 20,
		"claude":  68,
	}
	if len(decisions) != len(wantIssues) {
		t.Fatalf("decisions=%d want %d", len(decisions), len(wantIssues))
	}
	for _, decision := range decisions {
		if err := ValidateGraduationDecision(decision); err != nil {
			t.Fatalf("%s invalid: %v", decision.ClientID, err)
		}
		if decision.Status != GraduationResearchOnly || decision.Eligible {
			t.Fatalf("candidate graduated without complete evidence: %#v", decision)
		}
		if len(decision.Blockers) == 0 {
			t.Fatalf("research candidate %s has no explicit blockers", decision.ClientID)
		}
		if issue, ok := wantIssues[decision.ClientID]; !ok || issue != decision.ResearchIssue {
			t.Fatalf("unexpected candidate issue mapping: %#v", decision)
		}
		delete(wantIssues, decision.ClientID)
	}
	if len(wantIssues) != 0 {
		t.Fatalf("missing candidates: %v", wantIssues)
	}
}

func TestGraduationEligibilityRequiresEveryMandatoryCriterion(t *testing.T) {
	decision := GraduationDecision{
		ClientID:      "future-client",
		DisplayName:   "Future Client",
		ResearchIssue: 999,
		Status:        GraduationEligibleBeta,
		Eligible:      true,
		Criteria:      graduationCriterionSet(nil),
		EvidenceRefs:  []string{"github:issue/999"},
	}
	for i := range decision.Criteria {
		decision.Criteria[i].Status = MaturityCriterionMet
	}
	if err := ValidateGraduationDecision(decision); err != nil {
		t.Fatalf("fully evidenced candidate rejected: %v", err)
	}

	decision.Criteria[0].Status = MaturityCriterionLimited
	decision.Blockers = []string{decision.Criteria[0].ID}
	if err := ValidateGraduationDecision(decision); err == nil {
		t.Fatal("eligible candidate accepted with a limited mandatory criterion")
	}
	decision.Status = GraduationResearchOnly
	decision.Eligible = false
	if err := ValidateGraduationDecision(decision); err != nil {
		t.Fatalf("research-only fallback rejected: %v", err)
	}
}

func TestGraduationDecisionCannotHideBlockersOrInventCriteria(t *testing.T) {
	decisions, err := CurrentGraduationDecisions()
	if err != nil {
		t.Fatal(err)
	}
	decision := decisions[0]
	decision.Blockers = decision.Blockers[:1]
	if err := ValidateGraduationDecision(decision); err == nil {
		t.Fatal("candidate hid mandatory blockers")
	}

	decision = decisions[0]
	decision.Criteria = append(decision.Criteria, MaturityCriterion{ID: "special_exception", Status: MaturityCriterionMet})
	if err := ValidateGraduationDecision(decision); err == nil {
		t.Fatal("candidate-specific weaker exception criterion was accepted")
	}

	decision = decisions[0]
	decision.EvidenceRefs = append(decision.EvidenceRefs, decision.EvidenceRefs[0])
	if err := ValidateGraduationDecision(decision); err == nil {
		t.Fatal("duplicate graduation evidence reference accepted")
	}
}

func TestShippedLiveAdapterIDsComeOnlyFromValidMaturityCatalog(t *testing.T) {
	ids, err := ShippedLiveAdapterIDs()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"antigravity", "codex", "cursor"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("shipped ids=%v want %v", ids, want)
	}
	for _, id := range []string{"copilot", "vscode", "chatgpt", "claude"} {
		shipped, err := IsShippedLiveAdapter(id)
		if err != nil {
			t.Fatal(err)
		}
		if shipped {
			t.Fatalf("research candidate %q bypassed graduation policy", id)
		}
	}
}
