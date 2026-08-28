package interop

import (
	"reflect"
	"regexp"
	"testing"
)

func TestV1CandidateReasonCodesRemainStableAndExtensible(t *testing.T) {
	got := []ReasonCode{
		ReasonDCRUnsupported,
		ReasonDCRFailed,
		ReasonOAuthCallbackPortConflict,
		ReasonRegistrationStrategyUnsupported,
		ReasonTokenAuthMethodMismatch,
		ReasonClientAuthRejected,
		ReasonTokenRequestRejected,
		ReasonResourceMismatch,
		ReasonRedirectURIMismatch,
		ReasonPKCES256Missing,
		ReasonPKCEVerifierMissing,
		ReasonAccessTokenNotAttached,
		ReasonTokenSignatureInvalid,
		ReasonTokenIssuerMismatch,
		ReasonTokenAudienceMismatch,
		ReasonTokenExpired,
		ReasonInsufficientScope,
		ReasonToolOAuthMetadataMissing,
		ReasonToolOAuthChallengeMissing,
		ReasonToolOAuthChallengeInvalid,
	}
	want := []ReasonCode{
		"DCR_UNSUPPORTED",
		"DCR_FAILED",
		"OAUTH_CALLBACK_PORT_CONFLICT",
		"REGISTRATION_STRATEGY_UNSUPPORTED",
		"TOKEN_AUTH_METHOD_MISMATCH",
		"CLIENT_AUTH_REJECTED",
		"TOKEN_REQUEST_REJECTED",
		"RESOURCE_MISMATCH",
		"REDIRECT_URI_MISMATCH",
		"PKCE_S256_MISSING",
		"PKCE_VERIFIER_MISSING",
		"ACCESS_TOKEN_NOT_ATTACHED",
		"TOKEN_SIGNATURE_INVALID",
		"TOKEN_ISSUER_MISMATCH",
		"TOKEN_AUDIENCE_MISMATCH",
		"TOKEN_EXPIRED",
		"INSUFFICIENT_SCOPE",
		"TOOL_OAUTH_METADATA_MISSING",
		"TOOL_OAUTH_CHALLENGE_MISSING",
		"TOOL_OAUTH_CHALLENGE_INVALID",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stable reason-code contract changed:\n got=%v\nwant=%v", got, want)
	}
	pattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	seen := map[ReasonCode]bool{}
	for _, code := range got {
		if seen[code] || !pattern.MatchString(string(code)) {
			t.Fatalf("invalid or duplicate stable reason code %q", code)
		}
		seen[code] = true
	}

	// ReasonCode is deliberately an open string enum: future producers can add
	// a new machine code without forcing older readers to reinterpret it.
	var future ReasonCode = "FUTURE_DIRECT_EVIDENCE_CODE"
	if future == "" || !pattern.MatchString(string(future)) {
		t.Fatal("future reason-code strings must remain representable")
	}
}
