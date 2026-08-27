package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestNewProtectedEndpointIdentityNeverPersistsOrHashesProtectedPath(t *testing.T) {
	const secret = "path-secret-low-entropy"
	got, err := NewProtectedEndpointIdentity(
		"https://EXAMPLE.com:8443/mcp/"+secret+"?token=query-secret",
		"production-a",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.IdentityKind != EndpointIdentityDeploymentID || got.Identity != "production-a" {
		t.Fatalf("unexpected protected identity: %#v", got)
	}
	if got.Origin != "https://example.com:8443" {
		t.Fatalf("origin = %q, want canonical public origin", got.Origin)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	secretSum := sha256.Sum256([]byte(secret))
	for _, forbidden := range []string{
		secret,
		"query-secret",
		hex.EncodeToString(secretSum[:]),
		"/mcp/",
	} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("protected endpoint persisted secret-derived material %q: %s", forbidden, data)
		}
	}
}

func TestNewRunV2ProtectedPathRoundTrip(t *testing.T) {
	const secret = "opaque-protected-capability"
	result := interop.NewResult("codex", "Codex CLI", "codex-cli 2.0.0", "https://example.com/mcp/"+secret)
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "detail must not be copied")
	}
	run, err := NewRunV2ProtectedPath(
		result,
		result.Endpoint,
		"production-a",
		time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC),
		"default",
		EvidenceProvenance{Kind: ProvenanceRealClientAdapter, AdapterID: "codex"},
		"v-test",
		"deadbeef",
	)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v2.json")
	if err := WriteFile(path, NewArtifactV2([]Run{run})); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret) || strings.Contains(string(data), "/mcp/") {
		t.Fatalf("v2 artifact leaked protected path: %s", data)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != SchemaVersionV2 || got.Runs[0].Endpoint.Identity != "production-a" {
		t.Fatalf("unexpected v2 round trip: %#v", got)
	}
}

func TestValidateArtifactV1RejectsV2EndpointFields(t *testing.T) {
	run := validRun(t)
	run.Endpoint.IdentityKind = EndpointIdentityDeploymentID
	run.Endpoint.Origin = "https://example.com"
	if err := ValidateArtifact(NewArtifact([]Run{run})); err == nil || !strings.Contains(err.Error(), "v1 endpoint") {
		t.Fatalf("expected v1/v2 field separation error, got %v", err)
	}
}

func TestValidateArtifactV2RejectsPathInOrigin(t *testing.T) {
	run := validRunV2(t, "production-a")
	run.Endpoint.Origin = "https://example.com/mcp/secret"
	if err := ValidateArtifact(NewArtifactV2([]Run{run})); err == nil || !strings.Contains(err.Error(), "origin is not canonical") {
		t.Fatalf("expected protected origin validation error, got %v", err)
	}
}

func TestValidateArtifactV2RejectsDuplicateDeploymentIdentity(t *testing.T) {
	run := validRunV2(t, "production-a")
	if err := ValidateArtifact(NewArtifactV2([]Run{run, run})); err == nil || !strings.Contains(err.Error(), "duplicate comparison identity") {
		t.Fatalf("expected duplicate v2 identity rejection, got %v", err)
	}
}

func TestValidateDeploymentID(t *testing.T) {
	for _, value := range []string{"production-a", "prod_01", "A.B"} {
		if err := ValidateDeploymentID(value); err != nil {
			t.Fatalf("valid deployment id %q rejected: %v", value, err)
		}
	}
	for _, value := range []string{"", " production", "production/a", "production a", strings.Repeat("a", 129)} {
		if err := ValidateDeploymentID(value); err == nil {
			t.Fatalf("invalid deployment id %q accepted", value)
		}
	}
}

func validRunV2(t *testing.T, deploymentID string) Run {
	t.Helper()
	result := interop.NewResult("codex", "Codex CLI", "codex-cli 2.0.0", "https://example.com/mcp/protected-secret")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}
	run, err := NewRunV2ProtectedPath(
		result,
		result.Endpoint,
		deploymentID,
		time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC),
		"default",
		EvidenceProvenance{Kind: ProvenanceRealClientAdapter, AdapterID: "codex"},
		"v-test",
		"deadbeef",
	)
	if err != nil {
		t.Fatal(err)
	}
	return run
}
