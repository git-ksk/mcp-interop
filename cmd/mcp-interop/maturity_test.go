package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/client"
)

func TestRunMaturityJSONSeparatesTierFromEvidenceMaturity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if rc := runMaturity([]string{"--json"}, &stdout, &stderr); rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	var report maturityReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != maturityReportSchemaVersion || report.ArtifactType != maturityReportArtifactType {
		t.Fatalf("unexpected report contract: %#v", report)
	}
	if len(report.Decisions) != 3 {
		t.Fatalf("decisions=%d", len(report.Decisions))
	}
	want := map[string]client.Maturity{"codex": client.MaturityStable, "cursor": client.MaturityStable, "antigravity": client.MaturityStable}
	for _, decision := range report.Decisions {
		if decision.Tier != client.TierV1 || decision.Maturity != want[decision.ClientID] {
			t.Fatalf("tier/maturity unexpected: %#v", decision)
		}
	}
	for _, forbidden := range []string{"executable", "path", "version", "installed"} {
		if bytes.Contains(stdout.Bytes(), []byte(`"`+forbidden+`"`)) {
			t.Fatalf("maturity report unexpectedly contains detection field %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunMaturityHumanShowsCurrentStableAdapters(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if rc := runMaturity(nil, &stdout, &stderr); rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	for _, want := range []string{"Codex CLI", "Cursor CLI", "Antigravity CLI", "stable"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("maturity output missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), client.CriterionAdvertisedPlatformCoverage) || strings.Contains(stdout.String(), client.CriterionMeasurementSurfaceStability) {
		t.Fatalf("stable adapters unexpectedly expose blockers:\n%s", stdout.String())
	}
}

func TestRunMaturityRejectsPositionalArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if rc := runMaturity([]string{"codex"}, &stdout, &stderr); rc != 2 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
}
