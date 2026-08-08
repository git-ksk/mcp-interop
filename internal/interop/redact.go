package interop

import (
	"net/url"
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)[^\s"']+`),
	regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(`(?i)([?&](?:access_token|refresh_token|client_secret|code)=)[^&\s]+`),
	regexp.MustCompile(`(?i)("(?:access_token|refresh_token|client_secret|authorization_code)"\s*:\s*")[^"]+(")`),
}

var sensitiveQueryKeyParts = []string{
	"token",
	"secret",
	"api_key",
	"apikey",
	"authorization",
	"auth",
	"code",
	"signature",
	"sig",
}

// Redact removes common OAuth and bearer credential material from diagnostic
// text before it can be emitted in reports or logs.
func Redact(value string) string {
	value = secretPatterns[0].ReplaceAllString(value, `${1}[REDACTED]`)
	value = secretPatterns[1].ReplaceAllString(value, `${1}[REDACTED]`)
	value = secretPatterns[2].ReplaceAllString(value, `${1}[REDACTED]`)
	value = secretPatterns[3].ReplaceAllString(value, `${1}[REDACTED]${2}`)
	return value
}

// SanitizeEndpoint masks credential-like query values while preserving routing
// and non-sensitive query parameters that can matter when reproducing a test.
func SanitizeEndpoint(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return Redact(raw)
	}

	u.User = nil
	query := u.Query()
	for key := range query {
		if sensitiveQueryKey(key) {
			query.Set(key, "REDACTED")
		}
	}
	u.RawQuery = query.Encode()
	return Redact(u.String())
}

func sensitiveQueryKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, part := range sensitiveQueryKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

// RedactResult applies credential redaction to all user-visible report fields.
func RedactResult(result Result) Result {
	result.Endpoint = SanitizeEndpoint(result.Endpoint)
	for i := range result.Stages {
		result.Stages[i].Message = Redact(result.Stages[i].Message)
	}
	return result
}
