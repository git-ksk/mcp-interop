package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	diagnosepkg "github.com/git-ksk/mcp-interop/internal/diagnose"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestReadRuntimeEvidenceAcceptsStructuredV2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.json")
	body := `{
		"schema_version": 2,
		"registration": {
			"strategy": "cimd",
			"client_metadata_url": "https://chatgpt.com/oauth/test/client.json"
		},
		"authorization_request": {
			"resource_matches": true,
			"redirect_uri_matches": true,
			"pkce_s256": true
		},
		"token_request": {
			"resource_matches": true,
			"code_verifier_present": true,
			"client_assertion_present": false,
			"client_assertion_type_present": false
		},
		"resource_request": {
			"bearer_present": false
		}
	}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	evidence, err := readRuntimeEvidence(path)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.SchemaVersion != 2 {
		t.Fatalf("schema=%d", evidence.SchemaVersion)
	}
	if evidence.EffectiveClientID() != "https://chatgpt.com/oauth/test/client.json" {
		t.Fatalf("client=%q", evidence.EffectiveClientID())
	}
}

func TestWriteDiagnoseReportShowsReferencePatternLayer(t *testing.T) {
	report := diagnosepkg.Report{
		Profile:  "chatgpt",
		Endpoint: "https://example.com/mcp",
		Checks: []diagnosepkg.Check{{
			ID: "authorization_server_metadata", Status: diagnosepkg.StatusPass, Blocking: true, Message: "ok",
		}},
		RuntimeEvidence: &diagnosepkg.RuntimeEvidenceReport{
			SchemaVersion:        2,
			RegistrationStrategy: "cimd",
			Status:               diagnosepkg.StatusFail,
			ReasonCode:           interop.ReasonTokenAuthMethodMismatch,
			Checks: []diagnosepkg.RuntimeCheck{{
				ID: "token_auth_method", Status: diagnosepkg.StatusFail, Expected: "private_key_jwt", Observed: "none", ReasonCode: interop.ReasonTokenAuthMethodMismatch, Message: "mismatch",
			}},
			OpenAIReference: &diagnosepkg.ReferencePatternReport{
				Status:          diagnosepkg.StatusFail,
				ProfileRevision: "2026-08-10.1",
				ObservedDate:    "2026-08-10",
				Source:          "OpenAI authenticated MCP reference pattern",
				Checks: []diagnosepkg.RuntimeCheck{{
					ID: "token_auth", Status: diagnosepkg.StatusFail, Expected: "private_key_jwt", Observed: "none", ReasonCode: interop.ReasonTokenAuthMethodMismatch, Message: "mismatch",
				}},
			},
		},
	}
	var output bytes.Buffer
	if err := writeDiagnoseReport(&output, report); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, expected := range []string{"PREFLIGHT PASS", "RUNTIME EVIDENCE", "SCHEMA", "v2", "TOKEN_AUTH_METHOD_MISMATCH", "OPENAI REFERENCE PATTERN", "PROFILE_REVISION", "2026-08-10.1", "OBSERVED_DATE", "2026-08-10"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("output missing %q:\n%s", expected, text)
		}
	}
}
