package toolinventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	SchemaVersion = 1
	ArtifactType  = "mcp-interop/tool-expectation-evidence"
	maxBytes      = 1 << 20
)

// Evidence stores only operator-declared tool names and their directly
// observed membership. Never persist the entire real-client tool catalog.
// The attached live-run is protected-path schema v2, not the legacy JSON.
type Evidence struct {
	SchemaVersion int                     `json:"schema_version"`
	ArtifactType  string                  `json:"artifact_type"`
	Run           artifact.Run            `json:"run"`
	Expectation   interop.ToolExpectation `json:"expectation"`
}

func New(run artifact.Run, check interop.ToolExpectation) (Evidence, error) {
	value := Evidence{SchemaVersion: SchemaVersion, ArtifactType: ArtifactType, Run: run, Expectation: check}
	return value, Validate(value)
}

func Validate(value Evidence) error {
	if value.SchemaVersion != SchemaVersion || value.ArtifactType != ArtifactType {
		return errors.New("unsupported tool evidence schema identity")
	}
	if err := artifact.ValidateRunV2ProtectedPath(value.Run); err != nil {
		return fmt.Errorf("protected-path run: %w", err)
	}
	check := value.Expectation
	if len(check.ExpectedNames) == 0 && check.ExpectedCount == nil {
		return errors.New("at least one explicit name or count expectation is required")
	}
	if len(check.ExpectedNames) > interop.MaxExpectedToolNames {
		return errors.New("too many expected tool names")
	}
	for i, name := range check.ExpectedNames {
		if err := interop.ValidateExpectedToolName(name); err != nil {
			return fmt.Errorf("expected name %d: %w", i, err)
		}
		if i > 0 && check.ExpectedNames[i-1] >= name {
			return errors.New("expected names must be unique and sorted")
		}
	}
	if check.ExpectedCount != nil && (*check.ExpectedCount < 0 || *check.ExpectedCount > 4096) {
		return errors.New("expected count outside permitted bounds")
	}
	if check.ObservedCount != nil && (*check.ObservedCount < 0 || *check.ObservedCount > 4096) {
		return errors.New("observed count outside permitted bounds")
	}
	expectedSet := make(map[string]struct{}, len(check.ExpectedNames))
	for _, name := range check.ExpectedNames {
		expectedSet[name] = struct{}{}
	}
	for i, name := range check.MissingNames {
		if _, ok := expectedSet[name]; !ok {
			return errors.New("missing name not in declared expectations")
		}
		if i > 0 && check.MissingNames[i-1] >= name {
			return errors.New("missing names must be unique and sorted")
		}
	}
	if check.ObservedCount == nil {
		if check.Status != interop.StatusUnknown || check.ReasonCode != interop.ToolReasonUnobservable || len(check.MissingNames) != 0 {
			return errors.New("unobserved inventory must be unknown with no inferred missing names")
		}
		return nil
	}
	if value.Run.EvidenceProvenance.Kind != artifact.ProvenanceRealClientAdapter {
		return errors.New("observed inventory requires real-client provenance")
	}
	for _, stage := range value.Run.Stages {
		if stage.Status != interop.StatusPass {
			return errors.New("observed inventory requires all core stages PASS")
		}
	}
	if len(check.ExpectedNames)-len(check.MissingNames) > *check.ObservedCount {
		return errors.New("observed count cannot be smaller than present expected names")
	}
	status := interop.StatusPass
	reason := interop.ToolReasonMatched
	if len(check.MissingNames) != 0 {
		status, reason = interop.StatusFail, interop.ToolReasonMissing
	} else if check.ExpectedCount != nil && *check.ExpectedCount != *check.ObservedCount {
		status, reason = interop.StatusFail, interop.ToolReasonCountMismatch
	}
	if check.Status != status || check.ReasonCode != reason {
		return errors.New("expectation verdict contradicts recorded evidence")
	}
	return nil
}

