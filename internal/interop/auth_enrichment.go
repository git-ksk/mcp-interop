package interop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const (
	authCapabilityDiagnosticID = "auth_registration_capability"
	maxAuthMetadataBytes       = int64(2 << 20)
)

var authResourceMetadataPattern = regexp.MustCompile(`(?i)resource_metadata\s*=\s*"([^"]+)"`)

type authProtectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
}

type authAuthorizationServerMetadata struct {
	Issuer                            string `json:"issuer"`
	ClientIDMetadataDocumentSupported bool   `json:"client_id_metadata_document_supported"`
	RegistrationEndpoint              string `json:"registration_endpoint"`
}

// EnrichAuthFailure correlates explicit real-client DCR failures with
// independently discovered MCP/OAuth metadata. It never changes the four-stage
// verdict and intentionally runs only for stable DCR reason codes.
func EnrichAuthFailure(ctx context.Context, endpoint string, result *Result, httpClient *http.Client) {
	if result == nil {
		return
	}
	auth, ok := result.Get(StageAuth)
	if !ok || auth.Status != StatusFail {
		return
	}
	if auth.ReasonCode != ReasonDCRUnsupported && auth.ReasonCode != ReasonDCRFailed {
		return
	}

	diagnostic := Diagnostic{
		ID:         authCapabilityDiagnosticID,
		Stage:      StageAuth,
		ReasonCode: auth.ReasonCode,
	}
	capabilities, err := discoverAuthCapabilities(ctx, endpoint, httpClient)
	if err != nil {
		diagnostic.Conclusion = "inconclusive"
		diagnostic.Message = "Authorization-server registration capabilities could not be correlated safely from MCP/OAuth metadata"
		result.AddDiagnostic(diagnostic)
		return
	}
	diagnostic.AuthCapabilities = &capabilities

	if capabilities.SelectionAmbiguous || capabilities.CIMDAdvertised == nil || capabilities.DCRAdvertised == nil {
		diagnostic.Conclusion = "inconclusive"
		diagnostic.Message = "Multiple authorization servers are advertised, so the real client's selected issuer cannot be inferred from public metadata"
		result.AddDiagnostic(diagnostic)
		return
	}

	cimd := *capabilities.CIMDAdvertised
	dcr := *capabilities.DCRAdvertised
	switch auth.ReasonCode {
	case ReasonDCRUnsupported:
		switch {
		case cimd && !dcr:
			diagnostic.Conclusion = "registration_strategy_incompatibility"
			diagnostic.Message = "The real client requires DCR while the authorization server advertises CIMD and does not advertise DCR"
		case dcr:
			diagnostic.Conclusion = "client_server_evidence_conflict"
			diagnostic.Message = "The real client reported DCR as unsupported, but the authorization server metadata advertises a registration endpoint"
		default:
			diagnostic.Conclusion = "dcr_not_advertised"
			diagnostic.Message = "The authorization server does not advertise DCR; no registration URL was guessed or probed"
		}
	case ReasonDCRFailed:
		switch {
		case dcr:
			diagnostic.Conclusion = "dcr_attempt_failed"
			diagnostic.Message = "The authorization server advertises DCR, so the observed client failure occurred during a registration attempt rather than capability discovery"
		case cimd:
			diagnostic.Conclusion = "registration_strategy_incompatibility"
			diagnostic.Message = "The client attempted DCR while the authorization server advertises CIMD and does not advertise DCR"
		default:
			diagnostic.Conclusion = "dcr_not_advertised"
			diagnostic.Message = "The authorization server does not advertise DCR; the client-side registration failure is consistent with absent DCR capability"
		}
	}
	result.AddDiagnostic(diagnostic)
}

