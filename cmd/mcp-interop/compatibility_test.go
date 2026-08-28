package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/client"
	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

type compatibilityFakeDetector struct {
	detection client.Detection
}

func (d compatibilityFakeDetector) Detect(context.Context, client.Spec) client.Detection {
	return d.detection
}

func TestCompatibilityQueryAutoUpdatedUnseenVersionIsUntested(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	observed := writeCLISuiteCompareSet(t, manifest, "observed", "1.0.0", interop.StatusPass, "")
	output := runCompatibilityQueryJSON(t, []string{
		"--client", "codex",
		"--target", "production-a",
		"--deployment-id", "production-a",
		"--observation", observed,
		"--json",
	}, "2.0.0")
	if output.Classification.State != suite.CompatibilityUntested || output.Classification.Point != nil {
		t.Fatalf("unseen installed version classified as %#v", output.Classification)
	}
	if strings.Join(output.Classification.ObservedVersions, ",") != "1.0.0" {
		t.Fatalf("observed versions=%v", output.Classification.ObservedVersions)
	}
}

func TestCompatibilityQueryDistinguishesKnownBrokenAndRegression(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	broken := writeCLISuiteCompareSet(t, manifest, "broken", "2.0.0", interop.StatusFail, interop.ReasonDCRUnsupported)
	knownBroken := runCompatibilityQueryJSON(t, []string{
		"--client=codex", "--target=production-a", "--deployment-id=production-a",
		"--observation", broken, "--json",
	}, "2.0.0")
	if knownBroken.Classification.State != suite.CompatibilityKnownBroken {
		t.Fatalf("known failure state=%s", knownBroken.Classification.State)
	}

	baselineSource := writeCLISuiteCompareSet(t, manifest, "baseline", "1.0.0", interop.StatusPass, "")
	baselineDir := filepath.Join(t.TempDir(), "accepted")
	var baselineOut, baselineErr bytes.Buffer
	if rc := runBaselineCreate([]string{baselineSource, "--output-dir", baselineDir}, &baselineOut, &baselineErr, time.Now); rc != 0 {
		t.Fatalf("baseline create rc=%d stderr=%s", rc, baselineErr.String())
	}
	regressed := runCompatibilityQueryJSON(t, []string{
		"--client", "codex", "--target", "production-a", "--deployment-id", "production-a",
		"--baseline", baselineDir, "--observation", broken, "--json",
	}, "2.0.0")
	if regressed.Classification.State != suite.CompatibilityRegressed {
		t.Fatalf("true regression state=%s", regressed.Classification.State)
	}
}

