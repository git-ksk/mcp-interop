package client

import (
	"errors"
	"fmt"
	"sort"
)

const (
	GraduationResearchOnly = "research_only"
	GraduationEligibleBeta = "eligible_for_beta"

	GraduationDirectBoundary       = "direct_real_client_boundary"
	GraduationIsolatedState        = "isolated_state"
	GraduationOwnedCleanup         = "owned_cleanup"
	GraduationSecretSafety         = "secret_safety"
	GraduationFailureSemantics     = "conservative_failure_semantics"
	GraduationControlledFixtureE2E = "controlled_fixture_e2e"
	GraduationExactVersionPlatform = "exact_version_platform_evidence"
	GraduationSupportedScope       = "supported_platform_scope"
)

var preGraduationShippedAdapters = map[string]struct{}{
	"codex":       {},
	"cursor":      {},
	"antigravity": {},
}

var graduationCriteria = []string{
	GraduationDirectBoundary,
	GraduationIsolatedState,
	GraduationOwnedCleanup,
	GraduationSecretSafety,
	GraduationFailureSemantics,
	GraduationControlledFixtureE2E,
	GraduationExactVersionPlatform,
	GraduationSupportedScope,
}

type GraduationDecision struct {
	ClientID      string              `json:"client_id"`
	DisplayName   string              `json:"display_name"`
	ResearchIssue int                 `json:"research_issue"`
	Status        string              `json:"status"`
	Eligible      bool                `json:"eligible"`
	Criteria      []MaturityCriterion `json:"criteria"`
	Blockers      []string            `json:"blockers,omitempty"`
	EvidenceRefs  []string            `json:"evidence_refs"`
}

// graduationDecisionDrafts returns the reviewed evidence inputs for research
// candidates. CurrentGraduationDecisions fills the exact blocker set and
// validates every decision before exposing it.
func graduationDecisionDrafts() []GraduationDecision {
	return []GraduationDecision{
		{
			ClientID:      "copilot",
			DisplayName:   "GitHub Copilot CLI",
			ResearchIssue: 48,
			Status:        GraduationResearchOnly,
			Eligible:      false,
			Criteria: graduationCriterionSet(map[string]MaturityCriterionStatus{
				GraduationDirectBoundary:       MaturityCriterionLimited,
				GraduationIsolatedState:        MaturityCriterionLimited,
				GraduationOwnedCleanup:         MaturityCriterionMet,
				GraduationSecretSafety:         MaturityCriterionLimited,
				GraduationFailureSemantics:     MaturityCriterionMet,
				GraduationControlledFixtureE2E: MaturityCriterionLimited,
				GraduationExactVersionPlatform: MaturityCriterionMet,
				GraduationSupportedScope:       MaturityCriterionMissing,
			}),
			EvidenceRefs: []string{
				"github:issue/48",
				"github:pr/49",
				"github:pr/69",
				"docs/copilot-cli-poc.md",
			},
		},
		{
			ClientID:      "vscode",
			DisplayName:   "VS Code",
			ResearchIssue: 6,
			Status:        GraduationResearchOnly,
			Eligible:      false,
			Criteria: graduationCriterionSet(map[string]MaturityCriterionStatus{
				GraduationDirectBoundary:       MaturityCriterionMissing,
				GraduationIsolatedState:        MaturityCriterionMet,
				GraduationOwnedCleanup:         MaturityCriterionLimited,
				GraduationSecretSafety:         MaturityCriterionLimited,
				GraduationFailureSemantics:     MaturityCriterionMet,
				GraduationControlledFixtureE2E: MaturityCriterionLimited,
				GraduationExactVersionPlatform: MaturityCriterionMet,
				GraduationSupportedScope:       MaturityCriterionMissing,
			}),
			EvidenceRefs: []string{
				"github:issue/6",
				"docs/vscode-agent-plugin-poc.md",
			},
		},
		{
			ClientID:      "chatgpt",
			DisplayName:   "ChatGPT",
			ResearchIssue: 20,
			Status:        GraduationResearchOnly,
			Eligible:      false,
			Criteria: graduationCriterionSet(map[string]MaturityCriterionStatus{
				GraduationDirectBoundary:       MaturityCriterionMissing,
				GraduationIsolatedState:        MaturityCriterionMissing,
				GraduationOwnedCleanup:         MaturityCriterionMissing,
				GraduationSecretSafety:         MaturityCriterionMissing,
				GraduationFailureSemantics:     MaturityCriterionMet,
				GraduationControlledFixtureE2E: MaturityCriterionMissing,
				GraduationExactVersionPlatform: MaturityCriterionMissing,
				GraduationSupportedScope:       MaturityCriterionMissing,
			}),
			EvidenceRefs: []string{
				"github:issue/20",
				"docs/chatgpt-diagnostics.md",
			},
		},
		{
			ClientID:      "claude",
			DisplayName:   "Claude web/Desktop",
			ResearchIssue: 68,
			Status:        GraduationResearchOnly,
			Eligible:      false,
			Criteria: graduationCriterionSet(map[string]MaturityCriterionStatus{
				GraduationDirectBoundary:       MaturityCriterionMissing,
				GraduationIsolatedState:        MaturityCriterionMissing,
				GraduationOwnedCleanup:         MaturityCriterionMissing,
				GraduationSecretSafety:         MaturityCriterionMissing,
				GraduationFailureSemantics:     MaturityCriterionMet,
				GraduationControlledFixtureE2E: MaturityCriterionMissing,
				GraduationExactVersionPlatform: MaturityCriterionMissing,
				GraduationSupportedScope:       MaturityCriterionMissing,
			}),
			EvidenceRefs: []string{
				"github:issue/68",
			},
		},
	}
}

