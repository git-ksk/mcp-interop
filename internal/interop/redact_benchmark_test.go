package interop

import "testing"

func BenchmarkRedactResult(b *testing.B) {
	result := NewResult("codex", "Codex CLI", "codex-cli test", "https://example.com/mcp?tenant=acme&access_token=secret-token&api_key=secret-key")
	result.Set(StageReach, StatusPass, "Authorization: Bearer abcdefghijklmnop")
	result.Set(StageAuth, StatusFail, "callback failed?code=secret-code&client_secret=secret")
	result.AddDiagnostic(Diagnostic{ID: "auth", Message: `{"refresh_token":"secret-refresh"}`})

	b.ReportAllocs()
	for b.Loop() {
		redacted := RedactResult(result)
		if redacted.Endpoint == result.Endpoint {
			b.Fatal("endpoint was not sanitized")
		}
	}
}

func BenchmarkSanitizeEndpoint(b *testing.B) {
	endpoint := "https://example.com/mcp?tenant=acme&access_token=secret-token&refresh_token=secret-refresh&X-Amz-Credential=secret&X-Amz-Signature=secret&author=alice"

	b.ReportAllocs()
	for b.Loop() {
		sanitized := SanitizeEndpoint(endpoint)
		if sanitized == endpoint {
			b.Fatal("endpoint was not sanitized")
		}
	}
}
