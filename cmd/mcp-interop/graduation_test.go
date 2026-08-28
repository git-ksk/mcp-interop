package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/client"
)

func TestRunGraduationJSONKeepsEveryCandidateResearchOnly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if rc := runGraduation([]string{"--json"}, &stdout, &stderr); rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	var report graduationReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != graduationReportSchemaVersion || report.ArtifactType != graduationReportArtifactType {
		t.Fatalf("unexpected report contract: %#v", report)
	}
	if len(report.Decisions) != 4 {
		t.Fatalf("decisions=%d want 4", len(report.Decisions))
	}
	for _, decision := range report.Decisions {
		if decision.Status != client.GraduationResearchOnly || decision.Eligible {
			t.Fatalf("candidate unexpectedly eligible: %#v", decision)
		}
		if len(decision.Blockers) == 0 {
			t.Fatalf("candidate has no blockers: %#v", decision)
		}
	}
	for _, forbidden := range []string{"installed", "executable", "path", "endpoint", "token", "cookie"} {
		if bytes.Contains(bytes.ToLower(stdout.Bytes()), []byte(`"`+forbidden+`"`)) {
			t.Fatalf("graduation report contains runtime/secret field %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunGraduationHumanShowsCandidatesIssuesAndBlockers(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if rc := runGraduation(nil, &stdout, &stderr); rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	for _, want := range []string{
		"GitHub Copilot CLI", "#48",
		"VS Code", "#6",
		"ChatGPT", "#20",
		"Claude web/Desktop", "#68",
		"research_only", "NO",
		client.GraduationDirectBoundary,
		client.GraduationSupportedScope,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("graduation output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunGraduationRejectsArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if rc := runGraduation([]string{"copilot"}, &stdout, &stderr); rc != 2 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
}
