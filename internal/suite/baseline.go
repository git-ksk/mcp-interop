package suite

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/privatefile"
)

const (
	BaselineSchemaVersion = 1
	BaselineArtifactType  = "mcp-interop/suite-baseline"
	BaselineDescriptor    = "baseline.json"
	BaselineResultsDir    = "results"

	maxBaselineDescriptorBytes = 64 << 10
)

// Baseline records the immutable identity of one accepted suite result-set
// snapshot. Endpoint values and local source paths are intentionally absent.
type Baseline struct {
	SchemaVersion       int              `json:"schema_version"`
	ArtifactType        string           `json:"artifact_type"`
	CreatedAt           time.Time        `json:"created_at"`
	ManifestFingerprint string           `json:"manifest_fingerprint"`
	ExecutionContext    ExecutionContext `json:"execution_context"`
	ResultSetDigest     string           `json:"result_set_digest"`
	Supersedes          string           `json:"supersedes,omitempty"`
}

// LoadedBaseline is a validated baseline descriptor and its immutable result
// snapshot. Directory is local-only context and is never serialized.
type LoadedBaseline struct {
	Descriptor Baseline
	Directory  string
	ResultSet  LoadedResultSet
}

// CreateBaseline accepts one complete real-client result-set as a baseline.
// The destination is reserved with mkdir so an existing path is never replaced.
// A baseline becomes readable only after baseline.json is written last.
func CreateBaseline(
	source LoadedResultSet,
	outputDir string,
	supersedes *LoadedBaseline,
	createdAt time.Time,
) (LoadedBaseline, error) {
	if strings.TrimSpace(outputDir) == "" || outputDir == "-" {
		return LoadedBaseline{}, errors.New("baseline output directory is required")
	}
	if strings.TrimSpace(outputDir) != outputDir {
		return LoadedBaseline{}, errors.New("baseline output directory must not have surrounding whitespace")
	}
	if err := validateBaselineSource(source); err != nil {
		return LoadedBaseline{}, fmt.Errorf("baseline source: %w", err)
	}
	if createdAt.IsZero() {
		return LoadedBaseline{}, errors.New("baseline created_at is required")
	}
	createdAt = createdAt.UTC()

	var supersedesFingerprint string
	if supersedes != nil {
		if err := validateSupersedingSource(*supersedes, source); err != nil {
			return LoadedBaseline{}, err
		}
		var err error
		supersedesFingerprint, err = BaselineFingerprint(supersedes.Descriptor)
		if err != nil {
			return LoadedBaseline{}, err
		}
	}

	parent := filepath.Dir(outputDir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return LoadedBaseline{}, fmt.Errorf("create baseline output parent: %w", err)
	}
	if err := os.Mkdir(outputDir, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return LoadedBaseline{}, errors.New("baseline output directory already exists")
		}
		return LoadedBaseline{}, fmt.Errorf("reserve baseline output directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(outputDir)
		}
	}()

	staging := filepath.Join(outputDir, ".staging")
	if err := os.Mkdir(staging, 0o700); err != nil {
		return LoadedBaseline{}, fmt.Errorf("create baseline staging directory: %w", err)
	}
	stagedResults := filepath.Join(staging, BaselineResultsDir)
	if err := writeResultSetSnapshot(source, stagedResults); err != nil {
		return LoadedBaseline{}, err
	}
	stagedSet, err := ReadResultSet(filepath.Join(stagedResults, "index.json"))
	if err != nil {
		return LoadedBaseline{}, fmt.Errorf("validate staged baseline result set: %w", err)
	}
	digest, err := ResultSetDigest(stagedSet)
	if err != nil {
		return LoadedBaseline{}, err
	}

	descriptor := Baseline{
		SchemaVersion:       BaselineSchemaVersion,
		ArtifactType:        BaselineArtifactType,
		CreatedAt:           createdAt,
		ManifestFingerprint: stagedSet.Index.ManifestFingerprint,
		ExecutionContext:    stagedSet.Index.ExecutionContext,
		ResultSetDigest:     digest,
		Supersedes:          supersedesFingerprint,
	}
	if err := ValidateBaseline(descriptor); err != nil {
		return LoadedBaseline{}, err
	}

	finalResults := filepath.Join(outputDir, BaselineResultsDir)
	if err := os.Rename(stagedResults, finalResults); err != nil {
		return LoadedBaseline{}, fmt.Errorf("publish baseline result snapshot: %w", err)
	}
	if err := os.Remove(staging); err != nil {
		return LoadedBaseline{}, fmt.Errorf("remove baseline staging directory: %w", err)
	}
	if err := privatefile.WriteJSON(filepath.Join(outputDir, BaselineDescriptor), descriptor); err != nil {
		return LoadedBaseline{}, fmt.Errorf("write baseline descriptor: %w", err)
	}
	committed = true
	return ReadBaseline(outputDir)
}

