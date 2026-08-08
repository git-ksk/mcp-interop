package interop

import "testing"

func TestPassedRequiresEveryStageToPass(t *testing.T) {
	result := NewResult("codex", "Codex CLI", "test", "https://example.com/mcp")
	if result.Passed() {
		t.Fatal("new result must not pass while stages are unknown")
	}

	for _, stage := range OrderedStages {
		result.Set(stage, StatusPass, "observed")
	}
	if !result.Passed() {
		t.Fatal("expected complete pass")
	}

	result.Set(StageTools, StatusUnknown, "inconclusive")
	if result.Passed() {
		t.Fatal("unknown stage must make the result incomplete")
	}
}
