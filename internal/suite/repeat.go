package suite

import (
	"errors"
	"fmt"
	"sort"
)

const (
	RepeatReportSchemaVersion = 1
	RepeatReportArtifactType  = "mcp-interop/suite-repeat-report"

	RepeatDecisionClean           = RepeatDecision("clean")
	RepeatDecisionNonPass         = RepeatDecision("non_pass")
	RepeatDecisionUnstable        = RepeatDecision("unstable")
	RepeatDecisionNonPassUnstable = RepeatDecision("non_pass_and_unstable")
	RepeatDecisionIncomplete      = RepeatDecision("incomplete")
)

// RepeatDecision describes execution consistency, not a regression relative
// to an independently accepted baseline. Use suite compare for regressions.
type RepeatDecision string

// RepeatReport is an additive derived report. It never changes the strict
// suite manifest v1 or result-index v1 and contains no resolved endpoint URL.
type RepeatReport struct {
	SchemaVersion       int            `json:"schema_version"`
	ArtifactType        string         `json:"artifact_type"`
	ManifestFingerprint string         `json:"manifest_fingerprint"`
	RequestedAttempts   int            `json:"requested_attempts"`
	CompletedAttempts   int            `json:"completed_attempts"`
	Complete            bool           `json:"complete"`
	Decision            RepeatDecision `json:"decision"`
	HasNonPass          bool           `json:"has_non_pass"`
	HasUnstable         bool           `json:"has_unstable"`
	AttemptIndexes      []string       `json:"attempt_indexes"`
	Runs                []RepeatRun    `json:"runs"`
}

type RepeatRun struct {
	TargetID     string          `json:"target_id"`
	DeploymentID string          `json:"deployment_id"`
	ClientID     string          `json:"client_id"`
	AuthMode     AuthMode        `json:"auth_mode"`
	Unstable     bool            `json:"unstable"`
	AllPass      bool            `json:"all_pass"`
	Attempts     []RepeatAttempt `json:"attempts"`
}

type RepeatAttempt struct {
	Attempt  int          `json:"attempt"`
	Evidence *RunEvidence `json:"evidence"`
}

// AnalyzeRepeatedResultSets retains every completed attempt. It requires
// exactly matching trusted manifest fingerprints and identities. A mixed
// outcome is unstable even if the last attempt passed. Unlike suite compare,
// the first attempt participates in this assessment rather than becoming an
// accepted regression baseline.
func AnalyzeRepeatedResultSets(sets []LoadedResultSet, indexes []string, requested int) (RepeatReport, error) {
	if requested < 2 || requested > 5 {
		return RepeatReport{}, errors.New("repeated suite requires 2..5 attempts")
	}
	if len(sets) == 0 || len(sets) > requested || len(sets) != len(indexes) {
		return RepeatReport{}, errors.New("repeat evidence and index references do not match")
	}
	base := sets[0].Index
	if err := ValidateResultIndex(base); err != nil {
		return RepeatReport{}, fmt.Errorf("attempt[1] index: %w", err)
	}
	baseEntries := resultEntryMap(base)
	for i, set := range sets {
		if err := ValidateResultIndex(set.Index); err != nil {
			return RepeatReport{}, fmt.Errorf("attempt[%d] index: %w", i+1, err)
		}
		if set.Index.ManifestFingerprint != base.ManifestFingerprint || set.Index.ExecutionContext != base.ExecutionContext {
			return RepeatReport{}, fmt.Errorf("attempt[%d] has a different manifest or execution context", i+1)
		}
		entries := resultEntryMap(set.Index)
		if len(entries) != len(baseEntries) {
			return RepeatReport{}, fmt.Errorf("attempt[%d] has different run identities", i+1)
		}
		for key := range baseEntries {
			if _, found := entries[key]; !found {
				return RepeatReport{}, fmt.Errorf("attempt[%d] has a missing run identity", i+1)
			}
		}
	}

	report := RepeatReport{
		SchemaVersion:       RepeatReportSchemaVersion,
		ArtifactType:        RepeatReportArtifactType,
		ManifestFingerprint: base.ManifestFingerprint,
		RequestedAttempts:   requested,
		CompletedAttempts:   len(sets),
		Complete:            len(sets) == requested,
		AttemptIndexes:      append([]string(nil), indexes...),
		Runs:                make([]RepeatRun, 0, len(baseEntries)),
	}
	keys := make([]string, 0, len(baseEntries))
	for key := range baseEntries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := baseEntries[key]
		run := RepeatRun{
			TargetID: entry.TargetID, DeploymentID: entry.DeploymentID,
			ClientID: entry.ClientID, AuthMode: entry.AuthMode,
			AllPass: true, Attempts: make([]RepeatAttempt, 0, len(sets)),
		}
		signatures := map[string]struct{}{}
		for i, set := range sets {
			current := resultEntryMap(set.Index)[key]
			evidence := evidenceForEntry(set, current)
			run.Attempts = append(run.Attempts, RepeatAttempt{Attempt: i + 1, Evidence: evidence})
			if current.Outcome != OutcomePass {
				run.AllPass = false
			}
			if current.Outcome == OutcomeError {
				signatures["execution_error"] = struct{}{}
				continue
			}
			value, ok := set.Artifacts[key]
			if !ok || len(value.Runs) != 1 {
				return RepeatReport{}, fmt.Errorf("attempt[%d] %s has no validated artifact", i+1, current.ClientID)
			}
			if err := ValidateResultArtifact(current, value); err != nil {
				return RepeatReport{}, fmt.Errorf("attempt[%d] %s invalid artifact: %w", i+1, current.ClientID, err)
			}
			observed := value.Runs[0]
			// Include the exact client version: a mid-run client upgrade
			// invalidates the assumption that these are identical trials.
			signatures[evidenceSignature(observed, current.Outcome)+"\x00"+observed.Client.Version] = struct{}{}
		}
		run.Unstable = len(signatures) > 1
		report.HasNonPass = report.HasNonPass || !run.AllPass
		report.HasUnstable = report.HasUnstable || run.Unstable
		report.Runs = append(report.Runs, run)
	}
	switch {
	case !report.Complete:
		report.Decision = RepeatDecisionIncomplete
	case report.HasNonPass && report.HasUnstable:
		report.Decision = RepeatDecisionNonPassUnstable
	case report.HasNonPass:
		report.Decision = RepeatDecisionNonPass
	case report.HasUnstable:
		report.Decision = RepeatDecisionUnstable
	default:
		report.Decision = RepeatDecisionClean
	}
	for i, ref := range indexes {
		if ref != fmt.Sprintf("attempt-%02d/index.json", i+1) {
			return RepeatReport{}, errors.New("repeat index reference does not match ordered attempt identity")
		}
	}
	return report, nil
}
