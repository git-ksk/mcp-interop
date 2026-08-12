package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestNewEndpointIdentityDropsAllQueryValuesBeforeFingerprinting(t *testing.T) {
	raw := "https://EXAMPLE.com:8443/mcp?tenant=customer-a&opaque_super_secret=do-not-store#fragment"
	got, err := NewEndpointIdentity(raw)
	if err != nil {
		t.Fatal(err)
	}
	const wantIdentity = "https://example.com:8443/mcp"
	if got.Identity != wantIdentity {
		t.Fatalf("identity = %q, want %q", got.Identity, wantIdentity)
	}
	if strings.Contains(got.Identity, "customer-a") || strings.Contains(got.Identity, "do-not-store") {
		t.Fatalf("identity persisted query value: %q", got.Identity)
	}
	sum := sha256.Sum256([]byte(wantIdentity))
	wantFingerprint := "sha256:" + hex.EncodeToString(sum[:])
	if got.Fingerprint != wantFingerprint {
		t.Fatalf("fingerprint = %q, want %q", got.Fingerprint, wantFingerprint)
	}
}

func TestNewRunPreservesStageStatusAndReasonWithoutMessages(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "codex-cli 9.9.9", "https://example.com/mcp?opaque_secret=value")
	result.Set(interop.StageReach, interop.StatusPass, "contains detail that must not be copied")
	result.SetWithReason(interop.StageAuth, interop.StatusFail, interop.ReasonDCRUnsupported, "secret-ish detail")
	result.Set(interop.StageInit, interop.StatusUnknown, "unknown detail")
	result.Set(interop.StageTools, interop.StatusSkip, "skip detail")

	run, err := NewRun(
		result,
		time.Date(2026, 8, 13, 0, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		"oauth",
		EvidenceProvenance{Kind: ProvenanceRealClientAdapter, AdapterID: "codex"},
		"v-test",
		"deadbeef",
	)
	if err != nil {
		t.Fatal(err)
	}
	if run.ExecutedAt.Location() != time.UTC {
		t.Fatalf("executed_at location = %v, want UTC", run.ExecutedAt.Location())
	}
	if run.Endpoint.Identity != "https://example.com/mcp" {
		t.Fatalf("unexpected endpoint identity %q", run.Endpoint.Identity)
	}
	if got := run.Stages[1]; got.Status != interop.StatusFail || got.ReasonCode != interop.ReasonDCRUnsupported {
		t.Fatalf("auth stage not preserved: %#v", got)
	}
}

func TestValidateRunRejectsRunnerGeneratedPass(t *testing.T) {
	run := validRun(t)
	run.EvidenceProvenance = EvidenceProvenance{Kind: ProvenanceRunnerObservation}
	run.Client.Version = ""
	if err := ValidateRun(run); err == nil || !strings.Contains(err.Error(), "cannot produce pass") {
		t.Fatalf("expected runner PASS rejection, got %v", err)
	}
}

func TestValidateArtifactRejectsDuplicateComparisonIdentity(t *testing.T) {
	run := validRun(t)
	value := NewArtifact([]Run{run, run})
	if err := ValidateArtifact(value); err == nil || !strings.Contains(err.Error(), "duplicate comparison identity") {
		t.Fatalf("expected duplicate identity rejection, got %v", err)
	}
}

func TestReadFileRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.json")
	content := `{
  "schema_version": 1,
  "artifact_type": "mcp-interop/live-results",
  "runs": [],
  "access_token": "must-not-be-accepted"
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}
}

func TestWriteReadFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	want := NewArtifact([]Run{validRun(t)})
	if err := WriteFile(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != SchemaVersion || got.ArtifactType != ArtifactType || len(got.Runs) != 1 {
		t.Fatalf("unexpected round-trip value: %#v", got)
	}
	if got.Runs[0].Endpoint != want.Runs[0].Endpoint {
		t.Fatalf("endpoint changed across round trip: %#v != %#v", got.Runs[0].Endpoint, want.Runs[0].Endpoint)
	}
}

func validRun(t *testing.T) Run {
	t.Helper()
	result := interop.NewResult("codex", "Codex CLI", "codex-cli 1.0.0", "https://example.com/mcp")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}
	run, err := NewRun(
		result,
		time.Date(2026, 8, 12, 15, 0, 0, 0, time.UTC),
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
