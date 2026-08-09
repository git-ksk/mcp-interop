package diagnose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	maxMetadataBytes = 2 << 20
	defaultTimeout   = 12 * time.Second
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Check struct {
	ID       string `json:"id"`
	Status   Status `json:"status"`
	Blocking bool   `json:"blocking"`
	Message  string `json:"message"`
}

type AuthorizationServer struct {
	Issuer                            string   `json:"issuer"`
	MetadataURL                       string   `json:"metadata_url,omitempty"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint                     string   `json:"token_endpoint,omitempty"`
	ClientIDMetadataDocumentSupported bool     `json:"client_id_metadata_document_supported"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
}

type ChatGPTClientEvidence struct {
	ClientID                          string   `json:"client_id"`
	RedirectURIs                      []string `json:"redirect_uris,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	JWKSURI                           string   `json:"jwks_uri,omitempty"`
	JWKSKeyCount                      int      `json:"jwks_key_count,omitempty"`
}

type Report struct {
	Profile                      string                 `json:"profile"`
	Endpoint                     string                 `json:"endpoint"`
	ProtectedResourceMetadataURL string                 `json:"protected_resource_metadata_url,omitempty"`
	Resource                     string                 `json:"resource,omitempty"`
	AuthorizationServers         []AuthorizationServer  `json:"authorization_servers,omitempty"`
	Client                       *ChatGPTClientEvidence `json:"client,omitempty"`
	Checks                       []Check                `json:"checks"`
	RuntimeEvidence              *RuntimeEvidenceReport `json:"runtime_evidence,omitempty"`
}

func (r Report) PreflightPassed() bool {
	for _, check := range r.Checks {
		if check.Blocking && check.Status == StatusFail {
			return false
		}
	}
	return true
}

func (r Report) Passed() bool {
	if !r.PreflightPassed() {
		return false
	}
	return r.RuntimeEvidence == nil || r.RuntimeEvidence.Passed()
}

type ChatGPTOptions struct {
	HTTPClient      *http.Client
	ClientID        string
	RedirectURI     string
	RuntimeEvidence *ChatGPTRuntimeEvidence
}

type protectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

type authorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	ClientIDMetadataDocumentSupported bool     `json:"client_id_metadata_document_supported"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
}

type clientMetadata struct {
	ClientID                          string          `json:"client_id"`
	RedirectURIs                      []string        `json:"redirect_uris"`
	TokenEndpointAuthMethodsSupported []string        `json:"token_endpoint_auth_methods_supported"`
	TokenEndpointAuthMethod           string          `json:"token_endpoint_auth_method"`
	JWKSURI                           string          `json:"jwks_uri"`
	JWKS                              json.RawMessage `json:"jwks"`
}

type jwksDocument struct {
	Keys []json.RawMessage `json:"keys"`
}

var resourceMetadataPattern = regexp.MustCompile(`(?i)resource_metadata\s*=\s*"([^"]+)"`)

