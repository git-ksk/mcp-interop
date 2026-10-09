// Package reporting renders privacy-bounded derived views of already validated
// suite comparison evidence. It never reads arbitrary URLs or client log text.
package reporting

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

const (
	MaxRuns        = 192
	MaxAttempts    = 20
	MaxOutputBytes = 2 << 20
)

type suiteView struct {
	Decision     string
	Regression   bool
	Unstable     bool
	AttemptCount int
	Runs         []suiteRunView
}

type suiteRunView struct {
	Target     string
	Client     string
	Auth       string
	Baseline   string
	Regression bool
	Unstable   bool
	Attempts   []attemptView
}

type attemptView struct {
	Number         int
	State          string
	Outcome        string
	Regression     bool
	VersionChanged bool
	Stages         []stageView
}

type stageView struct {
	Stage         string
	From          string
	To            string
	ReasonChanged bool
	Regression    bool
}

// safeEnum accepts only values from stable code-owned enumerations, never
// untrusted free-form versions, paths, messages or reason code payloads.
func safeEnum(value string, allowed ...string) string {
	for _, ok := range allowed {
		if value == ok {
			return value
		}
	}
	return "unknown"
}

func safeStage(value interop.Stage) string {
	return safeEnum(string(value), "reach", "auth", "init", "tools")
}
func safeStatus(value interop.Status) string {
	return safeEnum(string(value), "pass", "fail", "skip", "unknown")
}
func safeOutcome(value suite.RunOutcome) string {
	return safeEnum(string(value), "pass", "non_pass", "error")
}
func safeState(value string) string {
	return safeEnum(value, suite.AttemptCompared, suite.AttemptNewOnly, suite.AttemptMissingEvidence, suite.AttemptExecutionError, suite.AttemptBaselineUnavailable, suite.AttemptIdentityChanged)
}

func viewOf(report suite.RegressionReport) (suiteView, error) {
	if report.SchemaVersion != suite.RegressionReportSchemaVersion || report.ArtifactType != suite.RegressionReportArtifactType {
		return suiteView{}, errors.New("unsupported suite regression report schema")
	}
	decision := safeEnum(string(report.Decision), string(suite.DecisionClean), string(suite.DecisionRegression), string(suite.DecisionUnstable), string(suite.DecisionRegressionAndUnstable))
	if decision == "unknown" {
		return suiteView{}, errors.New("unknown suite decision")
	}
	if report.AttemptCount < 1 || report.AttemptCount > MaxAttempts || len(report.Runs) > MaxRuns {
		return suiteView{}, fmt.Errorf("report exceeds %d attempts or %d runs", MaxAttempts, MaxRuns)
	}
	v := suiteView{Decision: decision, Regression: report.HasRegression, Unstable: report.HasUnstable, AttemptCount: report.AttemptCount, Runs: make([]suiteRunView, 0, len(report.Runs))}
	for _, run := range report.Runs {
		// Identity fields originate in suite manifest v1 and are validated there.
		// Do not render deployment id, URLs, raw client version, local artifact
		// references, reason codes, runtime logs, fingerprints or diagnostics.
		if len(run.Attempts) != report.AttemptCount {
			return suiteView{}, errors.New("attempt evidence count mismatch")
		}
		auth := safeEnum(string(run.AuthMode), "none", "oauth")
		if auth == "unknown" {
			return suiteView{}, errors.New("unexpected auth value")
		}
		name := safeEnum(run.ClientID, "codex", "cursor", "antigravity")
		if name == "unknown" {
			return suiteView{}, errors.New("unrecognized client id")
		}
		// Treat target as an untrusted label even though manifest v1 limits it.
		target := safeTarget(run.TargetID)
		if target == "" {
			return suiteView{}, errors.New("invalid target label")
		}
		r := suiteRunView{Target: target, Client: name, Auth: auth, Regression: run.Regression, Unstable: run.Unstable, Attempts: make([]attemptView, 0, len(run.Attempts))}
		r.Baseline = "not_observed"
		if run.Baseline != nil {
			r.Baseline = safeOutcome(run.Baseline.Outcome)
		}
		for i, a := range run.Attempts {
			if a.Attempt != i+1 {
				return suiteView{}, errors.New("attempt ordering mismatch")
			}
			item := attemptView{Number: a.Attempt, State: safeState(a.State), Regression: a.Regression, VersionChanged: a.ClientVersionChanged, Outcome: "not_observed"}
			if a.Evidence != nil {
				item.Outcome = safeOutcome(a.Evidence.Outcome)
			}
			for _, change := range a.StageChanges {
				if len(item.Stages) >= 4 {
					return suiteView{}, errors.New("too many stage changes")
				}
				item.Stages = append(item.Stages, stageView{Stage: safeStage(change.Stage), From: safeStatus(change.OldStatus), To: safeStatus(change.NewStatus), ReasonChanged: change.OldReasonCode != change.NewReasonCode, Regression: change.Regression})
			}
			r.Attempts = append(r.Attempts, item)
		}
		v.Runs = append(v.Runs, r)
	}
	return v, nil
}

