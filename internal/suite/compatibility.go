package suite

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	interopcompare "github.com/git-ksk/mcp-interop/internal/compare"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	CompatibilityEnvelopeSchemaVersion = 1
	CompatibilityEnvelopeArtifactType  = "mcp-interop/compatibility-envelope"

	CompatibilityTested      = CompatibilityState("tested")
	CompatibilityUntested    = CompatibilityState("untested")
	CompatibilityStale       = CompatibilityState("stale")
	CompatibilityKnownBroken = CompatibilityState("known_broken")
	CompatibilityRegressed   = CompatibilityState("regressed")
	CompatibilityUnknown     = CompatibilityState("unknown")

	CompatibilitySourceBaseline = "baseline"
	CompatibilitySourceAttempt  = "attempt"

	CompatibilityGapExecutionError      = "execution_error"
	CompatibilityGapMissingRunEvidence  = "missing_run_evidence"
	CompatibilityGapNonRealClient       = "non_real_client_evidence"
	CompatibilityGapClientVersionAbsent = "client_version_unavailable"

	CompatibilityStaleByAge           = "age_limit_exceeded"
	CompatibilityStaleByVersionChange = "later_client_version_observed"
)

// CompatibilityState classifies one exact client-version/platform/auth/
// deployment observation. Untested is query-derived and never fabricates a
// point that is absent from the observed envelope.
type CompatibilityState string

// CompatibilityStalePolicy makes staleness an explicit input. A zero max age
// disables age-based staleness. Version-change staleness compares observation
// times only; it never parses or orders version strings semantically.
type CompatibilityStalePolicy struct {
	MaxAgeSeconds              int64 `json:"max_age_seconds,omitempty"`
	StaleOnClientVersionChange bool  `json:"stale_on_client_version_change,omitempty"`
}

// CompatibilityEnvelope is a deterministic set of exact observed points for
// one manifest/execution context. It contains no inferred version range.
type CompatibilityEnvelope struct {
	SchemaVersion       int                        `json:"schema_version"`
	ArtifactType        string                     `json:"artifact_type"`
	ManifestFingerprint string                     `json:"manifest_fingerprint"`
	ExecutionContext    ExecutionContext           `json:"execution_context"`
	EvaluatedAt         time.Time                  `json:"evaluated_at"`
	StalePolicy         CompatibilityStalePolicy   `json:"stale_policy"`
	BaselineFingerprint string                     `json:"baseline_fingerprint,omitempty"`
	Points              []CompatibilityPoint       `json:"points"`
	EvidenceGaps        []CompatibilityEvidenceGap `json:"evidence_gaps,omitempty"`
}

// CompatibilityPoint is one exact observed client-version point. Multiple
// observations of the same exact point are retained rather than collapsed.
type CompatibilityPoint struct {
	TargetID                   string                     `json:"target_id"`
	DeploymentID               string                     `json:"deployment_id"`
	DeploymentFingerprint      string                     `json:"deployment_fingerprint"`
	ClientID                   string                     `json:"client_id"`
	ClientVersion              string                     `json:"client_version"`
	Platform                   artifact.Platform          `json:"platform"`
	AuthMode                   AuthMode                   `json:"auth_mode"`
	State                      CompatibilityState         `json:"state"`
	Unstable                   bool                       `json:"unstable"`
	LastObservedAt             time.Time                  `json:"last_observed_at"`
	ContextLastObservedVersion string                     `json:"context_last_observed_version"`
	ContextLastObservedAt      time.Time                  `json:"context_last_observed_at"`
	StaleReasons               []string                   `json:"stale_reasons,omitempty"`
	Observations               []CompatibilityObservation `json:"observations"`
}

// CompatibilityObservation preserves the material, secret-safe evidence for
// one exact point observation and its relationship to the accepted baseline.
type CompatibilityObservation struct {
	Source             string                      `json:"source"`
	Attempt            int                         `json:"attempt,omitempty"`
	ExecutedAt         time.Time                   `json:"executed_at"`
	Outcome            RunOutcome                  `json:"outcome"`
	ResultSetDigest    string                      `json:"result_set_digest"`
	Runtime            artifact.Runtime            `json:"runtime"`
	EvidenceProvenance artifact.EvidenceProvenance `json:"evidence_provenance"`
	Stages             []artifact.StageResult      `json:"stages"`
	Regression         bool                        `json:"regression"`
	RegressionKinds    []string                    `json:"regression_kinds,omitempty"`
}

// CompatibilityEvidenceGap records a suite attempt that cannot become an exact
// observed point, for example because no artifact or exact real-client version
// exists. Such a gap can carry regression evidence without inventing a version.
type CompatibilityEvidenceGap struct {
	TargetID           string                       `json:"target_id"`
	DeploymentID       string                       `json:"deployment_id"`
	ClientID           string                       `json:"client_id"`
	AuthMode           AuthMode                     `json:"auth_mode"`
	Source             string                       `json:"source"`
	Attempt            int                          `json:"attempt,omitempty"`
	Kind               string                       `json:"kind"`
	ResultSetDigest    string                       `json:"result_set_digest"`
	ClientVersion      string                       `json:"client_version,omitempty"`
	Platform           *artifact.Platform           `json:"platform,omitempty"`
	ExecutedAt         *time.Time                   `json:"executed_at,omitempty"`
	EvidenceProvenance *artifact.EvidenceProvenance `json:"evidence_provenance,omitempty"`
	Regression         bool                         `json:"regression"`
	RegressionKinds    []string                     `json:"regression_kinds,omitempty"`
}

