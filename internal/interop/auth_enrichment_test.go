package interop

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestEnrichAuthFailureCorrelatesCIMDOnlyWithDCRUnsupported(t *testing.T) {
	var baseURL string
	var guessedRegistrationHits atomic.Int32
	server := newLocalTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+baseURL+`/.well-known/oauth-protected-resource/mcp"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/.well-known/oauth-protected-resource/mcp":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resource":              baseURL + "/mcp",
				"authorization_servers": []string{baseURL},
			})
		case "/.well-known/oauth-authorization-server":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                baseURL,
				"client_id_metadata_document_supported": true,
			})
		case "/register", "/oauth/register":
			guessedRegistrationHits.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	result := NewResult("codex", "Codex CLI", "test", baseURL+"/mcp")
	result.SetWithReason(StageAuth, StatusFail, ReasonDCRUnsupported, "Dynamic client registration not supported")
	EnrichAuthFailure(context.Background(), baseURL+"/mcp", &result, server.Client())

	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(result.Diagnostics))
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Conclusion != "registration_strategy_incompatibility" {
		t.Fatalf("conclusion = %q", diagnostic.Conclusion)
	}
	if diagnostic.AuthCapabilities == nil || diagnostic.AuthCapabilities.CIMDAdvertised == nil || !*diagnostic.AuthCapabilities.CIMDAdvertised {
		t.Fatalf("CIMD capability not proven: %#v", diagnostic.AuthCapabilities)
	}
	if diagnostic.AuthCapabilities.DCRAdvertised == nil || *diagnostic.AuthCapabilities.DCRAdvertised {
		t.Fatalf("DCR capability = %#v, want false", diagnostic.AuthCapabilities.DCRAdvertised)
	}
	if guessedRegistrationHits.Load() != 0 {
		t.Fatalf("guessed registration paths were probed %d time(s)", guessedRegistrationHits.Load())
	}
	auth, _ := result.Get(StageAuth)
	if auth.ReasonCode != ReasonDCRUnsupported || auth.Status != StatusFail {
		t.Fatalf("auth stage mutated: %#v", auth)
	}
}

func TestEnrichAuthFailureDistinguishesAdvertisedDCRFailure(t *testing.T) {
	var baseURL string
	server := newLocalTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+baseURL+`/.well-known/oauth-protected-resource/mcp"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/.well-known/oauth-protected-resource/mcp":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resource":              baseURL + "/mcp",
				"authorization_servers": []string{baseURL},
			})
		case "/.well-known/oauth-authorization-server":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                baseURL,
				"registration_endpoint": baseURL + "/registration",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	result := NewResult("client", "Client", "test", baseURL+"/mcp")
	result.SetWithReason(StageAuth, StatusFail, ReasonDCRFailed, "registration failed")
	EnrichAuthFailure(context.Background(), baseURL+"/mcp", &result, server.Client())

	if got := result.Diagnostics[0].Conclusion; got != "dcr_attempt_failed" {
		t.Fatalf("conclusion = %q", got)
	}
	if caps := result.Diagnostics[0].AuthCapabilities; caps == nil || caps.DCRAdvertised == nil || !*caps.DCRAdvertised {
		t.Fatalf("DCR capability not proven: %#v", caps)
	}
}