func TestCompatibilityQuerySurfacesStaleObservedVersion(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	old := writeCLISuiteCompareSet(t, manifest, "old", "1.0.0", interop.StatusPass, "")
	newer := writeCLISuiteCompareSet(t, manifest, "newer", "next-build", interop.StatusPass, "")
	setCLIResultExecutedAt(t, old, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	setCLIResultExecutedAt(t, newer, time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC))

	output := runCompatibilityQueryJSONAt(t, []string{
		"--client", "codex", "--target", "production-a", "--deployment-id", "production-a",
		"--observation", old, "--observation", newer,
		"--max-age-seconds", "1209600", "--trust-executed-at-clock",
		"--stale-on-client-version-change", "--json",
	}, "1.0.0", time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if output.Classification.State != suite.CompatibilityStale || output.Classification.Point == nil {
		t.Fatalf("stale classification=%#v", output.Classification)
	}
	if len(output.Classification.Point.StaleReasons) != 2 {
		t.Fatalf("stale reasons=%v", output.Classification.Point.StaleReasons)
	}
}

func TestCompatibilityQueryJSONIsSecretSafeAndOmitsLocalPaths(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	observed := writeCLISuiteCompareSet(t, manifest, "observed", "1.0.0", interop.StatusPass, "")
	var stdout, stderr bytes.Buffer
	detector := compatibilityFakeDetector{detection: client.Detection{
		ID: "codex", DisplayName: "Codex CLI", Tier: client.TierV1, Installed: true,
		Executable: "codex", Path: "/Users/private/bin/codex", Version: "1.0.0",
	}}
	rc := runCompatibilityQueryWith(context.Background(), []string{
		"--client", "codex", "--target", "production-a", "--deployment-id", "production-a",
		"--observation", observed, "--json",
	}, &stdout, &stderr, detector, func() time.Time { return time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC) }, artifact.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	for _, forbidden := range []string{"protected-value", "https://example.com/mcp", "/Users/private/bin/codex", observed} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("compatibility JSON leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestCompatibilityQueryRejectsKnownClientExecutableArchitectureMismatch(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	observed := writeCLISuiteCompareSet(t, manifest, "observed", "1.0.0", interop.StatusPass, "")
	var stdout, stderr bytes.Buffer
	detector := compatibilityFakeDetector{detection: client.Detection{
		ID: "codex", DisplayName: "Codex CLI", Tier: client.TierV1, Installed: true,
		Version: "1.0.0", ExecutableArchitectures: []string{"other-arch"},
	}}
	rc := runCompatibilityQueryWith(context.Background(), []string{
		"--client", "codex", "--target", "production-a", "--deployment-id", "production-a",
		"--observation", observed,
	}, &stdout, &stderr, detector, time.Now, artifact.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
	if rc != 2 || !strings.Contains(stderr.String(), "client executable architecture") {
		t.Fatalf("rc=%d stderr=%q, want fail-closed architecture mismatch", rc, stderr.String())
	}
}

func TestCompatibilityQueryAllowsUnknownWrapperArchitecture(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	observed := writeCLISuiteCompareSet(t, manifest, "observed", "1.0.0", interop.StatusPass, "")
	var stdout, stderr bytes.Buffer
	detector := compatibilityFakeDetector{detection: client.Detection{
		ID: "codex", DisplayName: "Codex CLI", Tier: client.TierV1, Installed: true,
		Version: "1.0.0",
	}}
	rc := runCompatibilityQueryWith(context.Background(), []string{
		"--client", "codex", "--target", "production-a", "--deployment-id", "production-a",
		"--observation", observed,
	}, &stdout, &stderr, detector, time.Now, artifact.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%q, unknown wrapper architecture must remain conservative/allowed", rc, stderr.String())
	}
}

func TestCompatibilityQueryRequiresInstalledExactVersion(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	observed := writeCLISuiteCompareSet(t, manifest, "observed", "1.0.0", interop.StatusPass, "")
	baseArgs := []string{"--client", "codex", "--target", "production-a", "--deployment-id", "production-a", "--observation", observed}
	for _, detection := range []client.Detection{
		{ID: "codex", DisplayName: "Codex CLI", Tier: client.TierV1, Installed: false},
		{ID: "codex", DisplayName: "Codex CLI", Tier: client.TierV1, Installed: true, Error: "version command failed"},
	} {
		var stdout, stderr bytes.Buffer
		rc := runCompatibilityQueryWith(context.Background(), baseArgs, &stdout, &stderr, compatibilityFakeDetector{detection: detection}, time.Now, artifact.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
		if rc != 1 {
			t.Fatalf("detection=%#v rc=%d stderr=%s", detection, rc, stderr.String())
		}
	}
}

func TestParseCompatibilityQueryRejectsMissingEvidenceAndRanges(t *testing.T) {
	if _, err := parseCompatibilityQueryOptions([]string{"--client", "codex", "--target", "production-a", "--deployment-id", "production-a"}); err == nil {
		t.Fatal("expected missing evidence input to fail")
	}
	if _, err := parseCompatibilityQueryOptions([]string{"--client", "codex", "--target", "production-a", "--deployment-id", "production-a", "--observation", "x", "--version-range", "1.0-2.0"}); err == nil {
		t.Fatal("version-range option must not exist")
	}
}

func TestParseCompatibilityQueryRequiresExplicitClockTrustForAge(t *testing.T) {
	base := []string{
		"--client", "codex",
		"--target", "production-a",
		"--deployment-id", "production-a",
		"--observation", "result-set",
	}
	withoutTrust := append(append([]string(nil), base...), "--max-age-seconds", "3600")
	if _, err := parseCompatibilityQueryOptions(withoutTrust); err == nil || !strings.Contains(err.Error(), "trust-executed-at-clock") {
		t.Fatalf("expected explicit clock-trust error, got %v", err)
	}
	withoutAge := append(append([]string(nil), base...), "--trust-executed-at-clock")
	if _, err := parseCompatibilityQueryOptions(withoutAge); err == nil || !strings.Contains(err.Error(), "max-age-seconds") {
		t.Fatalf("expected max-age pairing error, got %v", err)
	}
}

func TestParseCompatibilityQueryBoundsObservationInputs(t *testing.T) {
	args := []string{"--client", "codex", "--target", "production-a", "--deployment-id", "production-a"}
	for i := 0; i < maxCompatibilityObservationInputs+1; i++ {
		args = append(args, "--observation", "result-set")
	}
	if _, err := parseCompatibilityQueryOptions(args); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("expected bounded observation input error, got %v", err)
	}
}

func runCompatibilityQueryJSON(t *testing.T, args []string, version string) compatibilityQueryOutput {
	t.Helper()
	return runCompatibilityQueryJSONAt(t, args, version, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
}

func runCompatibilityQueryJSONAt(t *testing.T, args []string, version string, now time.Time) compatibilityQueryOutput {
	t.Helper()
	var stdout, stderr bytes.Buffer
	detector := compatibilityFakeDetector{detection: client.Detection{
		ID: "codex", DisplayName: "Codex CLI", Tier: client.TierV1, Installed: true, Version: version,
	}}
	rc := runCompatibilityQueryWith(context.Background(), args, &stdout, &stderr, detector, func() time.Time { return now }, artifact.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%s stdout=%s", rc, stderr.String(), stdout.String())
	}
	var output compatibilityQueryOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode compatibility JSON: %v\n%s", err, stdout.String())
	}
	return output
}

func setCLIResultExecutedAt(t *testing.T, root string, executedAt time.Time) {
	t.Helper()
	index, err := suite.ReadResultIndex(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Runs) != 1 || index.Runs[0].Artifact == "" {
		t.Fatalf("unexpected result index: %#v", index)
	}
	path := filepath.Join(root, filepath.FromSlash(index.Runs[0].Artifact))
	value, err := artifact.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	value.Runs[0].ExecutedAt = executedAt.UTC()
	if err := artifact.WriteFile(path, value); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestCompatibilityMatrixRetainsExactPointsAndRetries(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	observed := writeCLISuiteCompareSet(t, manifest, "observed", "1.0.0", interop.StatusPass, "")
	failed := writeCLISuiteCompareSet(t, manifest, "retry-failed", "2.0.0", interop.StatusUnknown, "")
	passed := writeCLISuiteCompareSet(t, manifest, "retry-passed", "2.0.0", interop.StatusPass, "")

	var stdout, stderr bytes.Buffer
	rc := runCompatibilityMatrixWith([]string{
		"--observation", observed,
		"--observation", failed,
		"--observation", passed,
		"--json",
	}, &stdout, &stderr, func() time.Time { return time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC) })
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	var envelope suite.CompatibilityEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode matrix: %v\n%s", err, stdout.String())
	}
	if len(envelope.Points) != 2 {
		t.Fatalf("points=%d want 2: %#v", len(envelope.Points), envelope.Points)
	}
	var first, second *suite.CompatibilityPoint
	for i := range envelope.Points {
		switch envelope.Points[i].ClientVersion {
		case "1.0.0":
			first = &envelope.Points[i]
		case "2.0.0":
			second = &envelope.Points[i]
		}
	}
	if first == nil || first.State != suite.CompatibilityTested || len(first.Observations) != 1 {
		t.Fatalf("first exact point=%#v", first)
	}
	if second == nil || second.State != suite.CompatibilityUnknown || !second.Unstable || len(second.Observations) != 2 {
		t.Fatalf("retry point=%#v", second)
	}
	if second.Observations[0].Outcome != suite.OutcomeNonPass || second.Observations[1].Outcome != suite.OutcomePass {
		t.Fatalf("retry observations were collapsed/reordered: %#v", second.Observations)
	}
}

func TestCompatibilityMatrixHumanOutputKeepsAttemptHistory(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	failed := writeCLISuiteCompareSet(t, manifest, "retry-failed", "2.0.0", interop.StatusUnknown, "")
	passed := writeCLISuiteCompareSet(t, manifest, "retry-passed", "2.0.0", interop.StatusPass, "")

	var stdout, stderr bytes.Buffer
	rc := runCompatibilityMatrixWith([]string{
		"--observation", failed,
		"--observation", passed,
	}, &stdout, &stderr, time.Now)
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	for _, want := range []string{"2.0.0", "unknown", "2", "YES", "attempt1:non_pass", "attempt2:pass"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("matrix missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestCompatibilityMatrixExecutionErrorIsEvidenceGapNotFailurePoint(t *testing.T) {
	manifest := cliSuiteCompareManifest()
	root := filepath.Join(t.TempDir(), "unavailable")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	index, err := suite.NewResultIndex(manifest, []suite.ResultEntry{{
		TargetID:     "production-a",
		DeploymentID: "production-a",
		ClientID:     "codex",
		AuthMode:     suite.AuthNone,
		Outcome:      suite.OutcomeError,
		ExitCode:     1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := suite.WriteResultIndex(filepath.Join(root, "index.json"), index); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rc := runCompatibilityMatrixWith([]string{"--observation", root, "--json"}, &stdout, &stderr, time.Now)
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	var envelope suite.CompatibilityEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Points) != 0 || len(envelope.EvidenceGaps) != 1 {
		t.Fatalf("unavailable run became a tested failure: %#v", envelope)
	}
	if envelope.EvidenceGaps[0].Kind != suite.CompatibilityGapExecutionError {
		t.Fatalf("gap kind=%q", envelope.EvidenceGaps[0].Kind)
	}
}

func TestParseCompatibilityMatrixRejectsMissingEvidenceRangesAndClockAmbiguity(t *testing.T) {
	if _, err := parseCompatibilityMatrixOptions(nil); err == nil {
		t.Fatal("expected missing evidence input to fail")
	}
	if _, err := parseCompatibilityMatrixOptions([]string{"--observation", "set", "--version-range", "1-2"}); err == nil {
		t.Fatal("version ranges must not be accepted")
	}
	if _, err := parseCompatibilityMatrixOptions([]string{"--observation", "set", "--max-age-seconds", "60"}); err == nil || !strings.Contains(err.Error(), "trust-executed-at-clock") {
		t.Fatalf("expected explicit clock-trust error, got %v", err)
	}
}
