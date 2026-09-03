package client

import (
	"testing"
)

func TestCurrentShippedMaturityDecisionsAreValidAndConservative(t *testing.T) {
	decisions := MaturityDecisions()
	if len(decisions) != 3 {
		t.Fatalf("decisions=%d want 3", len(decisions))
	}
	want := map[string]Maturity{
		"codex":       MaturityStable,
		"cursor":      MaturityBeta,
		"antigravity": MaturityBeta,
	}
	for _, decision := range decisions {
		if err := ValidateMaturityDecision(decision); err != nil {
			t.Fatalf("%s maturity invalid: %v", decision.ClientID, err)
		}
		if got, ok := want[decision.ClientID]; !ok || decision.Maturity != got {
			t.Fatalf("unexpected maturity for %s: %q", decision.ClientID, decision.Maturity)
		}
		if decision.Tier != TierV1 {
			t.Fatalf("%s tier=%q; delivery tier must remain separate from maturity", decision.ClientID, decision.Tier)
		}
		delete(want, decision.ClientID)
	}
	if len(want) != 0 {
		t.Fatalf("missing shipped maturity decisions: %v", want)
	}
}

func TestStableMaturityRequiresEveryStableCriterion(t *testing.T) {
	decision := MaturityDecisions()[0]
	decision.Maturity = MaturityStable
	decision.Blockers = nil
	for i := range decision.Criteria {
		decision.Criteria[i].Status = MaturityCriterionMet
	}
	if err := ValidateMaturityDecision(decision); err != nil {
		t.Fatalf("fully evidenced stable decision rejected: %v", err)
	}

	for i := range decision.Criteria {
		if decision.Criteria[i].ID == CriterionAdvertisedPlatformCoverage {
			decision.Criteria[i].Status = MaturityCriterionLimited
			break
		}
	}
	if err := ValidateMaturityDecision(decision); err == nil {
		t.Fatal("stable maturity accepted limited platform evidence")
	}
}

func TestBetaMaturityRequiresAllBetaCriteriaAndDocumentedBlocker(t *testing.T) {
	decision := MaturityDecisions()[1]
	decision.Blockers = nil
	if err := ValidateMaturityDecision(decision); err == nil {
		t.Fatal("beta maturity without a stable blocker was accepted")
	}

	decision = MaturityDecisions()[1]
	for i := range decision.Criteria {
		if decision.Criteria[i].ID == CriterionSecretSafety {
			decision.Criteria[i].Status = MaturityCriterionLimited
			break
		}
	}
	if err := ValidateMaturityDecision(decision); err == nil {
		t.Fatal("beta maturity accepted limited secret-safety evidence")
	}
}

func TestBetaMaturityMustExposeEveryLimitedOrMissingCriterionAsBlocker(t *testing.T) {
	decision := MaturityDecisions()[1]
	decision.Blockers = decision.Blockers[:1]
	if err := ValidateMaturityDecision(decision); err == nil {
		t.Fatal("beta maturity hid limited stable criteria from blocker list")
	}
}

func TestMaturityDecisionRejectsUnknownCriteriaAndDuplicateEvidenceRefs(t *testing.T) {
	decision := MaturityDecisions()[0]
	decision.Criteria = append(decision.Criteria, MaturityCriterion{ID: "future_guess", Status: MaturityCriterionMet})
	if err := ValidateMaturityDecision(decision); err == nil {
		t.Fatal("unknown maturity criterion was accepted")
	}

	decision = MaturityDecisions()[0]
	decision.EvidenceRefs = append(decision.EvidenceRefs, decision.EvidenceRefs[0])
	if err := ValidateMaturityDecision(decision); err == nil {
		t.Fatal("duplicate maturity evidence reference was accepted")
	}
}
