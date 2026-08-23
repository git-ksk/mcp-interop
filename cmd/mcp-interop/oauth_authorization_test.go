package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaybeAutoAuthorizeLoopbackRefusesNonLoopbackRedirect(t *testing.T) {
	t.Setenv(autoAuthorizeLoopbackEnv, "1")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, nil, "http://example.com/authorization", http.StatusFound)
	}))
	defer server.Close()

	handled, err := maybeAutoAuthorizeLoopback(context.Background(), server.URL)
	if !handled {
		t.Fatal("expected explicit E2E auto-authorization mode to handle the request")
	}
	if err == nil || !strings.Contains(err.Error(), "redirect refused") {
		t.Fatalf("expected non-loopback redirect refusal, got %v", err)
	}
}

func TestMaybeAutoAuthorizeLoopbackAllowsLoopbackRedirect(t *testing.T) {
	t.Setenv(autoAuthorizeLoopbackEnv, "1")
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/done", http.StatusFound)
	})
	mux.HandleFunc("/done", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handled, err := maybeAutoAuthorizeLoopback(context.Background(), server.URL+"/start")
	if !handled || err != nil {
		t.Fatalf("expected loopback redirect to succeed, handled=%v err=%v", handled, err)
	}
}

func TestValidateAutoAuthorizeLoopbackURLRejectsUnsafeForms(t *testing.T) {
	for _, raw := range []string{
		"https://127.0.0.1:8080/authorize",
		"http://example.com/authorize",
		"http://user:pass@127.0.0.1:8080/authorize",
		"http://127.0.0.1:8080/authorize#fragment",
	} {
		if err := validateAutoAuthorizeLoopbackURL(raw); err == nil {
			t.Fatalf("expected unsafe URL to fail: %s", raw)
		}
	}
	if err := validateAutoAuthorizeLoopbackURL("http://127.0.0.1:8080/authorize"); err != nil {
		t.Fatalf("expected loopback HTTP URL to pass: %v", err)
	}
	if err := validateAutoAuthorizeLoopbackURL("http://[::1]:8080/authorize"); err != nil {
		t.Fatalf("expected IPv6 loopback HTTP URL to pass: %v", err)
	}
}
