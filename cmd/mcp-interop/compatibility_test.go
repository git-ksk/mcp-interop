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
		"--max-age-seconds", "1209600", "--stale-on-client-version-change", "--json",
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