func ChatGPT(ctx context.Context, endpoint string, options ChatGPTOptions) (Report, error) {
	report := Report{Profile: "chatgpt", Endpoint: interop.SanitizeEndpoint(endpoint)}
	if err := (interop.Target{Endpoint: endpoint}).Validate(); err != nil {
		return report, err
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return report, err
	}
	if endpointURL.Scheme == "https" {
		report.add("endpoint_https", StatusPass, true, "Remote MCP endpoint uses HTTPS")
	} else {
		report.add("endpoint_https", StatusFail, true, "ChatGPT remote MCP connections require an HTTPS endpoint")
	}

	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	metadataURL, challengeObserved, err := discoverProtectedResourceMetadata(ctx, client, endpointURL)
	if err != nil {
		report.add("protected_resource_metadata", StatusFail, true, "Protected Resource Metadata could not be discovered: "+safeError(err))
		return report, nil
	}
	if challengeObserved {
		report.add("oauth_challenge", StatusPass, true, "MCP endpoint advertised Protected Resource Metadata in WWW-Authenticate")
	} else {
		report.add("oauth_challenge", StatusWarn, false, "No resource_metadata parameter was observed; using the standardized Protected Resource Metadata location")
	}

	var prm protectedResourceMetadata
	if err := fetchJSON(ctx, client, metadataURL, &prm); err != nil {
		report.add("protected_resource_metadata", StatusFail, true, "Protected Resource Metadata could not be fetched: "+safeError(err))
		return report, nil
	}
	report.ProtectedResourceMetadataURL = sanitizePublicURL(metadataURL)
	report.Resource = interop.SanitizeEndpoint(prm.Resource)
	if prm.Resource == "" {
		report.add("protected_resource_metadata", StatusFail, true, "Protected Resource Metadata is missing resource")
	} else if len(prm.AuthorizationServers) == 0 {
		report.add("protected_resource_metadata", StatusFail, true, "Protected Resource Metadata is missing authorization_servers")
	} else {
		report.add("protected_resource_metadata", StatusPass, true, fmt.Sprintf("Protected Resource Metadata advertises %d authorization server(s)", len(prm.AuthorizationServers)))
	}

	if prm.Resource != "" {
		if equivalentResource(prm.Resource, endpoint) {
			report.add("resource_consistency", StatusPass, false, "Protected Resource Metadata resource matches the MCP endpoint")
		} else {
			report.add("resource_consistency", StatusWarn, false, "Protected Resource Metadata resource differs from the supplied MCP endpoint; verify token audience/resource validation accepts the advertised canonical resource")
		}
	}

	for _, issuer := range prm.AuthorizationServers {
		server, err := discoverAuthorizationServer(ctx, client, issuer)
		if err != nil {
			report.add("authorization_server_candidate", StatusWarn, false, "One advertised authorization server could not be discovered: "+safeError(err))
			continue
		}
		report.AuthorizationServers = append(report.AuthorizationServers, server)
	}

	if len(report.AuthorizationServers) == 0 {
		report.add("authorization_server_metadata", StatusFail, true, "No usable authorization server metadata was discovered")
		return report, nil
	}
	report.add("authorization_server_metadata", StatusPass, true, fmt.Sprintf("Discovered metadata for %d authorization server(s)", len(report.AuthorizationServers)))
	if len(prm.AuthorizationServers) > 1 {
		report.add("authorization_server_selection", StatusWarn, false, "Multiple authorization servers are advertised; this preflight does not claim which issuer ChatGPT will select")
	}

	evaluateServerCapabilities(&report)

	clientID := options.ClientID
	if clientID == "" && options.RuntimeEvidence != nil {
		clientID = options.RuntimeEvidence.ClientID
	}
	if clientID != "" {
		evaluateChatGPTClientMetadata(ctx, client, &report, clientID, options.RedirectURI)
	} else if options.RedirectURI != "" {
		report.add("chatgpt_client_metadata", StatusFail, true, "--redirect-uri requires --client-id so the advertised redirect URI can be verified")
	} else {
		report.add("chatgpt_client_metadata", StatusWarn, false, "No observed ChatGPT client_id was supplied; server-side preflight is complete, but the exact ChatGPT CIMD document/JWKS/redirect URI was not verified")
	}

	if options.RuntimeEvidence != nil {
		if err := options.RuntimeEvidence.Validate(); err != nil {
			return report, fmt.Errorf("invalid ChatGPT runtime evidence: %w", err)
		}
		evaluateRuntimeEvidence(&report, *options.RuntimeEvidence)
	}

	return report, nil
}

