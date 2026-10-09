package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/reporting"
)

func TestReportSuiteCreatesPrivateOfflineHTMLAndCISummary(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	base := writeCLISuiteCompareSet(t, manifest, "baseline", "1.0", interop.StatusPass, "")
	bad := writeCLISuiteCompareSet(t, manifest, "attempt-1", "2.0", interop.StatusUnknown, "")
	good := writeCLISuiteCompareSet(t, manifest, "attempt-2", "2.0", interop.StatusPass, "")
	root := t.TempDir()
	html := filepath.Join(root, "report.html")
	ci := filepath.Join(root, "summary.md")
	var stdout, stderr bytes.Buffer
	rc := runReport([]string{"suite", base, bad, good, "--html", html, "--ci-summary", ci, "--fail-on-regression"}, &stdout, &stderr)
	if rc != 1 {
		t.Fatalf("regression not gated: rc=%d stdout=%s stderr=%s", rc, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "REGRESSION_AND_UNSTABLE") || !strings.Contains(stdout.String(), "ATTEMPTS\t2") {
		t.Fatalf("missing decision: %s", stdout.String())
	}
	htmlText := readFileString(t, html)
	ciText := readFileString(t, ci)
	for _, part := range []string{"<html", "default-src 'none'", "production-a", "non_pass", "unknown", "pass"} {
		if !strings.Contains(strings.ToLower(htmlText), strings.ToLower(part)) {
			t.Fatalf("HTML missing %q: %s", part, htmlText)
		}
	}
	for _, part := range []string{"Regression: **true**", "| production-a | codex | 1", "| production-a | codex | 2"} {
		if !strings.Contains(ciText, part) {
			t.Fatalf("CI summary missing %q: %s", part, ciText)
		}
	}
	for _, data := range []string{htmlText, ciText} {
		for _, part := range []string{"/mcp/protected-value", "https://example.com", "/artifacts/", "/Users/", "Authorization", "token=", "/tmp/"} {
			if strings.Contains(data, part) {
				t.Fatalf("sensitive text %q in report", part)
			}
		}
	}
	if runtime.GOOS != "windows" {
		for _, path := range []string{html, ci} {
			stat, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if stat.Mode().Perm() != 0o600 {
				t.Fatalf("unsafe output mode: %s", stat.Mode().Perm())
			}
		}
	}
	// Results are no-clobber and replaying a command does not overwrite evidence.
	stdout.Reset()
	stderr.Reset()
	if rc := runReport([]string{"suite", base, bad, good, "--html", html, "--ci-summary", ci}, &stdout, &stderr); rc != 2 {
		t.Fatalf("output overwrite accepted: rc=%d, stderr=%s", rc, stderr.String())
	}
}

func TestReportSuiteCleanAndCIOnly(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	base := writeCLISuiteCompareSet(t, manifest, "baseline", "1.0", interop.StatusPass, "")
	good := writeCLISuiteCompareSet(t, manifest, "attempt", "2.0", interop.StatusPass, "")
	summary := filepath.Join(t.TempDir(), "ci.md")
	var out, stderr bytes.Buffer
	if rc := runReport([]string{"suite", base, good, "--ci-summary=" + summary, "--fail-on-regression"}, &out, &stderr); rc != 0 {
		t.Fatalf("clean report falsely failed: rc=%d, stderr=%s", rc, stderr.String())
	}
	if !strings.Contains(readFileString(t, summary), "CLEAN") {
		t.Fatal("no clean decision")
	}
}

func TestReportSuiteStrictArgumentsAndNoPartialOutputs(t *testing.T) {
	base := cliSuiteCompareManifest()
	baseline := writeCLISuiteCompareSet(t, base, "baseline", "1.0", interop.StatusPass, "")
	current := writeCLISuiteCompareSet(t, base, "current", "1.0", interop.StatusPass, "")
	root := t.TempDir()
	html := filepath.Join(root, "report.html")
	ci := filepath.Join(root, "ci.md")
	inputs := [][]string{
		{}, {"wrong", baseline, current, "--html", html}, {"suite", baseline, "--html", html},
		{"suite", baseline, current}, {"suite", baseline, current, "--html"},
		{"suite", baseline, current, "--html", html, "--html", html},
		{"suite", baseline, current, "--html", html, "--ci-summary", html},
		{"suite", baseline, current, "--html", html, "--wat"},
		{"suite", baseline, current, "--html", filepath.Join(root, "absent", "report.html")},
		{"suite", baseline, current, "--html", html, "--ci-summary", filepath.Join(root, "absent", "ci.md")},
	}
	for _, args := range inputs {
		var out, stderr bytes.Buffer
		if rc := runReport(args, &out, &stderr); rc != 2 {
			t.Errorf("invalid args accepted: %v rc=%d stderr=%s", args, rc, stderr.String())
		}
		if _, err := os.Stat(html); !os.IsNotExist(err) {
			t.Fatal("invalid invocation unexpectedly created HTML")
		}
	}
	// Fail safely before output if any attempt is untrusted or unreadable.
	var out, stderr bytes.Buffer
	if rc := runReport([]string{"suite", baseline, "/nonexistent/attempt", "--html", html, "--ci-summary", ci}, &out, &stderr); rc != 2 {
		t.Fatalf("invalid input not rejected: rc=%d", rc)
	}
	if _, err := os.Stat(html); !os.IsNotExist(err) {
		t.Fatal("output created before source validation")
	}
}

func TestReportSuiteAttemptCountBound(t *testing.T) {
	root := t.TempDir()
	args := []string{"suite", "baseline"}
	for i := 0; i <= reporting.MaxAttempts; i++ {
		args = append(args, "attempt")
	}
	args = append(args, "--html", filepath.Join(root, "report.html"))
	if _, err := parseReportSuiteOptions(args[1:]); err == nil {
		t.Fatal("unbounded report input accepted")
	}
}
