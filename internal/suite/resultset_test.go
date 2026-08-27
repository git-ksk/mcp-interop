package suite

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResultIndexContainsNoEndpointFields(t *testing.T) {
	manifest := Manifest{
		SchemaVersion:    SchemaVersionV1,
		ExecutionContext: ExecutionTrusted,
		Targets: []Target{{
			ID:           "production-a",
			Endpoint:     EndpointReference{Source: EndpointEnvironment, Variable: "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"},
			DeploymentID: "production-a",
			Clients:      []ClientSelection{{ID: "codex", Auth: AuthNone}},
		}},
	}
	index, err := NewResultIndex(manifest, []ResultEntry{{
		TargetID: "production-a",
		ClientID: "codex",
		AuthMode: AuthNone,
		Outcome:  OutcomePass,
		ExitCode: 0,
		Artifact: "artifacts/production-a--codex--none.json",
	}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"endpoint", "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("suite index leaked %q: %s", forbidden, data)
		}
	}
}

func TestResultIndexRejectsOutOfOrderEntries(t *testing.T) {
	index := ResultIndex{
		SchemaVersion:         ResultSetSchemaVersion,
		ArtifactType:          ResultSetArtifactType,
		ManifestSchemaVersion: SchemaVersionV1,
		ManifestFingerprint:   "sha256:" + strings.Repeat("a", 64),
		ExecutionContext:      ExecutionTrusted,
		ArtifactSchemaVersion: 2,
		Runs: []ResultEntry{
			{TargetID: "zeta", ClientID: "codex", AuthMode: AuthNone, Outcome: OutcomePass, ExitCode: 0, Artifact: "artifacts/zeta--codex--none.json"},
			{TargetID: "alpha", ClientID: "codex", AuthMode: AuthNone, Outcome: OutcomePass, ExitCode: 0, Artifact: "artifacts/alpha--codex--none.json"},
		},
	}
	if err := ValidateResultIndex(index); err == nil || !strings.Contains(err.Error(), "deterministic order") {
		t.Fatalf("expected deterministic-order failure, got %v", err)
	}
}

func TestResultIndexRepresentsExecutionErrorWithoutArtifact(t *testing.T) {
	index := ResultIndex{
		SchemaVersion:         ResultSetSchemaVersion,
		ArtifactType:          ResultSetArtifactType,
		ManifestSchemaVersion: SchemaVersionV1,
		ManifestFingerprint:   "sha256:" + strings.Repeat("b", 64),
		ExecutionContext:      ExecutionTrusted,
		ArtifactSchemaVersion: 2,
		Runs: []ResultEntry{{
			TargetID: "production-a",
			ClientID: "cursor",
			AuthMode: AuthNone,
			Outcome:  OutcomeError,
			ExitCode: 1,
		}},
	}
	if err := ValidateResultIndex(index); err != nil {
		t.Fatal(err)
	}
}
