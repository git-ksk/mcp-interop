package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

func repeatFixture(t *testing.T) (string, string, string, suite.EndpointLookup) {
	t.Helper()
	root := t.TempDir()
	manifestPath := filepath.Join(root, "manifest.json")
	outputDir := filepath.Join(root, "repeat")
	manifest := `{"schema_version":1,"execution_context":"trusted_real_client","targets":[{"id":"test-a","endpoint":{"source":"environment","variable":"MCP_INTEROP_SUITE_ENDPOINT_TEST_A"},"deployment_id":"test-a","clients":[{"id":"codex","auth":"none"}]}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	secretEndpoint := "https://example.test/mcp/PRIVATE-TOKEN?key=PRIVATE-QUERY"
	lookup := func(name string) (string, bool) { return secretEndpoint, name == "MCP_INTEROP_SUITE_ENDPOINT_TEST_A" }
	return manifestPath, outputDir, secretEndpoint, lookup
}

func writeFakeRepeatRun(t *testing.T, args []string, status interop.Status) int {
	t.Helper()
	if got := optionValue(t, args, "--timeout"); got != "5s" {
		t.Fatalf("timeout not forwarded: %s", got)
	}
	result := interop.NewResult("codex", "Codex CLI", "0.152.1", args[0])
	for _, stage := range interop.OrderedStages {
		stageStatus := interop.StatusPass
		if stage == interop.StageTools {
			stageStatus = status
		}
		result.Set(stage, stageStatus, "test")
	}
	run, err := artifact.NewRunV2ProtectedPath(result, args[0], "test-a", time.Now().UTC(), "default", artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"}, "test", "deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	if err := artifact.WriteFile(optionValue(t, args, "--output"), artifact.NewArtifactV2([]artifact.Run{run})); err != nil {
		t.Fatal(err)
	}
	if status == interop.StatusPass {
		return 0
	}
	return 1
}

func TestParseSuiteRepeatOptionsBounds(t *testing.T) {
	valid := []string{"manifest.json", "--output-dir", "out", "--attempts=3", "--timeout", "45s", "--json"}
	got, err := parseSuiteRepeatOptions(valid)
	if err != nil || got.attempts != 3 || got.timeout != 45*time.Second || !got.json {
		t.Fatalf("valid args rejected: %#v %v", got, err)
	}
	for _, flags := range [][]string{
		{"--timeout=5s"}, {"--attempts=3"}, {"--attempts=1", "--timeout=5s"},
		{"--attempts=6", "--timeout=5s"}, {"--attempts=9999999999999", "--timeout=5s"},
		{"--attempts=3", "--timeout=0s"}, {"--attempts=3", "--timeout=11m"},
		{"--attempts=3", "--attempts=4", "--timeout=5s"},
		{"--attempts=3", "--timeout=5s", "--timeout=6s"},
		{"--attempts=3", "--timeout=5s", "--unrecognized"},
	} {
		args := append([]string{"manifest.json", "--output-dir", "out"}, flags...)
		if _, err := parseSuiteRepeatOptions(args); err == nil {
			t.Errorf("unsafe options accepted: %#v", args)
		}
	}
}

func TestSuiteRepeatPersistsAllAttemptsAndFailsOnAnyNonPass(t *testing.T) {
	for _, tc := range []struct {
		name     string
		statuses []interop.Status
		want     suite.RepeatDecision
		exit     int
	}{
		{"both pass", []interop.Status{interop.StatusPass, interop.StatusPass}, suite.RepeatDecisionClean, 0},
		{"first failed", []interop.Status{interop.StatusUnknown, interop.StatusPass}, suite.RepeatDecisionNonPassUnstable, 1},
		{"last failed", []interop.Status{interop.StatusPass, interop.StatusFail}, suite.RepeatDecisionNonPassUnstable, 1},
		{"both failed", []interop.Status{interop.StatusFail, interop.StatusFail}, suite.RepeatDecisionNonPass, 1},
		{"later retry failed", []interop.Status{interop.StatusPass, interop.StatusFail, interop.StatusPass}, suite.RepeatDecisionNonPassUnstable, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest, output, secret, lookup := repeatFixture(t)
			calls := 0
			fake := func(_ context.Context, args []string, _ io.Writer, _ io.Writer) int {
				if args[0] != secret {
					t.Fatal("repeat did not reuse exact protected endpoint")
				}
				status := tc.statuses[calls]
				calls++
				return writeFakeRepeatRun(t, args, status)
			}
			args := []string{manifest, "--output-dir", output, "--attempts", string(rune('0' + len(tc.statuses))), "--timeout", "5s", "--json"}
			var out, stderr bytes.Buffer
			rc := runSuiteRepeatWith(context.Background(), args, &out, &stderr, lookup, fake)
			if rc != tc.exit || calls != len(tc.statuses) {
				t.Fatalf("rc=%d calls=%d stderr=%s", rc, calls, stderr.String())
			}
			var report suite.RepeatReport
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatalf("JSON output: %v, %s", err, out.String())
			}
			if report.Decision != tc.want || !report.Complete || report.CompletedAttempts != len(tc.statuses) || len(report.Runs) != 1 || len(report.Runs[0].Attempts) != len(tc.statuses) {
				t.Fatalf("incomplete/wrong report: %#v", report)
			}
			if !bytes.Contains([]byte(readFileString(t, filepath.Join(output, "repeat-report.json"))), []byte(`"schema_version": 1`)) {
				t.Fatal("missing persisted versioned report")
			}
			for i, status := range tc.statuses {
				ref := filepath.Join(output, report.AttemptIndexes[i])
				if _, err := suite.ReadResultSet(ref); err != nil {
					t.Fatalf("attempt %d unreadable: %v", i, err)
				}
				wantOutcome := suite.OutcomePass
				if status != interop.StatusPass {
					wantOutcome = suite.OutcomeNonPass
				}
				if report.Runs[0].Attempts[i].Evidence.Outcome != wantOutcome {
					t.Fatalf("attempt %d lost outcome", i)
				}
			}
			for _, data := range []string{out.String(), readFileString(t, filepath.Join(output, "repeat-report.json")), readFileString(t, filepath.Join(output, "manifest.json"))} {
				if strings.Contains(data, "PRIVATE-TOKEN") || strings.Contains(data, "PRIVATE-QUERY") {
					t.Fatalf("secret value emitted: %s", data)
				}
			}
		})
	}
}

func TestSuiteRepeatCancellationPreservesPriorAttempt(t *testing.T) {
	manifest, output, _, lookup := repeatFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	fake := func(_ context.Context, args []string, _ io.Writer, _ io.Writer) int {
		calls++
		rc := writeFakeRepeatRun(t, args, interop.StatusPass)
		cancel()
		return rc
	}
	var out, stderr bytes.Buffer
	rc := runSuiteRepeatWith(ctx, []string{manifest, "--output-dir", output, "--attempts=3", "--timeout=5s", "--json"}, &out, &stderr, lookup, fake)
	if rc != 1 || calls != 1 {
		t.Fatalf("canceled attempt unexpectedly retried: rc=%d calls=%d stderr=%s", rc, calls, stderr.String())
	}
	var report suite.RepeatReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Complete || report.Decision != suite.RepeatDecisionIncomplete || report.CompletedAttempts != 1 {
		t.Fatalf("interrupted repeat marked clean: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(output, "attempt-01", "index.json")); err != nil {
		t.Fatal("completed attempt lost: ", err)
	}
	if _, err := os.Stat(filepath.Join(output, "attempt-02")); !os.IsNotExist(err) {
		t.Fatal("canceled repeat launched additional attempt")
	}
}

func TestSuiteRepeatExecutionErrorIsKeptEvenWhenLaterPasses(t *testing.T) {
	manifest, output, _, lookup := repeatFixture(t)
	calls := 0
	fake := func(_ context.Context, args []string, _ io.Writer, _ io.Writer) int {
		calls++
		if calls == 1 {
			return 1
		} // no artifact: stage evidence unavailable
		return writeFakeRepeatRun(t, args, interop.StatusPass)
	}
	var out, stderr bytes.Buffer
	if rc := runSuiteRepeatWith(context.Background(), []string{manifest, "--output-dir", output, "--attempts=2", "--timeout=5s", "--json"}, &out, &stderr, lookup, fake); rc != 1 {
		t.Fatalf("execution error hidden: rc=%d stderr=%s", rc, stderr.String())
	}
	var report suite.RepeatReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Decision != suite.RepeatDecisionNonPassUnstable || report.Runs[0].Attempts[0].Evidence.Outcome != suite.OutcomeError {
		t.Fatalf("execution error fabricated: %#v", report)
	}
	if _, err := suite.ReadResultSet(filepath.Join(output, "attempt-01", "index.json")); err != nil {
		t.Fatal(err)
	}
}

func TestSuiteRepeatFailsClosedBeforeLaunching(t *testing.T) {
	manifest, output, _, lookup := repeatFixture(t)
	launched := 0
	fake := func(_ context.Context, _ []string, _ io.Writer, _ io.Writer) int { launched++; return 0 }
	for _, tc := range []struct {
		name   string
		args   []string
		lookup suite.EndpointLookup
	}{
		{"budget", []string{manifest, "--output-dir", output, "--attempts=5", "--timeout=10m"}, lookup},
		{"bad endpoint", []string{manifest, "--output-dir", output, "--attempts=2", "--timeout=5s"}, func(string) (string, bool) { return "", false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errs bytes.Buffer
			if rc := runSuiteRepeatWith(context.Background(), tc.args, &out, &errs, tc.lookup, fake); rc != 2 || launched != 0 {
				t.Fatalf("preflight launched clients: rc=%d launches=%d stderr=%s", rc, launched, errs.String())
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("preflight created output: %v", err)
			}
		})
	}
	// Existing output cannot be overwritten.
	if err := os.Mkdir(output, 0o700); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if rc := runSuiteRepeatWith(context.Background(), []string{manifest, "--output-dir", output, "--attempts=2", "--timeout=5s"}, &out, &errs, lookup, fake); rc != 2 || launched != 0 {
		t.Fatalf("clobber not rejected: rc=%d launched=%d stderr=%s", rc, launched, errs.String())
	}
}