// CompatibilityQuery asks for one exact client version on a platform within a
// known target/deployment/client/auth context. It never accepts a version range.
type CompatibilityQuery struct {
	TargetID      string            `json:"target_id"`
	DeploymentID  string            `json:"deployment_id"`
	ClientID      string            `json:"client_id"`
	ClientVersion string            `json:"client_version"`
	Platform      artifact.Platform `json:"platform"`
	AuthMode      AuthMode          `json:"auth_mode"`
}

// CompatibilityClassification is the exact-query result. When State is
// untested, Point is nil and ObservedVersions lists exact observations only.
type CompatibilityClassification struct {
	Query                      CompatibilityQuery         `json:"query"`
	State                      CompatibilityState         `json:"state"`
	Point                      *CompatibilityPoint        `json:"point,omitempty"`
	ObservedVersions           []string                   `json:"observed_versions,omitempty"`
	ContextLastObservedVersion string                     `json:"context_last_observed_version,omitempty"`
	ContextLastObservedAt      *time.Time                 `json:"context_last_observed_at,omitempty"`
	EvidenceGaps               []CompatibilityEvidenceGap `json:"evidence_gaps,omitempty"`
}

type compatibilityPointBuilder struct {
	point      CompatibilityPoint
	signatures map[string]struct{}
}

type compatibilityContextLast struct {
	version string
	at      time.Time
}

// BuildCompatibilityEnvelope aggregates exact observed points from an optional
// accepted baseline and one or more current/historical suite result sets.
// Manifest, execution-context, logical-run, and deployment identity mismatches
// fail closed. Platform differences remain distinct observed points.
func BuildCompatibilityEnvelope(
	baseline *LoadedBaseline,
	observations []LoadedResultSet,
	policy CompatibilityStalePolicy,
	evaluatedAt time.Time,
) (CompatibilityEnvelope, error) {
	if err := validateCompatibilityPolicy(policy); err != nil {
		return CompatibilityEnvelope{}, err
	}
	if evaluatedAt.IsZero() {
		return CompatibilityEnvelope{}, errors.New("compatibility evaluated_at is required")
	}
	evaluatedAt = evaluatedAt.UTC()
	if baseline == nil && len(observations) == 0 {
		return CompatibilityEnvelope{}, errors.New("compatibility envelope requires a baseline or observed result set")
	}

	var manifestFingerprint string
	var executionContext ExecutionContext
	var baselineFingerprint string
	var expectedIndex ResultIndex
	var baselineSet *LoadedResultSet

	if baseline != nil {
		if err := validateLoadedBaselineForCompatibility(*baseline); err != nil {
			return CompatibilityEnvelope{}, fmt.Errorf("invalid compatibility baseline: %w", err)
		}
		manifestFingerprint = baseline.Descriptor.ManifestFingerprint
		executionContext = baseline.Descriptor.ExecutionContext
		expectedIndex = baseline.ResultSet.Index
		baselineSet = &baseline.ResultSet
		var err error
		baselineFingerprint, err = BaselineFingerprint(baseline.Descriptor)
		if err != nil {
			return CompatibilityEnvelope{}, err
		}
	} else {
		if err := validateCompatibilityResultSet(observations[0]); err != nil {
			return CompatibilityEnvelope{}, fmt.Errorf("observation[0]: %w", err)
		}
		manifestFingerprint = observations[0].Index.ManifestFingerprint
		executionContext = observations[0].Index.ExecutionContext
		expectedIndex = observations[0].Index
	}

	expectedEntries := resultEntryMap(expectedIndex)
	deploymentFingerprints := make(map[string]string, len(expectedEntries))
	builders := make(map[string]*compatibilityPointBuilder)
	gaps := make([]CompatibilityEvidenceGap, 0)

	if baselineSet != nil {
		if err := addCompatibilityResultSet(
			builders,
			&gaps,
			deploymentFingerprints,
			expectedEntries,
			nil,
			*baselineSet,
			CompatibilitySourceBaseline,
			0,
		); err != nil {
			return CompatibilityEnvelope{}, fmt.Errorf("baseline observations: %w", err)
		}
	}

	for i, set := range observations {
		if err := validateCompatibilityResultSet(set); err != nil {
			return CompatibilityEnvelope{}, fmt.Errorf("observation[%d]: %w", i, err)
		}
		if set.Index.ManifestFingerprint != manifestFingerprint {
			return CompatibilityEnvelope{}, fmt.Errorf("observation[%d] manifest fingerprint differs from compatibility context", i)
		}
		if set.Index.ExecutionContext != executionContext {
			return CompatibilityEnvelope{}, fmt.Errorf("observation[%d] execution context differs from compatibility context", i)
		}
		if err := validateCompatibilityRunSet(expectedEntries, set.Index); err != nil {
			return CompatibilityEnvelope{}, fmt.Errorf("observation[%d] logical run identity differs: %w", i, err)
		}
		if err := addCompatibilityResultSet(
			builders,
			&gaps,
			deploymentFingerprints,
			expectedEntries,
			baselineSet,
			set,
			CompatibilitySourceAttempt,
			i+1,
		); err != nil {
			return CompatibilityEnvelope{}, fmt.Errorf("observation[%d]: %w", i, err)
		}
	}

	points := make([]CompatibilityPoint, 0, len(builders))
	for _, builder := range builders {
		sortCompatibilityObservations(builder.point.Observations)
		builder.point.LastObservedAt = builder.point.Observations[len(builder.point.Observations)-1].ExecutedAt
		builder.point.Unstable = len(builder.signatures) > 1
		builder.point.State = baseCompatibilityState(builder.point, builder.signatures)
		points = append(points, builder.point)
	}
	sortCompatibilityPoints(points)

	contextLast := compatibilityContextLastObserved(points)
	for i := range points {
		contextKey := compatibilityPointContextKey(points[i])
		last := contextLast[contextKey]
		points[i].ContextLastObservedVersion = last.version
		points[i].ContextLastObservedAt = last.at
		if points[i].State != CompatibilityTested {
			continue
		}
		if policy.MaxAgeSeconds > 0 && evaluatedAt.After(points[i].LastObservedAt) {
			age := evaluatedAt.Sub(points[i].LastObservedAt)
			if age > time.Duration(policy.MaxAgeSeconds)*time.Second {
				points[i].StaleReasons = append(points[i].StaleReasons, CompatibilityStaleByAge)
			}
		}
		if policy.StaleOnClientVersionChange &&
			last.version != "" &&
			last.version != points[i].ClientVersion &&
			last.at.After(points[i].LastObservedAt) {
			points[i].StaleReasons = append(points[i].StaleReasons, CompatibilityStaleByVersionChange)
		}
		if len(points[i].StaleReasons) > 0 {
			points[i].State = CompatibilityStale
		}
	}

	sortCompatibilityGaps(gaps)
	envelope := CompatibilityEnvelope{
		SchemaVersion:       CompatibilityEnvelopeSchemaVersion,
		ArtifactType:        CompatibilityEnvelopeArtifactType,
		ManifestFingerprint: manifestFingerprint,
		ExecutionContext:    executionContext,
		EvaluatedAt:         evaluatedAt,
		StalePolicy:         policy,
		BaselineFingerprint: baselineFingerprint,
		Points:              points,
		EvidenceGaps:        gaps,
	}
	if err := ValidateCompatibilityEnvelope(envelope); err != nil {
		return CompatibilityEnvelope{}, err
	}
	return envelope, nil
}

