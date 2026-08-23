package interop

import (
	"strings"
	"testing"
)

func TestRedactJSONCredentialFieldVariants(t *testing.T) {
	input := `{"accessToken":"access-secret","refresh_token":"refresh-secret","clientSecret":"client-secret-value","code_verifier":"verifier-secret","apiKey":"api-secret","password":"password-secret","authorization":"Basic credential-secret","token":"generic-token-secret"}`
	got := Redact(input)
	for _, secret := range []string{
		"access-secret",
		"refresh-secret",
		"client-secret-value",
		"verifier-secret",
		"api-secret",
		"password-secret",
		"credential-secret",
		"generic-token-secret",
	} {
		if strings.Contains(got, secret) {
			t.Fatalf("JSON credential field leaked %q: %s", secret, got)
		}
	}
}

func TestRedactCompactCredentialQueryKeys(t *testing.T) {
	input := "request failed: https://example.com/mcp?tenant=acme&accessToken=access-secret&clientSecret=client-secret&codeVerifier=verifier-secret&apiKey=api-secret"
	got := Redact(input)
	for _, secret := range []string{"access-secret", "client-secret", "verifier-secret", "api-secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("compact query credential leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "tenant=acme") {
		t.Fatalf("ordinary query value was unexpectedly removed: %s", got)
	}
}

func TestRedactJSONDoesNotRedactPublicKeyField(t *testing.T) {
	input := `{"key":"public-key-id","monkey":"banana","tenant":"acme"}`
	if got := Redact(input); got != input {
		t.Fatalf("ordinary JSON fields were unexpectedly redacted: %s", got)
	}
}