func evaluateServerCapabilities(report *Report) {
	cimdAvailable := false
	dcrAvailable := false
	compatibleCIMDTokenAuth := false
	pkceExplicitPass := false
	pkceExplicitFail := false
	offlineAccess := false
	endpointsComplete := false

	for _, server := range report.AuthorizationServers {
		if server.AuthorizationEndpoint != "" && server.TokenEndpoint != "" {
			endpointsComplete = true
		}
		if server.ClientIDMetadataDocumentSupported {
			cimdAvailable = true
			if intersects(server.TokenEndpointAuthMethodsSupported, []string{"none", "private_key_jwt"}) {
				compatibleCIMDTokenAuth = true
			}
		}
		if server.RegistrationEndpoint != "" {
			dcrAvailable = true
		}
		if contains(server.CodeChallengeMethodsSupported, "S256") {
			pkceExplicitPass = true
		} else if len(server.CodeChallengeMethodsSupported) > 0 {
			pkceExplicitFail = true
		}
		if contains(server.ScopesSupported, "offline_access") {
			offlineAccess = true
		}
	}

	if endpointsComplete {
		report.add("authorization_endpoints", StatusPass, true, "Authorization and token endpoints are advertised")
	} else {
		report.add("authorization_endpoints", StatusFail, true, "No discovered authorization server advertises both authorization_endpoint and token_endpoint")
	}

	switch {
	case cimdAvailable && dcrAvailable:
		report.add("client_registration", StatusPass, true, "Authorization server advertises CIMD and DCR; ChatGPT can use CIMD without requiring DCR")
	case cimdAvailable:
		report.add("client_registration", StatusPass, true, "Authorization server advertises Client ID Metadata Documents (CIMD); DCR is not required for ChatGPT")
	case dcrAvailable:
		report.add("client_registration", StatusPass, true, "Authorization server does not advertise CIMD but does advertise a Dynamic Client Registration endpoint")
	default:
		report.add("client_registration", StatusFail, true, "Neither Client ID Metadata Documents nor a Dynamic Client Registration endpoint is advertised")
	}

	switch {
	case compatibleCIMDTokenAuth:
		report.add("token_endpoint_auth", StatusPass, true, "CIMD-capable authorization server advertises a ChatGPT-compatible token endpoint auth method (none or private_key_jwt)")
	case cimdAvailable && dcrAvailable:
		report.add("token_endpoint_auth", StatusWarn, false, "CIMD metadata does not advertise a ChatGPT-compatible token endpoint auth method, but a DCR fallback is advertised")
	case cimdAvailable:
		report.add("token_endpoint_auth", StatusFail, true, "CIMD is advertised, but no ChatGPT-compatible token endpoint auth method (none or private_key_jwt) is advertised")
	default:
		report.add("token_endpoint_auth", StatusWarn, false, "CIMD is not advertised; token endpoint auth compatibility depends on the DCR/pre-registered client metadata")
	}

	if pkceExplicitPass {
		report.add("pkce_s256", StatusPass, true, "Authorization server explicitly advertises PKCE S256")
	} else if pkceExplicitFail {
		report.add("pkce_s256", StatusFail, true, "Authorization server advertises PKCE methods but not S256")
	} else {
		report.add("pkce_s256", StatusWarn, false, "Authorization server does not advertise code_challenge_methods_supported; ChatGPT uses PKCE S256, so support cannot be proven from metadata")
	}

	if offlineAccess {
		report.add("offline_access", StatusPass, false, "Authorization server advertises offline_access for refresh-token capable sessions")
	} else {
		report.add("offline_access", StatusWarn, false, "offline_access is not advertised; initial OAuth may work, but long-lived ChatGPT access can fail if refresh tokens are not issued")
	}
}

