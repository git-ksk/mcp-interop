package diagnose

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMergeRuntimeEvidenceNormalizesV2AndV3ToV3(t *testing.T) {
	v2 := ChatGPTRuntimeEvidence{
		SchemaVersion: 2,
		Registration:  &RegistrationEvidence{Strategy: "cimd", ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json"},
		ToolAuth: &ToolAuthEvidence{
			OAuth2SecuritySchemePresent: boolPtr(true),
		},
	}
	v3 := ChatGPTRuntimeEvidence{
		SchemaVersion: 3,
		ResourceRequest: &ResourceRequestEvidence{
			BearerPresent: boolPtr(true),
		},
		ToolChallenge: &ToolChallengeEvidence{
			Expected: boolPtr(false),
		},
	}
	merged, err := MergeRuntimeEvidence([]ChatGPTRuntimeEvidence{v2, v3})
	if err != nil {
		t.Fatal(err)
	}
	if merged.SchemaVersion != 3 || merged.ToolAuth != nil || merged.ToolMetadata == nil || merged.ToolChallenge == nil {
		t.Fatalf("merged=%#v", merged)
	}
	if merged.ToolMetadata.OAuth2SecuritySchemePresent == nil || !*merged.ToolMetadata.OAuth2SecuritySchemePresent {
		t.Fatalf("metadata=%#v", merged.ToolMetadata)
	}
	if merged.ToolChallenge.Expected == nil || *merged.ToolChallenge.Expected {
		t.Fatalf("challenge=%#v", merged.ToolChallenge)
	}
}

func TestMergeRuntimeEvidenceRejectsConflicts(t *testing.T) {
	left := ChatGPTRuntimeEvidence{SchemaVersion: 3, ResourceRequest: &ResourceRequestEvidence{BearerPresent: boolPtr(true)}}
	right := ChatGPTRuntimeEvidence{SchemaVersion: 3, ResourceRequest: &ResourceRequestEvidence{BearerPresent: boolPtr(false)}}
	_, err := MergeRuntimeEvidence([]ChatGPTRuntimeEvidence{left, right})
	if err == nil || !strings.Contains(err.Error(), "resource_request.bearer_present") {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCanonicalRuntimeEvidenceV3DropsLegacyTopLevelShape(t *testing.T) {
	body := []byte(`{
		"client_id": "https://chatgpt.com/oauth/test/client.json",
		"resource_matches": true,
		"code_verifier_present": true,
		"client_assertion_present": false
	}`)
	var legacy ChatGPTRuntimeEvidence
	if err := json.Unmarshal(body, &legacy); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Validate(); err != nil {
		t.Fatal(err)
	}
	canonical := CanonicalRuntimeEvidenceV3(legacy)
	encoded, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	if _, ok := object["client_id"]; ok {
		t.Fatalf("legacy top-level client_id leaked into canonical v3: %s", encoded)
	}
	if canonical.SchemaVersion != 3 || canonical.Registration == nil || canonical.TokenRequest == nil {
		t.Fatalf("canonical=%#v", canonical)
	}
}

func TestSummarizeRuntimeEvidenceReportsStructureOnly(t *testing.T) {
	evidence := ChatGPTRuntimeEvidence{
		SchemaVersion: 3,
		Registration:  &RegistrationEvidence{Strategy: "cimd", ClientMetadataURL: "https://chatgpt.com/oauth/test/client.json"},
		ResourceRequest: &ResourceRequestEvidence{
			BearerPresent:   boolPtr(true),
			SignatureValid:  boolPtr(true),
			AudienceMatches: boolPtr(true),
		},
		ToolChallenge: &ToolChallengeEvidence{Expected: boolPtr(false)},
	}
	summary := SummarizeRuntimeEvidence(evidence)
	if summary.SchemaVersion != 3 || summary.TotalSupplied != 6 {
		t.Fatalf("summary=%#v", summary)
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"chatgpt.com", "client.json", "true", "false"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary exposed observed value %q: %s", forbidden, text)
		}
	}
}
