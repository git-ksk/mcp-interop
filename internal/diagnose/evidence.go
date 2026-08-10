package diagnose

import (
	"errors"
	"fmt"
	"strings"
)

// RuntimeEvidenceSectionSummary reports only structural coverage. It never
// includes observed values, metadata URLs, OAuth errors, or credential-like
// material.
type RuntimeEvidenceSectionSummary struct {
	Section  string `json:"section"`
	Supplied int    `json:"supplied"`
}

type RuntimeEvidenceInputSummary struct {
	SchemaVersion int                             `json:"schema_version"`
	Sections      []RuntimeEvidenceSectionSummary `json:"sections"`
	TotalSupplied int                             `json:"total_supplied"`
}

// CanonicalRuntimeEvidenceV3 converts legacy v1/v2 evidence into the schema v3
// structural shape without inventing observations. Secret-bearing fields are
// impossible to carry through this typed representation because the strict
// decoder rejects them before this function is reached.
func CanonicalRuntimeEvidenceV3(input ChatGPTRuntimeEvidence) ChatGPTRuntimeEvidence {
	out := ChatGPTRuntimeEvidence{
		SchemaVersion:        runtimeEvidenceSchemaV3,
		Registration:         cloneRegistration(input.Registration),
		AuthorizationRequest: cloneAuthorizationRequest(input.AuthorizationRequest),
		TokenRequest:         cloneTokenRequest(input.TokenRequest),
		ResourceRequest:      cloneResourceRequest(input.ResourceRequest),
	}
	if metadata := input.effectiveToolMetadata(); metadata != nil {
		out.ToolMetadata = cloneToolMetadata(metadata)
	}
	if challenge := input.effectiveToolChallenge(); challenge != nil {
		out.ToolChallenge = cloneToolChallenge(challenge)
	}
	return out
}

// MergeRuntimeEvidence combines independently produced secret-free evidence
// fragments. Conflicting observations fail closed instead of letting later
// files silently overwrite earlier evidence.
func MergeRuntimeEvidence(inputs []ChatGPTRuntimeEvidence) (ChatGPTRuntimeEvidence, error) {
	if len(inputs) == 0 {
		return ChatGPTRuntimeEvidence{}, errors.New("at least one runtime evidence input is required")
	}
	merged := ChatGPTRuntimeEvidence{SchemaVersion: runtimeEvidenceSchemaV3}
	for i, input := range inputs {
		if err := input.Validate(); err != nil {
			return ChatGPTRuntimeEvidence{}, fmt.Errorf("input %d: %w", i+1, err)
		}
		canonical := CanonicalRuntimeEvidenceV3(input)
		var err error
		merged.Registration, err = mergeRegistration(merged.Registration, canonical.Registration)
		if err != nil {
			return ChatGPTRuntimeEvidence{}, err
		}
		merged.AuthorizationRequest, err = mergeAuthorizationRequest(merged.AuthorizationRequest, canonical.AuthorizationRequest)
		if err != nil {
			return ChatGPTRuntimeEvidence{}, err
		}
		merged.TokenRequest, err = mergeTokenRequest(merged.TokenRequest, canonical.TokenRequest)
		if err != nil {
			return ChatGPTRuntimeEvidence{}, err
		}
		merged.ResourceRequest, err = mergeResourceRequest(merged.ResourceRequest, canonical.ResourceRequest)
		if err != nil {
			return ChatGPTRuntimeEvidence{}, err
		}
		merged.ToolMetadata, err = mergeToolMetadata(merged.ToolMetadata, canonical.ToolMetadata)
		if err != nil {
			return ChatGPTRuntimeEvidence{}, err
		}
		merged.ToolChallenge, err = mergeToolChallenge(merged.ToolChallenge, canonical.ToolChallenge)
		if err != nil {
			return ChatGPTRuntimeEvidence{}, err
		}
	}
	if err := merged.Validate(); err != nil {
		return ChatGPTRuntimeEvidence{}, err
	}
	return merged, nil
}

func SummarizeRuntimeEvidence(input ChatGPTRuntimeEvidence) RuntimeEvidenceInputSummary {
	canonical := CanonicalRuntimeEvidenceV3(input)
	summary := RuntimeEvidenceInputSummary{SchemaVersion: input.SchemaVersion}
	add := func(section string, supplied int, present bool) {
		if !present {
			return
		}
		summary.Sections = append(summary.Sections, RuntimeEvidenceSectionSummary{Section: section, Supplied: supplied})
		summary.TotalSupplied += supplied
	}
	if v := canonical.Registration; v != nil {
		add("registration", countStrings(v.Strategy, v.ClientMetadataURL), true)
	}
	if v := canonical.AuthorizationRequest; v != nil {
		add("authorization_request", countBools(v.ResourceMatches, v.RedirectURIMatches, v.PKCES256), true)
	}
	if v := canonical.TokenRequest; v != nil {
		add("token_request", countBools(v.ResourceMatches, v.CodeVerifierPresent, v.ClientAssertionPresent, v.ClientAssertionTypePresent)+countStrings(v.OAuthError), true)
	}
	if v := canonical.ResourceRequest; v != nil {
		add("resource_request", countBools(v.BearerPresent, v.SignatureValid, v.IssuerMatches, v.AudienceMatches, v.ExpiryValid, v.ScopesSufficient), true)
	}
	if v := canonical.ToolMetadata; v != nil {
		add("tool_metadata", countBools(v.OAuth2SecuritySchemePresent), true)
	}
	if v := canonical.ToolChallenge; v != nil {
		add("tool_challenge", countBools(v.Expected, v.WWWAuthenticatePresent, v.WWWAuthenticateHasError, v.WWWAuthenticateHasErrorDescription), true)
	}
	return summary
}

