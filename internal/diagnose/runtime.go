package diagnose

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	runtimeEvidenceSchemaV2            = 2
	runtimeEvidenceSchemaV3            = 3
	openAIReferenceProfileRevision     = "2026-08-10.1"
	openAIReferenceProfileObservedDate = "2026-08-10"
	openAIReferenceProfileSource       = "OpenAI authenticated MCP reference pattern"
)

var oauthErrorPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// ChatGPTRuntimeEvidence is a deliberately secret-free observation of a
// ChatGPT OAuth/MCP flow. Schema v2 separates registration, authorization,
// token, resource-server, and combined tool-level signals. Schema v3 splits
// static tool metadata from runtime reauthorization challenges. Legacy v1 and
// v2 inputs remain accepted so existing evidence files keep working.
//
// Only presence/match booleans and a sanitized OAuth error code are accepted.
// Tokens, authorization codes, PKCE verifier values, client assertions,
// cookies, credentials, and arbitrary fields are rejected by the CLI decoder.
type ChatGPTRuntimeEvidence struct {
	SchemaVersion int `json:"schema_version,omitempty"`

	Registration         *RegistrationEvidence         `json:"registration,omitempty"`
	AuthorizationRequest *AuthorizationRequestEvidence `json:"authorization_request,omitempty"`
	TokenRequest         *TokenRequestEvidence         `json:"token_request,omitempty"`
	ResourceRequest      *ResourceRequestEvidence      `json:"resource_request,omitempty"`
	ToolAuth             *ToolAuthEvidence             `json:"tool_auth,omitempty"`
	ToolMetadata         *ToolMetadataEvidence         `json:"tool_metadata,omitempty"`
	ToolChallenge        *ToolChallengeEvidence        `json:"tool_challenge,omitempty"`

	// Legacy schema v1. These fields are intentionally retained for backwards
	// compatibility and are normalized into the v2 sections during decoding.
	ClientID                   string `json:"client_id,omitempty"`
	ResourceMatches            *bool  `json:"resource_matches,omitempty"`
	CodeVerifierPresent        *bool  `json:"code_verifier_present,omitempty"`
	ClientAssertionPresent     *bool  `json:"client_assertion_present,omitempty"`
	ClientAssertionTypePresent *bool  `json:"client_assertion_type_present,omitempty"`
}

type RegistrationEvidence struct {
	Strategy          string `json:"strategy"`
	ClientMetadataURL string `json:"client_metadata_url,omitempty"`
}

type AuthorizationRequestEvidence struct {
	ResourceMatches    *bool `json:"resource_matches,omitempty"`
	RedirectURIMatches *bool `json:"redirect_uri_matches,omitempty"`
	PKCES256           *bool `json:"pkce_s256,omitempty"`
}

type TokenRequestEvidence struct {
	ResourceMatches            *bool  `json:"resource_matches,omitempty"`
	CodeVerifierPresent        *bool  `json:"code_verifier_present,omitempty"`
	ClientAssertionPresent     *bool  `json:"client_assertion_present,omitempty"`
	ClientAssertionTypePresent *bool  `json:"client_assertion_type_present,omitempty"`
	OAuthError                 string `json:"oauth_error,omitempty"`
}

type ResourceRequestEvidence struct {
	BearerPresent    *bool `json:"bearer_present,omitempty"`
	SignatureValid   *bool `json:"signature_valid,omitempty"`
	IssuerMatches    *bool `json:"issuer_matches,omitempty"`
	AudienceMatches  *bool `json:"audience_matches,omitempty"`
	ExpiryValid      *bool `json:"expiry_valid,omitempty"`
	ScopesSufficient *bool `json:"scopes_sufficient,omitempty"`
}

// ToolAuthEvidence is the schema v2 combined tool-level shape. New evidence
// producers should use ToolMetadataEvidence and ToolChallengeEvidence in v3.
type ToolAuthEvidence struct {
	ChallengeExpected                  *bool `json:"challenge_expected,omitempty"`
	OAuth2SecuritySchemePresent        *bool `json:"oauth2_security_scheme_present,omitempty"`
	WWWAuthenticatePresent             *bool `json:"www_authenticate_present,omitempty"`
	WWWAuthenticateHasError            *bool `json:"www_authenticate_has_error,omitempty"`
	WWWAuthenticateHasErrorDescription *bool `json:"www_authenticate_has_error_description,omitempty"`
}

