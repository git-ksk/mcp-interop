package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSuiteValidateAcceptsTrustedManifestWithoutPrintingEndpoint(t *testing.T) {
	path := writeSuiteManifest(t, `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [{
    "id": "production-a",
    "endpoint": {"source": "environment", "variable": "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"},
    "deployment_id": "production-a",
    "clients": [{"id": "codex", "auth": "none"}, {"id": "cursor", "auth": "oauth"}]
  }]
}`)
	var stdout, stderr bytes.Buffer
	if code := runSuiteValidate([]string{path, "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	for _, want := range []string{`"valid": true`, `"execution_context": "trusted_real_client"`, `"runs": 2`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A") {
		t.Fatalf("validation summary leaked resolver detail: %s", stdout.String())
	}
}

func TestRunSuiteValidateRejectsUnsafeManifest(t *testing.T) {
	path := writeSuiteManifest(t, `{"schema_version":1,"execution_context":"trusted_real_client","token":"secret","targets":[]}`)
	var stdout, stderr bytes.Buffer
	if code := runSuiteValidate([]string{path}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `unknown field "token"`) {
		t.Fatalf("unexpected error: %s", stderr.String())
	}
	if strings.Contains(stderr.String(), "secret") {
		t.Fatalf("error leaked rejected field value: %s", stderr.String())
	}
}

func writeSuiteManifest(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "suite.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
