package reporting

import (
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	interopcompare "github.com/git-ksk/mcp-interop/internal/compare"
	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

func reportingFixture() suite.RegressionReport {
	return suite.RegressionReport{
		SchemaVersion: suite.RegressionReportSchemaVersion,
		ArtifactType:  suite.RegressionReportArtifactType,
		Decision:      suite.DecisionRegressionAndUnstable,
		HasRegression: true, HasUnstable: true, AttemptCount: 2,
		Runs: []suite.RegressionRun{{
			TargetID: "target-a", DeploymentID: "public-label", ClientID: "codex", AuthMode: suite.AuthNone,
			Regression: true, Unstable: true,
			Baseline: &suite.RunEvidence{Outcome: suite.OutcomePass, ClientVersion: "evil: https://example.org/?token=SUPERSECRET", EndpointFingerprint: "sha256:SECRET"},
			Attempts: []suite.AttemptComparison{
				{Attempt: 1, State: suite.AttemptCompared, Regression: true, Evidence: &suite.RunEvidence{
					Outcome: suite.OutcomeNonPass, ClientVersion: "<script>alert('xss')</script>", Artifact: "/Users/private/secret.json", Stages: []artifact.StageResult{{Stage: interop.StageTools, Status: interop.StatusFail, ReasonCode: "Bearer ACCESS_PRIVATE_SECRET"}},
				}, StageChanges: []interopcompare.StageChange{{Stage: interop.StageTools, OldStatus: interop.StatusPass, NewStatus: interop.StatusFail, OldReasonCode: "SAFE", NewReasonCode: "TOKEN-SECRET", Regression: true}}},
				{Attempt: 2, State: suite.AttemptCompared, Evidence: &suite.RunEvidence{Outcome: suite.OutcomePass, ClientVersion: "https://evil.example/x", EndpointFingerprint: "secret-session-token"}},
			},
		}},
	}
}

func TestSuiteRenderOfflineHTMLAndCIRedactsUntrustedContent(t *testing.T) {
	report := reportingFixture()
	html, err := SuiteHTML(report)
	if err != nil {
		t.Fatal(err)
	}
	md, err := SuiteCI(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{string(html), string(md)} {
		for _, forbidden := range []string{"SUPERSECRET", "ACCESS_PRIVATE_SECRET", "<script", "https://", "/Users/private", "TOKEN-SECRET", "secret-session-token", "secret.json", "<iframe", "http-equiv=\"Refresh\""} {
			if strings.Contains(data, forbidden) {
				t.Fatalf("unsafe content %q in derived report: %s", forbidden, data)
			}
		}
		for _, required := range []string{"target-a", "codex", "regression_and_unstable", "non_pass", "pass"} {
			if !strings.Contains(strings.ToLower(data), required) {
				t.Fatalf("missing %q from report: %s", required, data)
			}
		}
	}
	if !strings.Contains(string(html), "tools") {
		t.Fatal("HTML omitted stage transition")
	}
	if !strings.Contains(string(html), `default-src 'none'`) {
		t.Fatal("offline HTML lacks restrictive CSP")
	}
	if strings.Contains(string(html), "<script") || strings.Contains(string(html), "src=") || strings.Contains(string(html), "href=") {
		t.Fatal("HTML contains script, link, or external asset")
	}
	if !strings.Contains(string(md), "| target-a | codex | 1 | compared | non_pass | true |") {
		t.Fatal("CI summary omitted first failing attempt")
	}
	if !strings.Contains(string(md), "| target-a | codex | 2 | compared | pass | false |") {
		t.Fatal("CI summary omitted second successful attempt")
	}
}

func TestSuiteReportFailsClosedOnUnsafeIdentityAndShape(t *testing.T) {
	cases := map[string]func(*suite.RegressionReport){
		"invalid identity":  func(r *suite.RegressionReport) { r.ArtifactType = "unknown" },
		"unknown decision":  func(r *suite.RegressionReport) { r.Decision = "mystery" },
		"target inject":     func(r *suite.RegressionReport) { r.Runs[0].TargetID = "a<script>" },
		"client inject":     func(r *suite.RegressionReport) { r.Runs[0].ClientID = "evil" },
		"auth inject":       func(r *suite.RegressionReport) { r.Runs[0].AuthMode = "fake" },
		"attempt reorder":   func(r *suite.RegressionReport) { r.Runs[0].Attempts[0].Attempt = 2 },
		"missing attempts":  func(r *suite.RegressionReport) { r.Runs[0].Attempts = r.Runs[0].Attempts[:1] },
		"too many attempts": func(r *suite.RegressionReport) { r.AttemptCount = MaxAttempts + 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := reportingFixture()
			mutate(&r)
			if _, err := SuiteHTML(r); err == nil {
				t.Fatal("HTML accepted invalid report")
			}
			if _, err := SuiteCI(r); err == nil {
				t.Fatal("CI summary accepted invalid report")
			}
		})
	}
}

func TestSuiteReportDistinguishesUntestedUnknownAndExecutionError(t *testing.T) {
	r := reportingFixture()
	r.Runs[0].Attempts[0].State = suite.AttemptMissingEvidence
	r.Runs[0].Attempts[0].Evidence = nil
	r.Runs[0].Attempts[1].State = suite.AttemptExecutionError
	r.Runs[0].Attempts[1].Evidence = &suite.RunEvidence{Outcome: suite.OutcomeError}
	data, err := SuiteHTML(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "missing_evidence") || !strings.Contains(string(data), "not_observed") || !strings.Contains(string(data), "execution_error") {
		t.Fatalf("missing/unknown evidence silently promoted: %s", data)
	}
}