type ToolMetadataEvidence struct {
	OAuth2SecuritySchemePresent *bool `json:"oauth2_security_scheme_present,omitempty"`
}

type ToolChallengeEvidence struct {
	Expected                           *bool `json:"expected,omitempty"`
	WWWAuthenticatePresent             *bool `json:"www_authenticate_present,omitempty"`
	WWWAuthenticateHasError            *bool `json:"www_authenticate_has_error,omitempty"`
	WWWAuthenticateHasErrorDescription *bool `json:"www_authenticate_has_error_description,omitempty"`
}

type runtimeEvidenceWire ChatGPTRuntimeEvidence

func (e *ChatGPTRuntimeEvidence) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire runtimeEvidenceWire
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return err
	}

	value := ChatGPTRuntimeEvidence(wire)
	hasCommon := value.Registration != nil || value.AuthorizationRequest != nil || value.TokenRequest != nil || value.ResourceRequest != nil
	hasV2Tool := value.ToolAuth != nil
	hasV3Tool := value.ToolMetadata != nil || value.ToolChallenge != nil
	hasLegacy := value.ClientID != "" || value.ResourceMatches != nil || value.CodeVerifierPresent != nil || value.ClientAssertionPresent != nil || value.ClientAssertionTypePresent != nil

	if value.SchemaVersion != 0 && value.SchemaVersion != 1 && value.SchemaVersion != runtimeEvidenceSchemaV2 && value.SchemaVersion != runtimeEvidenceSchemaV3 {
		return fmt.Errorf("unsupported runtime evidence schema_version %d", value.SchemaVersion)
	}
	if hasV2Tool && hasV3Tool {
		return errors.New("runtime evidence cannot mix schema v2 tool_auth with schema v3 tool_metadata/tool_challenge")
	}
	if (hasCommon || hasV2Tool || hasV3Tool) && hasLegacy {
		return errors.New("runtime evidence cannot mix legacy v1 fields with structured evidence sections")
	}
	if value.SchemaVersion == 1 && (hasCommon || hasV2Tool || hasV3Tool) {
		return errors.New("schema_version 1 cannot contain structured runtime evidence sections")
	}
	if value.SchemaVersion == runtimeEvidenceSchemaV2 && hasV3Tool {
		return errors.New("schema_version 2 cannot contain tool_metadata or tool_challenge")
	}
	if value.SchemaVersion == runtimeEvidenceSchemaV3 && hasV2Tool {
		return errors.New("schema_version 3 cannot contain legacy tool_auth")
	}
	if (value.SchemaVersion == runtimeEvidenceSchemaV2 || value.SchemaVersion == runtimeEvidenceSchemaV3) && hasLegacy {
		return fmt.Errorf("schema_version %d cannot contain legacy v1 runtime evidence fields", value.SchemaVersion)
	}

	switch {
	case value.SchemaVersion == runtimeEvidenceSchemaV3 || hasV3Tool:
		value.SchemaVersion = runtimeEvidenceSchemaV3
	case value.SchemaVersion == runtimeEvidenceSchemaV2 || hasCommon || hasV2Tool:
		value.SchemaVersion = runtimeEvidenceSchemaV2
	default:
		value.SchemaVersion = 1
		value.normalizeLegacy()
	}
	*e = value
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("runtime evidence must contain exactly one JSON object")
		}
		return err
	}
	return nil
}

func (e *ChatGPTRuntimeEvidence) normalizeLegacy() {
	if e.ClientID != "" {
		e.Registration = &RegistrationEvidence{Strategy: "cimd", ClientMetadataURL: e.ClientID}
	}
	if e.ResourceMatches != nil || e.CodeVerifierPresent != nil || e.ClientAssertionPresent != nil || e.ClientAssertionTypePresent != nil {
		e.TokenRequest = &TokenRequestEvidence{
			ResourceMatches:            e.ResourceMatches,
			CodeVerifierPresent:        e.CodeVerifierPresent,
			ClientAssertionPresent:     e.ClientAssertionPresent,
			ClientAssertionTypePresent: e.ClientAssertionTypePresent,
		}
	}
}

