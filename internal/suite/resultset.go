package suite

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/privatefile"
)

const (
	ResultSetSchemaVersion = 1
	ResultSetArtifactType  = "mcp-interop/suite-results"
	maxResultIndexBytes    = 1 << 20

	OutcomePass    = RunOutcome("pass")
	OutcomeNonPass = RunOutcome("non_pass")
	OutcomeError   = RunOutcome("error")
)

// RunOutcome is the suite-level execution outcome. Stage-level evidence stays
// in the referenced portable live-result artifact.
type RunOutcome string

// ResultIndex is the secret-safe index for one coherent suite artifact set.
type ResultIndex struct {
	SchemaVersion         int              `json:"schema_version"`
	ArtifactType          string           `json:"artifact_type"`
	ManifestSchemaVersion int              `json:"manifest_schema_version"`
	ManifestFingerprint   string           `json:"manifest_fingerprint"`
	ExecutionContext      ExecutionContext `json:"execution_context"`
	ArtifactSchemaVersion int              `json:"artifact_schema_version"`
	Runs                  []ResultEntry    `json:"runs"`
}

// ResultEntry points to one versioned live-result artifact. Endpoint values are
// intentionally absent from the index.
type ResultEntry struct {
	TargetID     string     `json:"target_id"`
	DeploymentID string     `json:"deployment_id"`
	ClientID     string     `json:"client_id"`
	AuthMode     AuthMode   `json:"auth_mode"`
	Outcome      RunOutcome `json:"outcome"`
	ExitCode     int        `json:"exit_code"`
	Artifact     string     `json:"artifact,omitempty"`
}

// NewResultIndex builds and validates a deterministic trusted-suite index.
func NewResultIndex(manifest Manifest, entries []ResultEntry) (ResultIndex, error) {
	fingerprint, err := ManifestFingerprint(manifest)
	if err != nil {
		return ResultIndex{}, err
	}
	index := ResultIndex{
		SchemaVersion:         ResultSetSchemaVersion,
		ArtifactType:          ResultSetArtifactType,
		ManifestSchemaVersion: manifest.SchemaVersion,
		ManifestFingerprint:   fingerprint,
		ExecutionContext:      manifest.ExecutionContext,
		ArtifactSchemaVersion: artifact.SchemaVersionV2,
		Runs:                  entries,
	}
	if err := ValidateResultIndex(index); err != nil {
		return ResultIndex{}, err
	}
	return index, nil
}

// ValidateResultIndex validates portable index structure and deterministic run
// ordering without reading the referenced artifacts.
func ValidateResultIndex(index ResultIndex) error {
	if index.SchemaVersion != ResultSetSchemaVersion {
		return fmt.Errorf("unsupported suite result schema_version %d", index.SchemaVersion)
	}
	if index.ArtifactType != ResultSetArtifactType {
		return fmt.Errorf("unsupported suite result artifact_type %q", index.ArtifactType)
	}
	if index.ManifestSchemaVersion != SchemaVersionV1 {
		return fmt.Errorf("unsupported manifest_schema_version %d", index.ManifestSchemaVersion)
	}
	if !strings.HasPrefix(index.ManifestFingerprint, "sha256:") || len(index.ManifestFingerprint) != len("sha256:")+64 {
		return errors.New("manifest_fingerprint must be a sha256 fingerprint")
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(index.ManifestFingerprint, "sha256:")); err != nil {
		return errors.New("manifest_fingerprint must contain lowercase hexadecimal sha256 bytes")
	}
	if index.ManifestFingerprint != strings.ToLower(index.ManifestFingerprint) {
		return errors.New("manifest_fingerprint must use lowercase hexadecimal")
	}
	if index.ExecutionContext != ExecutionTrusted {
		return errors.New("suite result index currently requires trusted_real_client execution_context")
	}
	if index.ArtifactSchemaVersion != artifact.SchemaVersionV2 {
		return fmt.Errorf("artifact_schema_version must be %d", artifact.SchemaVersionV2)
	}
	if len(index.Runs) == 0 {
		return errors.New("suite result index requires at least one run")
	}

	previous := ""
	seen := make(map[string]struct{}, len(index.Runs))
	for i, entry := range index.Runs {
		if err := validateResultEntry(entry); err != nil {
			return fmt.Errorf("runs[%d]: %w", i, err)
		}
		key := resultEntryKey(entry)
		if previous != "" && key < previous {
			return fmt.Errorf("runs[%d]: entries are not in deterministic order", i)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("runs[%d]: duplicate suite run identity", i)
		}
		seen[key] = struct{}{}
		previous = key
	}
	return nil
}

func validateResultEntry(entry ResultEntry) error {
	if err := validateTargetID(entry.TargetID); err != nil {
		return fmt.Errorf("target_id: %w", err)
	}
	if err := artifact.ValidateDeploymentID(entry.DeploymentID); err != nil {
		return fmt.Errorf("deployment_id: %w", err)
	}
	if err := validateClientSelection(ExecutionTrusted, ClientSelection{ID: entry.ClientID, Auth: entry.AuthMode}); err != nil {
		return err
	}
	switch entry.Outcome {
	case OutcomePass:
		if entry.ExitCode != 0 {
			return errors.New("pass outcome requires exit_code 0")
		}
		if err := validateArtifactReference(entry.Artifact); err != nil {
			return err
		}
	case OutcomeNonPass:
		if entry.ExitCode != 1 {
			return errors.New("non_pass outcome requires exit_code 1")
		}
		if err := validateArtifactReference(entry.Artifact); err != nil {
			return err
		}
	case OutcomeError:
		if entry.ExitCode != 1 {
			return errors.New("error outcome requires exit_code 1")
		}
		if entry.Artifact != "" {
			return errors.New("error outcome must not reference an artifact")
		}
	default:
		return fmt.Errorf("unsupported outcome %q", entry.Outcome)
	}
	return nil
}

func validateArtifactReference(reference string) error {
	if reference == "" {
		return errors.New("artifact reference is required")
	}
	if strings.Contains(reference, "\\") || path.IsAbs(reference) || path.Clean(reference) != reference || strings.HasPrefix(reference, "../") {
		return errors.New("artifact reference must be a clean relative slash path")
	}
	if !strings.HasPrefix(reference, "artifacts/") || !strings.HasSuffix(reference, ".json") {
		return errors.New("artifact reference must point below artifacts/ to a JSON file")
	}
	return nil
}

func resultEntryKey(entry ResultEntry) string {
	return entry.TargetID + "\x00" + entry.DeploymentID + "\x00" + entry.ClientID + "\x00" + string(entry.AuthMode)
}

// WriteResultIndex writes the index as a private JSON file.
func WriteResultIndex(filePath string, index ResultIndex) error {
	if err := ValidateResultIndex(index); err != nil {
		return err
	}
	return privatefile.WriteJSON(filePath, index)
}

// ReadResultIndex strictly reads a result-set index for later regression work.
func ReadResultIndex(filePath string) (ResultIndex, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return ResultIndex{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxResultIndexBytes+1))
	if err != nil {
		return ResultIndex{}, fmt.Errorf("read suite result index: %w", err)
	}
	if len(data) > maxResultIndexBytes {
		return ResultIndex{}, fmt.Errorf("suite result index exceeds %d bytes", maxResultIndexBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var index ResultIndex
	if err := decoder.Decode(&index); err != nil {
		return ResultIndex{}, fmt.Errorf("decode suite result index: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return ResultIndex{}, err
	}
	if err := ValidateResultIndex(index); err != nil {
		return ResultIndex{}, err
	}
	return index, nil
}
