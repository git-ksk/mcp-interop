package main

import (
	"context"
	"testing"
)

func TestValidateInteractiveAuthorizationURL(t *testing.T) {
	for _, raw := range []string{
		"https://auth.example/authorize?client_id=test&state=s",
		"http://127.0.0.1:8080/authorize?state=s",
		"http://[::1]:8080/authorize?state=s",
	} {
		if err := validateInteractiveAuthorizationURL(raw); err != nil {
			t.Fatalf("expected authorization URL to be accepted %q: %v", raw, err)
		}
	}

	for _, raw := range []string{
		"http://auth.example/authorize",
		"javascript:alert(1)",
		"file:///tmp/authorize",
		"https://user:pass@auth.example/authorize",
		"https://auth.example/authorize#fragment",
	} {
		if err := validateInteractiveAuthorizationURL(raw); err == nil {
			t.Fatalf("expected authorization URL to be rejected: %q", raw)
		}
	}
}

func TestMaybeAutoAuthorizeValidatesBeforeDisplayBoundary(t *testing.T) {
	t.Setenv(autoAuthorizeLoopbackEnv, "")

	handled, err := maybeAutoAuthorizeLoopback(context.Background(), "javascript:alert(1)")
	if !handled || err == nil {
		t.Fatalf("unsafe authorization URL must be stopped before display, handled=%v err=%v", handled, err)
	}

	handled, err = maybeAutoAuthorizeLoopback(context.Background(), "https://auth.example/authorize?state=s")
	if handled || err != nil {
		t.Fatalf("valid HTTPS authorization URL should remain interactive, handled=%v err=%v", handled, err)
	}
}
