package cursor

import "testing"

func TestRequiresOAuthRecognizesHTTPAuthBoundary(t *testing.T) {
	for _, input := range []string{
		"HTTP 401 from MCP server",
		"request failed: Unauthorized",
	} {
		if !requiresOAuth(input) {
			t.Fatalf("expected OAuth boundary for %q", input)
		}
	}
}