func evaluateChatGPTClientMetadata(ctx context.Context, client *http.Client, report *Report, clientID, redirectURI string) {
	clientURL, err := url.Parse(clientID)
	if err != nil || clientURL.Scheme != "https" || clientURL.Host == "" || clientURL.RawQuery != "" || clientURL.Fragment != "" {
		report.add("chatgpt_client_metadata", StatusFail, true, "Observed ChatGPT client_id must be a stable HTTPS metadata URL without query or fragment")
		return
	}

	var metadata clientMetadata
	if err := fetchJSON(ctx, client, clientID, &metadata); err != nil {
		report.add("chatgpt_client_metadata", StatusFail, true, "Observed ChatGPT client_id metadata could not be fetched: "+safeError(err))
		return
	}
	if metadata.ClientID == "" || metadata.ClientID != clientID {
		report.add("chatgpt_client_metadata", StatusFail, true, "CIMD document client_id is missing or does not exactly match the metadata document URL")
		return
	}
	if len(metadata.RedirectURIs) == 0 {
		report.add("chatgpt_client_metadata", StatusFail, true, "CIMD document does not advertise redirect_uris")
		return
	}

	methods := append([]string(nil), metadata.TokenEndpointAuthMethodsSupported...)
	if len(methods) == 0 && metadata.TokenEndpointAuthMethod != "" {
		methods = []string{metadata.TokenEndpointAuthMethod}
	}

	evidence := &ChatGPTClientEvidence{
		ClientID:                          sanitizePublicURL(clientID),
		RedirectURIs:                      append([]string(nil), metadata.RedirectURIs...),
		TokenEndpointAuthMethodsSupported: append([]string(nil), methods...),
		JWKSURI:                           sanitizePublicURL(metadata.JWKSURI),
	}
	report.Client = evidence
	report.add("chatgpt_client_metadata", StatusPass, true, "Fetched and validated the observed ChatGPT CIMD client metadata document")

	if redirectURI != "" {
		if contains(metadata.RedirectURIs, redirectURI) {
			report.add("chatgpt_redirect_uri", StatusPass, true, "Observed ChatGPT redirect_uri is registered by the supplied client metadata document")
		} else {
			report.add("chatgpt_redirect_uri", StatusFail, true, "Observed ChatGPT redirect_uri is not present in the supplied client metadata document")
		}
	}

	serverMethods := make([]string, 0)
	for _, server := range report.AuthorizationServers {
		if server.ClientIDMetadataDocumentSupported {
			serverMethods = append(serverMethods, server.TokenEndpointAuthMethodsSupported...)
		}
	}
	compatible := intersection(methods, serverMethods)
	compatible = intersection(compatible, []string{"none", "private_key_jwt"})
	if len(compatible) == 0 {
		report.add("chatgpt_token_endpoint_auth", StatusFail, true, "No compatible token endpoint auth method is shared by the ChatGPT client metadata and authorization server metadata")
		return
	}
	sort.Strings(compatible)
	report.add("chatgpt_token_endpoint_auth", StatusPass, true, "ChatGPT client metadata and authorization server share token endpoint auth method(s): "+strings.Join(compatible, ", "))

	if contains(compatible, "private_key_jwt") {
		if metadata.JWKSURI == "" && len(metadata.JWKS) == 0 {
			report.add("chatgpt_jwks", StatusFail, true, "private_key_jwt is compatible, but the ChatGPT client metadata does not expose jwks_uri or embedded jwks")
			return
		}
		if metadata.JWKSURI != "" {
			var jwks jwksDocument
			if err := fetchJSON(ctx, client, metadata.JWKSURI, &jwks); err != nil {
				report.add("chatgpt_jwks", StatusFail, true, "ChatGPT JWKS could not be fetched: "+safeError(err))
				return
			}
			evidence.JWKSKeyCount = len(jwks.Keys)
			if len(jwks.Keys) == 0 {
				report.add("chatgpt_jwks", StatusFail, true, "ChatGPT JWKS contains no keys")
				return
			}
			report.add("chatgpt_jwks", StatusPass, true, fmt.Sprintf("ChatGPT JWKS is reachable and contains %d key(s)", len(jwks.Keys)))
			return
		}
		var embedded jwksDocument
		if err := json.Unmarshal(metadata.JWKS, &embedded); err != nil || len(embedded.Keys) == 0 {
			report.add("chatgpt_jwks", StatusFail, true, "Embedded ChatGPT JWKS is malformed or contains no keys")
			return
		}
		evidence.JWKSKeyCount = len(embedded.Keys)
		report.add("chatgpt_jwks", StatusPass, true, fmt.Sprintf("Embedded ChatGPT JWKS contains %d key(s)", len(embedded.Keys)))
	}
}

func discoverProtectedResourceMetadata(ctx context.Context, client *http.Client, endpoint *url.URL) (string, bool, error) {
	requestBody := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"mcp-interop-diagnose","version":"dev"}}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), requestBody)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		for _, challenge := range resp.Header.Values("WWW-Authenticate") {
			if match := resourceMetadataPattern.FindStringSubmatch(challenge); len(match) == 2 {
				metadataURL, parseErr := url.Parse(match[1])
				if parseErr == nil && metadataURL.Scheme == "https" && metadataURL.Host != "" {
					return metadataURL.String(), true, nil
				}
			}
		}
	}

	for _, candidate := range protectedResourceCandidates(endpoint) {
		if urlExists(ctx, client, candidate) {
			return candidate, false, nil
		}
	}
	if err != nil {
		return "", false, fmt.Errorf("MCP probe failed and no standardized Protected Resource Metadata endpoint was reachable: %w", err)
	}
	return "", false, errors.New("MCP response did not advertise resource_metadata and no standardized Protected Resource Metadata endpoint was reachable")
}