func (e ChatGPTRuntimeEvidence) Validate() error {
	hasCommon := e.Registration != nil || e.AuthorizationRequest != nil || e.TokenRequest != nil || e.ResourceRequest != nil
	hasV2Tool := e.ToolAuth != nil
	hasV3Tool := e.ToolMetadata != nil || e.ToolChallenge != nil
	hasLegacy := e.ClientID != "" || e.ResourceMatches != nil || e.CodeVerifierPresent != nil || e.ClientAssertionPresent != nil || e.ClientAssertionTypePresent != nil

	if e.SchemaVersion == 0 {
		switch {
		case hasV3Tool:
			e.SchemaVersion = runtimeEvidenceSchemaV3
		case hasCommon || hasV2Tool:
			e.SchemaVersion = runtimeEvidenceSchemaV2
		default:
			e.SchemaVersion = 1
		}
	}
	if e.SchemaVersion != 1 && e.SchemaVersion != runtimeEvidenceSchemaV2 && e.SchemaVersion != runtimeEvidenceSchemaV3 {
		return fmt.Errorf("unsupported runtime evidence schema_version %d", e.SchemaVersion)
	}
	if hasV2Tool && hasV3Tool {
		return errors.New("runtime evidence cannot mix schema v2 tool_auth with schema v3 tool_metadata/tool_challenge")
	}
	if e.SchemaVersion == 1 {
		// Legacy v1 is normalized internally into registration/token_request while
		// retaining the original v1 fields. Explicit schema v1 input containing
		// structured sections is rejected during JSON decoding.
		if !hasLegacy && (hasCommon || hasV2Tool || hasV3Tool) {
			return errors.New("schema_version 1 cannot contain structured runtime evidence sections")
		}
		if e.ClientID == "" {
			return errors.New("client_id is required for legacy runtime evidence")
		}
		if err := validateStableHTTPSURL(e.ClientID, "client_id"); err != nil {
			return err
		}
		if e.ClientAssertionPresent == nil {
			return errors.New("client_assertion_present is required for legacy runtime evidence")
		}
		return nil
	}
	if hasLegacy {
		return fmt.Errorf("schema_version %d cannot contain legacy v1 runtime evidence fields", e.SchemaVersion)
	}
	if e.SchemaVersion == runtimeEvidenceSchemaV2 && hasV3Tool {
		return errors.New("schema_version 2 cannot contain tool_metadata or tool_challenge")
	}
	if e.SchemaVersion == runtimeEvidenceSchemaV3 && hasV2Tool {
		return errors.New("schema_version 3 cannot contain legacy tool_auth")
	}
	if !hasCommon && !hasV2Tool && !hasV3Tool {
		return fmt.Errorf("schema_version %d runtime evidence must contain at least one evidence section", e.SchemaVersion)
	}
	if e.Registration != nil {
		strategy := strings.ToLower(strings.TrimSpace(e.Registration.Strategy))
		switch strategy {
		case "cimd":
			if e.Registration.ClientMetadataURL == "" {
				return errors.New("registration.client_metadata_url is required when strategy is cimd")
			}
			if err := validateStableHTTPSURL(e.Registration.ClientMetadataURL, "registration.client_metadata_url"); err != nil {
				return err
			}
		case "dcr", "predefined":
			if e.Registration.ClientMetadataURL != "" {
				return errors.New("registration.client_metadata_url is only valid when strategy is cimd")
			}
		default:
			return errors.New("registration.strategy must be one of cimd, dcr, or predefined")
		}
	}
	if e.TokenRequest != nil && e.TokenRequest.OAuthError != "" && !oauthErrorPattern.MatchString(e.TokenRequest.OAuthError) {
		return errors.New("token_request.oauth_error must be a short OAuth error code without whitespace")
	}
	return nil
}

func validateStableHTTPSURL(raw, field string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must be a stable HTTPS URL without userinfo, query, or fragment", field)
	}
	return nil
}

func (e ChatGPTRuntimeEvidence) EffectiveClientID() string {
	if e.Registration != nil && strings.EqualFold(e.Registration.Strategy, "cimd") {
		return e.Registration.ClientMetadataURL
	}
	return e.ClientID
}

func (e ChatGPTRuntimeEvidence) EffectiveRegistrationStrategy() string {
	if e.Registration != nil {
		return strings.ToLower(strings.TrimSpace(e.Registration.Strategy))
	}
	if e.ClientID != "" {
		return "cimd"
	}
	return "unknown"
}

