package codex

import (
	"bytes"
	"strings"
	"testing"
)

func TestRPCClientDoesNotTreatServerRequestAsResponse(t *testing.T) {
	input := strings.Join([]string{
		`{"id":1,"method":"some/server/request","params":{}}`,
		`{"id":1,"result":{"value":"response"}}`,
		"",
	}, "\n")

	client := newRPCClient(strings.NewReader(input), &bytes.Buffer{})
	var result struct {
		Value string `json:"value"`
	}
	if err := client.call("client/request", map[string]any{}, &result); err != nil {
		t.Fatal(err)
	}
	if result.Value != "response" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
