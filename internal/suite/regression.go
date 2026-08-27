package suite

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	interopcompare "github.com/git-ksk/mcp-interop/internal/compare"
)

const (
	RegressionReportSchemaVersion = 1
	RegressionReportArtifactType  = "mcp-interop/suite-regression-report"

	DecisionClean                 = RegressionDecision("clean")
	DecisionRegression            = RegressionDecision("regression")
	DecisionUnstable              = RegressionDecision("unstable")
	DecisionRegressionAndUnstable = RegressionDecision("regression_and_unstable")

	AttemptCompared            = "compared"
	AttemptNewOnly             = "new_only"
	AttemptMissingEvidence     = "missing_evidence"
	AttemptExecutionError      = "execution_error"
	AttemptBaselineUnavailable = "baseline_unavailable"
	AttemptIdentityChanged     = "identity_changed"

	ProtocolEvidenceNotSerialized = "not_serialized_in_live_result_v2"
)

// RegressionDecision is the suite-level deterministic comparison decision.
type RegressionDecision string

// RegressionReport compares one baseline result set with every retained retry
// attempt. Attempts are never collapsed into a single latest result.
type RegressionReport struct {
	SchemaVersion               int                `json:"schema_version"`
	ArtifactType                string             `json:"artifact_type"`
	Decision                    RegressionDecision `json:"decision"`
	HasRegression               bool               `json:"has_regression"`
	HasUnstable                 bool               `json:"has_unstable"`
	BaselineManifestFingerprint string             `json:"baseline_manifest_fingerprint"`
	AttemptManifestFingerprint  string             `json:"attempt_manifest_fingerprint"`
	AttemptCount                int                `json:"attempt_count"`
	ProtocolEvidenceStatus      string             `json:"protocol_evidence_status"`
	Runs                        []RegressionRun    `json:"runs"`
}

// RegressionRun is one logical suite target/client/auth identity across the
// baseline and all current attempts.
type RegressionRun struct {
	TargetID        string              `json:"target_id"`
	DeploymentID    string              `json:"deployment_id"`
	ClientID        string              `json:"client_id"`
	AuthMode        AuthMode            `json:"auth_mode"`
	Baseline        *RunEvidence        `json:"baseline,omitempty"`
	Attempts        []AttemptComparison `json:"attempts"`
	Regression      bool                `json:"regression"`
	Unstable        bool                `json:"unstable"`
	RegressionKinds []string            `json:"regression_kinds,omitempty"`
}

// RunEvidence is the portable material evidence retained from one result-set
// entry. Endpoint URLs and human diagnostic messages are intentionally absent.
type RunEvidence struct {
	Outcome             RunOutcome             `json:"outcome"`
	ExitCode            int                    `json:"exit_code"`
	Artifact            string                 `json:"artifact,omitempty"`
	ClientVersion       string                 `json:"client_version,omitempty"`
	Platform            *artifact.Platform     `json:"platform,omitempty"`
	EndpointFingerprint string                 `json:"endpoint_fingerprint,omitempty"`
	Stages              []artifact.StageResult `json:"stages,omitempty"`
}

// AttemptComparison retains one current attempt plus its direct artifact
// comparison against the baseline when both sides have usable evidence.
type AttemptComparison struct {
	Attempt              int                          `json:"attempt"`
	State                string                       `json:"state"`
	Evidence             *RunEvidence                 `json:"evidence,omitempty"`
	ClientVersionChanged bool                         `json:"client_version_changed,omitempty"`
	StageChanges         []interopcompare.StageChange `json:"stage_changes,omitempty"`
	Regression           bool                         `json:"regression"`
	RegressionKinds      []string                     `json:"regression_kinds,omitempty"`
}

