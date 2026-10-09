package interop

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func completeToolResult(t *testing.T, names []string) Result {
	t.Helper()
	r := NewResult("codex", "Codex CLI", "0.152.1", "https://example.test")
	for _, stage := range OrderedStages {
		r.Set(stage, StatusPass, "direct real-client success")
	}
	if !r.SetObservedToolNames(names) {
		t.Fatal("failed to set known inventory")
	}
	return r
}

func TestToolExpectationMatchesNamesAndCounts(t *testing.T) {
	result := completeToolResult(t, []string{"ping", "read_tool", "write_tool"})
	n := 3
	check := CheckToolExpectation(result, []string{"write_tool", "ping"}, &n)
	if check.Status != StatusPass || check.ReasonCode != ToolReasonMatched || len(check.MissingNames) != 0 {
		t.Fatalf("unexpected check: %#v", check)
	}
	if *check.ObservedCount != 3 || *check.ExpectedCount != 3 || !reflect.DeepEqual(check.ExpectedNames, []string{"ping", "write_tool"}) {
		t.Fatalf("unexpected expected/observed metadata: %#v", check)
	}
}

func TestToolExpectationMissingAndCountMismatch(t *testing.T) {
	result := completeToolResult(t, []string{"ping", "read_tool"})
	three := 3
	missing := CheckToolExpectation(result, []string{"write_tool", "ping", "new_tool"}, &three)
	if missing.Status != StatusFail || missing.ReasonCode != ToolReasonMissing || !reflect.DeepEqual(missing.MissingNames, []string{"new_tool", "write_tool"}) {
		t.Fatalf("missing names not deterministic: %#v", missing)
	}
	mismatch := CheckToolExpectation(result, nil, &three)
	if mismatch.Status != StatusFail || mismatch.ReasonCode != ToolReasonCountMismatch || *mismatch.ObservedCount != 2 {
		t.Fatalf("count mismatch not detected: %#v", mismatch)
	}
}

func TestToolExpectationUnknownDoesNotInferFromCorePass(t *testing.T) {
	result := NewResult("cursor", "Cursor CLI", "1.0", "https://example.test")
	for _, stage := range OrderedStages {
		result.Set(stage, StatusPass, "real-client inventory was listed")
	}
	n := 3
	check := CheckToolExpectation(result, []string{"ping"}, &n)
	if check.Status != StatusUnknown || check.ReasonCode != ToolReasonUnobservable || check.ObservedCount != nil || len(check.MissingNames) != 0 {
		t.Fatalf("unobserved inventory inferred from core PASS: %#v", check)
	}
	result.Set(StageTools, StatusUnknown, "could not prove listing")
	second := CheckToolExpectation(result, []string{"ping"}, nil)
	if second.Status != StatusUnknown || second.ObservedCount != nil {
		t.Fatalf("unknown core incorrectly promoted to tool check: %#v", second)
	}
}

func TestToolExpectationZeroCountAndNoNameLeak(t *testing.T) {
	result := completeToolResult(t, []string{"private_tool", "ping"})
	two := 2
	result.ToolExpectation = ptrCheck(CheckToolExpectation(result, []string{"ping"}, &two))
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private_tool") {
		t.Fatalf("unexpected observed tool name leaked into public JSON: %s", data)
	}
	if !strings.Contains(string(data), "tool_expectation") || !strings.Contains(string(data), "ping") {
		t.Fatalf("expected opt-in assertion missing: %s", data)
	}
	// The core PASS remains intact after checking a separate expectation.
	if !result.Passed() {
		t.Fatal("the core PASS changed")
	}
	empty := completeToolResult(t, nil)
	zero := 0
	match := CheckToolExpectation(empty, nil, &zero)
	if match.Status != StatusPass || match.ObservedCount == nil || *match.ObservedCount != 0 {
		t.Fatalf("known empty inventory not handled: %#v", match)
	}
}

func ptrCheck(check ToolExpectation) *ToolExpectation { return &check }

func TestToolExpectationNameValidation(t *testing.T) {
	for _, name := range []string{"ping", "_underscore", "resource.read-2", "A", "日本語", "outil_été", strings.Repeat("a", 128)} {
		if err := ValidateExpectedToolName(name); err != nil {
			t.Errorf("valid %q: %v", name, err)
		}
	}
	for _, name := range []string{"", "-prefix", "space name", "too/long", "a\nb", "📦", "a\u200btool", strings.Repeat("a", 129), "bearer abc", "token=secret"} {
		if err := ValidateExpectedToolName(name); err == nil {
			t.Errorf("invalid name accepted: %q", name)
		}
	}
}

func TestToolExpectationInvalidatedAfterCoreStageChanges(t *testing.T) {
	result := completeToolResult(t, []string{"ping"})
	check := CheckToolExpectation(result, []string{"ping"}, nil)
	result.ToolExpectation = &check
	result.Set(StageReach, StatusUnknown, "evidence lost")
	if result.ToolExpectation != nil {
		t.Fatal("stale tool expectation survived stage change")
	}
	if _, known := result.ObservedToolNames(); known {
		t.Fatal("stale tool inventory survived stage change")
	}
}

func TestToolExpectationRedactResultClonesOptionalNames(t *testing.T) {
	result := completeToolResult(t, []string{"ping"})
	check := CheckToolExpectation(result, []string{"ping", "missing"}, nil)
	result.ToolExpectation = &check
	safe := RedactResult(result)
	safe.ToolExpectation.ExpectedNames[0] = "modified"
	if result.ToolExpectation.ExpectedNames[0] == "modified" {
		t.Fatal("redaction leaked a mutable alias")
	}
	if result.ToolExpectation.MissingNames[0] != "missing" {
		t.Fatal("redaction changed original missing tool name")
	}
}
