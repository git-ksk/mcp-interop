package interop

// ReasonCode is a stable, machine-readable explanation for a stage result.
// Codes are added conservatively when mcp-interop can distinguish a failure
// mode from evidence returned by the real client.
type ReasonCode string

const (
	// ReasonDCRUnsupported means the real client explicitly reported that
	// Dynamic Client Registration is not supported for the OAuth target.
	ReasonDCRUnsupported ReasonCode = "DCR_UNSUPPORTED"

	// ReasonDCRFailed means the real client explicitly reported that a Dynamic
	// Client Registration attempt failed for a reason other than unsupported.
	ReasonDCRFailed ReasonCode = "DCR_FAILED"

	// ReasonTokenAuthMethodMismatch means sanitized runtime evidence shows that
	// the client used a different token endpoint authentication method from the
	// method selected by the published client/server metadata.
	ReasonTokenAuthMethodMismatch ReasonCode = "TOKEN_AUTH_METHOD_MISMATCH"
)
