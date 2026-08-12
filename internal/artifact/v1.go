package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	SchemaVersion = 1
	ArtifactType  = "mcp-interop/live-results"

	ProvenanceRealClientAdapter = "real_client_adapter"
	ProvenanceRunnerObservation = "runner_observation"
)

// Artifact is the portable, versioned live interoperability result document.
// It is intentionally separate from the legacy `test --json` contract.
type Artifact struct {
	SchemaVersion int    `json:"schema_version"`
	ArtifactType  string `json:"artifact_type"`
	Runs          []Run  `json:"runs"`
}

// Run is one client observation with enough secret-safe context for later
// regression comparison.
type Run struct {
	ExecutedAt         time.Time          `json:"executed_at"`
	Endpoint           EndpointIdentity   `json:"endpoint"`
	Client             ClientIdentity     `json:"client"`
	Platform           Platform           `json:"platform"`
	Runtime            Runtime            `json:"runtime"`
	AuthMode           string             `json:"auth_mode"`
	EvidenceProvenance EvidenceProvenance `json:"evidence_provenance"`
	Stages             []StageResult      `json:"stages"`
}

type EndpointIdentity struct {
	Identity    string `json:"identity"`
	Fingerprint string `json:"fingerprint"`
}

type ClientIdentity struct {
	ID      string `json:"id"`
	Product string `json:"product"`
	Version string `json:"version,omitempty"`
}

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type Runtime struct {
	MCPInteropVersion string `json:"mcp_interop_version"`
	MCPInteropCommit  string `json:"mcp_interop_commit"`
	GoVersion         string `json:"go_version"`
}

type EvidenceProvenance struct {
	Kind      string `json:"kind"`
	AdapterID string `json:"adapter_id,omitempty"`
}

type StageResult struct {
	Stage      interop.Stage      `json:"stage"`
	Status     interop.Status     `json:"status"`
	ReasonCode interop.ReasonCode `json:"reason_code,omitempty"`
}

// NewArtifact creates a validated v1 artifact from already-redacted live
// results. Human diagnostic messages are deliberately not copied.
func NewArtifact(runs []Run) Artifact {
	return Artifact{
		SchemaVersion: SchemaVersion,
		ArtifactType:  ArtifactType,
		Runs:          runs,
	}
}

// NewRun projects one existing interoperability Result into the v1 portable
// schema without changing any stage status or reason-code semantics.
func NewRun(result interop.Result, executedAt time.Time, authMode string, provenance EvidenceProvenance, runnerVersion, runnerCommit string) (Run, error) {
	endpoint, err := NewEndpointIdentity(result.Endpoint)
	if err != nil {
		return Run{}, err
	}

	stages := make([]StageResult, 0, len(result.Stages))
	for _, stage := range result.Stages {
		stages = append(stages, StageResult{
			Stage:      stage.Stage,
			Status:     stage.Status,
			ReasonCode: stage.ReasonCode,
		})
	}

	run := Run{
		ExecutedAt: executedAt.UTC(),
		Endpoint:   endpoint,
		Client: ClientIdentity{
			ID:      result.ClientID,
			Product: result.ClientName,
			Version: result.ClientVersion,
		},
		Platform: Platform{OS: runtime.GOOS, Arch: runtime.GOARCH},
		Runtime: Runtime{
			MCPInteropVersion: runnerVersion,
			MCPInteropCommit:  runnerCommit,
			GoVersion:         runtime.Version(),
		},
		AuthMode:           authMode,
		EvidenceProvenance: provenance,
		Stages:             stages,
	}
	if err := ValidateRun(run); err != nil {
		return Run{}, err
	}
	return run, nil
}

// NewEndpointIdentity removes all query values and other non-essential URL
// components before deriving the portable fingerprint. The raw endpoint is
// never hashed or persisted by this package.
func NewEndpointIdentity(raw string) (EndpointIdentity, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return EndpointIdentity{}, fmt.Errorf("parse endpoint: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return EndpointIdentity{}, errors.New("endpoint must use http or https")
	}
	if u.Host == "" {
		return EndpointIdentity{}, errors.New("endpoint must include a host")
	}

	scheme := strings.ToLower(u.Scheme)
	hostname := strings.ToLower(u.Hostname())
	if hostname == "" {
		return EndpointIdentity{}, errors.New("endpoint must include a hostname")
	}
	host := hostname
	if port := u.Port(); port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	identity := scheme + "://" + host + path
	sum := sha256.Sum256([]byte(identity))
	return EndpointIdentity{
		Identity:    identity,
		Fingerprint: "sha256:" + hex.EncodeToString(sum[:]),
	}, nil
}