func countBools(values ...*bool) int {
	count := 0
	for _, value := range values {
		if value != nil {
			count++
		}
	}
	return count
}

func countStrings(values ...string) int {
	count := 0
	for _, value := range values {
		if value != "" {
			count++
		}
	}
	return count
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneRegistration(value *RegistrationEvidence) *RegistrationEvidence {
	if value == nil {
		return nil
	}
	return &RegistrationEvidence{Strategy: strings.ToLower(strings.TrimSpace(value.Strategy)), ClientMetadataURL: value.ClientMetadataURL}
}

func cloneAuthorizationRequest(value *AuthorizationRequestEvidence) *AuthorizationRequestEvidence {
	if value == nil {
		return nil
	}
	return &AuthorizationRequestEvidence{ResourceMatches: cloneBool(value.ResourceMatches), RedirectURIMatches: cloneBool(value.RedirectURIMatches), PKCES256: cloneBool(value.PKCES256)}
}

func cloneTokenRequest(value *TokenRequestEvidence) *TokenRequestEvidence {
	if value == nil {
		return nil
	}
	return &TokenRequestEvidence{
		ResourceMatches:            cloneBool(value.ResourceMatches),
		CodeVerifierPresent:        cloneBool(value.CodeVerifierPresent),
		ClientAssertionPresent:     cloneBool(value.ClientAssertionPresent),
		ClientAssertionTypePresent: cloneBool(value.ClientAssertionTypePresent),
		OAuthError:                 value.OAuthError,
	}
}

func cloneResourceRequest(value *ResourceRequestEvidence) *ResourceRequestEvidence {
	if value == nil {
		return nil
	}
	return &ResourceRequestEvidence{
		BearerPresent:    cloneBool(value.BearerPresent),
		SignatureValid:   cloneBool(value.SignatureValid),
		IssuerMatches:    cloneBool(value.IssuerMatches),
		AudienceMatches:  cloneBool(value.AudienceMatches),
		ExpiryValid:      cloneBool(value.ExpiryValid),
		ScopesSufficient: cloneBool(value.ScopesSufficient),
	}
}

func cloneToolMetadata(value *ToolMetadataEvidence) *ToolMetadataEvidence {
	if value == nil {
		return nil
	}
	return &ToolMetadataEvidence{OAuth2SecuritySchemePresent: cloneBool(value.OAuth2SecuritySchemePresent)}
}

func cloneToolChallenge(value *ToolChallengeEvidence) *ToolChallengeEvidence {
	if value == nil {
		return nil
	}
	return &ToolChallengeEvidence{
		Expected:                           cloneBool(value.Expected),
		WWWAuthenticatePresent:             cloneBool(value.WWWAuthenticatePresent),
		WWWAuthenticateHasError:            cloneBool(value.WWWAuthenticateHasError),
		WWWAuthenticateHasErrorDescription: cloneBool(value.WWWAuthenticateHasErrorDescription),
	}
}

func mergeBool(field string, left, right *bool) (*bool, error) {
	if left == nil {
		return cloneBool(right), nil
	}
	if right == nil {
		return left, nil
	}
	if *left != *right {
		return nil, fmt.Errorf("conflicting evidence for %s", field)
	}
	return left, nil
}

func mergeString(field, left, right string) (string, error) {
	if left == "" {
		return right, nil
	}
	if right == "" {
		return left, nil
	}
	if left != right {
		return "", fmt.Errorf("conflicting evidence for %s", field)
	}
	return left, nil
}

func mergeRegistration(left, right *RegistrationEvidence) (*RegistrationEvidence, error) {
	if left == nil {
		return cloneRegistration(right), nil
	}
	if right == nil {
		return left, nil
	}
	strategy, err := mergeString("registration.strategy", strings.ToLower(strings.TrimSpace(left.Strategy)), strings.ToLower(strings.TrimSpace(right.Strategy)))
	if err != nil {
		return nil, err
	}
	clientURL, err := mergeString("registration.client_metadata_url", left.ClientMetadataURL, right.ClientMetadataURL)
	if err != nil {
		return nil, err
	}
	return &RegistrationEvidence{Strategy: strategy, ClientMetadataURL: clientURL}, nil
}

func mergeAuthorizationRequest(left, right *AuthorizationRequestEvidence) (*AuthorizationRequestEvidence, error) {
	if left == nil {
		return cloneAuthorizationRequest(right), nil
	}
	if right == nil {
		return left, nil
	}
	resource, err := mergeBool("authorization_request.resource_matches", left.ResourceMatches, right.ResourceMatches)
	if err != nil {
		return nil, err
	}
	redirect, err := mergeBool("authorization_request.redirect_uri_matches", left.RedirectURIMatches, right.RedirectURIMatches)
	if err != nil {
		return nil, err
	}
	pkce, err := mergeBool("authorization_request.pkce_s256", left.PKCES256, right.PKCES256)
	if err != nil {
		return nil, err
	}
	return &AuthorizationRequestEvidence{ResourceMatches: resource, RedirectURIMatches: redirect, PKCES256: pkce}, nil
}

func mergeTokenRequest(left, right *TokenRequestEvidence) (*TokenRequestEvidence, error) {
	if left == nil {
		return cloneTokenRequest(right), nil
	}
	if right == nil {
		return left, nil
	}
	resource, err := mergeBool("token_request.resource_matches", left.ResourceMatches, right.ResourceMatches)
	if err != nil {
		return nil, err
	}
	verifier, err := mergeBool("token_request.code_verifier_present", left.CodeVerifierPresent, right.CodeVerifierPresent)
	if err != nil {
		return nil, err
	}
	assertion, err := mergeBool("token_request.client_assertion_present", left.ClientAssertionPresent, right.ClientAssertionPresent)
	if err != nil {
		return nil, err
	}
	assertionType, err := mergeBool("token_request.client_assertion_type_present", left.ClientAssertionTypePresent, right.ClientAssertionTypePresent)
	if err != nil {
		return nil, err
	}
	oauthError, err := mergeString("token_request.oauth_error", left.OAuthError, right.OAuthError)
	if err != nil {
		return nil, err
	}
	return &TokenRequestEvidence{ResourceMatches: resource, CodeVerifierPresent: verifier, ClientAssertionPresent: assertion, ClientAssertionTypePresent: assertionType, OAuthError: oauthError}, nil
}

func mergeResourceRequest(left, right *ResourceRequestEvidence) (*ResourceRequestEvidence, error) {
	if left == nil {
		return cloneResourceRequest(right), nil
	}
	if right == nil {
		return left, nil
	}
	bearer, err := mergeBool("resource_request.bearer_present", left.BearerPresent, right.BearerPresent)
	if err != nil {
		return nil, err
	}
	signature, err := mergeBool("resource_request.signature_valid", left.SignatureValid, right.SignatureValid)
	if err != nil {
		return nil, err
	}
	issuer, err := mergeBool("resource_request.issuer_matches", left.IssuerMatches, right.IssuerMatches)
	if err != nil {
		return nil, err
	}
	audience, err := mergeBool("resource_request.audience_matches", left.AudienceMatches, right.AudienceMatches)
	if err != nil {
		return nil, err
	}
	expiry, err := mergeBool("resource_request.expiry_valid", left.ExpiryValid, right.ExpiryValid)
	if err != nil {
		return nil, err
	}
	scopes, err := mergeBool("resource_request.scopes_sufficient", left.ScopesSufficient, right.ScopesSufficient)
	if err != nil {
		return nil, err
	}
	return &ResourceRequestEvidence{BearerPresent: bearer, SignatureValid: signature, IssuerMatches: issuer, AudienceMatches: audience, ExpiryValid: expiry, ScopesSufficient: scopes}, nil
}

func mergeToolMetadata(left, right *ToolMetadataEvidence) (*ToolMetadataEvidence, error) {
	if left == nil {
		return cloneToolMetadata(right), nil
	}
	if right == nil {
		return left, nil
	}
	scheme, err := mergeBool("tool_metadata.oauth2_security_scheme_present", left.OAuth2SecuritySchemePresent, right.OAuth2SecuritySchemePresent)
	if err != nil {
		return nil, err
	}
	return &ToolMetadataEvidence{OAuth2SecuritySchemePresent: scheme}, nil
}

func mergeToolChallenge(left, right *ToolChallengeEvidence) (*ToolChallengeEvidence, error) {
	if left == nil {
		return cloneToolChallenge(right), nil
	}
	if right == nil {
		return left, nil
	}
	expected, err := mergeBool("tool_challenge.expected", left.Expected, right.Expected)
	if err != nil {
		return nil, err
	}
	present, err := mergeBool("tool_challenge.www_authenticate_present", left.WWWAuthenticatePresent, right.WWWAuthenticatePresent)
	if err != nil {
		return nil, err
	}
	hasError, err := mergeBool("tool_challenge.www_authenticate_has_error", left.WWWAuthenticateHasError, right.WWWAuthenticateHasError)
	if err != nil {
		return nil, err
	}
	hasDescription, err := mergeBool("tool_challenge.www_authenticate_has_error_description", left.WWWAuthenticateHasErrorDescription, right.WWWAuthenticateHasErrorDescription)
	if err != nil {
		return nil, err
	}
	return &ToolChallengeEvidence{Expected: expected, WWWAuthenticatePresent: present, WWWAuthenticateHasError: hasError, WWWAuthenticateHasErrorDescription: hasDescription}, nil
}
