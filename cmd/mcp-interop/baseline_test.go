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

func TestRunBaselineVerifyReportsLocalConsistencyWithoutAuthenticatedProvenance(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	source := writeCLISuiteCompareSet(t, manifest, "source", "1.2.3", interop.StatusPass, "")
	baselineDir := filepath.Join(t.TempDir(), "baseline-private-path")
	var createOut, createErr bytes.Buffer
	if rc := runBaselineCreate(
		[]string{source, "--output-dir", baselineDir},
		&createOut,
		&createErr,
		time.Now,
	); rc != 0 {
		t.Fatalf("create rc=%d stderr=%s", rc, createErr.String())
	}

	var stdout, stderr bytes.Buffer
	if rc := runBaselineVerify([]string{baselineDir, "--json"}, &stdout, &stderr); rc != 0 {
		t.Fatalf("verify rc=%d stderr=%s", rc, stderr.String())
	}
	var got baselineVerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Valid || got.IntegrityScope != baselineIntegrityLocalConsistency || got.AuthenticatedProvenance {
		t.Fatalf("verification overclaimed provenance: %#v", got)
	}
	if got.SchemaVersion != baselineVerificationSchemaVersion || got.ArtifactType != baselineVerificationArtifactType {
		t.Fatalf("unexpected verification contract: %#v", got)
	}
	for _, forbidden := range []string{baselineDir, source, "protected-value", "https://example.com/mcp"} {
		if bytes.Contains(stdout.Bytes(), []byte(forbidden)) {
			t.Fatalf("verification JSON leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunBaselineVerifyChecksExplicitSupersedesLink(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	firstSource := writeCLISuiteCompareSet(t, manifest, "first", "1.0.0", interop.StatusPass, "")
	firstDir := filepath.Join(t.TempDir(), "baseline-1")
	var stdout, stderr bytes.Buffer
	if rc := runBaselineCreate([]string{firstSource, "--output-dir", firstDir}, &stdout, &stderr, time.Now); rc != 0 {
		t.Fatalf("first create rc=%d stderr=%s", rc, stderr.String())
	}

	secondSource := writeCLISuiteCompareSet(t, manifest, "second", "2.0.0", interop.StatusPass, "")
	secondDir := filepath.Join(t.TempDir(), "baseline-2")
	stdout.Reset()
	stderr.Reset()
	if rc := runBaselineCreate(
		[]string{secondSource, "--output-dir", secondDir, "--supersedes", firstDir},
		&stdout,
		&stderr,
		time.Now,
	); rc != 0 {
		t.Fatalf("second create rc=%d stderr=%s", rc, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if rc := runBaselineVerify([]string{secondDir, "--predecessor", firstDir, "--json"}, &stdout, &stderr); rc != 0 {
		t.Fatalf("verify rc=%d stderr=%s", rc, stderr.String())
	}
	var got baselineVerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Predecessor == nil || !got.Predecessor.Verified || got.Predecessor.Relation != "supersedes" {
		t.Fatalf("predecessor link not verified: %#v", got)
	}
}

func TestRunBaselineVerifyRejectsWrongPredecessor(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	firstSource := writeCLISuiteCompareSet(t, manifest, "first", "1.0.0", interop.StatusPass, "")
	firstDir := filepath.Join(t.TempDir(), "baseline-1")
	var stdout, stderr bytes.Buffer
	if rc := runBaselineCreate([]string{firstSource, "--output-dir", firstDir}, &stdout, &stderr, time.Now); rc != 0 {
		t.Fatal(stderr.String())
	}
	secondSource := writeCLISuiteCompareSet(t, manifest, "second", "2.0.0", interop.StatusPass, "")
	secondDir := filepath.Join(t.TempDir(), "baseline-2")
	stdout.Reset()
	stderr.Reset()
	if rc := runBaselineCreate([]string{secondSource, "--output-dir", secondDir, "--supersedes", firstDir}, &stdout, &stderr, time.Now); rc != 0 {
		t.Fatal(stderr.String())
	}
	wrongSource := writeCLISuiteCompareSet(t, manifest, "wrong", "1.5.0", interop.StatusPass, "")
	wrongDir := filepath.Join(t.TempDir(), "baseline-wrong")
	stdout.Reset()
	stderr.Reset()
	if rc := runBaselineCreate([]string{wrongSource, "--output-dir", wrongDir}, &stdout, &stderr, time.Now); rc != 0 {
		t.Fatal(stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if rc := runBaselineVerify([]string{secondDir, "--predecessor", wrongDir}, &stdout, &stderr); rc != 2 || !bytes.Contains(stderr.Bytes(), []byte("fingerprint does not match")) {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
}

func TestParseBaselineVerifyOptionsRejectsAmbiguousInput(t *testing.T) {
	if _, err := parseBaselineVerifyOptions(nil); err == nil {
		t.Fatal("expected missing baseline path error")
	}
	if _, err := parseBaselineVerifyOptions([]string{"one", "two"}); err == nil {
		t.Fatal("expected multiple baseline path error")
	}
	if _, err := parseBaselineVerifyOptions([]string{"one", "--predecessor", " two "}); err == nil {
		t.Fatal("expected predecessor whitespace error")
	}
	if _, err := parseBaselineVerifyOptions([]string{"one", "--predecessor="}); err == nil {
		t.Fatal("expected empty predecessor error")
	}
	if _, err := parseBaselineVerifyOptions([]string{"one", "--predecessor", "two", "--predecessor", "three"}); err == nil {
		t.Fatal("expected duplicate predecessor error")
	}
}
