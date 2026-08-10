package interop

// ReasonCode is a stable, machine-readable explanation for an interoperability
// or diagnostic result. Codes are added conservatively when mcp-interop has
// explicit client, server, or sanitized runtime evidence for a specific failure.
type ReasonCode string

const (
	// ReasonDCRUnsupported means the real client explicitly reported that
	// Dynamic Client Registration is not supported for the OAuth target.
	ReasonDCRUnsupported ReasonCode = "DCR_UNSUPPORTED"

	// ReasonDCRFailed means the real client explicitly reported that a Dynamic
	// Client Registration attempt failed for a reason other than unsupported.
	ReasonDCRFailed ReasonCode = "DCR_FAILED"

	// ReasonOAuthCallbackPortConflict means the real client explicitly reported
	// that its loopback OAuth callback listener could not bind the selected port.
	ReasonOAuthCallbackPortConflict ReasonCode = "OAUTH_CALLBACK_PORT_CONFLICT"

	// ReasonRegistrationStrategyUnsupported means sanitized runtime evidence says
	// a registration strategy was used that the discovered authorization-server
	// metadata does not advertise.
	ReasonRegistrationStrategyUnsupported ReasonCode = "REGISTRATION_STRATEGY_UNSUPPORTED"

	// ReasonTokenAuthMethodMismatch means sanitized runtime evidence shows that
	// the client used a different token endpoint authentication method from the
	// method selected by the published client/server metadata.
	ReasonTokenAuthMethodMismatch ReasonCode = "TOKEN_AUTH_METHOD_MISMATCH"

	// ReasonClientAuthRejected means the token endpoint returned the standard
	// OAuth invalid_client error for the observed request.
	ReasonClientAuthRejected ReasonCode = "CLIENT_AUTH_REJECTED"

	// ReasonTokenRequestRejected means the token endpoint returned a sanitized
	// OAuth error that is not specific enough for a narrower classification.
	ReasonTokenRequestRejected ReasonCode = "TOKEN_REQUEST_REJECTED"

	// ReasonResourceMismatch means an observed OAuth request carried a resource
	// value that did not match the canonical MCP protected resource.
	ReasonResourceMismatch ReasonCode = "RESOURCE_MISMATCH"

	// ReasonRedirectURIMismatch means the observed redirect URI did not match the
	// client metadata / connection configuration being diagnosed.
	ReasonRedirectURIMismatch ReasonCode = "REDIRECT_URI_MISMATCH"

	// ReasonPKCES256Missing means the observed authorization request did not use
	// the S256 PKCE method expected by the ChatGPT profile.
	ReasonPKCES256Missing ReasonCode = "PKCE_S256_MISSING"

	// ReasonPKCEVerifierMissing means the observed token request omitted the PKCE
	// code_verifier presence signal.
	ReasonPKCEVerifierMissing ReasonCode = "PKCE_VERIFIER_MISSING"

	// ReasonAccessTokenNotAttached means an authenticated MCP resource request was
	// observed without an Authorization bearer token.
	ReasonAccessTokenNotAttached ReasonCode = "ACCESS_TOKEN_NOT_ATTACHED"

	// ReasonTokenSignatureInvalid means the resource server reported that the
	// bearer token signature could not be validated.
	ReasonTokenSignatureInvalid ReasonCode = "TOKEN_SIGNATURE_INVALID"

	// ReasonTokenIssuerMismatch means the resource server reported that the token
	// issuer did not match the configured issuer.
	ReasonTokenIssuerMismatch ReasonCode = "TOKEN_ISSUER_MISMATCH"

	// ReasonTokenAudienceMismatch means the resource server reported that the
	// token audience/resource did not match the protected MCP resource.
	ReasonTokenAudienceMismatch ReasonCode = "TOKEN_AUDIENCE_MISMATCH"

	// ReasonTokenExpired means the resource server reported that the bearer token
	// was expired at verification time.
	ReasonTokenExpired ReasonCode = "TOKEN_EXPIRED"

	// ReasonInsufficientScope means the resource server reported that the token
	// did not contain the scopes required for the attempted MCP operation.
	ReasonInsufficientScope ReasonCode = "INSUFFICIENT_SCOPE"

	// ReasonToolOAuthMetadataMissing means an auth-required tool was observed
	// without an oauth2 securitySchemes declaration needed for ChatGPT's
	// tool-level linking flow.
	ReasonToolOAuthMetadataMissing ReasonCode = "TOOL_OAUTH_METADATA_MISSING"

	// ReasonToolOAuthChallengeMissing means an auth-required tool error was
	// observed without the mcp/www_authenticate runtime challenge.
	ReasonToolOAuthChallengeMissing ReasonCode = "TOOL_OAUTH_CHALLENGE_MISSING"

	// ReasonToolOAuthChallengeInvalid means the runtime challenge was present but
	// did not carry the expected OAuth error / error_description signals.
	ReasonToolOAuthChallengeInvalid ReasonCode = "TOOL_OAUTH_CHALLENGE_INVALID"
)