// CompareResultSets produces one evidence-derived report. All current attempts
// must come from the exact same validated manifest declaration so they are true
// retries rather than silently different test suites.
func CompareResultSets(baseline LoadedResultSet, attempts []LoadedResultSet) (RegressionReport, error) {
	if len(attempts) == 0 {
		return RegressionReport{}, errors.New("at least one current suite attempt is required")
	}
	attemptFingerprint := attempts[0].Index.ManifestFingerprint
	if baseline.Index.ManifestFingerprint != attemptFingerprint {
		return RegressionReport{}, errors.New("baseline manifest fingerprint differs from current attempts")
	}
	for i, attempt := range attempts {
		if attempt.Index.ManifestFingerprint != attemptFingerprint {
			return RegressionReport{}, fmt.Errorf("attempt[%d] manifest fingerprint differs from attempt[0]", i)
		}
		if attempt.Index.ExecutionContext != baseline.Index.ExecutionContext {
			return RegressionReport{}, fmt.Errorf("attempt[%d] execution context differs from baseline", i)
		}
	}

	baselineEntries := resultEntryMap(baseline.Index)
	attemptEntries := make([]map[string]ResultEntry, len(attempts))
	keySet := make(map[string]struct{}, len(baselineEntries)+len(attempts)*len(attempts[0].Index.Runs))
	for key := range baselineEntries {
		keySet[key] = struct{}{}
	}
	for i, attempt := range attempts {
		attemptEntries[i] = resultEntryMap(attempt.Index)
		for key := range attemptEntries[i] {
			keySet[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	report := RegressionReport{
		SchemaVersion:               RegressionReportSchemaVersion,
		ArtifactType:                RegressionReportArtifactType,
		BaselineManifestFingerprint: baseline.Index.ManifestFingerprint,
		AttemptManifestFingerprint:  attemptFingerprint,
		AttemptCount:                len(attempts),
		ProtocolEvidenceStatus:      ProtocolEvidenceNotSerialized,
		Runs:                        make([]RegressionRun, 0, len(keys)),
	}
	for _, key := range keys {
		baseEntry, hasBaseline := baselineEntries[key]
		identity := baseEntry
		if !hasBaseline {
			for i := range attemptEntries {
				if entry, ok := attemptEntries[i][key]; ok {
					identity = entry
					break
				}
			}
		}
		runReport := RegressionRun{
			TargetID:     identity.TargetID,
			DeploymentID: identity.DeploymentID,
			ClientID:     identity.ClientID,
			AuthMode:     identity.AuthMode,
			Attempts:     make([]AttemptComparison, 0, len(attempts)),
		}
		var baselineArtifact artifact.Artifact
		baselineUsable := false
		if hasBaseline {
			runReport.Baseline = evidenceForEntry(baseline, baseEntry)
			if baseEntry.Artifact != "" {
				baselineArtifact, baselineUsable = baseline.Artifacts[key]
			}
			if !baselineUsable {
				runReport.Unstable = true
			}
		}

		signatures := make(map[string]struct{}, len(attempts))
		for i, attempt := range attempts {
			entry, found := attemptEntries[i][key]
			comparison := AttemptComparison{Attempt: i + 1}
			switch {
			case !found:
				comparison.State = AttemptMissingEvidence
				comparison.Regression = hasBaseline
				if hasBaseline {
					comparison.RegressionKinds = []string{interopcompare.RegressionRunMissing}
				}
				runReport.Unstable = true
				signatures["missing"] = struct{}{}
			case entry.Outcome == OutcomeError:
				comparison.State = AttemptExecutionError
				comparison.Evidence = evidenceForEntry(attempt, entry)
				comparison.Regression = hasBaseline
				if hasBaseline {
					comparison.RegressionKinds = []string{interopcompare.RegressionRunMissing}
				}
				runReport.Unstable = true
				signatures["error"] = struct{}{}
			default:
				comparison.Evidence = evidenceForEntry(attempt, entry)
				currentArtifact, usable := attempt.Artifacts[key]
				if !usable {
					return RegressionReport{}, fmt.Errorf("attempt[%d] run %s/%s has no loaded artifact", i, entry.TargetID, entry.ClientID)
				}
				signatures[evidenceSignature(currentArtifact.Runs[0], entry.Outcome)] = struct{}{}
				if !hasBaseline {
					comparison.State = AttemptNewOnly
				} else if !baselineUsable {
					comparison.State = AttemptBaselineUnavailable
					runReport.Unstable = true
				} else {
					comparison.State = AttemptCompared
					direct, err := interopcompare.Artifacts(baselineArtifact, currentArtifact)
					if err != nil {
						return RegressionReport{}, fmt.Errorf("attempt[%d] compare %s/%s: %w", i, entry.TargetID, entry.ClientID, err)
					}
					comparison.Regression = direct.HasRegression
					comparison.RegressionKinds = directRegressionKinds(direct)
					comparison.StageChanges = directStageChanges(direct)
					if len(direct.Runs) != 1 || direct.Runs[0].State != interopcompare.RunCompared {
						comparison.State = AttemptIdentityChanged
					}
					if runReport.Baseline != nil && comparison.Evidence != nil {
						comparison.ClientVersionChanged = runReport.Baseline.ClientVersion != comparison.Evidence.ClientVersion
					}
				}
			}
			runReport.Attempts = append(runReport.Attempts, comparison)
			if comparison.Regression {
				runReport.Regression = true
				for _, kind := range comparison.RegressionKinds {
					runReport.RegressionKinds = appendUniqueString(runReport.RegressionKinds, kind)
				}
			}
		}
		if len(signatures) > 1 {
			runReport.Unstable = true
		}
		report.HasRegression = report.HasRegression || runReport.Regression
		report.HasUnstable = report.HasUnstable || runReport.Unstable
		report.Runs = append(report.Runs, runReport)
	}

	switch {
	case report.HasRegression && report.HasUnstable:
		report.Decision = DecisionRegressionAndUnstable
	case report.HasRegression:
		report.Decision = DecisionRegression
	case report.HasUnstable:
		report.Decision = DecisionUnstable
	default:
		report.Decision = DecisionClean
	}
	return report, nil
}

func resultEntryMap(index ResultIndex) map[string]ResultEntry {
	out := make(map[string]ResultEntry, len(index.Runs))
	for _, entry := range index.Runs {
		out[resultEntryKey(entry)] = entry
	}
	return out
}

func evidenceForEntry(set LoadedResultSet, entry ResultEntry) *RunEvidence {
	evidence := &RunEvidence{
		Outcome:  entry.Outcome,
		ExitCode: entry.ExitCode,
		Artifact: entry.Artifact,
	}
	if entry.Artifact == "" {
		return evidence
	}
	value, ok := set.Artifacts[resultEntryKey(entry)]
	if !ok || len(value.Runs) != 1 {
		return evidence
	}
	run := value.Runs[0]
	platform := run.Platform
	evidence.ClientVersion = run.Client.Version
	evidence.Platform = &platform
	evidence.EndpointFingerprint = run.Endpoint.Fingerprint
	evidence.Stages = append([]artifact.StageResult(nil), run.Stages...)
	return evidence
}

func evidenceSignature(run artifact.Run, outcome RunOutcome) string {
	parts := []string{string(outcome), run.Endpoint.Fingerprint, run.Platform.OS, run.Platform.Arch}
	for _, stage := range run.Stages {
		parts = append(parts, string(stage.Stage), string(stage.Status), string(stage.ReasonCode))
	}
	return strings.Join(parts, "\x00")
}

func directRegressionKinds(report interopcompare.Report) []string {
	var kinds []string
	for _, run := range report.Runs {
		for _, kind := range run.RegressionKinds {
			kinds = appendUniqueString(kinds, kind)
		}
	}
	return kinds
}

func directStageChanges(report interopcompare.Report) []interopcompare.StageChange {
	var changes []interopcompare.StageChange
	for _, run := range report.Runs {
		if run.State == interopcompare.RunCompared {
			changes = append(changes, run.StageChanges...)
		}
	}
	return changes
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
