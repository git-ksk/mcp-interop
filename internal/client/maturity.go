package client

import (
	"errors"
	"fmt"
)

// Maturity is an evidence-reviewed adapter lifecycle state. It is deliberately
// separate from Tier, which describes roadmap/delivery placement rather than a
// claim that the adapter has stable interoperability evidence.
type Maturity string

const (
	MaturityResearchOnly Maturity = "research_only"
	MaturityBeta         Maturity = "beta"
	MaturityStable       Maturity = "stable"
)

type MaturityCriterionStatus string

const (
	MaturityCriterionMet     MaturityCriterionStatus = "met"
	MaturityCriterionLimited MaturityCriterionStatus = "limited"
	MaturityCriterionMissing MaturityCriterionStatus = "missing"
)

const (
	CriterionDirectRealClientBoundary    = "direct_real_client_boundary"
	CriterionIsolation                   = "isolated_state"
	CriterionCleanup                     = "owned_cleanup"
	CriterionSecretSafety                = "secret_safety"
	CriterionFailureSemantics            = "conservative_failure_semantics"
	CriterionFixtureE2E                  = "controlled_fixture_e2e"
	CriterionExactVersionPlatform        = "exact_version_platform_evidence"
	CriterionRepeatPathVersionCoverage   = "repeat_path_version_coverage"
	CriterionAdvertisedPlatformCoverage  = "advertised_platform_coverage"
	CriterionMeasurementSurfaceStability = "measurement_surface_stability"
	CriterionMaintenancePath             = "regression_maintenance_path"
)

var betaMaturityCriteria = []string{
	CriterionDirectRealClientBoundary,
	CriterionIsolation,
	CriterionCleanup,
	CriterionSecretSafety,
	CriterionFailureSemantics,
	CriterionFixtureE2E,
	CriterionExactVersionPlatform,
}

var stableMaturityCriteria = []string{
	CriterionDirectRealClientBoundary,
	CriterionIsolation,
	CriterionCleanup,
	CriterionSecretSafety,
	CriterionFailureSemantics,
	CriterionFixtureE2E,
	CriterionExactVersionPlatform,
	CriterionRepeatPathVersionCoverage,
	CriterionAdvertisedPlatformCoverage,
	CriterionMeasurementSurfaceStability,
	CriterionMaintenancePath,
}

type MaturityCriterion struct {
	ID     string                  `json:"id"`
	Status MaturityCriterionStatus `json:"status"`
}

type MaturityDecision struct {
	ClientID     string              `json:"client_id"`
	DisplayName  string              `json:"display_name"`
	Tier         Tier                `json:"tier"`
	Maturity     Maturity            `json:"maturity"`
	Criteria     []MaturityCriterion `json:"criteria"`
	Blockers     []string            `json:"blockers,omitempty"`
	EvidenceRefs []string            `json:"evidence_refs"`
}

