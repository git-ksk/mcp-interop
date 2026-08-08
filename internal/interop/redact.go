package interop

import "regexp"

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)[^\s"']+`),
	regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(`(?i)([?&](?:access_token|refresh_token|client_secret|code)=)[^&\s]+`),
	regexp.MustCompile(`(?i)("(?:access_token|refresh_token|client_secret|authorization_code)"\s*:\s*")[^"]+(")`),
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

// RedactResult applies Redact to every human-readable diagnostic in a result.
func RedactResult(result Result) Result {
	for i := range result.Stages {
		result.Stages[i].Message = Redact(result.Stages[i].Message)
	}
	return result
}