func (e ChatGPTRuntimeEvidence) effectiveToolMetadata() *ToolMetadataEvidence {
	if e.SchemaVersion == runtimeEvidenceSchemaV3 || e.ToolMetadata != nil || e.ToolChallenge != nil {
		return e.ToolMetadata
	}
	if e.ToolAuth == nil {
		return nil
	}
	return &ToolMetadataEvidence{OAuth2SecuritySchemePresent: e.ToolAuth.OAuth2SecuritySchemePresent}
}

func (e ChatGPTRuntimeEvidence) effectiveToolChallenge() *ToolChallengeEvidence {
	if e.SchemaVersion == runtimeEvidenceSchemaV3 || e.ToolMetadata != nil || e.ToolChallenge != nil {
		return e.ToolChallenge
	}
	if e.ToolAuth == nil {
		return nil
	}
	return &ToolChallengeEvidence{
		Expected:                           e.ToolAuth.ChallengeExpected,
		WWWAuthenticatePresent:             e.ToolAuth.WWWAuthenticatePresent,
		WWWAuthenticateHasError:            e.ToolAuth.WWWAuthenticateHasError,
		WWWAuthenticateHasErrorDescription: e.ToolAuth.WWWAuthenticateHasErrorDescription,
	}
}

type RuntimeCheck struct {
	ID         string             `json:"id"`
	Status     Status             `json:"status"`
	Expected   string             `json:"expected"`
	Observed   string             `json:"observed"`
	ReasonCode interop.ReasonCode `json:"reason_code,omitempty"`
	Message    string             `json:"message"`
}

type ReferencePatternReport struct {
	Status          Status         `json:"status"`
	ProfileRevision string         `json:"profile_revision"`
	ObservedDate    string         `json:"observed_date"`
	Source          string         `json:"source"`
	Checks          []RuntimeCheck `json:"checks"`
}

type EvidenceCoverage struct {
	Observed      int `json:"observed"`
	Passed        int `json:"passed"`
	Failed        int `json:"failed"`
	Unknown       int `json:"unknown"`
	NotApplicable int `json:"not_applicable"`
}

type RuntimeEvidenceReport struct {
	SchemaVersion        int                     `json:"schema_version"`
	RegistrationStrategy string                  `json:"registration_strategy,omitempty"`
	ClientID             string                  `json:"client_id,omitempty"`
	Status               Status                  `json:"status"`
	ReasonCode           interop.ReasonCode      `json:"reason_code,omitempty"`
	Checks               []RuntimeCheck          `json:"checks"`
	Coverage             EvidenceCoverage        `json:"coverage"`
	OpenAIReference      *ReferencePatternReport `json:"openai_reference_pattern,omitempty"`
}

func (r RuntimeEvidenceReport) Passed() bool {
	return r.Status != StatusFail
}

func evaluateRuntimeEvidence(report *Report, evidence ChatGPTRuntimeEvidence) {
	if evidence.SchemaVersion == 0 {
		switch {
		case evidence.ToolMetadata != nil || evidence.ToolChallenge != nil:
			evidence.SchemaVersion = runtimeEvidenceSchemaV3
		case evidence.Registration != nil || evidence.AuthorizationRequest != nil || evidence.TokenRequest != nil || evidence.ResourceRequest != nil || evidence.ToolAuth != nil:
			evidence.SchemaVersion = runtimeEvidenceSchemaV2
		default:
			evidence.SchemaVersion = 1
			evidence.normalizeLegacy()
		}
	}

	strategy := evidence.EffectiveRegistrationStrategy()
	runtime := &RuntimeEvidenceReport{
		SchemaVersion:        evidence.SchemaVersion,
		RegistrationStrategy: strategy,
		ClientID:             interop.SanitizeEndpoint(evidence.EffectiveClientID()),
		Status:               StatusPass,
	}

	evaluateRegistration(report, runtime, evidence)
	evaluateAuthorizationRequest(runtime, evidence.AuthorizationRequest)
	evaluateTokenRequest(report, runtime, evidence)
	evaluateResourceRequest(runtime, evidence.ResourceRequest)
	evaluateToolAuth(runtime, evidence.effectiveToolMetadata(), evidence.effectiveToolChallenge())
	runtime.Coverage = coverageForChecks(runtime.Checks)
	runtime.OpenAIReference = buildOpenAIReferencePattern(runtime)
	report.RuntimeEvidence = runtime
}

