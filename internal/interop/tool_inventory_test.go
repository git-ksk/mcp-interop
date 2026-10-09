package interop

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestObservedToolNamesRequiresDirectSuccessfulStage(t *testing.T) {
	r := NewResult("codex", "Codex", "1.0", "https://example.test/mcp")
	if r.SetObservedToolNames([]string{"alpha"}) {
		t.Fatal("unknown tools stage must not accept inventory")
	}
	if _, known := r.ObservedToolNames(); known {
		t.Fatal("unknown tools stage must leave inventory unknown")
	}
	r.Set(StageTools, StatusPass, "client listed tools")
	if !r.SetObservedToolNames([]string{"alpha"}) {
		t.Fatal("successful real-client stage must accept valid names")
	}
}

func TestObservedToolNamesSortedCopiedAndNotSerialized(t *testing.T) {
	r := NewResult("codex", "Codex", "1.0", "https://example.test/mcp")
	r.Set(StageTools, StatusPass, "real-client inventory")
	input := []string{"zeta", "あいう", "alpha"}
	if !r.SetObservedToolNames(input) {
		t.Fatal("valid inventory rejected")
	}
	input[0] = "modified"
	got, known := r.ObservedToolNames()
	want := []string{"alpha", "zeta", "あいう"}
	if !known || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, known=%v, want %q", got, known, want)
	}
	got[0] = "modified"
	got2, _ := r.ObservedToolNames()
	if !reflect.DeepEqual(got2, want) {
		t.Fatalf("getter returned mutable storage: %q", got2)
	}
	encoded, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "alpha") || strings.Contains(string(encoded), "observedTool") {
		t.Fatalf("private inventory was serialized: %s", encoded)
	}
}

func TestObservedToolNamesInvalidInputsFailClosed(t *testing.T) {
	tests := map[string][]string{
		"duplicate":    {"same", "same"},
		"empty":        {""},
		"space":        {" leading"},
		"control":      {"x\ny"},
		"invalid utf8": {string([]byte{0xff})},
		"too long":     {strings.Repeat("x", maxObservedToolNameBytes+1)},
		"too many":     make([]string, maxObservedToolNames+1),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			r := NewResult("codex", "Codex", "1.0", "https://example.test")
			r.Set(StageTools, StatusPass, "client listed tools")
			if r.SetObservedToolNames(input) {
				t.Fatal("invalid inventory accepted")
			}
			if _, known := r.ObservedToolNames(); known {
				t.Fatal("rejected inventory must remain unknown")
			}
		})
	}
}

func TestObservedToolNamesEmptyIsKnownOnlyWhenExplicitlySet(t *testing.T) {
	r := NewResult("test", "test", "1", "https://example.test")
	r.Set(StageTools, StatusPass, "client explicitly established zero tools")
	if !r.SetObservedToolNames(nil) {
		t.Fatal("an explicit proved zero-tool inventory must be representable")
	}
	tools, known := r.ObservedToolNames()
	if !known || len(tools) != 0 {
		t.Fatalf("got %v, known=%v", tools, known)
	}
}

func TestObservedToolNamesRejectedReplacementClearsPreviousEvidence(t *testing.T) {
	r := NewResult("codex", "Codex", "1", "https://example.test")
	for _, stage := range OrderedStages {
		r.Set(stage, StatusPass, "direct real-client success")
	}
	if !r.SetObservedToolNames([]string{"alpha"}) {
		t.Fatal("valid initial inventory rejected")
	}
	if r.SetObservedToolNames([]string{"duplicate", "duplicate"}) {
		t.Fatal("invalid replacement accepted")
	}
	if names, known := r.ObservedToolNames(); known {
		t.Fatalf("rejected replacement retained stale evidence: %q", names)
	}
	if !r.Passed() {
		t.Fatal("optional inventory rejection changed the core PASS status")
	}
}