func graduationCriterionSet(statuses map[string]MaturityCriterionStatus) []MaturityCriterion {
	criteria := make([]MaturityCriterion, 0, len(graduationCriteria))
	for _, id := range graduationCriteria {
		criteria = append(criteria, MaturityCriterion{ID: id, Status: statuses[id]})
	}
	return criteria
}

func ValidateGraduationDecision(value GraduationDecision) error {
	if value.ClientID == "" || value.DisplayName == "" {
		return errors.New("graduation decision requires client identity")
	}
	if value.ResearchIssue <= 0 {
		return errors.New("graduation decision requires research_issue")
	}
	if value.Status != GraduationResearchOnly && value.Status != GraduationEligibleBeta {
		return fmt.Errorf("unsupported graduation status %q", value.Status)
	}
	allowed := make(map[string]struct{}, len(graduationCriteria))
	for _, id := range graduationCriteria {
		allowed[id] = struct{}{}
	}
	statuses := make(map[string]MaturityCriterionStatus, len(value.Criteria))
	for _, criterion := range value.Criteria {
		if _, ok := allowed[criterion.ID]; !ok {
			return fmt.Errorf("unsupported graduation criterion %q", criterion.ID)
		}
		if _, duplicate := statuses[criterion.ID]; duplicate {
			return fmt.Errorf("duplicate graduation criterion %q", criterion.ID)
		}
		switch criterion.Status {
		case MaturityCriterionMet, MaturityCriterionLimited, MaturityCriterionMissing:
		default:
			return fmt.Errorf("unsupported graduation criterion status %q", criterion.Status)
		}
		statuses[criterion.ID] = criterion.Status
	}
	for _, id := range graduationCriteria {
		if _, ok := statuses[id]; !ok {
			return fmt.Errorf("missing graduation criterion %q", id)
		}
	}
	if len(value.EvidenceRefs) == 0 {
		return errors.New("graduation decision requires retained evidence references")
	}
	refs := make(map[string]struct{}, len(value.EvidenceRefs))
	for _, ref := range value.EvidenceRefs {
		if ref == "" {
			return errors.New("graduation evidence reference must not be empty")
		}
		if _, duplicate := refs[ref]; duplicate {
			return fmt.Errorf("duplicate graduation evidence reference %q", ref)
		}
		refs[ref] = struct{}{}
	}

	expectedBlockers := make(map[string]struct{})
	for _, id := range graduationCriteria {
		if statuses[id] != MaturityCriterionMet {
			expectedBlockers[id] = struct{}{}
		}
	}
	seenBlockers := make(map[string]struct{}, len(value.Blockers))
	for _, blocker := range value.Blockers {
		status, ok := statuses[blocker]
		if !ok {
			return fmt.Errorf("graduation blocker %q is not a criterion", blocker)
		}
		if status == MaturityCriterionMet {
			return fmt.Errorf("graduation blocker %q is already met", blocker)
		}
		if _, duplicate := seenBlockers[blocker]; duplicate {
			return fmt.Errorf("duplicate graduation blocker %q", blocker)
		}
		seenBlockers[blocker] = struct{}{}
	}
	if len(seenBlockers) != len(expectedBlockers) {
		return errors.New("graduation blockers must list every limited or missing criterion")
	}
	for blocker := range expectedBlockers {
		if _, ok := seenBlockers[blocker]; !ok {
			return fmt.Errorf("graduation blocker %q is not documented", blocker)
		}
	}

	allMet := len(expectedBlockers) == 0
	if value.Eligible != allMet {
		return errors.New("graduation eligible flag must match complete mandatory criteria")
	}
	if allMet && value.Status != GraduationEligibleBeta {
		return errors.New("complete graduation criteria require eligible_for_beta status")
	}
	if !allMet && value.Status != GraduationResearchOnly {
		return errors.New("incomplete graduation criteria require research_only status")
	}
	return nil
}