func safeTarget(s string) string {
	if len(s) < 1 || len(s) > 63 {
		return ""
	}
	for i, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '-' && i > 0 && i < len(s)-1)) {
			return ""
		}
	}
	return s
}

const offlineTemplate = `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; object-src 'none'; frame-src 'none'">
<title>MCP Interop — suite regression</title>
<style>
:root{color-scheme:light dark;font-family:system-ui,-apple-system,sans-serif}
body{max-width:980px;margin:2.5rem auto;padding:0 1.5rem;line-height:1.5}
header{border-bottom:1px solid #aaa;padding-bottom:1rem;margin-bottom:1.5rem}
small{color:#777}h1{margin-bottom:.3rem}h2{margin-top:2rem}table{border-collapse:collapse;width:100%;margin-bottom:1.3rem;font-size:.92rem}
th,td{padding:.58rem;text-align:left;border-bottom:1px solid #aaa;vertical-align:top}
th{font-weight:650}code{font-family:ui-monospace,monospace}section{margin-bottom:2rem}
.flag{font-weight:650}.note{border-left:3px solid #aaa;padding-left:1rem;color:#777}
</style></head><body>
<header><h1>MCP Interop — suite regression</h1><p>Decision: <strong>{{.Decision}}</strong> · Regression: <strong>{{if .Regression}}YES{{else}}NO{{end}}</strong> · Unstable: <strong>{{if .Unstable}}YES{{else}}NO{{end}}</strong> · Attempts: <strong>{{.AttemptCount}}</strong></p>
<p class="note">This offline report includes only bounded derived evidence. Unknown, missing, error and untested states are not a PASS. Optional named-tool evidence is separate.</p></header>
<main>
{{range .Runs}}<section><h2>{{.Target}} / {{.Client}} ({{.Auth}})</h2>
<p>Baseline: <strong>{{.Baseline}}</strong> · Regression: <strong>{{if .Regression}}YES{{else}}NO{{end}}</strong> · Unstable: <strong>{{if .Unstable}}YES{{else}}NO{{end}}</strong></p>
<table><thead><tr><th>Attempt</th><th>Evidence state</th><th>Outcome</th><th>Regression</th><th>Client version changed</th></tr></thead><tbody>
{{range .Attempts}}<tr><td>{{.Number}}</td><td>{{.State}}</td><td><strong>{{.Outcome}}</strong></td><td>{{if .Regression}}YES{{else}}NO{{end}}</td><td>{{if .VersionChanged}}YES{{else}}NO{{end}}</td></tr>
{{if .Stages}}<tr><td colspan="5"><small>Stage transitions (reason code values withheld):</small><table><thead><tr><th>Stage</th><th>Before</th><th>After</th><th>Reason changed</th><th>Regression</th></tr></thead><tbody>
{{range .Stages}}<tr><td>{{.Stage}}</td><td>{{.From}}</td><td>{{.To}}</td><td>{{if .ReasonChanged}}YES{{else}}NO{{end}}</td><td>{{if .Regression}}YES{{else}}NO{{end}}</td></tr>{{end}}
</tbody></table></td></tr>{{end}}
{{end}}</tbody></table></section>{{end}}
</main><footer><small>Generated locally from validated suite result sets. No scripts, remote assets, server URLs, raw logs, tokens, full client versions or file paths are included. This report is not a cryptographic attestation.</small></footer>
</body></html>`

var templateSuite = template.Must(template.New("suite").Parse(offlineTemplate))

// SuiteHTML creates an entirely offline HTML document from a deliberately
// minimized projection. html/template escapes all values automatically.
func SuiteHTML(report suite.RegressionReport) ([]byte, error) {
	view, err := viewOf(report)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := templateSuite.Execute(&out, view); err != nil {
		return nil, err
	}
	if out.Len() > MaxOutputBytes {
		return nil, errors.New("HTML report exceeds bounded output size")
	}
	return out.Bytes(), nil
}

// SuiteCI generates Markdown using only trusted enums and validated target
// names. It never embeds untrusted raw versions, paths, messages or links.
func SuiteCI(report suite.RegressionReport) ([]byte, error) {
	view, err := viewOf(report)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "## MCP Interop suite regression\n\nDecision: **%s** · Regression: **%t** · Unstable: **%t** · Attempts: **%d**\n\n", strings.ToUpper(view.Decision), view.Regression, view.Unstable, view.AttemptCount)
	fmt.Fprintln(&out, "| Target | Client | Attempt | Evidence | Outcome | Regression |")
	fmt.Fprintln(&out, "| --- | --- | ---: | --- | --- | --- |")
	for _, run := range view.Runs {
		for _, attempt := range run.Attempts {
			fmt.Fprintf(&out, "| %s | %s | %d | %s | %s | %t |\n", run.Target, run.Client, attempt.Number, attempt.State, attempt.Outcome, attempt.Regression)
		}
	}
	fmt.Fprintln(&out, "\nMissing, unknown, untested and execution-error evidence is not a passing observation. The source suite comparison retains all attempts; this summary deliberately omits URLs, raw logs, versions, credential material and local paths.")
	if out.Len() > MaxOutputBytes {
		return nil, errors.New("CI summary exceeds bounded output size")
	}
	return out.Bytes(), nil
}
