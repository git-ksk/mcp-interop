package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

func TestRunBaselineCreateAcceptsImmutableSnapshot(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	source := writeCLISuiteCompareSet(t, manifest, "source", "1.2.3", interop.StatusPass, "")
	output := filepath.Join(t.TempDir(), "baseline")
	fixed := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	var stdout, stderr bytes.Buffer

	rc := runBaselineCreate(
		[]string{source, "--output-dir", output, "--json"},
		&stdout,
		&stderr,
		func() time.Time { return fixed },
	)
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	var got baselineCreateResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Fingerprint == "" || got.Baseline.CreatedAt != fixed {
		t.Fatalf("unexpected baseline result: %#v", got)
	}
	loaded, err := suite.ReadBaseline(output)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Descriptor != got.Baseline {
		t.Fatalf("persisted descriptor differs: %#v vs %#v", loaded.Descriptor, got.Baseline)
	}
}

func TestRunBaselineCreateRecordsSupersedes(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	firstSource := writeCLISuiteCompareSet(t, manifest, "first", "1.0.0", interop.StatusPass, "")
	firstDir := filepath.Join(t.TempDir(), "baseline-1")
	var stdout, stderr bytes.Buffer
	if rc := runBaselineCreate(
		[]string{firstSource, "--output-dir", firstDir},
		&stdout,
		&stderr,
		time.Now,
	); rc != 0 {
		t.Fatalf("first rc=%d stderr=%s", rc, stderr.String())
	}

	secondSource := writeCLISuiteCompareSet(t, manifest, "second", "2.0.0", interop.StatusPass, "")
	secondDir := filepath.Join(t.TempDir(), "baseline-2")
	stdout.Reset()
	stderr.Reset()
	if rc := runBaselineCreate(
		[]string{secondSource, "--output-dir", secondDir, "--supersedes", firstDir, "--json"},
		&stdout,
		&stderr,
		time.Now,
	); rc != 0 {
		t.Fatalf("second rc=%d stderr=%s", rc, stderr.String())
	}
	var got baselineCreateResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Baseline.Supersedes == "" {
		t.Fatalf("supersedes missing: %#v", got)
	}
}

func TestRunBaselineCompareRetainsFailedRetry(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	baselineSource := writeCLISuiteCompareSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	baselineDir := filepath.Join(t.TempDir(), "accepted")
	var createOut, createErr bytes.Buffer
	if rc := runBaselineCreate(
		[]string{baselineSource, "--output-dir", baselineDir},
		&createOut,
		&createErr,
		time.Now,
	); rc != 0 {
		t.Fatalf("create rc=%d stderr=%s", rc, createErr.String())
	}
	failed := writeCLISuiteCompareSet(
		t,
		manifest,
		"attempt-1",
		"2.0.0",
		interop.StatusUnknown,
		interop.ReasonDCRUnsupported,
	)
	passed := writeCLISuiteCompareSet(t, manifest, "attempt-2", "2.0.0", interop.StatusPass, "")
	var stdout, stderr bytes.Buffer
	rc := runBaselineCompare(
		[]string{baselineDir, failed, passed, "--json", "--fail-on-regression"},
		&stdout,
		&stderr,
	)
	if rc != 1 {
		t.Fatalf("rc=%d stderr=%s stdout=%s", rc, stderr.String(), stdout.String())
	}
	var report suite.RegressionReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Decision != suite.DecisionRegressionAndUnstable || !report.HasRegression || !report.HasUnstable {
		t.Fatalf("retry evidence was collapsed: %#v", report)
	}
	if len(report.Runs) != 1 || len(report.Runs[0].Attempts) != 2 {
		t.Fatalf("expected both attempts, got %#v", report.Runs)
	}
}
