package suite

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestAnalyzeRepeatedResultSetsPreservesEveryAttempt(t *testing.T) {
	manifest := regressionTestManifest()
	makeSet := func(status interop.Status, name string) LoadedResultSet {
		return regressionTestSet(t, manifest, name, "0.152.1", status, "")
	}
	tests := []struct {
		name     string
		statuses []interop.Status
		want     RepeatDecision
		unstable bool
		nonpass  bool
	}{
		{"pass, pass", []interop.Status{interop.StatusPass, interop.StatusPass}, RepeatDecisionClean, false, false},
		{"fail, fail", []interop.Status{interop.StatusFail, interop.StatusFail}, RepeatDecisionNonPass, false, true},
		{"fail, pass", []interop.Status{interop.StatusFail, interop.StatusPass}, RepeatDecisionNonPassUnstable, true, true},
		{"pass, fail", []interop.Status{interop.StatusPass, interop.StatusFail}, RepeatDecisionNonPassUnstable, true, true},
		{"pass, fail, pass", []interop.Status{interop.StatusPass, interop.StatusFail, interop.StatusPass}, RepeatDecisionNonPassUnstable, true, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sets := make([]LoadedResultSet, 0, len(test.statuses))
			refs := make([]string, 0, len(test.statuses))
			for i, status := range test.statuses {
				sets = append(sets, makeSet(status, "attempt"+string(rune('a'+i))))
				refs = append(refs, fmt.Sprintf("attempt-%02d/index.json", i+1))
			}
			report, err := AnalyzeRepeatedResultSets(sets, refs, len(test.statuses))
			if err != nil {
				t.Fatal(err)
			}
			if report.Decision != test.want || report.HasUnstable != test.unstable || report.HasNonPass != test.nonpass || !report.Complete || report.CompletedAttempts != len(sets) {
				t.Fatalf("unexpected repeated result: %#v", report)
			}
			if len(report.Runs) != 1 || len(report.Runs[0].Attempts) != len(sets) {
				t.Fatalf("attempt evidence was dropped: %#v", report.Runs)
			}
			for i, attempt := range report.Runs[0].Attempts {
				if attempt.Attempt != i+1 || attempt.Evidence == nil {
					t.Fatalf("missing attempt %d: %#v", i+1, attempt)
				}
				if test.statuses[i] == interop.StatusPass && attempt.Evidence.Outcome != OutcomePass {
					t.Fatal("pass attempt overwritten")
				}
				if test.statuses[i] != interop.StatusPass && attempt.Evidence.Outcome != OutcomeNonPass {
					t.Fatal("nonpass attempt overwritten")
				}
			}
		})
	}
}

func TestAnalyzeRepeatedResultSetsDetectsIncompleteAndVersionChanges(t *testing.T) {
	manifest := regressionTestManifest()
	one := regressionTestSet(t, manifest, "attempt1", "0.152.1", interop.StatusPass, "")
	two := regressionTestSet(t, manifest, "attempt2", "0.153.0", interop.StatusPass, "")
	partial, err := AnalyzeRepeatedResultSets([]LoadedResultSet{one}, []string{"attempt-01/index.json"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if partial.Decision != RepeatDecisionIncomplete || partial.Complete || partial.CompletedAttempts != 1 || partial.HasNonPass {
		t.Fatalf("incomplete attempt incorrectly marked clean: %#v", partial)
	}
	changed, err := AnalyzeRepeatedResultSets([]LoadedResultSet{one, two}, []string{"attempt-01/index.json", "attempt-02/index.json"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Decision != RepeatDecisionUnstable || !changed.HasUnstable || changed.HasNonPass {
		t.Fatalf("client version shift not recorded as unstable: %#v", changed)
	}
}

func TestAnalyzeRepeatedResultSetsHandlesExecutionErrorsWithoutFabrication(t *testing.T) {
	manifest := regressionTestManifest()
	failed := regressionTestSet(t, manifest, "attempt1", "0.152.1", interop.StatusPass, "")
	failed.Index.Runs[0].Outcome = OutcomeError
	failed.Index.Runs[0].ExitCode = 1
	failed.Index.Runs[0].Artifact = ""
	failed.Artifacts = nil
	passed := regressionTestSet(t, manifest, "attempt2", "0.152.1", interop.StatusPass, "")
	report, err := AnalyzeRepeatedResultSets([]LoadedResultSet{failed, passed}, []string{"attempt-01/index.json", "attempt-02/index.json"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != RepeatDecisionNonPassUnstable || report.Runs[0].Attempts[0].Evidence.Outcome != OutcomeError || report.Runs[0].Attempts[1].Evidence.Outcome != OutcomePass {
		t.Fatalf("execution error was hidden: %#v", report)
	}
}

func TestAnalyzeRepeatedResultSetsRejectsIncompatibleInputs(t *testing.T) {
	manifest := regressionTestManifest()
	set := regressionTestSet(t, manifest, "attempt1", "0.152.1", interop.StatusPass, "")
	otherManifest := regressionTestManifest()
	otherManifest.Targets[0].Clients = append(otherManifest.Targets[0].Clients, ClientSelection{ID: "cursor", Auth: AuthNone})
	other := regressionTestSet(t, otherManifest, "attempt2", "0.152.1", interop.StatusPass, "")
	for _, tc := range []struct {
		name  string
		sets  []LoadedResultSet
		refs  []string
		count int
	}{
		{"too few", []LoadedResultSet{set}, []string{"attempt-01/index.json"}, 1},
		{"too many", []LoadedResultSet{set}, []string{"attempt-01/index.json"}, 6},
		{"none", nil, nil, 2},
		{"count mismatch", []LoadedResultSet{set}, nil, 2},
		{"manifest drift", []LoadedResultSet{set, other}, []string{"attempt-01/index.json", "attempt-02/index.json"}, 2},
		{"unsafe path", []LoadedResultSet{set}, []string{"../secret"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := AnalyzeRepeatedResultSets(tc.sets, tc.refs, tc.count); err == nil {
				t.Fatal("invalid repeat evidence accepted")
			}
		})
	}
}

func TestAnalyzeRepeatedResultSetsDoesNotModifyInputs(t *testing.T) {
	manifest := regressionTestManifest()
	set := regressionTestSet(t, manifest, "attempt1", "0.152.1", interop.StatusPass, "")
	refs := []string{"attempt-01/index.json"}
	report, err := AnalyzeRepeatedResultSets([]LoadedResultSet{set}, refs, 2)
	if err != nil {
		t.Fatal(err)
	}
	report.AttemptIndexes[0] = "changed"
	if !reflect.DeepEqual(refs, []string{"attempt-01/index.json"}) {
		t.Fatal("report aliased caller references")
	}
	if strings.Contains(report.ManifestFingerprint, "protected-value") {
		t.Fatal("secret endpoint persisted")
	}
}