// ClassifyCompatibilityExact returns only an exact observed-point state. If the
// requested version/platform point is absent inside a known logical context,
// the result is untested; nearby versions are listed but never generalized.
func ClassifyCompatibilityExact(
	envelope CompatibilityEnvelope,
	query CompatibilityQuery,
) (CompatibilityClassification, error) {
	if err := ValidateCompatibilityEnvelope(envelope); err != nil {
		return CompatibilityClassification{}, err
	}
	if err := validateCompatibilityQuery(query); err != nil {
		return CompatibilityClassification{}, err
	}

	staticContextKnown := false
	var exact *CompatibilityPoint
	platformPoints := make([]CompatibilityPoint, 0)
	for i := range envelope.Points {
		point := envelope.Points[i]
		if !compatibilityStaticContextMatches(point, query) {
			continue
		}
		staticContextKnown = true
		if point.Platform != query.Platform {
			continue
		}
		platformPoints = append(platformPoints, point)
		if point.ClientVersion == query.ClientVersion {
			copyPoint := point
			exact = &copyPoint
		}
	}
	matchingGaps := make([]CompatibilityEvidenceGap, 0)
	for _, gap := range envelope.EvidenceGaps {
		if compatibilityGapStaticContextMatches(gap, query) {
			staticContextKnown = true
			if gap.Platform == nil || *gap.Platform == query.Platform {
				matchingGaps = append(matchingGaps, gap)
			}
		}
	}
	if !staticContextKnown {
		return CompatibilityClassification{}, errors.New("compatibility query target/deployment/client/auth context is not present in envelope")
	}

	result := CompatibilityClassification{
		Query:        query,
		EvidenceGaps: matchingGaps,
	}
	if exact != nil {
		result.State = exact.State
		result.Point = exact
		return result, nil
	}

	result.State = CompatibilityUntested
	seenVersions := make(map[string]struct{}, len(platformPoints))
	var last *CompatibilityPoint
	for i := range platformPoints {
		point := platformPoints[i]
		seenVersions[point.ClientVersion] = struct{}{}
		if last == nil || point.LastObservedAt.After(last.LastObservedAt) ||
			(point.LastObservedAt.Equal(last.LastObservedAt) && point.ClientVersion > last.ClientVersion) {
			candidate := point
			last = &candidate
		}
	}
	for version := range seenVersions {
		result.ObservedVersions = append(result.ObservedVersions, version)
	}
	sort.Strings(result.ObservedVersions)
	if last != nil {
		result.ContextLastObservedVersion = last.ClientVersion
		at := last.LastObservedAt
		result.ContextLastObservedAt = &at
	}
	return result, nil
}

