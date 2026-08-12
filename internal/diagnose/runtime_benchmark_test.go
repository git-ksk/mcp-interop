package diagnose

import "testing"

func BenchmarkEvaluateRuntimeEvidenceReferencePattern(b *testing.B) {
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 3,
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

	b.ReportAllocs()
	for b.Loop() {
		report := referenceCIMDReport()
		evaluateRuntimeEvidence(&report, evidence)
		if report.RuntimeEvidence == nil || report.RuntimeEvidence.Status != StatusPass {
			b.Fatalf("unexpected runtime evidence: %#v", report.RuntimeEvidence)
		}
	}
}
