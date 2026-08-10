package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthorizeWritesPrivateCodeFileWithoutLoggingCode(t *testing.T) {
	codeFile := filepath.Join(t.TempDir(), "authorization-code")
	var log bytes.Buffer
	h := &server{
		baseURL:  "http://127.0.0.1:9999",
		log:      &log,
		codeFile: codeFile,
		clients: map[string][]string{
			"fixture-client": {"http://127.0.0.1:54321/callback"},
		},
		codes:  map[string]authorizationCode{},
		tokens: map[string]struct{}{},
	}
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {"fixture-client"},
		"redirect_uri":          {"http://127.0.0.1:54321/callback"},
		"state":                 {"state"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
	}
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9999/authorize?"+query.Encode(), nil)
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	content, err := os.ReadFile(codeFile)
	if err != nil {
		t.Fatal(err)
	}
	code := string(content)
	if !strings.HasPrefix(code, "fixture-code-") {
		t.Fatalf("unexpected private authorization code shape: %q", code)
	}
	if strings.Contains(log.String(), code) {
		t.Fatal("authorization code leaked into fixture log")
	}
	if !strings.Contains(log.String(), `"path":"/authorize"`) {
		t.Fatalf("secret-free authorize event missing from log: %s", log.String())
	}
}
