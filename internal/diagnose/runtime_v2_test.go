package diagnose

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func boolPtr(value bool) *bool { return &value }

func referenceCIMDReport() Report {
	clientID := "https://chatgpt.com/oauth/test/client.json"
	return Report{
		Profile:  "chatgpt",
		Endpoint: "https://example.com/mcp",
		AuthorizationServers: []AuthorizationServer{{
			Issuer:                            "https://auth.example.com",
			ClientIDMetadataDocumentSupported: true,
			TokenEndpointAuthMethodsSupported: []string{"none", "private_key_jwt"},
			CodeChallengeMethodsSupported:     []string{"S256"},
		}},
		Client: &ChatGPTClientEvidence{
			ClientID:                          clientID,
			TokenEndpointAuthMethodsSupported: []string{"none", "private_key_jwt"},
		},
	}
}

func TestRuntimeEvidenceV2OpenAIReferenceHappyPath(t *testing.T) {
	report := referenceCIMDReport()
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration: &RegistrationEvidence{
			Strategy:          "cimd",
			ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json",
		},
		AuthorizationRequest: &AuthorizationRequestEvidence{
			ResourceMatches:    boolPtr(true),
			RedirectURIMatches: boolPtr(true),
			PKCES256:           boolPtr(true),
		},
		TokenRequest: &TokenRequestEvidence{
			ResourceMatches:            boolPtr(true),
			CodeVerifierPresent:        boolPtr(true),
			ClientAssertionPresent:     boolPtr(true),
			ClientAssertionTypePresent: boolPtr(true),
		},
		ResourceRequest: &ResourceRequestEvidence{
			BearerPresent:    boolPtr(true),
			SignatureValid:   boolPtr(true),
			IssuerMatches:    boolPtr(true),
			AudienceMatches:  boolPtr(true),
			ExpiryValid:      boolPtr(true),
			ScopesSufficient: boolPtr(true),
		},
		ToolAuth: &ToolAuthEvidence{
			ChallengeExpected:                  boolPtr(true),
			OAuth2SecuritySchemePresent:        boolPtr(true),
			WWWAuthenticatePresent:             boolPtr(true),
			WWWAuthenticateHasError:            boolPtr(true),
			WWWAuthenticateHasErrorDescription: boolPtr(true),
		},
	}
	if err := evidence.Validate(); err != nil {
		t.Fatal(err)
	}
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil {
		t.Fatal("missing runtime evidence report")
	}
	if report.RuntimeEvidence.Status != StatusPass {
		t.Fatalf("runtime status=%s checks=%#v", report.RuntimeEvidence.Status, report.RuntimeEvidence.Checks)
	}
	if report.RuntimeEvidence.OpenAIReference == nil || report.RuntimeEvidence.OpenAIReference.Status != StatusPass {
		t.Fatalf("reference pattern=%#v", report.RuntimeEvidence.OpenAIReference)
	}
	reference := report.RuntimeEvidence.OpenAIReference
	if reference.ProfileRevision != openAIReferenceProfileRevision || reference.ObservedDate != openAIReferenceProfileObservedDate || reference.Source != openAIReferenceProfileSource {
		t.Fatalf("reference profile metadata=%#v", reference)
	}
	encoded, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"profile_revision":"2026-08-10.1"`, `"observed_date":"2026-08-10"`, `"source":"OpenAI authenticated MCP reference pattern"`} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("reference JSON missing %s: %s", want, encoded)
		}
	}
}

func TestRuntimeEvidenceV2DetectsChatGPTTokenAuthMismatch(t *testing.T) {
	report := referenceCIMDReport()
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration: &RegistrationEvidence{
			Strategy:          "cimd",
			ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json",
		},
		TokenRequest: &TokenRequestEvidence{
			ClientAssertionPresent:     boolPtr(false),
			ClientAssertionTypePresent: boolPtr(false),
		},
	}
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil || report.RuntimeEvidence.ReasonCode != interop.ReasonTokenAuthMethodMismatch {
		t.Fatalf("runtime=%#v", report.RuntimeEvidence)
	}
	assertRuntimeCheck(t, *report.RuntimeEvidence, "token_auth_method", StatusFail, "private_key_jwt", "none")
}

func TestRuntimeEvidenceV2DoesNotInferDCRTokenAuthMethod(t *testing.T) {
	report := referenceCIMDReport()
	report.AuthorizationServers[0].RegistrationEndpoint = "https://auth.example.com/register"
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration:  &RegistrationEvidence{Strategy: "dcr"},
		TokenRequest: &TokenRequestEvidence{
			ClientAssertionPresent: boolPtr(false),
		},
	}
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil {
		t.Fatal("missing runtime report")
	}
	assertRuntimeCheck(t, *report.RuntimeEvidence, "registration_strategy", StatusPass, "supported by discovered authorization metadata", "dcr")
	assertRuntimeCheck(t, *report.RuntimeEvidence, "token_auth_method", StatusWarn, "unknown", "none")
	if report.RuntimeEvidence.ReasonCode == interop.ReasonTokenAuthMethodMismatch {
		t.Fatal("DCR path must not infer CIMD private_key_jwt requirements")
	}
}

func TestRuntimeEvidenceV2DetectsResourceServerAudienceFailure(t *testing.T) {
	report := referenceCIMDReport()
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration: &RegistrationEvidence{
			Strategy:          "cimd",
			ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json",
		},
		ResourceRequest: &ResourceRequestEvidence{
			BearerPresent:    boolPtr(true),
			SignatureValid:   boolPtr(true),
			IssuerMatches:    boolPtr(true),
			AudienceMatches:  boolPtr(false),
			ExpiryValid:      boolPtr(true),
			ScopesSufficient: boolPtr(true),
		},
	}
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil || report.RuntimeEvidence.ReasonCode != interop.ReasonTokenAudienceMismatch {
		t.Fatalf("runtime=%#v", report.RuntimeEvidence)
	}
	if report.RuntimeEvidence.OpenAIReference == nil || report.RuntimeEvidence.OpenAIReference.Status != StatusFail {
		t.Fatalf("reference=%#v", report.RuntimeEvidence.OpenAIReference)
	}
}

func TestRuntimeEvidenceV2DetectsToolOAuthSignalFailures(t *testing.T) {
	report := referenceCIMDReport()
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration: &RegistrationEvidence{
			Strategy:          "cimd",
			ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json",
		},
		ToolAuth: &ToolAuthEvidence{
			ChallengeExpected:           boolPtr(true),
			OAuth2SecuritySchemePresent: boolPtr(false),
			WWWAuthenticatePresent:      boolPtr(false),
		},
	}
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil || report.RuntimeEvidence.ReasonCode != interop.ReasonToolOAuthMetadataMissing {
		t.Fatalf("runtime=%#v", report.RuntimeEvidence)
	}
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_security_scheme", StatusFail, "oauth2 securitySchemes metadata for an OAuth-protected tool", "false")
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_www_authenticate", StatusFail, "mcp/www_authenticate challenge when reauthorization is required", "false")
}

func TestRuntimeEvidenceV2MarksToolChallengeNotApplicableWhenGrantAlreadySatisfiesTool(t *testing.T) {
	report := referenceCIMDReport()
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration: &RegistrationEvidence{
			Strategy:          "cimd",
			ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json",
		},
		ToolAuth: &ToolAuthEvidence{
			ChallengeExpected:           boolPtr(false),
			OAuth2SecuritySchemePresent: boolPtr(true),
		},
	}
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil {
		t.Fatal("missing runtime evidence")
	}
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_security_scheme", StatusPass, "oauth2 securitySchemes metadata for an OAuth-protected tool", "true")
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_www_authenticate", StatusNA, "not required for the observed authorized tool call", "not applicable")
	coverage := report.RuntimeEvidence.Coverage
	if coverage.NotApplicable != 1 || coverage.Failed != 0 || coverage.Unknown != 0 {
		t.Fatalf("coverage=%#v", coverage)
	}
	if report.RuntimeEvidence.OpenAIReference == nil {
		t.Fatal("missing OpenAI reference report")
	}
	var toolSignals *RuntimeCheck
	for i := range report.RuntimeEvidence.OpenAIReference.Checks {
		if report.RuntimeEvidence.OpenAIReference.Checks[i].ID == "tool_oauth_signals" {
			toolSignals = &report.RuntimeEvidence.OpenAIReference.Checks[i]
			break
		}
	}
	if toolSignals == nil || toolSignals.Status != StatusPass {
		t.Fatalf("tool oauth reference=%#v", toolSignals)
	}
}

func TestRuntimeEvidenceV2MetadataMissingFailsEvenWhenChallengeNotExpected(t *testing.T) {
	report := referenceCIMDReport()
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration: &RegistrationEvidence{
			Strategy:          "cimd",
			ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json",
		},
		ToolAuth: &ToolAuthEvidence{
			ChallengeExpected:           boolPtr(false),
			OAuth2SecuritySchemePresent: boolPtr(false),
		},
	}
	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil || report.RuntimeEvidence.ReasonCode != interop.ReasonToolOAuthMetadataMissing {
		t.Fatalf("runtime=%#v", report.RuntimeEvidence)
	}
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_security_scheme", StatusFail, "oauth2 securitySchemes metadata for an OAuth-protected tool", "false")
	assertRuntimeCheck(t, *report.RuntimeEvidence, "tool_oauth_www_authenticate", StatusNA, "not required for the observed authorized tool call", "not applicable")
}

func TestRuntimeEvidenceV2RejectsUnknownSecretBearingFields(t *testing.T) {
	body := []byte(`{
		"schema_version": 2,
		"registration": {"strategy": "cimd", "client_metadata_url": "https://chatgpt.com/oauth/test/client.json"},
		"token_request": {"client_assertion_present": false},
		"access_token": "secret"
	}`)
	var evidence ChatGPTRuntimeEvidence
	if err := json.Unmarshal(body, &evidence); err == nil {
		t.Fatal("expected unknown secret-bearing field to be rejected")
	}
}

func TestRuntimeEvidenceLegacyV1StillNormalizesToCIMD(t *testing.T) {
	body := []byte(`{
		"client_id": "https://chatgpt.com/oauth/test/client.json",
		"resource_matches": true,
		"code_verifier_present": true,
		"client_assertion_present": false
	}`)
	var evidence ChatGPTRuntimeEvidence
	if err := json.Unmarshal(body, &evidence); err != nil {
		t.Fatal(err)
	}
	if err := evidence.Validate(); err != nil {
		t.Fatal(err)
	}
	if evidence.SchemaVersion != 1 || evidence.EffectiveRegistrationStrategy() != "cimd" {
		t.Fatalf("unexpected normalized evidence: %#v", evidence)
	}
	if evidence.TokenRequest == nil || evidence.TokenRequest.CodeVerifierPresent == nil || !*evidence.TokenRequest.CodeVerifierPresent {
		t.Fatalf("legacy token evidence was not normalized: %#v", evidence.TokenRequest)
	}
}