func evaluateRegistration(report *Report, runtime *RuntimeEvidenceReport, evidence ChatGPTRuntimeEvidence) {
	strategy := evidence.EffectiveRegistrationStrategy()
	check := RuntimeCheck{
		ID:       "registration_strategy",
		Expected: "supported by discovered authorization metadata",
		Observed: strategy,
		Message:  "Correlates the explicitly observed registration strategy with discovered authorization-server capabilities",
	}
	switch strategy {
	case "cimd":
		if !serverSupportsCIMD(report) {
			check.Status = StatusFail
			check.ReasonCode = interop.ReasonRegistrationStrategyUnsupported
		} else if report.Client == nil {
			check.Status = StatusWarn
			check.Message = "CIMD is advertised, but the exact observed client metadata document was not fetched"
		} else if report.Client.ClientID != sanitizePublicURL(evidence.EffectiveClientID()) {
			check.Status = StatusWarn
			check.Message = "CIMD is advertised, but the fetched client metadata identity did not correlate with the supplied runtime evidence"
		} else {
			check.Status = StatusPass
		}
	case "dcr":
		if serverSupportsDCR(report) {
			check.Status = StatusPass
		} else {
			check.Status = StatusFail
			check.ReasonCode = interop.ReasonRegistrationStrategyUnsupported
		}
	case "predefined":
		check.Status = StatusWarn
		check.Message = "Predefined client registration cannot be proven from public authorization-server metadata alone"
	default:
		check.Status = StatusWarn
		check.Observed = "unknown"
		check.Message = "Registration strategy was not included in the supplied sanitized runtime evidence"
	}
	runtime.add(check)
}

func evaluateAuthorizationRequest(runtime *RuntimeEvidenceReport, evidence *AuthorizationRequestEvidence) {
	if evidence == nil {
		return
	}
	runtime.add(matchCheck("authorization_resource", "canonical URL match", evidence.ResourceMatches, interop.ReasonResourceMismatch, "Authorization-request resource must match the canonical MCP protected resource"))
	runtime.add(matchCheck("authorization_redirect_uri", "registered redirect URI match", evidence.RedirectURIMatches, interop.ReasonRedirectURIMismatch, "Observed redirect URI should match the client metadata / connection configuration"))
	runtime.add(presenceCheck("authorization_pkce_s256", "S256", evidence.PKCES256, interop.ReasonPKCES256Missing, "Observed authorization request should use PKCE S256"))
}

func evaluateTokenRequest(report *Report, runtime *RuntimeEvidenceReport, evidence ChatGPTRuntimeEvidence) {
	token := evidence.TokenRequest
	if token == nil {
		return
	}
	runtime.add(matchCheck("token_resource", "canonical URL match", token.ResourceMatches, interop.ReasonResourceMismatch, "Token-request resource should match the canonical MCP protected resource"))
	runtime.add(presenceCheck("token_pkce_verifier", "present", token.CodeVerifierPresent, interop.ReasonPKCEVerifierMissing, "Only code_verifier presence is observed; the verifier value is never ingested"))

	expectedAuth := expectedChatGPTTokenAuthForStrategy(report, evidence.EffectiveRegistrationStrategy())
	observedAuth, conclusive := observedTokenAuth(token)
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
	}
	runtime.add(authCheck)

	if token.OAuthError != "" {
		reason := interop.ReasonTokenRequestRejected
		if token.OAuthError == "invalid_client" {
			reason = interop.ReasonClientAuthRejected
		}
		runtime.add(RuntimeCheck{
			ID:         "token_endpoint_result",
			Status:     StatusFail,
			Expected:   "token issued",
			Observed:   token.OAuthError,
			ReasonCode: reason,
			Message:    "Authorization server returned the supplied sanitized OAuth error code",
		})
	}
}

