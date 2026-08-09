package diagnose

import (
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestRuntimeEvidenceDoesNotCombineTokenAuthAcrossAuthorizationServers(t *testing.T) {
	report := referenceCIMDReport()
	report.AuthorizationServers = []AuthorizationServer{
		{
			Issuer:                            "https://auth-one.example.com",
			ClientIDMetadataDocumentSupported: true,
			TokenEndpointAuthMethodsSupported: []string{"private_key_jwt"},
		},
		{
			Issuer:                            "https://auth-two.example.com",
			ClientIDMetadataDocumentSupported: true,
			TokenEndpointAuthMethodsSupported: []string{"none"},
		},
	}

	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration: &RegistrationEvidence{
			Strategy:          "cimd",
			ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json",
		},
		TokenRequest: &TokenRequestEvidence{
			ClientAssertionPresent: boolPtr(false),
		},
	}

	evaluateRuntimeEvidence(&report, evidence)
	if report.RuntimeEvidence == nil {
		t.Fatal("missing runtime evidence report")
	}
	assertRuntimeCheck(t, *report.RuntimeEvidence, "token_auth_method", StatusWarn, "unknown", "none")
	if report.RuntimeEvidence.ReasonCode == interop.ReasonTokenAuthMethodMismatch {
		t.Fatal("ambiguous authorization-server selection must not produce TOKEN_AUTH_METHOD_MISMATCH")
	}
}
