package suite

import (
	"strings"
	"testing"
)

func TestParseValidHostedFixtureManifest(t *testing.T) {
	manifest := parseManifest(t, `{
  "schema_version": 1,
  "execution_context": "hosted_fixture",
  "targets": [{
    "id": "fixture-a",
    "endpoint": {"source": "fixture"},
    "clients": [
      {"id": "codex", "auth": "none"},
      {"id": "cursor", "auth": "none"}
    ]
  }]
}`)
	if manifest.ExecutionContext != ExecutionHosted || RunCount(manifest) != 2 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}

func TestParseValidTrustedOAuthManifest(t *testing.T) {
	manifest := parseManifest(t, `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [{
    "id": "production-a",
    "endpoint": {
      "source": "environment",
      "variable": "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"
    },
    "deployment_id": "production-a",
    "clients": [{"id": "antigravity", "auth": "oauth"}]
  }]
}`)
	if manifest.Targets[0].DeploymentID != "production-a" || RunCount(manifest) != 1 {
		t.Fatalf("unexpected trusted manifest: %#v", manifest)
	}
}

func TestParseRejectsUnknownSecretBearingField(t *testing.T) {
	assertParseErrorContains(t, `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "token": "must-not-be-accepted",
  "targets": []
}`, `unknown field "token"`)
}

func TestParseRejectsLiteralEndpointField(t *testing.T) {
	assertParseErrorContains(t, `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [{
    "id": "production-a",
    "endpoint": {"source": "environment", "url": "https://example.com/mcp/secret"},
    "deployment_id": "production-a",
    "clients": [{"id": "codex", "auth": "none"}]
  }]
}`, `unknown field "url"`)
}

func TestParseRejectsArbitraryEnvironmentVariable(t *testing.T) {
	assertParseErrorContains(t, `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [{
    "id": "production-a",
    "endpoint": {"source": "environment", "variable": "GITHUB_TOKEN"},
    "deployment_id": "production-a",
    "clients": [{"id": "codex", "auth": "none"}]
  }]
}`, `endpoint variable must be "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"`)
}

func TestParseRejectsOAuthInHostedFixtureContext(t *testing.T) {
	assertParseErrorContains(t, `{
  "schema_version": 1,
  "execution_context": "hosted_fixture",
  "targets": [{
    "id": "fixture-a",
    "endpoint": {"source": "fixture"},
    "clients": [{"id": "cursor", "auth": "oauth"}]
  }]
}`, "oauth requires trusted_real_client")
}

func TestParseRejectsUnknownClient(t *testing.T) {
	assertParseErrorContains(t, `{
  "schema_version": 1,
  "execution_context": "hosted_fixture",
  "targets": [{
    "id": "fixture-a",
    "endpoint": {"source": "fixture"},
    "clients": [{"id": "future-client", "auth": "none"}]
  }]
}`, `unsupported live client "future-client"`)
}

func TestParseRejectsTrustedTargetWithoutDeploymentID(t *testing.T) {
	assertParseErrorContains(t, `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [{
    "id": "production-a",
    "endpoint": {
      "source": "environment",
      "variable": "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"
    },
    "clients": [{"id": "codex", "auth": "none"}]
  }]
}`, "deployment id is required")
}

func TestParseRejectsDuplicateTargetsAndClients(t *testing.T) {
	assertParseErrorContains(t, `{
  "schema_version": 1,
  "execution_context": "hosted_fixture",
  "targets": [{
    "id": "fixture-a",
    "endpoint": {"source": "fixture"},
    "clients": [
      {"id": "codex", "auth": "none"},
      {"id": "codex", "auth": "none"}
    ]
  }]
}`, `duplicate client id "codex"`)

	assertParseErrorContains(t, `{
  "schema_version": 1,
  "execution_context": "hosted_fixture",
  "targets": [
    {"id": "fixture-a", "endpoint": {"source": "fixture"}, "clients": [{"id": "codex", "auth": "none"}]},
    {"id": "fixture-a", "endpoint": {"source": "fixture"}, "clients": [{"id": "cursor", "auth": "none"}]}
  ]
}`, `duplicate target id "fixture-a"`)
}

func TestParseRejectsTrailingDocument(t *testing.T) {
	assertParseErrorContains(t, `{"schema_version":1,"execution_context":"hosted_fixture","targets":[{"id":"fixture-a","endpoint":{"source":"fixture"},"clients":[{"id":"codex","auth":"none"}]}]} {}`, "exactly one JSON document")
}

func TestEndpointEnvNameIsDeterministicAndRestricted(t *testing.T) {
	got, err := EndpointEnvName("production-a")
	if err != nil {
		t.Fatal(err)
	}
	if got != "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A" {
		t.Fatalf("unexpected env name: %s", got)
	}
	if _, err := EndpointEnvName("production_a"); err == nil {
		t.Fatal("underscore target id should be rejected to keep env mapping unambiguous")
	}
}

func parseManifest(t *testing.T, input string) Manifest {
	t.Helper()
	manifest, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func assertParseErrorContains(t *testing.T, input, want string) {
	t.Helper()
	_, err := Parse(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %v", want, err)
	}
}