func evaluateResourceRequest(runtime *RuntimeEvidenceReport, evidence *ResourceRequestEvidence) {
	if evidence == nil {
		return
	}
	runtime.add(presenceCheck("resource_bearer", "present", evidence.BearerPresent, interop.ReasonAccessTokenNotAttached, "ChatGPT attaches the access token to subsequent MCP requests as a bearer token"))
	if evidence.BearerPresent != nil && !*evidence.BearerPresent {
		return
	}
	runtime.add(validityCheck("resource_token_signature", evidence.SignatureValid, interop.ReasonTokenSignatureInvalid, "Resource server should verify the bearer token signature"))
	runtime.add(matchCheck("resource_token_issuer", "configured issuer match", evidence.IssuerMatches, interop.ReasonTokenIssuerMismatch, "Resource server should verify the token issuer"))
	runtime.add(matchCheck("resource_token_audience", "protected resource match", evidence.AudienceMatches, interop.ReasonTokenAudienceMismatch, "Resource server should verify token audience/resource binding"))
	runtime.add(validityCheck("resource_token_expiry", evidence.ExpiryValid, interop.ReasonTokenExpired, "Resource server should reject expired bearer tokens"))
	runtime.add(presenceCheck("resource_token_scopes", "sufficient", evidence.ScopesSufficient, interop.ReasonInsufficientScope, "Resource server should enforce scopes required for the MCP operation"))
}

func evaluateToolAuth(runtime *RuntimeEvidenceReport, metadataEvidence *ToolMetadataEvidence, challengeEvidence *ToolChallengeEvidence) {
	if metadataEvidence != nil {
		metadata := RuntimeCheck{
			ID:       "tool_oauth_security_scheme",
			Expected: "oauth2 securitySchemes metadata for an OAuth-protected tool",
			Observed: boolObservation(metadataEvidence.OAuth2SecuritySchemePresent),
			Message:  "Per-tool OAuth metadata is independent of whether the current grant already satisfies the tool",
		}
		switch {
		case metadataEvidence.OAuth2SecuritySchemePresent == nil:
			metadata.Status = StatusWarn
		case *metadataEvidence.OAuth2SecuritySchemePresent:
			metadata.Status = StatusPass
		default:
			metadata.Status = StatusFail
			metadata.ReasonCode = interop.ReasonToolOAuthMetadataMissing
		}
		runtime.add(metadata)
	}

	if challengeEvidence == nil {
		return
	}
	challengeExpected := challengeEvidence.Expected != nil && *challengeEvidence.Expected
	challenge := RuntimeCheck{
		ID:       "tool_oauth_www_authenticate",
		Expected: "mcp/www_authenticate challenge when reauthorization is required",
		Observed: boolObservation(challengeEvidence.WWWAuthenticatePresent),
		Message:  "Runtime tool errors use _meta[mcp/www_authenticate] to trigger ChatGPT's tool-level OAuth UI",
	}
	switch {
	case challengeEvidence.Expected != nil && !*challengeEvidence.Expected:
		challenge.Status = StatusNA
		challenge.Expected = "not required for the observed authorized tool call"
		challenge.Observed = "not applicable"
		challenge.Message = "The current authorization already satisfied the tool, so no reauthorization challenge was expected"
	case challengeEvidence.WWWAuthenticatePresent == nil:
		challenge.Status = StatusWarn
	case *challengeEvidence.WWWAuthenticatePresent:
		challenge.Status = StatusPass
	case challengeExpected:
		challenge.Status = StatusFail
		challenge.ReasonCode = interop.ReasonToolOAuthChallengeMissing
	default:
		challenge.Status = StatusWarn
	}
	runtime.add(challenge)

	if challengeExpected && challengeEvidence.WWWAuthenticatePresent != nil && *challengeEvidence.WWWAuthenticatePresent {
		validateToolChallenge(runtime, challengeEvidence.WWWAuthenticateHasError, "tool_oauth_challenge_error", "error parameter")
		validateToolChallenge(runtime, challengeEvidence.WWWAuthenticateHasErrorDescription, "tool_oauth_challenge_error_description", "error_description parameter")
	}
}

func validateToolChallenge(runtime *RuntimeEvidenceReport, value *bool, id, expected string) {
	check := RuntimeCheck{
		ID:       id,
		Expected: expected + " present",
		Observed: boolObservation(value),
		Message:  "ChatGPT expects the tool-level WWW-Authenticate payload to carry a useful OAuth error and description",
	}
	switch {
	case value == nil:
		check.Status = StatusWarn
	case *value:
		check.Status = StatusPass
	default:
		check.Status = StatusFail
		check.ReasonCode = interop.ReasonToolOAuthChallengeInvalid
	}
	runtime.add(check)
}

