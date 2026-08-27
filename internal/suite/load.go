package suite

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

// LoadedResultSet is a validated suite index plus the individual artifacts it
// references. Endpoint paths remain confined to the already secret-safe v2
// artifacts and are never reconstructed from external inputs.
type LoadedResultSet struct {
	Index     ResultIndex
	IndexPath string
	Artifacts map[string]artifact.Artifact
}

// ReadResultSet validates an index and each referenced regular-file artifact.
// Symlink artifact references are rejected so a result set cannot redirect a
// later privileged report step to arbitrary local files.
func ReadResultSet(indexPath string) (LoadedResultSet, error) {
	info, err := os.Lstat(indexPath)
	if err != nil {
		return LoadedResultSet{}, fmt.Errorf("inspect suite result index: %w", err)
	}
	if !info.Mode().IsRegular() {
		return LoadedResultSet{}, errors.New("suite result index must be a regular file")
	}
	index, err := ReadResultIndex(indexPath)
	if err != nil {
		return LoadedResultSet{}, err
	}
	loaded := LoadedResultSet{
		Index:     index,
		IndexPath: indexPath,
		Artifacts: make(map[string]artifact.Artifact, len(index.Runs)),
	}
	base := filepath.Dir(indexPath)
	resolvedBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		return LoadedResultSet{}, fmt.Errorf("resolve suite result directory: %w", err)
	}
	for _, entry := range index.Runs {
		if entry.Artifact == "" {
			continue
		}
		artifactPath := filepath.Join(base, filepath.FromSlash(entry.Artifact))
		relative, err := filepath.Rel(base, artifactPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
			return LoadedResultSet{}, fmt.Errorf("run %s/%s: artifact reference escapes result set", entry.TargetID, entry.ClientID)
		}
		artifactInfo, err := os.Lstat(artifactPath)
		if err != nil {
			return LoadedResultSet{}, fmt.Errorf("run %s/%s: inspect artifact: %w", entry.TargetID, entry.ClientID, err)
		}
		if !artifactInfo.Mode().IsRegular() {
			return LoadedResultSet{}, fmt.Errorf("run %s/%s: artifact must be a regular file", entry.TargetID, entry.ClientID)
		}
		resolvedArtifact, err := filepath.EvalSymlinks(artifactPath)
		if err != nil {
			return LoadedResultSet{}, fmt.Errorf("run %s/%s: resolve artifact: %w", entry.TargetID, entry.ClientID, err)
		}
		resolvedRelative, err := filepath.Rel(resolvedBase, resolvedArtifact)
		if err != nil || resolvedRelative == ".." || strings.HasPrefix(resolvedRelative, ".."+string(os.PathSeparator)) {
			return LoadedResultSet{}, fmt.Errorf("run %s/%s: resolved artifact escapes result set", entry.TargetID, entry.ClientID)
		}
		value, err := artifact.ReadFile(artifactPath)
		if err != nil {
			return LoadedResultSet{}, fmt.Errorf("run %s/%s: read artifact: %w", entry.TargetID, entry.ClientID, err)
		}
		if err := ValidateResultArtifact(entry, value); err != nil {
			return LoadedResultSet{}, fmt.Errorf("run %s/%s: %w", entry.TargetID, entry.ClientID, err)
		}
		loaded.Artifacts[resultEntryKey(entry)] = value
	}
	return loaded, nil
}

// ValidateResultArtifact binds one index entry to exactly one schema-v2
// live-result artifact and verifies the suite-level outcome summary.
func ValidateResultArtifact(entry ResultEntry, value artifact.Artifact) error {
	if entry.Outcome == OutcomeError {
		return errors.New("error outcome must not have an artifact")
	}
	if value.SchemaVersion != artifact.SchemaVersionV2 || len(value.Runs) != 1 {
		return errors.New("suite artifact must contain exactly one schema-v2 run")
	}
	run := value.Runs[0]
	if run.Client.ID != entry.ClientID {
		return errors.New("artifact client does not match index")
	}
	expectedAuth := "default"
	if entry.AuthMode == AuthOAuth {
		expectedAuth = "oauth"
	}
	if run.AuthMode != expectedAuth {
		return errors.New("artifact auth mode does not match index")
	}
	if run.Endpoint.IdentityKind != artifact.EndpointIdentityDeploymentID || run.Endpoint.Identity != entry.DeploymentID {
		return errors.New("artifact deployment identity does not match index")
	}
	passed := artifactRunPassed(run)
	if entry.Outcome == OutcomePass && !passed {
		return errors.New("pass index outcome does not match artifact stages")
	}
	if entry.Outcome == OutcomeNonPass && passed {
		return errors.New("non_pass index outcome does not match artifact stages")
	}
	return nil
}

func artifactRunPassed(run artifact.Run) bool {
	if len(run.Stages) != len(interop.OrderedStages) {
		return false
	}
	for _, stage := range run.Stages {
		if stage.Status != interop.StatusPass {
			return false
		}
	}
	return true
}