// MaturityDecisions returns the current explicit evidence review for shipped
// live adapters. It performs no client detection or execution; a newly installed
// client version is handled by exact compatibility evidence, not by silently
// changing adapter maturity.
func MaturityDecisions() []MaturityDecision {
	return []MaturityDecision{
		{
			ClientID:    "codex",
			DisplayName: "Codex CLI",
			Tier:        TierV1,
			Maturity:    MaturityBeta,
			Criteria: maturityCriteria(map[string]MaturityCriterionStatus{
				CriterionDirectRealClientBoundary:    MaturityCriterionMet,
				CriterionIsolation:                   MaturityCriterionMet,
				CriterionCleanup:                     MaturityCriterionMet,
				CriterionSecretSafety:                MaturityCriterionMet,
				CriterionFailureSemantics:            MaturityCriterionMet,
				CriterionFixtureE2E:                  MaturityCriterionMet,
				CriterionExactVersionPlatform:        MaturityCriterionMet,
				CriterionRepeatPathVersionCoverage:   MaturityCriterionLimited,
				CriterionAdvertisedPlatformCoverage:  MaturityCriterionLimited,
				CriterionMeasurementSurfaceStability: MaturityCriterionMet,
				CriterionMaintenancePath:             MaturityCriterionMet,
			}),
			Blockers: []string{
				CriterionRepeatPathVersionCoverage,
				CriterionAdvertisedPlatformCoverage,
			},
			EvidenceRefs: []string{
				"docs/observed-coverage.md",
				"github:pr/108",
			},
		},
		{
			ClientID:    "cursor",
			DisplayName: "Cursor CLI",
			Tier:        TierV1,
			Maturity:    MaturityBeta,
			Criteria: maturityCriteria(map[string]MaturityCriterionStatus{
				CriterionDirectRealClientBoundary:    MaturityCriterionMet,
				CriterionIsolation:                   MaturityCriterionMet,
				CriterionCleanup:                     MaturityCriterionMet,
				CriterionSecretSafety:                MaturityCriterionMet,
				CriterionFailureSemantics:            MaturityCriterionMet,
				CriterionFixtureE2E:                  MaturityCriterionMet,
				CriterionExactVersionPlatform:        MaturityCriterionMet,
				CriterionRepeatPathVersionCoverage:   MaturityCriterionLimited,
				CriterionAdvertisedPlatformCoverage:  MaturityCriterionLimited,
				CriterionMeasurementSurfaceStability: MaturityCriterionLimited,
				CriterionMaintenancePath:             MaturityCriterionMet,
			}),
			Blockers: []string{
				CriterionRepeatPathVersionCoverage,
				CriterionAdvertisedPlatformCoverage,
				CriterionMeasurementSurfaceStability,
			},
			EvidenceRefs: []string{
				"docs/observed-coverage.md",
				"github:pr/39",
				"github:pr/108",
			},
		},
		{
			ClientID:    "antigravity",
			DisplayName: "Antigravity CLI",
			Tier:        TierV1,
			Maturity:    MaturityBeta,
			Criteria: maturityCriteria(map[string]MaturityCriterionStatus{
				CriterionDirectRealClientBoundary:    MaturityCriterionMet,
				CriterionIsolation:                   MaturityCriterionMet,
				CriterionCleanup:                     MaturityCriterionMet,
				CriterionSecretSafety:                MaturityCriterionMet,
				CriterionFailureSemantics:            MaturityCriterionMet,
				CriterionFixtureE2E:                  MaturityCriterionMet,
				CriterionExactVersionPlatform:        MaturityCriterionMet,
				CriterionRepeatPathVersionCoverage:   MaturityCriterionLimited,
				CriterionAdvertisedPlatformCoverage:  MaturityCriterionLimited,
				CriterionMeasurementSurfaceStability: MaturityCriterionLimited,
				CriterionMaintenancePath:             MaturityCriterionMet,
			}),
			Blockers: []string{
				CriterionRepeatPathVersionCoverage,
				CriterionAdvertisedPlatformCoverage,
				CriterionMeasurementSurfaceStability,
			},
			EvidenceRefs: []string{
				"docs/observed-coverage.md",
				"docs/antigravity-oauth.md",
				"github:pr/40",
				"github:pr/108",
			},
		},
	}
}

func maturityCriteria(statuses map[string]MaturityCriterionStatus) []MaturityCriterion {
	criteria := make([]MaturityCriterion, 0, len(stableMaturityCriteria))
	for _, id := range stableMaturityCriteria {
		criteria = append(criteria, MaturityCriterion{ID: id, Status: statuses[id]})
	}
	return criteria
}