// ValidateCompatibilityEnvelope checks the public machine-readable envelope
// structure and deterministic ordering. It does not infer missing evidence.
func ValidateCompatibilityEnvelope(envelope CompatibilityEnvelope) error {
	if envelope.SchemaVersion != CompatibilityEnvelopeSchemaVersion {
		return fmt.Errorf("unsupported compatibility schema_version %d", envelope.SchemaVersion)
	}
	if envelope.ArtifactType != CompatibilityEnvelopeArtifactType {
		return fmt.Errorf("unsupported compatibility artifact_type %q", envelope.ArtifactType)
	}
	if err := validateSHA256Fingerprint("manifest_fingerprint", envelope.ManifestFingerprint); err != nil {
		return err
	}
	if envelope.ExecutionContext != ExecutionTrusted {
		return errors.New("compatibility envelope currently requires trusted_real_client execution_context")
	}
	if envelope.EvaluatedAt.IsZero() {
		return errors.New("compatibility evaluated_at is required")
	}
	if _, offset := envelope.EvaluatedAt.Zone(); offset != 0 {
		return errors.New("compatibility evaluated_at must use UTC")
	}
	if err := validateCompatibilityPolicy(envelope.StalePolicy); err != nil {
		return err
	}
	if envelope.BaselineFingerprint != "" {
		if err := validateSHA256Fingerprint("baseline_fingerprint", envelope.BaselineFingerprint); err != nil {
			return err
		}
	}

	previousPoint := ""
	seenPoints := make(map[string]struct{}, len(envelope.Points))
	for i := range envelope.Points {
		point := envelope.Points[i]
		if err := validateCompatibilityPoint(point); err != nil {
			return fmt.Errorf("points[%d]: %w", i, err)
		}
		key := compatibilityPointKey(point)
		if previousPoint != "" && key < previousPoint {
			return fmt.Errorf("points[%d]: points are not in deterministic order", i)
		}
		if _, duplicate := seenPoints[key]; duplicate {
			return fmt.Errorf("points[%d]: duplicate exact compatibility point", i)
		}
		seenPoints[key] = struct{}{}
		previousPoint = key
	}

	previousGap := ""
	for i := range envelope.EvidenceGaps {
		gap := envelope.EvidenceGaps[i]
		if err := validateCompatibilityGap(gap); err != nil {
			return fmt.Errorf("evidence_gaps[%d]: %w", i, err)
		}
		key := compatibilityGapSortKey(gap)
		if previousGap != "" && key < previousGap {
			return fmt.Errorf("evidence_gaps[%d]: gaps are not in deterministic order", i)
		}
		previousGap = key
	}
	return nil
}