// ReadBaseline strictly validates a baseline descriptor, its copied result set,
// and the deterministic digest binding the two together.
func ReadBaseline(directory string) (LoadedBaseline, error) {
	info, err := os.Lstat(directory)
	if err != nil {
		return LoadedBaseline{}, fmt.Errorf("inspect baseline directory: %w", err)
	}
	if !info.IsDir() {
		return LoadedBaseline{}, errors.New("baseline path must be a directory, not a symlink or file")
	}
	descriptorPath := filepath.Join(directory, BaselineDescriptor)
	descriptor, err := readBaselineDescriptor(descriptorPath)
	if err != nil {
		return LoadedBaseline{}, err
	}
	resultSet, err := ReadResultSet(filepath.Join(directory, BaselineResultsDir, "index.json"))
	if err != nil {
		return LoadedBaseline{}, fmt.Errorf("read baseline result set: %w", err)
	}
	if err := validateBaselineSource(resultSet); err != nil {
		return LoadedBaseline{}, fmt.Errorf("baseline result set: %w", err)
	}
	if descriptor.ManifestFingerprint != resultSet.Index.ManifestFingerprint {
		return LoadedBaseline{}, errors.New("baseline manifest fingerprint does not match result snapshot")
	}
	if descriptor.ExecutionContext != resultSet.Index.ExecutionContext {
		return LoadedBaseline{}, errors.New("baseline execution context does not match result snapshot")
	}
	digest, err := ResultSetDigest(resultSet)
	if err != nil {
		return LoadedBaseline{}, err
	}
	if descriptor.ResultSetDigest != digest {
		return LoadedBaseline{}, errors.New("baseline result snapshot digest mismatch")
	}
	return LoadedBaseline{
		Descriptor: descriptor,
		Directory:  directory,
		ResultSet:  resultSet,
	}, nil
}

// ValidateBaseline validates the descriptor without reading its result snapshot.
func ValidateBaseline(value Baseline) error {
	if value.SchemaVersion != BaselineSchemaVersion {
		return fmt.Errorf("unsupported baseline schema_version %d", value.SchemaVersion)
	}
	if value.ArtifactType != BaselineArtifactType {
		return fmt.Errorf("unsupported baseline artifact_type %q", value.ArtifactType)
	}
	if value.CreatedAt.IsZero() {
		return errors.New("baseline created_at is required")
	}
	_, offset := value.CreatedAt.Zone()
	if offset != 0 {
		return errors.New("baseline created_at must use UTC")
	}
	if err := validateSHA256Fingerprint("manifest_fingerprint", value.ManifestFingerprint); err != nil {
		return err
	}
	if value.ExecutionContext != ExecutionTrusted {
		return errors.New("baseline currently requires trusted_real_client execution_context")
	}
	if err := validateSHA256Fingerprint("result_set_digest", value.ResultSetDigest); err != nil {
		return err
	}
	if value.Supersedes != "" {
		if err := validateSHA256Fingerprint("supersedes", value.Supersedes); err != nil {
			return err
		}
	}
	return nil
}

