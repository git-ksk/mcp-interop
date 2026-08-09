package diagnose

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatGPTCIMDOnlyServerPassesRegistrationPreflight(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		CIMD:             true,
		TokenAuthMethods: []string{"none", "private_key_jwt"},
		PKCEMethods:      []string{"S256"},
		Scopes:           []string{"openid", "offline_access"},
	})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("expected CIMD-only ChatGPT preflight to pass: %#v", report.Checks)
	}
	assertCheck(t, report, "client_registration", StatusPass, "DCR is not required for ChatGPT")
	assertCheck(t, report, "token_endpoint_auth", StatusPass, "private_key_jwt")
	assertCheck(t, report, "pkce_s256", StatusPass, "S256")
}

func TestChatGPTDCRFallbackPassesRegistrationPreflight(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		DCR:         true,
		PKCEMethods: []string{"S256"},
	})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})
	if err != nil {
		t.Fatal(err)
	}
	assertCheck(t, report, "client_registration", StatusPass, "Dynamic Client Registration")
	if !report.Passed() {
		t.Fatalf("expected DCR fallback preflight to pass: %#v", report.Checks)
	}
}

func TestChatGPTFailsWhenNoRegistrationStrategyIsAdvertised(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{PKCEMethods: []string{"S256"}})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})
	if err != nil {
		t.Fatal(err)
	}
	assertCheck(t, report, "client_registration", StatusFail, "Neither Client ID Metadata Documents nor")
	if report.Passed() {
		t.Fatal("expected registration-incompatible preflight to fail")
	}
}

func TestChatGPTFailsWhenCIMDTokenAuthMethodsDoNotIntersect(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		CIMD:             true,
		TokenAuthMethods: []string{"client_secret_basic"},
		PKCEMethods:      []string{"S256"},
	})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})
	if err != nil {
		t.Fatal(err)
	}
	assertCheck(t, report, "token_endpoint_auth", StatusFail, "no ChatGPT-compatible")
	if report.Passed() {
		t.Fatal("expected incompatible token auth preflight to fail")
	}
}

func TestChatGPTFailsWhenAdvertisedPKCEDoesNotIncludeS256(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		CIMD:             true,
		TokenAuthMethods: []string{"none"},
		PKCEMethods:      []string{"plain"},
	})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})
	if err != nil {
		t.Fatal(err)
	}
	assertCheck(t, report, "pkce_s256", StatusFail, "not S256")
	if report.Passed() {
		t.Fatal("expected PKCE-incompatible preflight to fail")
	}
}

func TestChatGPTMissingOfflineAccessIsWarningOnly(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		CIMD:             true,
		TokenAuthMethods: []string{"none"},
		PKCEMethods:      []string{"S256"},
		Scopes:           []string{"openid"},
	})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})
	if err != nil {
		t.Fatal(err)
	}
	assertCheck(t, report, "offline_access", StatusWarn, "refresh")
	if !report.Passed() {
		t.Fatalf("offline_access warning should not block initial preflight: %#v", report.Checks)
	}
}

func TestChatGPTUsesStandardPRMLocationWhenChallengeOmitsResourceMetadata(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		CIMD:                     true,
		TokenAuthMethods:         []string{"none"},
		PKCEMethods:              []string{"S256"},
		OmitResourceMetadataHint: true,
	})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})
	if err != nil {
		t.Fatal(err)
	}
	assertCheck(t, report, "oauth_challenge", StatusWarn, "standardized")
	assertCheck(t, report, "protected_resource_metadata", StatusPass, "advertises")
}

func TestChatGPTObservedClientMetadataChecksRedirectAndJWKS(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		CIMD:             true,
		TokenAuthMethods: []string{"private_key_jwt"},
		PKCEMethods:      []string{"S256"},
		ClientMetadata:   true,
	})
	defer fixture.Close()

	clientID := fixture.URL + "/chatgpt-client.json"
	redirectURI := "https://chatgpt.com/connector/oauth/test-callback"
	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{
		HTTPClient:  fixture.Client(),
		ClientID:    clientID,
		RedirectURI: redirectURI,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("expected observed ChatGPT metadata preflight to pass: %#v", report.Checks)
	}
	assertCheck(t, report, "chatgpt_client_metadata", StatusPass, "validated")
	assertCheck(t, report, "chatgpt_redirect_uri", StatusPass, "registered")
	assertCheck(t, report, "chatgpt_token_endpoint_auth", StatusPass, "private_key_jwt")
	assertCheck(t, report, "chatgpt_jwks", StatusPass, "1 key")
	if report.Client == nil || report.Client.JWKSKeyCount != 1 {
		t.Fatalf("unexpected client evidence: %#v", report.Client)
	}
}