func addCompatibilityResultSet(
	builders map[string]*compatibilityPointBuilder,
	gaps *[]CompatibilityEvidenceGap,
	deploymentFingerprints map[string]string,
	expectedEntries map[string]ResultEntry,
	baseline *LoadedResultSet,
	set LoadedResultSet,
	source string,
	attempt int,
) error {
	digest, err := ResultSetDigest(set)
	if err != nil {
		return err
	}
	entries := resultEntryMap(set.Index)
	if source == CompatibilitySourceAttempt {
		for key, expected := range expectedEntries {
			if _, ok := entries[key]; ok {
				continue
			}
			gap := compatibilityGapForEntry(expected, source, attempt, CompatibilityGapMissingRunEvidence, digest)
			if baseline != nil {
				if _, ok := resultEntryMap(baseline.Index)[key]; ok {
					gap.Regression = true
					gap.RegressionKinds = []string{interopcompare.RegressionRunMissing}
				}
			}
			*gaps = append(*gaps, gap)
		}
	}

	var baselineEntries map[string]ResultEntry
	if baseline != nil {
		baselineEntries = resultEntryMap(baseline.Index)
	}
	for _, entry := range set.Index.Runs {
		key := resultEntryKey(entry)
		if _, ok := expectedEntries[key]; !ok {
			return fmt.Errorf("run %s/%s/%s is not declared by the compatibility context", entry.TargetID, entry.ClientID, entry.AuthMode)
		}
		if entry.Outcome == OutcomeError || entry.Artifact == "" {
			gap := compatibilityGapForEntry(entry, source, attempt, CompatibilityGapExecutionError, digest)
			if source == CompatibilitySourceAttempt && baseline != nil {
				if _, ok := baselineEntries[key]; ok {
					gap.Regression = true
					gap.RegressionKinds = []string{interopcompare.RegressionRunMissing}
				}
			}
			*gaps = append(*gaps, gap)
			continue
		}

		value := set.Artifacts[key]
		run := value.Runs[0]
		if prior, ok := deploymentFingerprints[key]; ok && prior != run.Endpoint.Fingerprint {
			return fmt.Errorf("run %s/%s deployment fingerprint differs from prior observations", entry.TargetID, entry.ClientID)
		}
		deploymentFingerprints[key] = run.Endpoint.Fingerprint

		regression, kinds, err := compatibilityRegressionAgainstBaseline(baseline, baselineEntries, key, value)
		if err != nil {
			return err
		}
		if run.EvidenceProvenance.Kind != artifact.ProvenanceRealClientAdapter {
			gap := compatibilityGapForArtifact(entry, source, attempt, CompatibilityGapNonRealClient, digest, run)
			gap.Regression = regression
			gap.RegressionKinds = kinds
			*gaps = append(*gaps, gap)
			continue
		}
		if run.Client.Version == "" {
			gap := compatibilityGapForArtifact(entry, source, attempt, CompatibilityGapClientVersionAbsent, digest, run)
			gap.Regression = regression
			gap.RegressionKinds = kinds
			*gaps = append(*gaps, gap)
			continue
		}

		point := CompatibilityPoint{
			TargetID:              entry.TargetID,
			DeploymentID:          entry.DeploymentID,
			DeploymentFingerprint: run.Endpoint.Fingerprint,
			ClientID:              entry.ClientID,
			ClientVersion:         run.Client.Version,
			Platform:              run.Platform,
			AuthMode:              entry.AuthMode,
		}
		pointKey := compatibilityPointKey(point)
		builder, ok := builders[pointKey]
		if !ok {
			builder = &compatibilityPointBuilder{
				point:      point,
				signatures: make(map[string]struct{}),
			}
			builders[pointKey] = builder
		}
		observation := CompatibilityObservation{
			Source:             source,
			Attempt:            attempt,
			ExecutedAt:         run.ExecutedAt,
			Outcome:            entry.Outcome,
			ResultSetDigest:    digest,
			Runtime:            run.Runtime,
			EvidenceProvenance: run.EvidenceProvenance,
			Stages:             append([]artifact.StageResult(nil), run.Stages...),
			Regression:         regression,
			RegressionKinds:    append([]string(nil), kinds...),
		}
		builder.point.Observations = append(builder.point.Observations, observation)
		builder.signatures[evidenceSignature(run, entry.Outcome)] = struct{}{}
	}
	return nil
}

func compatibilityRegressionAgainstBaseline(
	baseline *LoadedResultSet,
	baselineEntries map[string]ResultEntry,
	key string,
	current artifact.Artifact,
) (bool, []string, error) {
	if baseline == nil {
		return false, nil, nil
	}
	baseEntry, ok := baselineEntries[key]
	if !ok || baseEntry.Artifact == "" {
		return false, nil, nil
	}
	baseArtifact, ok := baseline.Artifacts[key]
	if !ok || len(baseArtifact.Runs) != 1 || len(current.Runs) != 1 {
		return false, nil, nil
	}
	baseRun := baseArtifact.Runs[0]
	currentRun := current.Runs[0]
	if baseRun.Endpoint.Fingerprint != currentRun.Endpoint.Fingerprint {
		return false, nil, errors.New("deployment fingerprint differs from accepted baseline")
	}
	if baseRun.Platform != currentRun.Platform {
		return false, nil, nil
	}
	direct, err := interopcompare.Artifacts(baseArtifact, current)
	if err != nil {
		return false, nil, err
	}
	return direct.HasRegression, directRegressionKinds(direct), nil
}

func baseCompatibilityState(point CompatibilityPoint, signatures map[string]struct{}) CompatibilityState {
	for _, observation := range point.Observations {
		if observation.Regression {
			return CompatibilityRegressed
		}
	}
	if len(signatures) > 1 {
		return CompatibilityUnknown
	}
	if compatibilityObservationsAllPass(point.Observations) {
		return CompatibilityTested
	}
	if compatibilityObservationsContainFailure(point.Observations) {
		return CompatibilityKnownBroken
	}
	return CompatibilityUnknown
}

func compatibilityObservationEvidenceSignature(observation CompatibilityObservation) string {
	parts := []string{string(observation.Outcome)}
	for _, stage := range observation.Stages {
		parts = append(parts, string(stage.Stage), string(stage.Status), string(stage.ReasonCode))
	}
	return strings.Join(parts, "\x00")
}

func compatibilityObservationsAllPass(observations []CompatibilityObservation) bool {
	if len(observations) == 0 {
		return false
	}
	for _, observation := range observations {
		if observation.Outcome != OutcomePass {
			return false
		}
		for _, stage := range observation.Stages {
			if stage.Status != interop.StatusPass {
				return false
			}
		}
	}
	return true
}

