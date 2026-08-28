package diagnose

import (
	"encoding/json"
	"testing"
)

func TestV1CandidateRuntimeEvidenceVersionsRemainReadable(t *testing.T) {
	if runtimeEvidenceSchemaV2 != 2 || runtimeEvidenceSchemaV3 != 3 {
		t.Fatalf("runtime evidence schema constants changed: v2=%d v3=%d", runtimeEvidenceSchemaV2, runtimeEvidenceSchemaV3)
	}
	for _, input := range []string{
		`{"client_id":"https://chatgpt.com/oauth/test/client.json","resource_matches":true,"code_verifier_present":true,"client_assertion_present":false}`,
		`{"schema_version":2,"registration":{"strategy":"cimd","client_metadata_url":"https://chatgpt.com/oauth/test/client.json"}}`,
		`{"schema_version":3,"tool_challenge":{"expected":false}}`,
	} {
		var evidence ChatGPTRuntimeEvidence
		if err := json.Unmarshal([]byte(input), &evidence); err != nil {
			t.Fatalf("supported runtime evidence decode failed: %s: %v", input, err)
		}
		if err := evidence.Validate(); err != nil {
			t.Fatalf("supported runtime evidence became invalid: %s: %v", input, err)
		}
	}
}
