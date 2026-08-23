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

var queryParameterPattern = regexp.MustCompile(`([?&])([^=&\s#]+)=([^&\s#]*)`)

var sensitiveQueryWords = map[string]struct{}{
	"authorization": {},
	"code":          {},
	"credential":    {},
	"credentials":   {},
	"key":           {},
	"password":      {},
	"secret":        {},
	"sig":           {},
	"signature":     {},
	"token":         {},
}

// Redact removes common OAuth and bearer credential material from diagnostic
// text before it can be emitted in reports or logs. Sensitive URL query values
// are redacted even when a complete endpoint appears inside free-form text.
func Redact(value string) string {
	value = secretPatterns[0].ReplaceAllString(value, `${1}[REDACTED]`)
	value = secretPatterns[1].ReplaceAllString(value, `${1}[REDACTED]`)
	value = secretPatterns[2].ReplaceAllString(value, `${1}[REDACTED]`)
	value = secretPatterns[3].ReplaceAllString(value, `${1}[REDACTED]${2}`)
	value = queryParameterPattern.ReplaceAllStringFunc(value, redactSensitiveQueryParameter)
	return value
}

func redactSensitiveQueryParameter(parameter string) string {
	equals := strings.IndexByte(parameter, '=')
	if equals < 2 {
		return parameter
	}
	key := parameter[1:equals]
	if decoded, err := url.QueryUnescape(key); err == nil {
		key = decoded
	}
	if !sensitiveQueryKey(key) {
		return parameter
	}
	return parameter[:equals+1] + "[REDACTED]"
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
	normalized := strings.ToLower(key)
	normalized = strings.NewReplacer("-", "_", ".", "_", "[", "_", "]", "_").Replace(normalized)
	for _, word := range strings.FieldsFunc(normalized, func(r rune) bool { return r == '_' }) {
		if _, ok := sensitiveQueryWords[word]; ok {
			return true
		}
	}
	return normalized == "apikey" || normalized == "api_key"
}

// RedactResult applies credential redaction to all user-visible report fields.
func RedactResult(result Result) Result {
	result.Endpoint = SanitizeEndpoint(result.Endpoint)
	for i := range result.Stages {
		result.Stages[i].Message = Redact(result.Stages[i].Message)
	}
	for i := range result.Diagnostics {
		result.Diagnostics[i].Message = Redact(result.Diagnostics[i].Message)
	}
	return result
}