func compatibilityObservationsContainFailure(observations []CompatibilityObservation) bool {
	for _, observation := range observations {
		for _, stage := range observation.Stages {
			if stage.Status == interop.StatusFail {
				return true
			}
		}
	}
	return false
}

func validateCompatibilityResultSet(set LoadedResultSet) error {
	if err := ValidateResultIndex(set.Index); err != nil {
		return err
	}
	for _, entry := range set.Index.Runs {
		if entry.Outcome == OutcomeError {
			if entry.Artifact != "" {
				return fmt.Errorf("run %s/%s: error outcome unexpectedly references an artifact", entry.TargetID, entry.ClientID)
			}
			continue
		}
		value, ok := set.Artifacts[resultEntryKey(entry)]
		if !ok {
			return fmt.Errorf("run %s/%s: referenced artifact is not loaded", entry.TargetID, entry.ClientID)
		}
		if err := artifact.ValidateArtifact(value); err != nil {
			return fmt.Errorf("run %s/%s: invalid artifact: %w", entry.TargetID, entry.ClientID, err)
		}
		if err := ValidateResultArtifact(entry, value); err != nil {
			return fmt.Errorf("run %s/%s: artifact does not match index: %w", entry.TargetID, entry.ClientID, err)
		}
	}
	return nil
}

func validateLoadedBaselineForCompatibility(baseline LoadedBaseline) error {
	if err := ValidateBaseline(baseline.Descriptor); err != nil {
		return err
	}
	if err := validateBaselineSource(baseline.ResultSet); err != nil {
		return err
	}
	if baseline.Descriptor.ManifestFingerprint != baseline.ResultSet.Index.ManifestFingerprint {
		return errors.New("baseline manifest fingerprint does not match result set")
	}
	if baseline.Descriptor.ExecutionContext != baseline.ResultSet.Index.ExecutionContext {
		return errors.New("baseline execution context does not match result set")
	}
	digest, err := ResultSetDigest(baseline.ResultSet)
	if err != nil {
		return err
	}
	if digest != baseline.Descriptor.ResultSetDigest {
		return errors.New("baseline result set digest mismatch")
	}
	return nil
}

func validateCompatibilityRunSet(expected map[string]ResultEntry, index ResultIndex) error {
	current := resultEntryMap(index)
	for key, entry := range current {
		if _, ok := expected[key]; !ok {
			return fmt.Errorf("unexpected run %s/%s/%s", entry.TargetID, entry.ClientID, entry.AuthMode)
		}
	}
	return nil
}

func validateCompatibilityPolicy(policy CompatibilityStalePolicy) error {
	if policy.MaxAgeSeconds < 0 {
		return errors.New("compatibility stale max_age_seconds must not be negative")
	}
	return nil
}

func validateCompatibilityQuery(query CompatibilityQuery) error {
	if err := validateTargetID(query.TargetID); err != nil {
		return fmt.Errorf("target_id: %w", err)
	}
	if err := artifact.ValidateDeploymentID(query.DeploymentID); err != nil {
		return fmt.Errorf("deployment_id: %w", err)
	}
	if err := validateClientSelection(ExecutionTrusted, ClientSelection{ID: query.ClientID, Auth: query.AuthMode}); err != nil {
		return err
	}
	if strings.TrimSpace(query.ClientVersion) == "" {
		return errors.New("client_version is required for exact compatibility classification")
	}
	if query.Platform.OS == "" || query.Platform.Arch == "" {
		return errors.New("platform os and arch are required")
	}
	return nil
}

