package suite

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/client"
)

const (
	SchemaVersionV1     = 1
	maxManifestBytes    = 1 << 20
	maxTargets          = 64
	endpointEnvPrefix   = "MCP_INTEROP_SUITE_ENDPOINT_"
	ExecutionHosted     = ExecutionContext("hosted_fixture")
	ExecutionTrusted    = ExecutionContext("trusted_real_client")
	EndpointFixture     = EndpointSource("fixture")
	EndpointEnvironment = EndpointSource("environment")
	AuthNone            = AuthMode("none")
	AuthOAuth           = AuthMode("oauth")
)

var targetIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// ExecutionContext controls which endpoint/auth inputs a suite may request.
type ExecutionContext string

// EndpointSource identifies how an endpoint will be resolved at execution
// time. Endpoint URL values are deliberately absent from the manifest schema.
type EndpointSource string

// AuthMode is explicit per-client authentication intent. Ambient credentials
// never turn a no-auth entry into OAuth.
type AuthMode string

// Manifest is the versioned, secret-safe declaration for repeatable suites.
type Manifest struct {
	SchemaVersion    int              `json:"schema_version"`
	ExecutionContext ExecutionContext `json:"execution_context"`
	Targets          []Target         `json:"targets"`
}

// Target selects one logical Remote MCP deployment and the clients to run.
type Target struct {
	ID           string            `json:"id"`
	Endpoint     EndpointReference `json:"endpoint"`
	DeploymentID string            `json:"deployment_id,omitempty"`
	Clients      []ClientSelection `json:"clients"`
}

// EndpointReference contains only a safe resolver description. Real endpoint
// URL values are supplied outside the manifest at execution time.
type EndpointReference struct {
	Source   EndpointSource `json:"source"`
	Variable string         `json:"variable,omitempty"`
}

// ClientSelection chooses a shipped live adapter and explicit auth mode.
type ClientSelection struct {
	ID   string   `json:"id"`
	Auth AuthMode `json:"auth"`
}

// Parse decodes exactly one strict manifest and validates its safety contract.
func Parse(r io.Reader) (Manifest, error) {
	if r == nil {
		return Manifest{}, errors.New("manifest reader is required")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxManifestBytes+1))
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	if len(data) > maxManifestBytes {
		return Manifest{}, fmt.Errorf("manifest exceeds %d bytes", maxManifestBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Manifest{}, err
	}
	if err := Validate(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// ReadFile reads and validates a manifest without resolving endpoint values.
func ReadFile(path string) (Manifest, error) {
	if strings.TrimSpace(path) == "" {
		return Manifest{}, errors.New("manifest path is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open manifest: %w", err)
	}
	defer file.Close()
	return Parse(file)
}

// Validate enforces the v1 manifest trust and secret-safety boundary.
func Validate(manifest Manifest) error {
	if manifest.SchemaVersion != SchemaVersionV1 {
		return fmt.Errorf("unsupported suite schema_version %d", manifest.SchemaVersion)
	}
	if manifest.ExecutionContext != ExecutionHosted && manifest.ExecutionContext != ExecutionTrusted {
		return fmt.Errorf("unsupported execution_context %q", manifest.ExecutionContext)
	}
	if len(manifest.Targets) == 0 {
		return errors.New("suite requires at least one target")
	}
	if len(manifest.Targets) > maxTargets {
		return fmt.Errorf("suite exceeds %d targets", maxTargets)
	}

	targetIDs := make(map[string]struct{}, len(manifest.Targets))
	deploymentIDs := make(map[string]struct{}, len(manifest.Targets))
	for index, target := range manifest.Targets {
		if err := validateTarget(manifest.ExecutionContext, target); err != nil {
			return fmt.Errorf("target[%d]: %w", index, err)
		}
		if _, duplicate := targetIDs[target.ID]; duplicate {
			return fmt.Errorf("target[%d]: duplicate target id %q", index, target.ID)
		}
		targetIDs[target.ID] = struct{}{}
		if target.DeploymentID != "" {
			if _, duplicate := deploymentIDs[target.DeploymentID]; duplicate {
				return fmt.Errorf("target[%d]: duplicate deployment_id %q", index, target.DeploymentID)
			}
			deploymentIDs[target.DeploymentID] = struct{}{}
		}
	}
	return nil
}

// EndpointEnvName returns the only environment-variable name a trusted target
// may reference. This prevents a manifest from selecting arbitrary runner vars.
func EndpointEnvName(targetID string) (string, error) {
	if err := validateTargetID(targetID); err != nil {
		return "", err
	}
	name := strings.ToUpper(strings.ReplaceAll(targetID, "-", "_"))
	return endpointEnvPrefix + name, nil
}

// RunCount returns the number of declared target/client executions.
func RunCount(manifest Manifest) int {
	count := 0
	for _, target := range manifest.Targets {
		count += len(target.Clients)
	}
	return count
}

func validateTarget(context ExecutionContext, target Target) error {
	if err := validateTargetID(target.ID); err != nil {
		return fmt.Errorf("id: %w", err)
	}
	if len(target.Clients) == 0 {
		return errors.New("at least one client is required")
	}

	switch context {
	case ExecutionHosted:
		if target.Endpoint.Source != EndpointFixture {
			return errors.New("hosted_fixture targets must use endpoint source fixture")
		}
		if target.Endpoint.Variable != "" {
			return errors.New("fixture endpoint must not declare an environment variable")
		}
		if target.DeploymentID != "" {
			return errors.New("hosted_fixture target must not declare deployment_id")
		}
	case ExecutionTrusted:
		if target.Endpoint.Source != EndpointEnvironment {
			return errors.New("trusted_real_client targets must use endpoint source environment")
		}
		expected, _ := EndpointEnvName(target.ID)
		if target.Endpoint.Variable != expected {
			return fmt.Errorf("endpoint variable must be %q", expected)
		}
		if err := artifact.ValidateDeploymentID(target.DeploymentID); err != nil {
			return fmt.Errorf("deployment_id: %w", err)
		}
	}

	seenClients := make(map[string]struct{}, len(target.Clients))
	for index, selection := range target.Clients {
		if err := validateClientSelection(context, selection); err != nil {
			return fmt.Errorf("client[%d]: %w", index, err)
		}
		if _, duplicate := seenClients[selection.ID]; duplicate {
			return fmt.Errorf("client[%d]: duplicate client id %q", index, selection.ID)
		}
		seenClients[selection.ID] = struct{}{}
	}
	return nil
}

func validateTargetID(value string) error {
	if !targetIDPattern.MatchString(value) {
		return errors.New("must use 1-63 lowercase ASCII letters, digits, and internal '-' characters")
	}
	return nil
}

func validateClientSelection(context ExecutionContext, selection ClientSelection) error {
	shipped, err := client.IsShippedLiveAdapter(selection.ID)
	if err != nil {
		return fmt.Errorf("live adapter graduation policy is invalid: %w", err)
	}
	if !shipped {
		return fmt.Errorf("unsupported live client %q", selection.ID)
	}
	switch selection.Auth {
	case AuthNone:
		return nil
	case AuthOAuth:
		if context != ExecutionTrusted {
			return errors.New("oauth requires trusted_real_client execution_context")
		}
		return nil
	default:
		return fmt.Errorf("unsupported auth mode %q", selection.Auth)
	}
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing manifest data: %w", err)
	}
	return errors.New("manifest must contain exactly one JSON document")
}