func discoverAuthCapabilities(ctx context.Context, endpoint string, httpClient *http.Client) (AuthCapabilities, error) {
	capabilities := AuthCapabilities{}
	if err := (Target{Endpoint: endpoint}).Validate(); err != nil {
		return capabilities, err
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return capabilities, err
	}
	client := httpClient
	if client == nil {
		client = newAuthMetadataHTTPClient(endpointURL)
	}

	metadataURL, err := discoverAuthProtectedResourceMetadata(ctx, client, endpointURL)
	if err != nil {
		return capabilities, err
	}
	var protected authProtectedResourceMetadata
	if err := fetchAuthJSON(ctx, client, metadataURL, &protected); err != nil {
		return capabilities, err
	}
	if protected.Resource == "" {
		return capabilities, errors.New("protected resource metadata is missing resource")
	}
	if protected.Resource != endpointURL.String() {
		return capabilities, errors.New("protected resource metadata resource does not match target endpoint")
	}
	capabilities.AuthorizationServerCount = len(protected.AuthorizationServers)
	if len(protected.AuthorizationServers) == 0 {
		return capabilities, errors.New("protected resource metadata has no authorization server")
	}
	if len(protected.AuthorizationServers) > 1 {
		capabilities.SelectionAmbiguous = true
		return capabilities, nil
	}

	server, err := discoverAuthAuthorizationServer(ctx, client, protected.AuthorizationServers[0])
	if err != nil {
		return capabilities, err
	}
	cimd := server.ClientIDMetadataDocumentSupported
	dcr := server.RegistrationEndpoint != ""
	capabilities.CIMDAdvertised = &cimd
	capabilities.DCRAdvertised = &dcr
	return capabilities, nil
}

func discoverAuthProtectedResourceMetadata(ctx context.Context, client *http.Client, endpoint *url.URL) (string, error) {
	requestBody := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"mcp-interop-auth-diagnostic","version":"dev"}}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), requestBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, requestErr := client.Do(req)
	if requestErr == nil {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		for _, challenge := range resp.Header.Values("WWW-Authenticate") {
			if match := authResourceMetadataPattern.FindStringSubmatch(challenge); len(match) == 2 {
				metadataURL, parseErr := url.Parse(match[1])
				if parseErr == nil && metadataURL.Scheme == "https" && metadataURL.Host != "" && metadataURL.User == nil && metadataURL.Fragment == "" {
					return metadataURL.String(), nil
				}
			}
		}
	}

	for _, candidate := range authProtectedResourceCandidates(endpoint) {
		var discard json.RawMessage
		if fetchAuthJSON(ctx, client, candidate, &discard) == nil {
			return candidate, nil
		}
	}
	if requestErr != nil {
		return "", fmt.Errorf("MCP probe failed and protected resource metadata was not discovered: %w", requestErr)
	}
	return "", errors.New("protected resource metadata was not discovered")
}

func authProtectedResourceCandidates(endpoint *url.URL) []string {
	origin := endpoint.Scheme + "://" + endpoint.Host
	path := strings.TrimPrefix(endpoint.EscapedPath(), "/")
	query := ""
	if endpoint.RawQuery != "" {
		query = "?" + endpoint.RawQuery
	}
	candidates := make([]string, 0, 2)
	if path != "" {
		candidates = append(candidates, origin+"/.well-known/oauth-protected-resource/"+path+query)
	}
	candidates = append(candidates, origin+"/.well-known/oauth-protected-resource"+query)
	return candidates
}

func discoverAuthAuthorizationServer(ctx context.Context, client *http.Client, issuer string) (authAuthorizationServerMetadata, error) {
	issuerURL, err := url.Parse(issuer)
	if err != nil || issuerURL.Scheme != "https" || issuerURL.Host == "" || issuerURL.User != nil || issuerURL.RawQuery != "" || issuerURL.Fragment != "" {
		return authAuthorizationServerMetadata{}, errors.New("authorization server issuer must be an HTTPS URL without user info, query, or fragment")
	}
	for _, metadataURL := range authAuthorizationServerCandidates(issuerURL) {
		var metadata authAuthorizationServerMetadata
		if err := fetchAuthJSON(ctx, client, metadataURL, &metadata); err != nil {
			continue
		}
		if metadata.Issuer != issuer {
			continue
		}
		return metadata, nil
	}
	return authAuthorizationServerMetadata{}, errors.New("matching authorization-server metadata was not discovered")
}

func authAuthorizationServerCandidates(issuer *url.URL) []string {
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

func fetchAuthJSON(ctx context.Context, client *http.Client, rawURL string, out any) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("metadata URL must be HTTPS without user info or fragment")
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

	limited := &io.LimitedReader{R: resp.Body, N: maxAuthMetadataBytes + 1}
	decoder := json.NewDecoder(limited)
	if err := decoder.Decode(out); err != nil {
		return errors.New("invalid JSON metadata")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("invalid JSON metadata")
	}
	if limited.N == 0 {
		return errors.New("metadata response exceeds size limit")
	}
	return nil
}