func buildOpenAIReferencePattern(runtime *RuntimeEvidenceReport) *ReferencePatternReport {
	reference := &ReferencePatternReport{
		Status:          StatusPass,
		ProfileRevision: openAIReferenceProfileRevision,
		ObservedDate:    openAIReferenceProfileObservedDate,
		Source:          openAIReferenceProfileSource,
	}
	reference.add(referenceCheck(runtime, "registration_strategy", "registration"))
	reference.add(firstReferenceCheck(runtime, []string{"authorization_pkce_s256", "token_pkce_verifier"}, "pkce"))
	reference.add(referenceCheck(runtime, "token_auth_method", "token_auth"))
	reference.add(referenceCheck(runtime, "resource_bearer", "bearer_delivery"))
	reference.add(aggregateReferenceChecks(runtime, []string{"resource_token_signature", "resource_token_issuer", "resource_token_audience", "resource_token_expiry", "resource_token_scopes"}, "resource_server_verification"))
	reference.add(aggregateReferenceChecks(runtime, []string{"tool_oauth_security_scheme", "tool_oauth_www_authenticate", "tool_oauth_challenge_error", "tool_oauth_challenge_error_description"}, "tool_oauth_signals"))
	return reference
}

func referenceCheck(runtime *RuntimeEvidenceReport, id, outputID string) RuntimeCheck {
	for _, check := range runtime.Checks {
		if check.ID == id {
			check.ID = outputID
			return check
		}
	}
	return RuntimeCheck{ID: outputID, Status: StatusWarn, Expected: "observed evidence", Observed: "unknown", Message: "No sanitized runtime observation was supplied for this OpenAI reference-pattern boundary"}
}

func firstReferenceCheck(runtime *RuntimeEvidenceReport, ids []string, outputID string) RuntimeCheck {
	for _, id := range ids {
		for _, check := range runtime.Checks {
			if check.ID == id {
				check.ID = outputID
				return check
			}
		}
	}
	return RuntimeCheck{ID: outputID, Status: StatusWarn, Expected: "observed evidence", Observed: "unknown", Message: "No sanitized runtime observation was supplied for this OpenAI reference-pattern boundary"}
}

func aggregateReferenceChecks(runtime *RuntimeEvidenceReport, ids []string, outputID string) RuntimeCheck {
	result := RuntimeCheck{ID: outputID, Status: StatusWarn, Expected: "all applicable checks pass", Observed: "unknown", Message: "No sanitized runtime observation was supplied for this OpenAI reference-pattern boundary"}
	found := 0
	applicable := 0
	passed := 0
	for _, id := range ids {
		for _, check := range runtime.Checks {
			if check.ID != id {
				continue
			}
			found++
			if check.Status == StatusNA {
				continue
			}
			applicable++
			if check.Status == StatusFail {
				result.Status = StatusFail
				result.Observed = "failed"
				result.ReasonCode = check.ReasonCode
				result.Message = "At least one applicable OpenAI reference-pattern check failed"
				return result
			}
			if check.Status == StatusPass {
				passed++
			}
		}
	}
	if found > 0 && applicable == 0 && found == len(ids) {
		result.Status = StatusNA
		result.Observed = "not applicable"
		result.Message = "No checks in this OpenAI reference-pattern boundary applied to the observed flow"
	} else if found > 0 && applicable == 0 {
		result.Observed = "partial"
		result.Message = "Observed checks were not applicable, but other observations for this OpenAI reference-pattern boundary were not supplied"
	} else if applicable > 0 && passed == applicable {
		result.Status = StatusPass
		result.Observed = "passed"
		result.Message = "All applicable supplied observations for this OpenAI reference-pattern boundary passed"
	} else if found > 0 {
		result.Observed = "partial"
		result.Message = "Some applicable observations for this OpenAI reference-pattern boundary remain unknown"
	}
	return result
}

func coverageForChecks(checks []RuntimeCheck) EvidenceCoverage {
	var coverage EvidenceCoverage
	for _, check := range checks {
		switch check.Status {
		case StatusPass:
			coverage.Observed++
			coverage.Passed++
		case StatusFail:
			coverage.Observed++
			coverage.Failed++
		case StatusNA:
			coverage.NotApplicable++
		default:
			coverage.Unknown++
		}
	}
	return coverage
}

func (r *ReferencePatternReport) add(check RuntimeCheck) {
	check = normalizeRuntimeCheck(check)
	r.Checks = append(r.Checks, check)
	if check.Status == StatusFail {
		r.Status = StatusFail
	} else if check.Status == StatusWarn && r.Status == StatusPass {
		r.Status = StatusWarn
	}
}

