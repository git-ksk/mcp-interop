package diagnose

import (
	"errors"
	"net/url"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

// ChatGPTRuntimeEvidence is a deliberately secret-free observation of the
// OAuth token request. It accepts presence/match booleans, never token values,
// authorization codes, PKCE verifiers, assertions, cookies, or credentials.
//
// client_assertion_present is the minimum runtime signal needed to compare a
// metadata-selected private_key_jwt flow with an observed unauthenticated token
// request. Other observations are optional and are reported as unknown when
// they were not captured by the server's sanitized logging.
type ChatGPTRuntimeEvidence struct {
	ClientID                   string `json:"client_id"`
	ResourceMatches            *bool  `json:"resource_matches,omitempty"`
	CodeVerifierPresent        *bool  `json:"code_verifier_present,omitempty"`
	ClientAssertionPresent     *bool  `json:"client_assertion_present"`
	ClientAssertionTypePresent *bool  `json:"client_assertion_type_present,omitempty"`
}

func (e ChatGPTRuntimeEvidence) Validate() error {
	clientURL, err := url.Parse(e.ClientID)
	if err != nil || clientURL.Scheme != "https" || clientURL.Host == "" || clientURL.RawQuery != "" || clientURL.Fragment != "" {
		return errors.New("client_id must be a stable HTTPS CIMD URL without query or fragment")
	}
	if e.ClientAssertionPresent == nil {
		return errors.New("client_assertion_present is required")
	}
	return nil
}

type RuntimeCheck struct {
	ID         string             `json:"id"`
	Status     Status             `json:"status"`
	Expected   string             `json:"expected"`
	Observed   string             `json:"observed"`
	ReasonCode interop.ReasonCode `json:"reason_code,omitempty"`
	Message    string             `json:"message"`
}

type RuntimeEvidenceReport struct {
	ClientID   string             `json:"client_id"`
	Status     Status             `json:"status"`
	ReasonCode interop.ReasonCode `json:"reason_code,omitempty"`
	Checks     []RuntimeCheck     `json:"checks"`
}

func (r RuntimeEvidenceReport) Passed() bool {
	return r.Status != StatusFail
}

func evaluateRuntimeEvidence(report *Report, evidence ChatGPTRuntimeEvidence) {
	runtime := &RuntimeEvidenceReport{
		ClientID: interop.SanitizeEndpoint(evidence.ClientID),
		Status:   StatusPass,
	}

	registrationObserved := "unknown"
	registrationStatus := StatusWarn
	if report.Client != nil && report.Client.ClientID == sanitizePublicURL(evidence.ClientID) {
		registrationObserved = "CIMD"
		registrationStatus = StatusPass
	}
	runtime.add(RuntimeCheck{
		ID:       "registration_strategy",
		Status:   registrationStatus,
		Expected: "CIMD",
		Observed: registrationObserved,
		Message:  "Observed client_id is correlated with the fetched ChatGPT CIMD document when available",
	})

	if evidence.ResourceMatches == nil {
		runtime.add(RuntimeCheck{
			ID:       "resource",
			Status:   StatusWarn,
			Expected: "canonical URL match",
			Observed: "unknown",
			Message:  "resource match was not included in the supplied sanitized runtime evidence",
		})
	} else {
		resourceObserved := "mismatch"
		resourceStatus := StatusFail
		if *evidence.ResourceMatches {
			resourceObserved = "canonical URL match"
			resourceStatus = StatusPass
		}
		runtime.add(RuntimeCheck{
			ID:       "resource",
			Status:   resourceStatus,
			Expected: "canonical URL match",
			Observed: resourceObserved,
			Message:  "Sanitized token-request evidence reports whether resource matched the canonical MCP resource",
		})
	}

	if evidence.CodeVerifierPresent == nil {
		runtime.add(RuntimeCheck{
			ID:       "pkce_verifier",
			Status:   StatusWarn,
			Expected: "present",
			Observed: "unknown",
			Message:  "code_verifier presence was not included in the supplied sanitized runtime evidence",
		})
	} else {
		verifierObserved := "absent"
		verifierStatus := StatusFail
		if *evidence.CodeVerifierPresent {
			verifierObserved = "present"
			verifierStatus = StatusPass
		}
		runtime.add(RuntimeCheck{
			ID:       "pkce_verifier",
			Status:   verifierStatus,
			Expected: "present",
			Observed: verifierObserved,
			Message:  "Only code_verifier presence is observed; the verifier value is never ingested",
		})
	}

	expectedAuth := expectedChatGPTTokenAuth(report)
	observedAuth, conclusive := observedTokenAuth(evidence)
	authCheck := RuntimeCheck{
		ID:       "token_auth_method",
		Expected: expectedAuth,
		Observed: observedAuth,
		Message:  "Compares metadata-selected token endpoint authentication with secret-free request evidence",
	}
	switch {
	case expectedAuth == "unknown" || !conclusive:
		authCheck.Status = StatusWarn
	case expectedAuth == observedAuth:
		authCheck.Status = StatusPass
	default:
		authCheck.Status = StatusFail
		authCheck.ReasonCode = interop.ReasonTokenAuthMethodMismatch
		runtime.ReasonCode = interop.ReasonTokenAuthMethodMismatch
	}
	runtime.add(authCheck)

	report.RuntimeEvidence = runtime
}

func (r *RuntimeEvidenceReport) add(check RuntimeCheck) {
	r.Checks = append(r.Checks, check)
	if check.Status == StatusFail {
		r.Status = StatusFail
	} else if check.Status == StatusWarn && r.Status == StatusPass {
		r.Status = StatusWarn
	}
}

func expectedChatGPTTokenAuth(report *Report) string {
	if report.Client == nil {
		return "unknown"
	}
	serverMethods := make([]string, 0)
	for _, server := range report.AuthorizationServers {
		if server.ClientIDMetadataDocumentSupported {
			serverMethods = append(serverMethods, server.TokenEndpointAuthMethodsSupported...)
		}
	}
	compatible := intersection(report.Client.TokenEndpointAuthMethodsSupported, serverMethods)
	if contains(compatible, "private_key_jwt") {
		return "private_key_jwt"
	}
	if contains(compatible, "none") {
		return "none"
	}
	return "unknown"
}

func observedTokenAuth(evidence ChatGPTRuntimeEvidence) (string, bool) {
	if !*evidence.ClientAssertionPresent {
		return "none", true
	}
	if evidence.ClientAssertionTypePresent == nil {
		return "client_assertion present", false
	}
	if *evidence.ClientAssertionTypePresent {
		return "private_key_jwt", true
	}
	return "malformed", true
}