func protectedResourceCandidates(endpoint *url.URL) []string {
	origin := endpoint.Scheme + "://" + endpoint.Host
	path := strings.TrimPrefix(endpoint.EscapedPath(), "/")
	candidates := make([]string, 0, 2)
	if path != "" {
		candidates = append(candidates, origin+"/.well-known/oauth-protected-resource/"+path)
	}
	candidates = append(candidates, origin+"/.well-known/oauth-protected-resource")
	return candidates
}

func discoverAuthorizationServer(ctx context.Context, client *http.Client, issuer string) (AuthorizationServer, error) {
	issuerURL, err := url.Parse(issuer)
	if err != nil || issuerURL.Scheme != "https" || issuerURL.Host == "" {
		return AuthorizationServer{}, errors.New("authorization server issuer is not a valid HTTPS URL")
	}
	for _, metadataURL := range authorizationServerCandidates(issuerURL) {
		var metadata authorizationServerMetadata
		if err := fetchJSON(ctx, client, metadataURL, &metadata); err != nil {
			continue
		}
		if metadata.Issuer == "" || metadata.Issuer != issuer {
			continue
		}
		return AuthorizationServer{
			Issuer:                            sanitizePublicURL(metadata.Issuer),
			MetadataURL:                       sanitizePublicURL(metadataURL),
			AuthorizationEndpoint:             sanitizePublicURL(metadata.AuthorizationEndpoint),
			TokenEndpoint:                     sanitizePublicURL(metadata.TokenEndpoint),
			ClientIDMetadataDocumentSupported: metadata.ClientIDMetadataDocumentSupported,
			RegistrationEndpoint:              sanitizePublicURL(metadata.RegistrationEndpoint),
			TokenEndpointAuthMethodsSupported: append([]string(nil), metadata.TokenEndpointAuthMethodsSupported...),
			CodeChallengeMethodsSupported:     append([]string(nil), metadata.CodeChallengeMethodsSupported...),
			ScopesSupported:                   append([]string(nil), metadata.ScopesSupported...),
		}, nil
	}
	return AuthorizationServer{}, errors.New("no OAuth/OIDC authorization-server metadata document with a matching issuer was found")
}

func authorizationServerCandidates(issuer *url.URL) []string {
	origin := issuer.Scheme + "://" + issuer.Host
	path := strings.TrimSuffix(issuer.EscapedPath(), "/")
	candidates := []string{origin + "/.well-known/oauth-authorization-server" + path}
	if path == "" {
		candidates = append(candidates, origin+"/.well-known/openid-configuration")
	} else {
		candidates = append(candidates, origin+path+"/.well-known/openid-configuration")
	}
	return candidates
}

func fetchJSON(ctx context.Context, client *http.Client, rawURL string, out any) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("metadata URL must be an HTTPS URL without user info or fragment")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxMetadataBytes+1))
	if err := decoder.Decode(out); err != nil {
		return errors.New("invalid JSON metadata")
	}
	return nil
}

func urlExists(ctx context.Context, client *http.Client, rawURL string) bool {
	var discard json.RawMessage
	return fetchJSON(ctx, client, rawURL, &discard) == nil
}

func (r *Report) add(id string, status Status, blocking bool, message string) {
	r.Checks = append(r.Checks, Check{ID: id, Status: status, Blocking: blocking, Message: interop.Redact(message)})
}

func equivalentResource(a, b string) bool {
	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	if errA != nil || errB != nil {
		return a == b
	}
	ua.Fragment = ""
	ub.Fragment = ""
	return ua.String() == ub.String()
}

func sanitizePublicURL(raw string) string {
	if raw == "" {
		return ""
	}
	return interop.SanitizeEndpoint(raw)
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	return interop.Redact(err.Error())
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func intersects(a, b []string) bool {
	return len(intersection(a, b)) > 0
}

func intersection(a, b []string) []string {
	set := make(map[string]struct{}, len(a))
	for _, value := range a {
		set[value] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, value := range b {
		if _, ok := set[value]; !ok {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