func ValidateArtifact(value Artifact) error {
	if value.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", value.SchemaVersion)
	}
	if value.ArtifactType != ArtifactType {
		return fmt.Errorf("unsupported artifact_type %q", value.ArtifactType)
	}
	if len(value.Runs) == 0 {
		return errors.New("artifact must contain at least one run")
	}
	seen := make(map[string]struct{}, len(value.Runs))
	for i, run := range value.Runs {
		if err := ValidateRun(run); err != nil {
			return fmt.Errorf("runs[%d]: %w", i, err)
		}
		key := ComparisonKey(run)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("runs[%d]: duplicate comparison identity", i)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func ValidateRun(run Run) error {
	if run.ExecutedAt.IsZero() {
		return errors.New("executed_at is required")
	}
	_, offset := run.ExecutedAt.Zone()
	if offset != 0 {
		return errors.New("executed_at must use UTC")
	}
	if run.Client.ID == "" || run.Client.Product == "" {
		return errors.New("client id and product are required")
	}
	if run.Platform.OS == "" || run.Platform.Arch == "" {
		return errors.New("platform os and arch are required")
	}
	if run.Runtime.MCPInteropVersion == "" || run.Runtime.MCPInteropCommit == "" || run.Runtime.GoVersion == "" {
		return errors.New("runtime version, commit, and go_version are required")
	}
	if run.AuthMode == "" {
		return errors.New("auth_mode is required")
	}

	expectedEndpoint, err := NewEndpointIdentity(run.Endpoint.Identity)
	if err != nil {
		return fmt.Errorf("endpoint identity: %w", err)
	}
	if expectedEndpoint.Identity != run.Endpoint.Identity {
		return errors.New("endpoint identity is not canonical secret-safe form")
	}
	if expectedEndpoint.Fingerprint != run.Endpoint.Fingerprint {
		return errors.New("endpoint fingerprint does not match identity")
	}

	switch run.EvidenceProvenance.Kind {
	case ProvenanceRealClientAdapter:
		if run.EvidenceProvenance.AdapterID == "" {
			return errors.New("real-client provenance requires adapter_id")
		}
		if run.Client.Version == "" {
			return errors.New("real-client provenance requires exact client version")
		}
	case ProvenanceRunnerObservation:
		if run.EvidenceProvenance.AdapterID != "" {
			return errors.New("runner observation must not claim adapter_id")
		}
	default:
		return fmt.Errorf("unsupported evidence provenance %q", run.EvidenceProvenance.Kind)
	}

	if len(run.Stages) != len(interop.OrderedStages) {
		return fmt.Errorf("stages must contain exactly %d entries", len(interop.OrderedStages))
	}
	for i, expected := range interop.OrderedStages {
		stage := run.Stages[i]
		if stage.Stage != expected {
			return fmt.Errorf("stages[%d] must be %q", i, expected)
		}
		switch stage.Status {
		case interop.StatusPass, interop.StatusFail, interop.StatusSkip, interop.StatusUnknown:
		default:
			return fmt.Errorf("stages[%d] has unsupported status %q", i, stage.Status)
		}
		if run.EvidenceProvenance.Kind == ProvenanceRunnerObservation && stage.Status == interop.StatusPass {
			return fmt.Errorf("stages[%d]: runner observation cannot produce pass", i)
		}
	}
	return nil
}

// ComparisonKey pairs runs while deliberately excluding client version and
// runner/runtime version so version-only changes do not become regressions.
func ComparisonKey(run Run) string {
	parts := []string{
		run.Endpoint.Fingerprint,
		run.Client.ID,
		run.AuthMode,
		run.Platform.OS,
		run.Platform.Arch,
	}
	return strings.Join(parts, "\x00")
}

func ReadFile(path string) (Artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var value Artifact
	if err := decoder.Decode(&value); err != nil {
		return Artifact{}, fmt.Errorf("decode artifact: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Artifact{}, errors.New("decode artifact: multiple JSON values")
		}
		return Artifact{}, fmt.Errorf("decode artifact trailing data: %w", err)
	}
	if err := ValidateArtifact(value); err != nil {
		return Artifact{}, err
	}
	return value, nil
}

func WriteFile(path string, value Artifact) error {
	if path == "" || path == "-" {
		return errors.New("artifact output must be a file path")
	}
	if err := ValidateArtifact(value); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		_ = file.Close()
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