func validateCompatibilityPoint(point CompatibilityPoint) error {
	if err := validateCompatibilityQuery(CompatibilityQuery{
		TargetID:      point.TargetID,
		DeploymentID:  point.DeploymentID,
		ClientID:      point.ClientID,
		ClientVersion: point.ClientVersion,
		Platform:      point.Platform,
		AuthMode:      point.AuthMode,
	}); err != nil {
		return err
	}
	if err := validateSHA256Fingerprint("deployment_fingerprint", point.DeploymentFingerprint); err != nil {
		return err
	}
	switch point.State {
	case CompatibilityTested, CompatibilityStale, CompatibilityKnownBroken, CompatibilityRegressed, CompatibilityUnknown:
	case CompatibilityUntested:
		return errors.New("untested must not be serialized as an observed compatibility point")
	default:
		return fmt.Errorf("unsupported compatibility state %q", point.State)
	}
	if point.LastObservedAt.IsZero() || point.ContextLastObservedAt.IsZero() {
		return errors.New("last observed timestamps are required")
	}
	if _, offset := point.LastObservedAt.Zone(); offset != 0 {
		return errors.New("last_observed_at must use UTC")
	}
	if _, offset := point.ContextLastObservedAt.Zone(); offset != 0 {
		return errors.New("context_last_observed_at must use UTC")
	}
	if point.ContextLastObservedVersion == "" {
		return errors.New("context_last_observed_version is required")
	}
	if len(point.Observations) == 0 {
		return errors.New("observed compatibility point requires evidence")
	}
	previousObservation := ""
	for i, observation := range point.Observations {
		if err := validateCompatibilityObservation(observation); err != nil {
			return fmt.Errorf("observations[%d]: %w", i, err)
		}
		key := compatibilityObservationSortKey(observation)
		if previousObservation != "" && key < previousObservation {
			return fmt.Errorf("observations[%d]: observations are not in deterministic order", i)
		}
		previousObservation = key
	}
	signatures := make(map[string]struct{}, len(point.Observations))
	for _, observation := range point.Observations {
		signatures[compatibilityObservationEvidenceSignature(observation)] = struct{}{}
	}
	wantUnstable := len(signatures) > 1
	if point.Unstable != wantUnstable {
		return fmt.Errorf("unstable=%t does not match retained observation evidence", point.Unstable)
	}
	wantBaseState := baseCompatibilityState(point, signatures)
	if point.State == CompatibilityStale {
		if wantBaseState != CompatibilityTested {
			return errors.New("stale point must otherwise be tested evidence")
		}
		if len(point.StaleReasons) == 0 {
			return errors.New("stale point requires a stale reason")
		}
	} else {
		if point.State != wantBaseState {
			return fmt.Errorf("compatibility state %q does not match retained observation evidence %q", point.State, wantBaseState)
		}
		if len(point.StaleReasons) != 0 {
			return errors.New("stale reasons are only valid for stale points")
		}
	}
	seenStaleReasons := make(map[string]struct{}, len(point.StaleReasons))
	for _, reason := range point.StaleReasons {
		switch reason {
		case CompatibilityStaleByAge, CompatibilityStaleByVersionChange:
		default:
			return fmt.Errorf("unsupported stale reason %q", reason)
		}
		if _, duplicate := seenStaleReasons[reason]; duplicate {
			return fmt.Errorf("duplicate stale reason %q", reason)
		}
		seenStaleReasons[reason] = struct{}{}
	}
	return nil
}

func validateCompatibilityObservation(observation CompatibilityObservation) error {
	switch observation.Source {
	case CompatibilitySourceBaseline:
		if observation.Attempt != 0 {
			return errors.New("baseline observation must not have attempt number")
		}
	case CompatibilitySourceAttempt:
		if observation.Attempt <= 0 {
			return errors.New("attempt observation requires a positive attempt number")
		}
	default:
		return fmt.Errorf("unsupported compatibility observation source %q", observation.Source)
	}
	if observation.ExecutedAt.IsZero() {
		return errors.New("executed_at is required")
	}
	if _, offset := observation.ExecutedAt.Zone(); offset != 0 {
		return errors.New("executed_at must use UTC")
	}
	if err := validateSHA256Fingerprint("result_set_digest", observation.ResultSetDigest); err != nil {
		return err
	}
	if observation.Runtime.MCPInteropVersion == "" || observation.Runtime.MCPInteropCommit == "" || observation.Runtime.GoVersion == "" {
		return errors.New("runtime version, commit, and go_version are required")
	}
	if observation.EvidenceProvenance.Kind != artifact.ProvenanceRealClientAdapter || observation.EvidenceProvenance.AdapterID == "" {
		return errors.New("compatibility point observation requires real-client adapter provenance")
	}
	if len(observation.Stages) != len(interop.OrderedStages) {
		return fmt.Errorf("stages must contain exactly %d entries", len(interop.OrderedStages))
	}
	for i, expected := range interop.OrderedStages {
		if observation.Stages[i].Stage != expected {
			return fmt.Errorf("stages[%d] must be %q", i, expected)
		}
	}
	if observation.Regression && len(observation.RegressionKinds) == 0 {
		return errors.New("regression observation requires regression kinds")
	}
	if !observation.Regression && len(observation.RegressionKinds) != 0 {
		return errors.New("non-regression observation must not carry regression kinds")
	}
	return nil
}

func validateCompatibilityGap(gap CompatibilityEvidenceGap) error {
	if err := validateTargetID(gap.TargetID); err != nil {
		return fmt.Errorf("target_id: %w", err)
	}
	if err := artifact.ValidateDeploymentID(gap.DeploymentID); err != nil {
		return fmt.Errorf("deployment_id: %w", err)
	}
	if err := validateClientSelection(ExecutionTrusted, ClientSelection{ID: gap.ClientID, Auth: gap.AuthMode}); err != nil {
		return err
	}
	switch gap.Source {
	case CompatibilitySourceBaseline:
		if gap.Attempt != 0 {
			return errors.New("baseline gap must not have attempt number")
		}
	case CompatibilitySourceAttempt:
		if gap.Attempt <= 0 {
			return errors.New("attempt gap requires a positive attempt number")
		}
	default:
		return fmt.Errorf("unsupported compatibility gap source %q", gap.Source)
	}
	switch gap.Kind {
	case CompatibilityGapExecutionError,
		CompatibilityGapMissingRunEvidence,
		CompatibilityGapNonRealClient,
		CompatibilityGapClientVersionAbsent:
	default:
		return fmt.Errorf("unsupported compatibility gap kind %q", gap.Kind)
	}
	if err := validateSHA256Fingerprint("result_set_digest", gap.ResultSetDigest); err != nil {
		return err
	}
	if gap.Regression && len(gap.RegressionKinds) == 0 {
		return errors.New("regression gap requires regression kinds")
	}
	if !gap.Regression && len(gap.RegressionKinds) != 0 {
		return errors.New("non-regression gap must not carry regression kinds")
	}
	return nil
}

