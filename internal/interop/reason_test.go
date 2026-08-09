package interop

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSetWithReasonAddsMachineReadableCode(t *testing.T) {
	result := NewResult("codex", "Codex CLI", "test", "https://example.com/mcp")
	if !result.SetWithReason(StageAuth, StatusFail, ReasonDCRUnsupported, "DCR is not supported") {
		t.Fatal("expected auth stage to update")
	}

	auth, ok := result.Get(StageAuth)
	if !ok {
		t.Fatal("missing auth stage")
	}
	if auth.ReasonCode != ReasonDCRUnsupported {
		t.Fatalf("reason code = %q, want %q", auth.ReasonCode, ReasonDCRUnsupported)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"reason_code":"DCR_UNSUPPORTED"`) {
		t.Fatalf("JSON does not contain reason_code: %s", encoded)
	}
}

func TestSetClearsPreviousReasonCode(t *testing.T) {
	result := NewResult("codex", "Codex CLI", "test", "https://example.com/mcp")
	result.SetWithReason(StageAuth, StatusFail, ReasonDCRFailed, "registration failed")
	result.Set(StageAuth, StatusPass, "authenticated")

	auth, ok := result.Get(StageAuth)
	if !ok {
		t.Fatal("missing auth stage")
	}
	if auth.ReasonCode != "" {
		t.Fatalf("stale reason code remained after replacement: %q", auth.ReasonCode)
	}
}
