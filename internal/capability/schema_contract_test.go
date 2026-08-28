package capability

import "testing"

func TestV1CandidateCapabilitySchemaIdentityRemainsStable(t *testing.T) {
	if SchemaVersion != 1 || ArtifactType != "mcp-interop/capability-profile" {
		t.Fatalf("capability schema identity changed: version=%d type=%q", SchemaVersion, ArtifactType)
	}
}
