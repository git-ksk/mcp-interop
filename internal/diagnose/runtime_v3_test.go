package diagnose

import (
	"encoding/json"
	"testing"
)

func TestRuntimeEvidenceV3SplitsToolMetadataAndChallenge(t *testing.T) {
	body := []byte(`{
		"schema_version": 3,
		"registration": {"strategy": "cimd", "client_metadata_url": "https://chatgpt.com/oauth/test/client.json"},
		"tool_metadata": {"oauth2_security_scheme_present": true},
		"tool_challenge": {"expected": false}
	}`)
	var evidence ChatGPTRuntimeEvidence
	if err := json.Unmarshal(body, &evidence); err != nil {
		t.Fatal(err)
	}
	if err := evidence.Validate(); err != nil {
		t.Fatal(err)
	}
	if evidence.SchemaVersion != 3 || evidence.ToolAuth != nil || evidence.ToolMetadata == nil || evidence.ToolChallenge == nil {
		t.Fatalf("unexpected v3 evidence: %#v", evidence)
	}

	report := referenceCIMDReport()
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil || report.RuntimeEvidence.SchemaVersion != 3 {
		t.Fatalf("runtime=%#v", report.RuntimeEvidence)
	}
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_security_scheme", StatusPass, "oauth2 securitySchemes metadata for an OAuth-protected tool", "true")
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_www_authenticate", StatusNA, "not required for the observed authorized tool call", "not applicable")
	if report.RuntimeEvidence.Coverage.NotApplicable != 1 || report.RuntimeEvidence.Coverage.Failed != 0 {
		t.Fatalf("coverage=%#v", report.RuntimeEvidence.Coverage)
	}
}

func TestRuntimeEvidenceV3InfersSchemaFromSplitToolSections(t *testing.T) {
	body := []byte(`{
		"tool_metadata": {"oauth2_security_scheme_present": true}
	}`)
	var evidence ChatGPTRuntimeEvidence
	if err := json.Unmarshal(body, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.SchemaVersion != 3 {
		t.Fatalf("schema=%d, want 3", evidence.SchemaVersion)
	}
	if err := evidence.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeEvidenceV3DoesNotInventMissingToolMetadataObservation(t *testing.T) {
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 3,
		ToolChallenge: &ToolChallengeEvidence{
			Expected: boolPtr(false),
		},
	}
	report := referenceCIMDReport()
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil {
		t.Fatal("missing runtime evidence")
	}
	for _, check := range report.RuntimeEvidence.Checks {
		if check.ID == "tool_oauth_security_scheme" {
			t.Fatalf("v3 should not evaluate absent tool_metadata section: %#v", check)
		}
	}
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_www_authenticate", StatusNA, "not required for the observed authorized tool call", "not applicable")
	if report.RuntimeEvidence.OpenAIReference == nil {
		t.Fatal("missing OpenAI reference report")
	}
	for _, check := range report.RuntimeEvidence.OpenAIReference.Checks {
		if check.ID != "tool_oauth_signals" {
			continue
		}
		if check.Status != StatusWarn || check.Observed != "partial" {
			t.Fatalf("tool_oauth_signals=%#v, want WARN/partial", check)
		}
		return
	}
	t.Fatal("missing tool_oauth_signals reference check")
}

func TestRuntimeEvidenceV2AndV3ToolEvidenceEvaluateEquivalently(t *testing.T) {
	v2 := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		ToolAuth: &ToolAuthEvidence{
			ChallengeExpected:                  boolPtr(true),
			OAuth2SecuritySchemePresent:        boolPtr(true),
			WWWAuthenticatePresent:             boolPtr(true),
			WWWAuthenticateHasError:            boolPtr(true),
			WWWAuthenticateHasErrorDescription: boolPtr(true),
		},
	}
	v3 := ChatGPTRuntimeEvidence{
		SchemaVersion: 3,
		ToolMetadata: &ToolMetadataEvidence{
			OAuth2SecuritySchemePresent: boolPtr(true),
		},
		ToolChallenge: &ToolChallengeEvidence{
			Expected:                           boolPtr(true),
			WWWAuthenticatePresent:             boolPtr(true),
			WWWAuthenticateHasError:            boolPtr(true),
			WWWAuthenticateHasErrorDescription: boolPtr(true),
		},
	}

	v2Report := referenceCIMDReport()
	v3Report := referenceCIMDReport()
	evaluateRuntimeEvidence(&v2Report, v2)
	evaluateRuntimeEvidence(&v3Report, v3)
	if v2Report.RuntimeEvidence == nil || v3Report.RuntimeEvidence == nil {
		t.Fatal("missing runtime evidence")
	}
	if len(v2Report.RuntimeEvidence.Checks) != len(v3Report.RuntimeEvidence.Checks) {
		t.Fatalf("check count differs: v2=%d v3=%d", len(v2Report.RuntimeEvidence.Checks), len(v3Report.RuntimeEvidence.Checks))
	}
	for i := range v2Report.RuntimeEvidence.Checks {
		a := v2Report.RuntimeEvidence.Checks[i]
		b := v3Report.RuntimeEvidence.Checks[i]
		if a.ID != b.ID || a.Status != b.Status || a.Expected != b.Expected || a.Observed != b.Observed || a.ReasonCode != b.ReasonCode {
			t.Fatalf("check %d differs:\nv2=%#v\nv3=%#v", i, a, b)
		}
	}
}

func TestRuntimeEvidenceV3RejectsMixedV2ToolShape(t *testing.T) {
	body := []byte(`{
		"schema_version": 3,
		"tool_auth": {"challenge_expected": true},
		"tool_metadata": {"oauth2_security_scheme_present": true}
	}`)
	var evidence ChatGPTRuntimeEvidence
	if err := json.Unmarshal(body, &evidence); err == nil {
		t.Fatal("expected mixed v2/v3 tool evidence to be rejected")
	}
}

func TestRuntimeEvidenceV2RejectsV3ToolSections(t *testing.T) {
	body := []byte(`{
		"schema_version": 2,
		"tool_metadata": {"oauth2_security_scheme_present": true}
	}`)
	var evidence ChatGPTRuntimeEvidence
	if err := json.Unmarshal(body, &evidence); err == nil {
		t.Fatal("expected v3 tool section in schema v2 to be rejected")
	}
}

func TestRuntimeEvidenceV3RejectsUnknownSecretBearingFields(t *testing.T) {
	body := []byte(`{
		"schema_version": 3,
		"tool_challenge": {"expected": true, "access_token": "secret"}
	}`)
	var evidence ChatGPTRuntimeEvidence
	if err := json.Unmarshal(body, &evidence); err == nil {
		t.Fatal("expected unknown secret-bearing field to be rejected")
	}
}
