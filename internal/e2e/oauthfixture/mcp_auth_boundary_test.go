package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnauthenticatedMCPGetReturnsOAuthChallenge(t *testing.T) {
	h := &server{
		baseURL: "http://127.0.0.1:9999",
		log:     io.Discard,
		clients: map[string][]string{},
		codes:   map[string]authorizationCode{},
		tokens:  map[string]struct{}{},
	}
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9999/mcp", nil)
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	challenge := recorder.Header().Get("WWW-Authenticate")
	if !strings.Contains(challenge, "resource_metadata=") {
		t.Fatalf("missing resource_metadata challenge: %q", challenge)
	}
}
