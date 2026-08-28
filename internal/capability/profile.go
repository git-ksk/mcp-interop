package capability

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/privatefile"
)

const (
	SchemaVersion = 1
	ArtifactType  = "mcp-interop/capability-profile"

	StatePass        State = "pass"
	StateFail        State = "fail"
	StateUnknown     State = "unknown"
	StateUnsupported State = "unsupported"
	StateUntested    State = "untested"

	EvidenceClientProtocol       EvidenceKind = "client_protocol"
	EvidenceClientControlSurface EvidenceKind = "client_control_surface"
	EvidenceClientObservedState  EvidenceKind = "client_observed_state"
	EvidenceAdapterPolicy        EvidenceKind = "adapter_policy"
	EvidenceNone                 EvidenceKind = "none"

	maxProfileBytes      = 1 << 20
	maxSafeIDBytes       = 96
	maxDisplayFieldBytes = 256
)

var safeIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

type State string

type EvidenceKind string

// Context is the exact, secret-safe execution identity for capability claims.
// It deliberately contains no endpoint URL or endpoint path. DeploymentID is
// an operator-chosen non-secret identity and DeploymentFingerprint binds the
// exact deployment without serializing a protected path.
type Context struct {
	ObservedAt            time.Time                   `json:"observed_at"`
	DeploymentID          string                      `json:"deployment_id"`
	DeploymentFingerprint string                      `json:"deployment_fingerprint"`
	Client                artifact.ClientIdentity     `json:"client"`
	Platform              artifact.Platform           `json:"platform"`
	Runtime               artifact.Runtime            `json:"runtime"`
	AuthMode              string                      `json:"auth_mode"`
	EvidenceProvenance    artifact.EvidenceProvenance `json:"evidence_provenance"`
}

// Observation is one optional capability claim. EvidenceID is a stable,
// project-defined, non-secret identifier for the direct evidence contract; it
// is never raw protocol/UI/client output.
type Observation struct {
	CapabilityID string       `json:"capability_id"`
	State        State        `json:"state"`
	EvidenceKind EvidenceKind `json:"evidence_kind"`
	EvidenceID   string       `json:"evidence_id,omitempty"`
}

// Profile is separate from the existing reach/auth/init/tools live-result
// schemas. Validating or emitting a capability profile cannot change core PASS.
type Profile struct {
	SchemaVersion int           `json:"schema_version"`
	ArtifactType  string        `json:"artifact_type"`
	Context       Context       `json:"context"`
	Capabilities  []Observation `json:"capabilities"`
}

// ContextFromLiveRunV2 copies exact, secret-safe context from a validated
// schema-v2 live-result run. Protected endpoint paths are not available to this
// function and therefore cannot be copied into a capability profile.
func ContextFromLiveRunV2(run artifact.Run) (Context, error) {
	if err := artifact.ValidateRunV2ProtectedPath(run); err != nil {
		return Context{}, fmt.Errorf("live-result v2 run: %w", err)
	}
	context := Context{
		ObservedAt:            run.ExecutedAt,
		DeploymentID:          run.Endpoint.Identity,
		DeploymentFingerprint: run.Endpoint.Fingerprint,
		Client:                run.Client,
		Platform:              run.Platform,
		Runtime:               run.Runtime,
		AuthMode:              run.AuthMode,
		EvidenceProvenance:    run.EvidenceProvenance,
	}
	if err := validateContext(context); err != nil {
		return Context{}, err
	}
	return context, nil
}

