package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	EndpointIdentityDeploymentID = "deployment_id"
	maxDeploymentIDBytes         = 128
)

var deploymentIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// NewArtifactV2 creates a schema-v2 artifact. V2 currently supports the
// explicit deployment-id identity mode used for protected-path endpoints.
func NewArtifactV2(runs []Run) Artifact {
	return Artifact{
		SchemaVersion: SchemaVersionV2,
		ArtifactType:  ArtifactType,
		Runs:          runs,
	}
}

// ValidateDeploymentID checks syntax only. The operator remains responsible
// for choosing a stable, non-secret identifier that is not derived from a
// credential-bearing endpoint path.
func ValidateDeploymentID(value string) error {
	if value == "" {
		return errors.New("deployment id is required")
	}
	if len(value) > maxDeploymentIDBytes {
		return fmt.Errorf("deployment id exceeds %d bytes", maxDeploymentIDBytes)
	}
	if strings.TrimSpace(value) != value {
		return errors.New("deployment id must not have surrounding whitespace")
	}
	if !deploymentIDPattern.MatchString(value) {
		return errors.New("deployment id must use only ASCII letters, digits, '.', '_', and '-'")
	}
	return nil
}

// NewProtectedEndpointIdentity builds a v2 endpoint identity without reading,
// hashing, or persisting the endpoint path, query, userinfo, or fragment.
func NewProtectedEndpointIdentity(raw, deploymentID string) (EndpointIdentity, error) {
	if err := ValidateDeploymentID(deploymentID); err != nil {
		return EndpointIdentity{}, err
	}
	origin, err := canonicalOrigin(raw)
	if err != nil {
		return EndpointIdentity{}, err
	}
	return EndpointIdentity{
		Identity:     deploymentID,
		Fingerprint:  deploymentIDFingerprint(origin, deploymentID),
		IdentityKind: EndpointIdentityDeploymentID,
		Origin:       origin,
	}, nil
}

// NewRunV2ProtectedPath projects one live Result into schema v2. rawEndpoint is
// used only to derive the public origin; its path and query are never hashed or
// persisted in the artifact.
func NewRunV2ProtectedPath(result interop.Result, rawEndpoint, deploymentID string, executedAt time.Time, authMode string, provenance EvidenceProvenance, runnerVersion, runnerCommit string) (Run, error) {
	endpoint, err := NewProtectedEndpointIdentity(rawEndpoint, deploymentID)
	if err != nil {
		return Run{}, err
	}
	return newRunWithEndpoint(result, endpoint, executedAt, authMode, provenance, runnerVersion, runnerCommit, SchemaVersionV2)
}

func ValidateRunV2ProtectedPath(run Run) error {
	if err := validateRunCommon(run); err != nil {
		return err
	}
	if run.Endpoint.IdentityKind != EndpointIdentityDeploymentID {
		return fmt.Errorf("unsupported v2 endpoint identity_kind %q", run.Endpoint.IdentityKind)
	}
	if err := ValidateDeploymentID(run.Endpoint.Identity); err != nil {
		return fmt.Errorf("endpoint identity: %w", err)
	}
	origin, err := canonicalOrigin(run.Endpoint.Origin)
	if err != nil {
		return fmt.Errorf("endpoint origin: %w", err)
	}
	if origin != run.Endpoint.Origin {
		return errors.New("endpoint origin is not canonical")
	}
	if run.Endpoint.Fingerprint != deploymentIDFingerprint(run.Endpoint.Origin, run.Endpoint.Identity) {
		return errors.New("endpoint fingerprint does not match origin and deployment id")
	}
	return nil
}

func deploymentIDFingerprint(origin, deploymentID string) string {
	sum := sha256.Sum256([]byte("deployment_id\x00" + origin + "\x00" + deploymentID))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func canonicalOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse endpoint: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("endpoint must use http or https")
	}
	if u.User != nil {
		return "", errors.New("endpoint must not embed user info")
	}
	if u.Fragment != "" {
		return "", errors.New("endpoint must not include a URL fragment")
	}
	hostname := strings.ToLower(u.Hostname())
	if hostname == "" {
		return "", errors.New("endpoint must include a hostname")
	}
	host := hostname
	if port := u.Port(); port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	return strings.ToLower(u.Scheme) + "://" + host, nil
}