// BaselineFingerprint returns the stable fingerprint referenced by a successor.
func BaselineFingerprint(value Baseline) (string, error) {
	if err := ValidateBaseline(value); err != nil {
		return "", err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode baseline fingerprint input: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// ResultSetDigest binds the validated index and every referenced artifact using
// their logical JSON content, independent of source filesystem paths.
func ResultSetDigest(set LoadedResultSet) (string, error) {
	h := sha256.New()
	indexData, err := json.Marshal(set.Index)
	if err != nil {
		return "", fmt.Errorf("encode result index digest input: %w", err)
	}
	writeDigestPart(h, "index", indexData)
	for _, entry := range set.Index.Runs {
		writeDigestPart(h, "entry", []byte(resultEntryKey(entry)))
		if entry.Artifact == "" {
			continue
		}
		value, ok := set.Artifacts[resultEntryKey(entry)]
		if !ok {
			return "", fmt.Errorf("result set digest: missing artifact for %s/%s", entry.TargetID, entry.ClientID)
		}
		data, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("encode result artifact digest input: %w", err)
		}
		writeDigestPart(h, entry.Artifact, data)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// CompareBaselineResultSets performs the v0.7 regression comparison after
// enforcing baseline-specific deployment/platform identity fail-closed checks.
func CompareBaselineResultSets(
	baseline LoadedBaseline,
	attempts []LoadedResultSet,
) (RegressionReport, error) {
	for i, attempt := range attempts {
		if err := validateBaselineComparable(baseline.ResultSet, attempt); err != nil {
			return RegressionReport{}, fmt.Errorf("attempt[%d] is not comparable to baseline: %w", i, err)
		}
	}
	return CompareResultSets(baseline.ResultSet, attempts)
}

func validateBaselineSource(set LoadedResultSet) error {
	if err := ValidateResultIndex(set.Index); err != nil {
		return err
	}
	for _, entry := range set.Index.Runs {
		if entry.Outcome == OutcomeError || entry.Artifact == "" {
			return fmt.Errorf("run %s/%s has no complete live-result evidence", entry.TargetID, entry.ClientID)
		}
		value, ok := set.Artifacts[resultEntryKey(entry)]
		if !ok || len(value.Runs) != 1 {
			return fmt.Errorf("run %s/%s has no loaded single-run artifact", entry.TargetID, entry.ClientID)
		}
		if err := artifact.ValidateArtifact(value); err != nil {
			return fmt.Errorf("run %s/%s artifact is invalid: %w", entry.TargetID, entry.ClientID, err)
		}
		if err := ValidateResultArtifact(entry, value); err != nil {
			return fmt.Errorf("run %s/%s artifact does not match index: %w", entry.TargetID, entry.ClientID, err)
		}
		run := value.Runs[0]
		if run.EvidenceProvenance.Kind != artifact.ProvenanceRealClientAdapter {
			return fmt.Errorf("run %s/%s is not real-client adapter evidence", entry.TargetID, entry.ClientID)
		}
		if run.Client.Version == "" {
			return fmt.Errorf("run %s/%s has no exact client version", entry.TargetID, entry.ClientID)
		}
	}
	return nil
}

func validateSupersedingSource(previous LoadedBaseline, next LoadedResultSet) error {
	if previous.Descriptor.ManifestFingerprint != next.Index.ManifestFingerprint {
		return errors.New("superseding baseline manifest fingerprint differs from previous baseline")
	}
	if previous.Descriptor.ExecutionContext != next.Index.ExecutionContext {
		return errors.New("superseding baseline execution context differs from previous baseline")
	}
	if err := validateBaselineComparable(previous.ResultSet, next); err != nil {
		return fmt.Errorf("superseding baseline is not comparable to previous baseline: %w", err)
	}
	return nil
}

func validateBaselineComparable(baseline, attempt LoadedResultSet) error {
	if baseline.Index.ManifestFingerprint != attempt.Index.ManifestFingerprint {
		return errors.New("manifest fingerprint differs")
	}
	if baseline.Index.ExecutionContext != attempt.Index.ExecutionContext {
		return errors.New("execution context differs")
	}
	attemptEntries := resultEntryMap(attempt.Index)
	for _, baseEntry := range baseline.Index.Runs {
		currentEntry, ok := attemptEntries[resultEntryKey(baseEntry)]
		if !ok || baseEntry.Artifact == "" || currentEntry.Artifact == "" {
			continue
		}
		baseArtifact, baseOK := baseline.Artifacts[resultEntryKey(baseEntry)]
		currentArtifact, currentOK := attempt.Artifacts[resultEntryKey(currentEntry)]
		if !baseOK || !currentOK || len(baseArtifact.Runs) != 1 || len(currentArtifact.Runs) != 1 {
			continue
		}
		baseRun := baseArtifact.Runs[0]
		currentRun := currentArtifact.Runs[0]
		if baseRun.Endpoint.Fingerprint != currentRun.Endpoint.Fingerprint {
			return fmt.Errorf("run %s/%s deployment fingerprint differs", baseEntry.TargetID, baseEntry.ClientID)
		}
		if baseRun.Platform != currentRun.Platform {
			return fmt.Errorf("run %s/%s platform differs", baseEntry.TargetID, baseEntry.ClientID)
		}
	}
	return nil
}

func writeResultSetSnapshot(source LoadedResultSet, directory string) error {
	if err := os.Mkdir(directory, 0o700); err != nil {
		return fmt.Errorf("create baseline result directory: %w", err)
	}
	artifactsDir := filepath.Join(directory, "artifacts")
	if err := os.Mkdir(artifactsDir, 0o700); err != nil {
		return fmt.Errorf("create baseline artifact directory: %w", err)
	}
	for _, entry := range source.Index.Runs {
		if entry.Artifact == "" {
			continue
		}
		value, ok := source.Artifacts[resultEntryKey(entry)]
		if !ok {
			return fmt.Errorf("snapshot run %s/%s: loaded artifact is missing", entry.TargetID, entry.ClientID)
		}
		destination := filepath.Join(directory, filepath.FromSlash(entry.Artifact))
		if err := artifact.WriteFile(destination, value); err != nil {
			return fmt.Errorf("snapshot run %s/%s: %w", entry.TargetID, entry.ClientID, err)
		}
	}
	if err := WriteResultIndex(filepath.Join(directory, "index.json"), source.Index); err != nil {
		return fmt.Errorf("snapshot result index: %w", err)
	}
	return nil
}

func readBaselineDescriptor(path string) (Baseline, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Baseline{}, fmt.Errorf("inspect baseline descriptor: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Baseline{}, errors.New("baseline descriptor must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return Baseline{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBaselineDescriptorBytes+1))
	if err != nil {
		return Baseline{}, fmt.Errorf("read baseline descriptor: %w", err)
	}
	if len(data) > maxBaselineDescriptorBytes {
		return Baseline{}, fmt.Errorf("baseline descriptor exceeds %d bytes", maxBaselineDescriptorBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value Baseline
	if err := decoder.Decode(&value); err != nil {
		return Baseline{}, fmt.Errorf("decode baseline descriptor: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Baseline{}, err
	}
	if err := ValidateBaseline(value); err != nil {
		return Baseline{}, err
	}
	return value, nil
}

func validateSHA256Fingerprint(name, value string) error {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 {
		return fmt.Errorf("%s must be a sha256 fingerprint", name)
	}
	hexValue := strings.TrimPrefix(value, prefix)
	if _, err := hex.DecodeString(hexValue); err != nil || hexValue != strings.ToLower(hexValue) {
		return fmt.Errorf("%s must contain lowercase hexadecimal sha256 bytes", name)
	}
	return nil
}

func writeDigestPart(h hash.Hash, label string, data []byte) {
	_, _ = fmt.Fprintf(h, "%d:%s:%d:", len(label), label, len(data))
	_, _ = h.Write(data)
}
