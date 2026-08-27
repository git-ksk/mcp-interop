package suite

import (
	"strings"
	"testing"
)

func TestResolveTrustedExpandsDeterministicallyWithoutPersistingEndpoint(t *testing.T) {
	manifest := Manifest{
		SchemaVersion:    SchemaVersionV1,
		ExecutionContext: ExecutionTrusted,
		Targets: []Target{
			{
				ID:           "zeta",
				Endpoint:     EndpointReference{Source: EndpointEnvironment, Variable: "MCP_INTEROP_SUITE_ENDPOINT_ZETA"},
				DeploymentID: "zeta",
				Clients:      []ClientSelection{{ID: "cursor", Auth: AuthNone}, {ID: "codex", Auth: AuthOAuth}},
			},
			{
				ID:           "alpha",
				Endpoint:     EndpointReference{Source: EndpointEnvironment, Variable: "MCP_INTEROP_SUITE_ENDPOINT_ALPHA"},
				DeploymentID: "alpha",
				Clients:      []ClientSelection{{ID: "antigravity", Auth: AuthNone}},
			},
		},
	}
	values := map[string]string{
		"MCP_INTEROP_SUITE_ENDPOINT_ZETA":  "https://example.com/mcp/secret-zeta",
		"MCP_INTEROP_SUITE_ENDPOINT_ALPHA": "https://example.net/mcp/secret-alpha",
	}
	planned, err := ResolveTrusted(manifest, func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(planned) != 3 {
		t.Fatalf("planned runs = %d", len(planned))
	}
	got := []string{
		planned[0].TargetID + "/" + planned[0].Client.ID,
		planned[1].TargetID + "/" + planned[1].Client.ID,
		planned[2].TargetID + "/" + planned[2].Client.ID,
	}
	want := []string{"alpha/antigravity", "zeta/codex", "zeta/cursor"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	fingerprint, err := ManifestFingerprint(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret-zeta", "secret-alpha"} {
		if strings.Contains(fingerprint, secret) {
			t.Fatalf("manifest fingerprint leaked endpoint material %q", secret)
		}
	}
}

func TestResolveTrustedFailsBeforeExecutionInputsArePartial(t *testing.T) {
	manifest := Manifest{
		SchemaVersion:    SchemaVersionV1,
		ExecutionContext: ExecutionTrusted,
		Targets: []Target{
			{
				ID:           "alpha",
				Endpoint:     EndpointReference{Source: EndpointEnvironment, Variable: "MCP_INTEROP_SUITE_ENDPOINT_ALPHA"},
				DeploymentID: "alpha",
				Clients:      []ClientSelection{{ID: "codex", Auth: AuthNone}},
			},
			{
				ID:           "beta",
				Endpoint:     EndpointReference{Source: EndpointEnvironment, Variable: "MCP_INTEROP_SUITE_ENDPOINT_BETA"},
				DeploymentID: "beta",
				Clients:      []ClientSelection{{ID: "cursor", Auth: AuthNone}},
			},
		},
	}
	_, err := ResolveTrusted(manifest, func(name string) (string, bool) {
		if name == "MCP_INTEROP_SUITE_ENDPOINT_ALPHA" {
			return "https://example.com/mcp/alpha-secret", true
		}
		return "", false
	})
	if err == nil || !strings.Contains(err.Error(), `target "beta"`) {
		t.Fatalf("expected missing beta endpoint error, got %v", err)
	}
	if strings.Contains(err.Error(), "alpha-secret") {
		t.Fatalf("resolution error leaked another endpoint: %v", err)
	}
}

func TestResolveTrustedRejectsHostedFixtureExecution(t *testing.T) {
	manifest := Manifest{
		SchemaVersion:    SchemaVersionV1,
		ExecutionContext: ExecutionHosted,
		Targets: []Target{{
			ID:       "fixture-a",
			Endpoint: EndpointReference{Source: EndpointFixture},
			Clients:  []ClientSelection{{ID: "codex", Auth: AuthNone}},
		}},
	}
	if _, err := ResolveTrusted(manifest, func(string) (string, bool) { return "", false }); err == nil {
		t.Fatal("expected hosted fixture suite execution to remain gated")
	}
}