// CurrentGraduationDecisions validates and fills the exact blocker set for the
// reviewed candidate table. The returned decisions remain deterministic.
func CurrentGraduationDecisions() ([]GraduationDecision, error) {
	decisions := graduationDecisionDrafts()
	for i := range decisions {
		blockers := make([]string, 0)
		for _, criterion := range decisions[i].Criteria {
			if criterion.Status != MaturityCriterionMet {
				blockers = append(blockers, criterion.ID)
			}
		}
		sort.Strings(blockers)
		decisions[i].Blockers = blockers
		if err := ValidateGraduationDecision(decisions[i]); err != nil {
			return nil, fmt.Errorf("%s: %w", decisions[i].ClientID, err)
		}
	}
	return decisions, nil
}

// ShippedLiveAdapterIDs is the only source of client IDs allowed through the
// live-test parser. A future adapter must first have a valid shipped maturity
// decision; research candidates cannot become runnable through a parser/switch
// exception alone.
func ShippedLiveAdapterIDs() ([]string, error) {
	decisions := MaturityDecisions()
	ids := make([]string, 0, len(decisions))
	seen := make(map[string]struct{}, len(decisions))
	for _, decision := range decisions {
		if err := ValidateMaturityDecision(decision); err != nil {
			return nil, fmt.Errorf("invalid shipped maturity decision for %s: %w", decision.ClientID, err)
		}
		if decision.Maturity == MaturityResearchOnly {
			return nil, fmt.Errorf("research-only client %q cannot be a shipped live adapter", decision.ClientID)
		}
		if decision.Tier != TierV1 {
			return nil, fmt.Errorf("shipped live adapter %q must use tier %q", decision.ClientID, TierV1)
		}
		if _, duplicate := seen[decision.ClientID]; duplicate {
			return nil, fmt.Errorf("duplicate shipped live adapter %q", decision.ClientID)
		}
		if _, grandfathered := preGraduationShippedAdapters[decision.ClientID]; !grandfathered {
			eligible, err := candidateEligibleForBeta(decision.ClientID)
			if err != nil {
				return nil, err
			}
			if !eligible {
				return nil, fmt.Errorf("new shipped live adapter %q has not passed the common graduation gate", decision.ClientID)
			}
		}
		seen[decision.ClientID] = struct{}{}
		ids = append(ids, decision.ClientID)
	}

	tierV1 := make(map[string]struct{})
	for _, spec := range Specs() {
		if spec.Tier != TierV1 {
			continue
		}
		tierV1[spec.ID] = struct{}{}
		if _, ok := seen[spec.ID]; !ok {
			return nil, fmt.Errorf("tier-v1 client %q has no shipped maturity decision", spec.ID)
		}
	}
	for id := range seen {
		if _, ok := tierV1[id]; !ok {
			return nil, fmt.Errorf("shipped maturity decision %q has no tier-v1 client spec", id)
		}
	}

	sort.Strings(ids)
	return ids, nil
}

func candidateEligibleForBeta(id string) (bool, error) {
	decisions, err := CurrentGraduationDecisions()
	if err != nil {
		return false, fmt.Errorf("invalid research graduation policy: %w", err)
	}
	for _, decision := range decisions {
		if decision.ClientID == id {
			return decision.Eligible && decision.Status == GraduationEligibleBeta, nil
		}
	}
	return false, nil
}

func IsShippedLiveAdapter(id string) (bool, error) {
	ids, err := ShippedLiveAdapterIDs()
	if err != nil {
		return false, err
	}
	index := sort.SearchStrings(ids, id)
	return index < len(ids) && ids[index] == id, nil
}
