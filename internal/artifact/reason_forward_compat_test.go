package artifact

import (
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestV1CandidateArtifactAcceptsUnknownFutureReasonCode(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "future-version", "https://example.com/mcp")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}
	result.SetWithReason(interop.StageAuth, interop.StatusFail, interop.ReasonCode("FUTURE_DIRECT_EVIDENCE_CODE"), "classified by newer producer")
	run, err := NewRun(result, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), "default", EvidenceProvenance{Kind: ProvenanceRealClientAdapter, AdapterID: "codex"}, "v1-test", "commit")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifact(NewArtifact([]Run{run})); err != nil {
		t.Fatalf("older artifact validation rejected open-enum reason code: %v", err)
	}
}