func TestEnrichAuthFailureKeepsMultipleAuthorizationServersInconclusive(t *testing.T) {
	var baseURL string
	server := newLocalTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+baseURL+`/.well-known/oauth-protected-resource/mcp"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/.well-known/oauth-protected-resource/mcp":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resource":              baseURL + "/mcp",
				"authorization_servers": []string{baseURL + "/issuer-a", baseURL + "/issuer-b"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	result := NewResult("client", "Client", "test", baseURL+"/mcp")
	result.SetWithReason(StageAuth, StatusFail, ReasonDCRUnsupported, "unsupported")
	EnrichAuthFailure(context.Background(), baseURL+"/mcp", &result, server.Client())

	diagnostic := result.Diagnostics[0]
	if diagnostic.Conclusion != "inconclusive" {
		t.Fatalf("conclusion = %q", diagnostic.Conclusion)
	}
	caps := diagnostic.AuthCapabilities
	if caps == nil || !caps.SelectionAmbiguous || caps.AuthorizationServerCount != 2 {
		t.Fatalf("capabilities = %#v", caps)
	}
	if caps.CIMDAdvertised != nil || caps.DCRAdvertised != nil {
		t.Fatalf("ambiguous issuer selection must not invent capabilities: %#v", caps)
	}
}

func TestEnrichAuthFailureMalformedMetadataIsInconclusive(t *testing.T) {
	var baseURL string
	server := newLocalTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+baseURL+`/.well-known/oauth-protected-resource/mcp"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/.well-known/oauth-protected-resource/mcp":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resource":              baseURL + "/mcp",
				"authorization_servers": []string{baseURL},
			})
		case "/.well-known/oauth-authorization-server", "/.well-known/openid-configuration":
			_, _ = w.Write([]byte("not-json"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	result := NewResult("client", "Client", "test", baseURL+"/mcp")
	result.SetWithReason(StageAuth, StatusFail, ReasonDCRUnsupported, "unsupported")
	EnrichAuthFailure(context.Background(), baseURL+"/mcp", &result, server.Client())

	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Conclusion != "inconclusive" {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].AuthCapabilities != nil {
		t.Fatalf("malformed metadata must not produce capabilities: %#v", result.Diagnostics[0].AuthCapabilities)
	}
}

func TestEnrichAuthFailureRejectsMismatchedProtectedResourceMetadata(t *testing.T) {
	var baseURL string
	var authorizationServerHits atomic.Int32
	server := newLocalTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+baseURL+`/.well-known/oauth-protected-resource/mcp"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/.well-known/oauth-protected-resource/mcp":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resource":              baseURL + "/different-resource",
				"authorization_servers": []string{baseURL},
			})
		case "/.well-known/oauth-authorization-server", "/.well-known/openid-configuration":
			authorizationServerHits.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": baseURL})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	result := NewResult("client", "Client", "test", baseURL+"/mcp")
	result.SetWithReason(StageAuth, StatusFail, ReasonDCRUnsupported, "unsupported")
	EnrichAuthFailure(context.Background(), baseURL+"/mcp", &result, server.Client())

	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Conclusion != "inconclusive" {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].AuthCapabilities != nil {
		t.Fatalf("mismatched resource metadata must not produce capabilities: %#v", result.Diagnostics[0].AuthCapabilities)
	}
	if authorizationServerHits.Load() != 0 {
		t.Fatalf("authorization-server metadata was fetched after resource mismatch: %d", authorizationServerHits.Load())
	}
}

func TestAuthProtectedResourceCandidatesPreserveQuery(t *testing.T) {
	endpoint, err := url.Parse("https://example.com/mcp?tenant=acme&mode=readonly")
	if err != nil {
		t.Fatal(err)
	}
	got := authProtectedResourceCandidates(endpoint)
	want := []string{
		"https://example.com/.well-known/oauth-protected-resource/mcp?tenant=acme&mode=readonly",
		"https://example.com/.well-known/oauth-protected-resource?tenant=acme&mode=readonly",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %#v, want %#v", got, want)
	}
}

func TestEnrichAuthFailureIgnoresUnrelatedAuthFailures(t *testing.T) {
	result := NewResult("client", "Client", "test", "https://example.com/mcp")
	result.SetWithReason(StageAuth, StatusFail, ReasonClientAuthRejected, "invalid client")
	EnrichAuthFailure(context.Background(), result.Endpoint, &result, nil)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
}

func TestRedactResultRedactsDiagnosticMessages(t *testing.T) {
	result := NewResult("client", "Client", "test", "https://example.com/mcp?access_token=secret-value")
	result.AddDiagnostic(Diagnostic{ID: "test", Message: "Authorization: Bearer abcdefghijklmnop"})
	redacted := RedactResult(result)
	if strings.Contains(redacted.Endpoint, "secret-value") {
		t.Fatalf("endpoint leaked secret: %s", redacted.Endpoint)
	}
	if strings.Contains(redacted.Diagnostics[0].Message, "abcdefghijklmnop") {
		t.Fatalf("diagnostic leaked bearer token: %s", redacted.Diagnostics[0].Message)
	}
}

func newLocalTLSServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
			t.Skipf("local listener unavailable in this environment: %v", err)
		}
		t.Fatalf("start local listener: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.StartTLS()
	return server
}