// ValidateMaturityDecision keeps machine-readable maturity claims conservative.
// Stable requires every stable criterion to be met; beta requires every beta
// criterion and at least one documented stable blocker.
func ValidateMaturityDecision(value MaturityDecision) error {
	if value.ClientID == "" || value.DisplayName == "" {
		return errors.New("maturity decision requires client identity")
	}
	if value.Tier == "" {
		return errors.New("maturity decision requires adapter tier")
	}
	if value.Maturity != MaturityResearchOnly && value.Maturity != MaturityBeta && value.Maturity != MaturityStable {
		return fmt.Errorf("unsupported maturity %q", value.Maturity)
	}
	allowedCriteria := make(map[string]struct{}, len(stableMaturityCriteria))
	for _, id := range stableMaturityCriteria {
		allowedCriteria[id] = struct{}{}
	}
	statuses := make(map[string]MaturityCriterionStatus, len(value.Criteria))
	for _, criterion := range value.Criteria {
		if criterion.ID == "" {
			return errors.New("maturity criterion id is required")
		}
		if _, ok := allowedCriteria[criterion.ID]; !ok {
			return fmt.Errorf("unsupported maturity criterion %q", criterion.ID)
		}
		if _, duplicate := statuses[criterion.ID]; duplicate {
			return fmt.Errorf("duplicate maturity criterion %q", criterion.ID)
		}
		switch criterion.Status {
		case MaturityCriterionMet, MaturityCriterionLimited, MaturityCriterionMissing:
		default:
			return fmt.Errorf("unsupported maturity criterion status %q", criterion.Status)
		}
		statuses[criterion.ID] = criterion.Status
	}
	for _, id := range stableMaturityCriteria {
		if _, ok := statuses[id]; !ok {
			return fmt.Errorf("missing maturity criterion %q", id)
		}
	}
	if len(value.EvidenceRefs) == 0 {
		return errors.New("maturity decision requires retained evidence references")
	}
	evidenceRefs := make(map[string]struct{}, len(value.EvidenceRefs))
	for _, ref := range value.EvidenceRefs {
		if ref == "" {
			return errors.New("maturity evidence reference must not be empty")
		}
		if _, duplicate := evidenceRefs[ref]; duplicate {
			return fmt.Errorf("duplicate maturity evidence reference %q", ref)
		}
		evidenceRefs[ref] = struct{}{}
	}
	if value.Maturity == MaturityResearchOnly {
		return nil
	}
	for _, id := range betaMaturityCriteria {
		if statuses[id] != MaturityCriterionMet {
			return fmt.Errorf("%s maturity requires criterion %q to be met", value.Maturity, id)
		}
	}
	if value.Maturity == MaturityStable {
		for _, id := range stableMaturityCriteria {
			if statuses[id] != MaturityCriterionMet {
				return fmt.Errorf("stable maturity requires criterion %q to be met", id)
			}
		}
		if len(value.Blockers) != 0 {
			return errors.New("stable maturity cannot retain blockers")
		}
		return nil
	}
	expectedBlockers := make(map[string]struct{})
	for _, id := range stableMaturityCriteria {
		if statuses[id] != MaturityCriterionMet {
			expectedBlockers[id] = struct{}{}
		}
	}
	if len(expectedBlockers) == 0 {
		return errors.New("beta maturity requires at least one documented stable blocker")
	}
	seenBlockers := make(map[string]struct{}, len(value.Blockers))
	for _, blocker := range value.Blockers {
		status, ok := statuses[blocker]
		if !ok {
			return fmt.Errorf("maturity blocker %q is not a criterion", blocker)
		}
		if _, duplicate := seenBlockers[blocker]; duplicate {
			return fmt.Errorf("duplicate maturity blocker %q", blocker)
		}
		if status == MaturityCriterionMet {
			return fmt.Errorf("maturity blocker %q is already met", blocker)
		}
		seenBlockers[blocker] = struct{}{}
	}
	if len(seenBlockers) != len(expectedBlockers) {
		return errors.New("beta maturity blockers must list every limited or missing stable criterion")
	}
	for blocker := range expectedBlockers {
		if _, ok := seenBlockers[blocker]; !ok {
			return fmt.Errorf("beta maturity blocker %q is not documented", blocker)
		}
	}
	return nil
}
