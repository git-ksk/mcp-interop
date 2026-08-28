package main

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestV1CandidateTopLevelExitClasses(t *testing.T) {
	if got := run(context.Background(), nil); got != 2 {
		t.Fatalf("no-args exit=%d want 2", got)
	}
	if got := run(context.Background(), []string{"definitely-unknown"}); got != 2 {
		t.Fatalf("unknown-command exit=%d want 2", got)
	}
	if got := run(context.Background(), []string{"help"}); got != 0 {
		t.Fatalf("help exit=%d want 0", got)
	}
	if got := run(context.Background(), []string{"maturity", "unexpected"}); got != 2 {
		t.Fatalf("invalid maturity invocation exit=%d want 2", got)
	}
}

func TestV1CandidatePrimaryLiveResultJSONFieldsAndStageOrder(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "codex-cli 1.0", "https://example.com/mcp")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "ok")
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	wantKeys := []string{"client_id", "client_name", "client_version", "endpoint", "stages"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Fatalf("live-result JSON fields changed: got=%v want=%v", keys, wantKeys)
	}
	var stages []struct {
		Stage string `json:"stage"`
	}
	if err := json.Unmarshal(object["stages"], &stages); err != nil {
		t.Fatal(err)
	}
	var gotOrder []string
	for _, stage := range stages {
		gotOrder = append(gotOrder, stage.Stage)
	}
	if !reflect.DeepEqual(gotOrder, []string{"reach", "auth", "init", "tools"}) {
		t.Fatalf("core stage order changed: %v", gotOrder)
	}

	// Diagnostics are additive/optional and must not appear when absent.
	if bytes.Contains(data, []byte(`"diagnostics"`)) {
		t.Fatalf("empty diagnostics unexpectedly serialized: %s", data)
	}
}