func WriteFile(path string, value Evidence) error {
	if path == "" || path == "-" {
		return errors.New("tool evidence output requires a file path")
	}
	if err := Validate(value); err != nil {
		return err
	}
	// O_EXCL protects the destination against a symlink or last-moment
	// replacement. Any partial write is removed on returned failure; a hard
	// process crash leaves invalid JSON, which the strict reader rejects.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create private tool evidence: %w", err)
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func ReadFile(path string) (Evidence, error) {
	stat, err := os.Lstat(path)
	if err != nil {
		return Evidence{}, err
	}
	if !stat.Mode().IsRegular() {
		return Evidence{}, errors.New("tool evidence input must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return Evidence{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return Evidence{}, err
	}
	if len(data) > maxBytes {
		return Evidence{}, errors.New("tool evidence file exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value Evidence
	if err := decoder.Decode(&value); err != nil {
		return Evidence{}, fmt.Errorf("decode tool evidence: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Evidence{}, errors.New("tool evidence contains trailing data")
	}
	if err := Validate(value); err != nil {
		return Evidence{}, err
	}
	return value, nil
}

const (
	DecisionClean      = "clean"
	DecisionRegression = "regression"
	DecisionDrift      = "drift"
	DecisionRecovered  = "recovered"
	DecisionNonPass    = "non_pass"
	DecisionUnknown    = "unknown"
)

type Diff struct {
	SchemaVersion        int            `json:"schema_version"`
	ArtifactType         string         `json:"artifact_type"`
	Decision             string         `json:"decision"`
	BaselineStatus       interop.Status `json:"baseline_status"`
	CurrentStatus        interop.Status `json:"current_status"`
	BaselineCount        *int           `json:"baseline_count,omitempty"`
	CurrentCount         *int           `json:"current_count,omitempty"`
	NewlyMissing         []string       `json:"newly_missing,omitempty"`
	Recovered            []string       `json:"recovered,omitempty"`
	ClientVersionChanged bool           `json:"client_version_changed"`
}

func Compare(base, current Evidence) (Diff, error) {
	if err := Validate(base); err != nil {
		return Diff{}, fmt.Errorf("baseline: %w", err)
	}
	if err := Validate(current); err != nil {
		return Diff{}, fmt.Errorf("current: %w", err)
	}
	if base.Run.Endpoint != current.Run.Endpoint || base.Run.Client.ID != current.Run.Client.ID ||
		base.Run.Platform != current.Run.Platform || base.Run.AuthMode != current.Run.AuthMode ||
		!reflect.DeepEqual(base.Expectation.ExpectedNames, current.Expectation.ExpectedNames) ||
		!reflect.DeepEqual(base.Expectation.ExpectedCount, current.Expectation.ExpectedCount) {
		return Diff{}, errors.New("incompatible deployment, client, platform, auth or expectations")
	}
	diff := Diff{SchemaVersion: 1, ArtifactType: "mcp-interop/tool-expectation-diff",
		BaselineStatus: base.Expectation.Status, CurrentStatus: current.Expectation.Status,
		BaselineCount: base.Expectation.ObservedCount, CurrentCount: current.Expectation.ObservedCount,
		ClientVersionChanged: base.Run.Client.Version != current.Run.Client.Version,
	}
	if diff.BaselineStatus == interop.StatusUnknown || diff.CurrentStatus == interop.StatusUnknown {
		diff.Decision = DecisionUnknown
		return diff, nil
	}
	oldMissing := map[string]bool{}
	newMissing := map[string]bool{}
	for _, name := range base.Expectation.MissingNames {
		oldMissing[name] = true
	}
	for _, name := range current.Expectation.MissingNames {
		newMissing[name] = true
	}
	for _, name := range current.Expectation.ExpectedNames {
		if !oldMissing[name] && newMissing[name] {
			diff.NewlyMissing = append(diff.NewlyMissing, name)
		}
		if oldMissing[name] && !newMissing[name] {
			diff.Recovered = append(diff.Recovered, name)
		}
	}
	sort.Strings(diff.NewlyMissing)
	sort.Strings(diff.Recovered)
	switch {
	case len(diff.NewlyMissing) != 0 || (diff.BaselineCount != nil && diff.CurrentCount != nil && *diff.CurrentCount < *diff.BaselineCount):
		diff.Decision = DecisionRegression
	case current.Expectation.Status == interop.StatusFail:
		diff.Decision = DecisionNonPass
	case base.Expectation.Status == interop.StatusFail && current.Expectation.Status == interop.StatusPass:
		diff.Decision = DecisionRecovered
	case len(diff.Recovered) != 0:
		diff.Decision = DecisionRecovered
	case diff.BaselineCount != nil && diff.CurrentCount != nil && *diff.CurrentCount != *diff.BaselineCount:
		diff.Decision = DecisionDrift
	default:
		diff.Decision = DecisionClean
	}
	return diff, nil
}
