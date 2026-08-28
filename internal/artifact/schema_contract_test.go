package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV1CandidateLiveArtifactSchemaIdentitiesRemainStable(t *testing.T) {
	if SchemaVersionV1 != 1 || SchemaVersionV2 != 2 || SchemaVersion != SchemaVersionV1 {
		t.Fatalf("live artifact schema versions changed: v1=%d v2=%d default=%d", SchemaVersionV1, SchemaVersionV2, SchemaVersion)
	}
	if ArtifactType != "mcp-interop/live-results" {
		t.Fatalf("live artifact type changed: %q", ArtifactType)
	}
}

func TestV1CandidateLiveArtifactSameSchemaRejectsUnknownStructuralField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.json")
	input := `{"schema_version":1,"artifact_type":"mcp-interop/live-results","future_required_field":true,"runs":[]}`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadFile(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("same-schema unknown field must fail closed, got %v", err)
	}
}