func TestChatGPTObservedRedirectMismatchFails(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		CIMD:             true,
		TokenAuthMethods: []string{"private_key_jwt"},
		PKCEMethods:      []string{"S256"},
		ClientMetadata:   true,
	})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{
		HTTPClient:  fixture.Client(),
		ClientID:    fixture.URL + "/chatgpt-client.json",
		RedirectURI: "https://chatgpt.com/connector/oauth/not-registered",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCheck(t, report, "chatgpt_redirect_uri", StatusFail, "not present")
	if report.Passed() {
		t.Fatal("expected redirect mismatch to fail")
	}
}

func TestChatGPTWarnsWhenResourceDiffersFromEndpoint(t *testing.T) {
	fixture := newAuthFixture(t, authFixtureOptions{
		CIMD:             true,
		TokenAuthMethods: []string{"none"},
		PKCEMethods:      []string{"S256"},
		ResourcePath:     "/",
	})
	defer fixture.Close()

	report, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})
	if err != nil {
		t.Fatal(err)
	}
	assertCheck(t, report, "resource_consistency", StatusWarn, "differs")
	if !report.Passed() {
		t.Fatalf("canonical resource difference should be advisory: %#v", report.Checks)
	}
}

type authFixtureOptions struct {
	CIMD                     bool
	DCR                      bool
	TokenAuthMethods         []string
	PKCEMethods              []string
	Scopes                   []string
	OmitResourceMetadataHint bool
	ClientMetadata           bool
	ResourcePath             string
}

func newAuthFixture(t *testing.T, options authFixtureOptions) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	mux := http.NewServeMux()
	server = httptest.NewTLSServer(mux)

	resourcePath := options.ResourcePath
	if resourcePath == "" {
		resourcePath = "/mcp"
	}
	resource := strings.TrimSuffix(server.URL, "/") + resourcePath
	issuer := server.URL
	prmURL := server.URL + "/.well-known/oauth-protected-resource/mcp"

	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if options.OmitResourceMetadataHint {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp"`)
		} else {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s"`, prmURL))
		}
		w.WriteHeader(http.StatusUnauthorized)
	})

	mux.HandleFunc("/.well-known/oauth-protected-resource/mcp", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"resource":              resource,
			"authorization_servers": []string{issuer},
			"scopes_supported":      options.Scopes,
		})
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"resource":              resource,
			"authorization_servers": []string{issuer},
		})
	})

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		metadata := map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/oauth/authorize",
			"token_endpoint":                        issuer + "/oauth/token",
			"client_id_metadata_document_supported": options.CIMD,
			"token_endpoint_auth_methods_supported": options.TokenAuthMethods,
			"code_challenge_methods_supported":      options.PKCEMethods,
			"scopes_supported":                      options.Scopes,
		}
		if options.DCR {
			metadata["registration_endpoint"] = issuer + "/oauth/register"
		}
		writeJSON(t, w, metadata)
	})

	if options.ClientMetadata {
		mux.HandleFunc("/chatgpt-client.json", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, map[string]any{
				"client_id":                             server.URL + "/chatgpt-client.json",
				"redirect_uris":                         []string{"https://chatgpt.com/connector/oauth/test-callback"},
				"token_endpoint_auth_methods_supported": []string{"none", "private_key_jwt"},
				"jwks_uri":                              server.URL + "/jwks.json",
			})
		})
		mux.HandleFunc("/jwks.json", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, map[string]any{
				"keys": []map[string]any{{"kty": "RSA", "kid": "test", "n": "AQAB", "e": "AQAB"}},
			})
		})
	}
	return server
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode fixture JSON: %v", err)
	}
}

func assertCheck(t *testing.T, report Report, id string, status Status, messageContains string) {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID != id {
			continue
		}
		if check.Status != status {
			t.Fatalf("check %s status=%s, want %s: %#v", id, check.Status, status, check)
		}
		if messageContains != "" && !strings.Contains(check.Message, messageContains) {
			t.Fatalf("check %s message %q missing %q", id, check.Message, messageContains)
		}
		return
	}
	t.Fatalf("missing check %s: %#v", id, report.Checks)
}