func (r *RuntimeEvidenceReport) add(check RuntimeCheck) {
	if check.ID == "" {
		if r.Status == StatusPass {
			r.Status = StatusWarn
		}
		return
	}
	check = normalizeRuntimeCheck(check)
	r.Checks = append(r.Checks, check)
	if check.Status == StatusFail {
		r.Status = StatusFail
		if r.ReasonCode == "" && check.ReasonCode != "" {
			r.ReasonCode = check.ReasonCode
		}
	} else if check.Status == StatusWarn && r.Status == StatusPass {
		r.Status = StatusWarn
	}
}

func normalizeRuntimeCheck(check RuntimeCheck) RuntimeCheck {
	switch check.Status {
	case StatusPass, StatusWarn, StatusFail, StatusNA:
		return check
	default:
		check.Status = StatusWarn
		if check.Observed == "" {
			check.Observed = "unknown"
		}
		if check.Message == "" {
			check.Message = "Runtime Evidence check did not provide a conclusive status"
		}
		return check
	}
}

func expectedChatGPTTokenAuthForStrategy(report *Report, strategy string) string {
	if strategy != "cimd" {
		return "unknown"
	}
	return expectedChatGPTTokenAuth(report)
}

func expectedChatGPTTokenAuth(report *Report) string {
	// Multiple authorization servers are intentionally treated as ambiguous.
	// Until a real flow proves which issuer ChatGPT selected, do not combine
	// token-auth capabilities from unrelated issuers into one expectation.
	if report.Client == nil || len(report.AuthorizationServers) != 1 {
		return "unknown"
	}
	server := report.AuthorizationServers[0]
	if !server.ClientIDMetadataDocumentSupported {
		return "unknown"
	}
	compatible := intersection(report.Client.TokenEndpointAuthMethodsSupported, server.TokenEndpointAuthMethodsSupported)
	if contains(compatible, "private_key_jwt") {
		return "private_key_jwt"
	}
	if contains(compatible, "none") {
		return "none"
	}
	return "unknown"
}

func observedTokenAuth(evidence *TokenRequestEvidence) (string, bool) {
	if evidence == nil || evidence.ClientAssertionPresent == nil {
		return "unknown", false
	}
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

func serverSupportsCIMD(report *Report) bool {
	for _, server := range report.AuthorizationServers {
		if server.ClientIDMetadataDocumentSupported {
			return true
		}
	}
	return false
}

func serverSupportsDCR(report *Report) bool {
	for _, server := range report.AuthorizationServers {
		if server.RegistrationEndpoint != "" {
			return true
		}
	}
	return false
}

func matchCheck(id, expected string, value *bool, reason interop.ReasonCode, message string) RuntimeCheck {
	check := RuntimeCheck{ID: id, Expected: expected, Observed: boolObservation(value), Message: message}
	switch {
	case value == nil:
		check.Status = StatusWarn
	case *value:
		check.Status = StatusPass
		check.Observed = expected
	default:
		check.Status = StatusFail
		check.Observed = "mismatch"
		check.ReasonCode = reason
	}
	return check
}

func presenceCheck(id, expected string, value *bool, reason interop.ReasonCode, message string) RuntimeCheck {
	check := RuntimeCheck{ID: id, Expected: expected, Observed: boolObservation(value), Message: message}
	switch {
	case value == nil:
		check.Status = StatusWarn
	case *value:
		check.Status = StatusPass
		check.Observed = expected
	default:
		check.Status = StatusFail
		check.Observed = "absent"
		check.ReasonCode = reason
	}
	return check
}

func validityCheck(id string, value *bool, reason interop.ReasonCode, message string) RuntimeCheck {
	check := RuntimeCheck{ID: id, Expected: "valid", Observed: boolObservation(value), Message: message}
	switch {
	case value == nil:
		check.Status = StatusWarn
	case *value:
		check.Status = StatusPass
		check.Observed = "valid"
	default:
		check.Status = StatusFail
		check.Observed = "invalid"
		check.ReasonCode = reason
	}
	return check
}

func boolObservation(value *bool) string {
	if value == nil {
		return "unknown"
	}
	if *value {
		return "true"
	}
	return "false"
}
