package compare

import (
	"fmt"
	"sort"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	ReportSchemaVersionV1 = 1
	ReportSchemaVersionV2 = 2
	ReportSchemaVersion   = ReportSchemaVersionV1
)

const (
	RunCompared   = "compared"
	RunMissingNew = "missing_new_evidence"
	RunNewOnly    = "new_only"

	RegressionPassToFail    = "PASS_TO_FAIL"
	RegressionPassToUnknown = "PASS_TO_UNKNOWN"
	RegressionPassToSkip    = "PASS_TO_SKIP"
	RegressionReasonChanged = "REASON_CODE_CHANGED"
	RegressionRunMissing    = "NEW_EVIDENCE_MISSING"
)

type Report struct {
	SchemaVersion int             `json:"schema_version"`
	HasRegression bool            `json:"has_regression"`
	Runs          []RunComparison `json:"runs"`
}

type RunComparison struct {
	State            string                    `json:"state"`
	Endpoint         artifact.EndpointIdentity `json:"endpoint"`
	ClientID         string                    `json:"client_id"`
	ClientProduct    string                    `json:"client_product"`
	OldClientVersion string                    `json:"old_client_version,omitempty"`
	NewClientVersion string                    `json:"new_client_version,omitempty"`
	AuthMode         string                    `json:"auth_mode"`
	Platform         artifact.Platform         `json:"platform"`
	StageChanges     []StageChange             `json:"stage_changes,omitempty"`
	Regression       bool                      `json:"regression"`
	RegressionKinds  []string                  `json:"regression_kinds,omitempty"`
}

type StageChange struct {
	Stage           interop.Stage      `json:"stage"`
	OldStatus       interop.Status     `json:"old_status"`
	NewStatus       interop.Status     `json:"new_status"`
	OldReasonCode   interop.ReasonCode `json:"old_reason_code,omitempty"`
	NewReasonCode   interop.ReasonCode `json:"new_reason_code,omitempty"`
	Regression      bool               `json:"regression"`
	RegressionKinds []string           `json:"regression_kinds,omitempty"`
}

// Artifacts compares two already-validated artifacts of the same schema.
// Cross-schema comparison is rejected so v1 and v2 pairing semantics are never
// mixed implicitly. Baseline runs missing from the new artifact are evidence-loss
// regressions. New-only runs are kept in the report but are not regressions.
func Artifacts(oldArtifact, newArtifact artifact.Artifact) (Report, error) {
	if oldArtifact.SchemaVersion != newArtifact.SchemaVersion {
		return Report{}, fmt.Errorf("artifact schema mismatch: old=%d new=%d; migrate or regenerate before comparison", oldArtifact.SchemaVersion, newArtifact.SchemaVersion)
	}
	oldRuns := make(map[string]artifact.Run, len(oldArtifact.Runs))
	newRuns := make(map[string]artifact.Run, len(newArtifact.Runs))
	keys := make([]string, 0, len(oldArtifact.Runs)+len(newArtifact.Runs))
	seen := make(map[string]bool, len(oldArtifact.Runs)+len(newArtifact.Runs))

	for _, run := range oldArtifact.Runs {
		key := artifact.ComparisonKeyForSchema(run, oldArtifact.SchemaVersion)
		oldRuns[key] = run
		if !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	for _, run := range newArtifact.Runs {
		key := artifact.ComparisonKeyForSchema(run, newArtifact.SchemaVersion)
		newRuns[key] = run
		if !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}

	// Artifact ordering is normally stable, but sorting makes comparison output
	// deterministic even when artifacts were assembled by another v1 producer.
	sort.Strings(keys)

	reportSchemaVersion := ReportSchemaVersionV1
	if oldArtifact.SchemaVersion == artifact.SchemaVersionV2 {
		reportSchemaVersion = ReportSchemaVersionV2
	}
	report := Report{SchemaVersion: reportSchemaVersion, Runs: make([]RunComparison, 0, len(keys))}
	for _, key := range keys {
		oldRun, hadOld := oldRuns[key]
		newRun, hadNew := newRuns[key]
		switch {
		case hadOld && hadNew:
			item := compareRun(oldRun, newRun)
			report.Runs = append(report.Runs, item)
			report.HasRegression = report.HasRegression || item.Regression
		case hadOld:
			item := RunComparison{
				State:            RunMissingNew,
				Endpoint:         oldRun.Endpoint,
				ClientID:         oldRun.Client.ID,
				ClientProduct:    oldRun.Client.Product,
				OldClientVersion: oldRun.Client.Version,
				AuthMode:         oldRun.AuthMode,
				Platform:         oldRun.Platform,
				Regression:       true,
				RegressionKinds:  []string{RegressionRunMissing},
			}
			report.Runs = append(report.Runs, item)
			report.HasRegression = true
		case hadNew:
			report.Runs = append(report.Runs, RunComparison{
				State:            RunNewOnly,
				Endpoint:         newRun.Endpoint,
				ClientID:         newRun.Client.ID,
				ClientProduct:    newRun.Client.Product,
				NewClientVersion: newRun.Client.Version,
				AuthMode:         newRun.AuthMode,
				Platform:         newRun.Platform,
			})
		}
	}
	return report, nil
}

func compareRun(oldRun, newRun artifact.Run) RunComparison {
	result := RunComparison{
		State:            RunCompared,
		Endpoint:         oldRun.Endpoint,
		ClientID:         oldRun.Client.ID,
		ClientProduct:    oldRun.Client.Product,
		OldClientVersion: oldRun.Client.Version,
		NewClientVersion: newRun.Client.Version,
		AuthMode:         oldRun.AuthMode,
		Platform:         oldRun.Platform,
	}

	for i := range oldRun.Stages {
		oldStage := oldRun.Stages[i]
		newStage := newRun.Stages[i]
		if oldStage.Status == newStage.Status && oldStage.ReasonCode == newStage.ReasonCode {
			continue
		}

		change := StageChange{
			Stage:         oldStage.Stage,
			OldStatus:     oldStage.Status,
			NewStatus:     newStage.Status,
			OldReasonCode: oldStage.ReasonCode,
			NewReasonCode: newStage.ReasonCode,
		}
		if oldStage.Status == interop.StatusPass {
			switch newStage.Status {
			case interop.StatusFail:
				change.RegressionKinds = append(change.RegressionKinds, RegressionPassToFail)
			case interop.StatusUnknown:
				change.RegressionKinds = append(change.RegressionKinds, RegressionPassToUnknown)
			case interop.StatusSkip:
				change.RegressionKinds = append(change.RegressionKinds, RegressionPassToSkip)
			}
		}
		// A reason-code change remains visible in the StageChange itself, but a
		// transition that recovers to PASS is an improvement, not a regression.
		if oldStage.ReasonCode != newStage.ReasonCode && newStage.Status != interop.StatusPass {
			change.RegressionKinds = append(change.RegressionKinds, RegressionReasonChanged)
		}
		change.Regression = len(change.RegressionKinds) > 0
		result.StageChanges = append(result.StageChanges, change)
		if change.Regression {
			result.Regression = true
			for _, kind := range change.RegressionKinds {
				result.RegressionKinds = appendUnique(result.RegressionKinds, kind)
			}
		}
	}
	return result
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