func compatibilityGapForEntry(
	entry ResultEntry,
	source string,
	attempt int,
	kind string,
	digest string,
) CompatibilityEvidenceGap {
	return CompatibilityEvidenceGap{
		TargetID:        entry.TargetID,
		DeploymentID:    entry.DeploymentID,
		ClientID:        entry.ClientID,
		AuthMode:        entry.AuthMode,
		Source:          source,
		Attempt:         attempt,
		Kind:            kind,
		ResultSetDigest: digest,
	}
}

func compatibilityGapForArtifact(
	entry ResultEntry,
	source string,
	attempt int,
	kind string,
	digest string,
	run artifact.Run,
) CompatibilityEvidenceGap {
	gap := compatibilityGapForEntry(entry, source, attempt, kind, digest)
	gap.ClientVersion = run.Client.Version
	platform := run.Platform
	gap.Platform = &platform
	executedAt := run.ExecutedAt
	gap.ExecutedAt = &executedAt
	provenance := run.EvidenceProvenance
	gap.EvidenceProvenance = &provenance
	return gap
}

func compatibilityPointKey(point CompatibilityPoint) string {
	parts := []string{
		point.TargetID,
		point.DeploymentID,
		point.DeploymentFingerprint,
		point.ClientID,
		string(point.AuthMode),
		point.Platform.OS,
		point.Platform.Arch,
		point.ClientVersion,
	}
	return strings.Join(parts, "\x00")
}

func compatibilityPointContextKey(point CompatibilityPoint) string {
	parts := []string{
		point.TargetID,
		point.DeploymentID,
		point.DeploymentFingerprint,
		point.ClientID,
		string(point.AuthMode),
		point.Platform.OS,
		point.Platform.Arch,
	}
	return strings.Join(parts, "\x00")
}

func compatibilityContextLastObserved(points []CompatibilityPoint) map[string]compatibilityContextLast {
	out := make(map[string]compatibilityContextLast)
	for _, point := range points {
		key := compatibilityPointContextKey(point)
		current, ok := out[key]
		if !ok || point.LastObservedAt.After(current.at) ||
			(point.LastObservedAt.Equal(current.at) && point.ClientVersion > current.version) {
			out[key] = compatibilityContextLast{version: point.ClientVersion, at: point.LastObservedAt}
		}
	}
	return out
}

func compatibilityStaticContextMatches(point CompatibilityPoint, query CompatibilityQuery) bool {
	return point.TargetID == query.TargetID &&
		point.DeploymentID == query.DeploymentID &&
		point.ClientID == query.ClientID &&
		point.AuthMode == query.AuthMode
}

func compatibilityGapStaticContextMatches(gap CompatibilityEvidenceGap, query CompatibilityQuery) bool {
	return gap.TargetID == query.TargetID &&
		gap.DeploymentID == query.DeploymentID &&
		gap.ClientID == query.ClientID &&
		gap.AuthMode == query.AuthMode
}

func sortCompatibilityPoints(points []CompatibilityPoint) {
	sort.Slice(points, func(i, j int) bool {
		return compatibilityPointKey(points[i]) < compatibilityPointKey(points[j])
	})
}

func sortCompatibilityObservations(observations []CompatibilityObservation) {
	sort.SliceStable(observations, func(i, j int) bool {
		return compatibilityObservationSortKey(observations[i]) < compatibilityObservationSortKey(observations[j])
	})
}

func compatibilityObservationSortKey(observation CompatibilityObservation) string {
	return observation.ExecutedAt.Format(time.RFC3339Nano) + "\x00" + observation.Source + "\x00" + fmt.Sprintf("%09d", observation.Attempt)
}

func sortCompatibilityGaps(gaps []CompatibilityEvidenceGap) {
	sort.SliceStable(gaps, func(i, j int) bool {
		return compatibilityGapSortKey(gaps[i]) < compatibilityGapSortKey(gaps[j])
	})
}

func compatibilityGapSortKey(gap CompatibilityEvidenceGap) string {
	platform := ""
	if gap.Platform != nil {
		platform = gap.Platform.OS + "\x00" + gap.Platform.Arch
	}
	executedAt := ""
	if gap.ExecutedAt != nil {
		executedAt = gap.ExecutedAt.Format(time.RFC3339Nano)
	}
	parts := []string{
		gap.TargetID,
		gap.DeploymentID,
		gap.ClientID,
		string(gap.AuthMode),
		platform,
		gap.Source,
		fmt.Sprintf("%09d", gap.Attempt),
		gap.Kind,
		executedAt,
	}
	return strings.Join(parts, "\x00")
}
