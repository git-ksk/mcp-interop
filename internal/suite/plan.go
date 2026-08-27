package suite

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

// EndpointLookup resolves a trusted suite endpoint outside the manifest. The
// returned value may contain protected path material and must not be persisted
// by suite planning or index generation.
type EndpointLookup func(name string) (string, bool)

// PlannedRun is one deterministic trusted-suite execution. Endpoint is kept in
// memory only and is deliberately excluded from every portable suite schema.
type PlannedRun struct {
	TargetID     string
	DeploymentID string
	Client       ClientSelection
	Endpoint     string
}

// ResolveTrusted validates and expands a trusted suite before the first client
// is launched. All endpoint references are resolved up front so a missing or
// invalid environment value cannot cause a partially started privileged suite.
func ResolveTrusted(manifest Manifest, lookup EndpointLookup) ([]PlannedRun, error) {
	if err := Validate(manifest); err != nil {
		return nil, err
	}
	if manifest.ExecutionContext != ExecutionTrusted {
		return nil, errors.New("suite execution currently requires trusted_real_client execution_context")
	}
	if lookup == nil {
		return nil, errors.New("endpoint lookup is required")
	}

	planned := make([]PlannedRun, 0, RunCount(manifest))
	for _, target := range manifest.Targets {
		endpoint, ok := lookup(target.Endpoint.Variable)
		if !ok || endpoint == "" {
			return nil, fmt.Errorf("target %q: endpoint environment variable is not set", target.ID)
		}
		if strings.TrimSpace(endpoint) != endpoint {
			return nil, fmt.Errorf("target %q: endpoint environment value has surrounding whitespace", target.ID)
		}
		if err := (interop.Target{Endpoint: endpoint}).Validate(); err != nil {
			return nil, fmt.Errorf("target %q: invalid resolved endpoint: %w", target.ID, err)
		}
		if _, err := artifact.NewProtectedEndpointIdentity(endpoint, target.DeploymentID); err != nil {
			return nil, fmt.Errorf("target %q: invalid protected endpoint identity: %w", target.ID, err)
		}
		for _, selection := range target.Clients {
			planned = append(planned, PlannedRun{
				TargetID:     target.ID,
				DeploymentID: target.DeploymentID,
				Client:       selection,
				Endpoint:     endpoint,
			})
		}
	}

	sort.Slice(planned, func(i, j int) bool {
		left, right := planned[i], planned[j]
		if left.TargetID != right.TargetID {
			return left.TargetID < right.TargetID
		}
		if left.Client.ID != right.Client.ID {
			return left.Client.ID < right.Client.ID
		}
		return left.Client.Auth < right.Client.Auth
	})
	return planned, nil
}

// ManifestFingerprint identifies the exact validated declaration without
// including any runtime endpoint value.
func ManifestFingerprint(manifest Manifest) (string, error) {
	if err := Validate(manifest); err != nil {
		return "", err
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("encode manifest fingerprint input: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