// NewProfile builds a deterministic profile from an exact context and optional
// capability observations. It does not execute a client or infer capabilities.
func NewProfile(context Context, observations []Observation) (Profile, error) {
	copied := append([]Observation(nil), observations...)
	sort.Slice(copied, func(i, j int) bool { return copied[i].CapabilityID < copied[j].CapabilityID })
	profile := Profile{
		SchemaVersion: SchemaVersion,
		ArtifactType:  ArtifactType,
		Context:       context,
		Capabilities:  copied,
	}
	if err := Validate(profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func Validate(profile Profile) error {
	if profile.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported capability-profile schema_version %d", profile.SchemaVersion)
	}
	if profile.ArtifactType != ArtifactType {
		return fmt.Errorf("unsupported capability-profile artifact_type %q", profile.ArtifactType)
	}
	if err := validateContext(profile.Context); err != nil {
		return fmt.Errorf("context: %w", err)
	}
	if len(profile.Capabilities) == 0 {
		return errors.New("capability profile requires at least one capability observation")
	}
	previous := ""
	for i, observation := range profile.Capabilities {
		if err := validateObservation(observation); err != nil {
			return fmt.Errorf("capabilities[%d]: %w", i, err)
		}
		if previous != "" && observation.CapabilityID <= previous {
			if observation.CapabilityID == previous {
				return fmt.Errorf("capabilities[%d]: duplicate capability_id %q", i, observation.CapabilityID)
			}
			return fmt.Errorf("capabilities[%d]: capability observations must use deterministic capability_id order", i)
		}
		previous = observation.CapabilityID
	}
	return nil
}

func validateContext(context Context) error {
	if context.ObservedAt.IsZero() {
		return errors.New("observed_at is required")
	}
	_, offset := context.ObservedAt.Zone()
	if offset != 0 {
		return errors.New("observed_at must use UTC")
	}
	if err := artifact.ValidateDeploymentID(context.DeploymentID); err != nil {
		return fmt.Errorf("deployment_id: %w", err)
	}
	if err := validateSHA256("deployment_fingerprint", context.DeploymentFingerprint); err != nil {
		return err
	}
	if err := validateSafeID("client.id", context.Client.ID); err != nil {
		return err
	}
	if err := validateDisplayField("client.product", context.Client.Product); err != nil {
		return err
	}
	if err := validateDisplayField("client.version", context.Client.Version); err != nil {
		return err
	}
	if err := validateSafeID("platform.os", context.Platform.OS); err != nil {
		return err
	}
	if err := validateSafeID("platform.arch", context.Platform.Arch); err != nil {
		return err
	}
	if err := validateDisplayField("runtime.mcp_interop_version", context.Runtime.MCPInteropVersion); err != nil {
		return err
	}
	if err := validateDisplayField("runtime.mcp_interop_commit", context.Runtime.MCPInteropCommit); err != nil {
		return err
	}
	if err := validateDisplayField("runtime.go_version", context.Runtime.GoVersion); err != nil {
		return err
	}
	if context.AuthMode != "default" && context.AuthMode != "oauth" {
		return fmt.Errorf("unsupported auth_mode %q", context.AuthMode)
	}
	if context.EvidenceProvenance.Kind != artifact.ProvenanceRealClientAdapter {
		return errors.New("capability profile requires real-client adapter provenance")
	}
	if err := validateSafeID("evidence_provenance.adapter_id", context.EvidenceProvenance.AdapterID); err != nil {
		return err
	}
	if context.EvidenceProvenance.AdapterID != context.Client.ID {
		return errors.New("capability adapter_id must match client id")
	}
	return nil
}

func validateObservation(observation Observation) error {
	if err := validateSafeID("capability_id", observation.CapabilityID); err != nil {
		return err
	}
	switch observation.State {
	case StatePass, StateFail, StateUnknown:
		if !isDirectClientEvidence(observation.EvidenceKind) {
			return fmt.Errorf("state %q requires direct client evidence_kind", observation.State)
		}
		if err := validateSafeID("evidence_id", observation.EvidenceID); err != nil {
			return err
		}
	case StateUnsupported:
		if observation.EvidenceKind != EvidenceAdapterPolicy {
			return fmt.Errorf("state %q requires evidence_kind %q", observation.State, EvidenceAdapterPolicy)
		}
		if err := validateSafeID("evidence_id", observation.EvidenceID); err != nil {
			return err
		}
	case StateUntested:
		if observation.EvidenceKind != EvidenceNone {
			return fmt.Errorf("state %q requires evidence_kind %q", observation.State, EvidenceNone)
		}
		if observation.EvidenceID != "" {
			return errors.New("untested capability must not claim evidence_id")
		}
	default:
		return fmt.Errorf("unsupported capability state %q", observation.State)
	}
	return nil
}

func isDirectClientEvidence(kind EvidenceKind) bool {
	switch kind {
	case EvidenceClientProtocol, EvidenceClientControlSurface, EvidenceClientObservedState:
		return true
	default:
		return false
	}
}

func validateSafeID(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if len(value) > maxSafeIDBytes {
		return fmt.Errorf("%s exceeds %d bytes", field, maxSafeIDBytes)
	}
	if !safeIDPattern.MatchString(value) {
		return fmt.Errorf("%s must use lowercase ASCII letters, digits, '.', '_', and '-'", field)
	}
	return nil
}

func validateDisplayField(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if len(value) > maxDisplayFieldBytes {
		return fmt.Errorf("%s exceeds %d bytes", field, maxDisplayFieldBytes)
	}
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must not have surrounding whitespace", field)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", field)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s must not contain control characters", field)
		}
	}
	return nil
}

func validateSHA256(field, value string) error {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return fmt.Errorf("%s must be a sha256 fingerprint", field)
	}
	hexValue := strings.TrimPrefix(value, "sha256:")
	if _, err := hex.DecodeString(hexValue); err != nil || hexValue != strings.ToLower(hexValue) {
		return fmt.Errorf("%s must contain lowercase hexadecimal sha256 bytes", field)
	}
	return nil
}

func ReadFile(path string) (Profile, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Profile{}, err
	}
	if !info.Mode().IsRegular() {
		return Profile{}, errors.New("capability profile must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return Profile{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProfileBytes+1))
	if err != nil {
		return Profile{}, fmt.Errorf("read capability profile: %w", err)
	}
	if len(data) > maxProfileBytes {
		return Profile{}, fmt.Errorf("capability profile exceeds %d bytes", maxProfileBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var profile Profile
	if err := decoder.Decode(&profile); err != nil {
		return Profile{}, fmt.Errorf("decode capability profile: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Profile{}, errors.New("decode capability profile: multiple JSON values")
		}
		return Profile{}, fmt.Errorf("decode capability profile trailing data: %w", err)
	}
	if err := Validate(profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func WriteFile(path string, profile Profile) error {
	if err := Validate(profile); err != nil {
		return err
	}
	return privatefile.WriteJSON(path, profile)
}
