package cursor

import "testing"

func TestRequiresOAuthRecognizesStandaloneHTTP401(t *testing.T) {
	for _, input := range []string{
		"HTTP 401 Unauthorized",
		"request failed: status=401",
		"401",
		"server returned (401)",
	} {
		if !requiresOAuth(input) {
			t.Fatalf("expected standalone 401 to indicate OAuth boundary: %q", input)
		}
	}
}

func TestRequiresOAuthDoesNotMatch401InsideOtherNumbers(t *testing.T) {
	for _, input := range []string{
		"connection attempted on port 4010",
		"build 1401 failed",
		"server id=4012 is unavailable",
		"elapsed=40100ms",
	} {
		if requiresOAuth(input) {
			t.Fatalf("embedded 401 digits must not imply OAuth: %q", input)
		}
	}
}
