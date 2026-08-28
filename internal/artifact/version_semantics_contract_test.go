package artifact

import "testing"

func TestV1CandidateComparisonIdentityDoesNotUseClientVersion(t *testing.T) {
	runA := Run{Endpoint: EndpointIdentity{Fingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Client: ClientIdentity{ID: "codex", Version: "1.0"}, AuthMode: "default", Platform: Platform{OS: "darwin", Arch: "arm64"}}
	runB := runA
	runB.Client.Version = "2.0"
	if ComparisonKeyForSchema(runA, SchemaVersionV2) != ComparisonKeyForSchema(runB, SchemaVersionV2) {
		t.Fatal("client-version-only change became a different comparison identity")
	}
}
