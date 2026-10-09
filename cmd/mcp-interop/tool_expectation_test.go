package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestParseTestOptionsExpectToolAndCount(t *testing.T) {
	options, err := parseTestOptions([]string{
		"https://example.test/mcp", "--expect-tool", "ping", "--expect-tool=read_tool",
		"--expect-tool-count", "3", "--client=codex,cursor", "--json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(options.expectedTools, []string{"ping", "read_tool"}) || options.expectedCount == nil || *options.expectedCount != 3 {
		t.Fatalf("incorrect option parsing: %#v", options)
	}
	zero, err := parseTestOptions([]string{"https://example.test/mcp", "--expect-tool-count=0"})
	if err != nil || zero.expectedCount == nil || *zero.expectedCount != 0 {
		t.Fatalf("zero count must parse: %#v, %v", zero, err)
	}
}

func TestParseTestOptionsRejectsInvalidToolChecks(t *testing.T) {
	for _, flags := range [][]string{
		{"--expect-tool"}, {"--expect-tool="}, {"--expect-tool", "--json"},
		{"--expect-tool", "a/b"}, {"--expect-tool", "ping", "--expect-tool", "ping"},
		{"--expect-tool-count"}, {"--expect-tool-count="}, {"--expect-tool-count=-1"},
		{"--expect-tool-count=4097"}, {"--expect-tool-count=999999999999999999999999"},
		{"--expect-tool-count=1x"}, {"--expect-tool-count=+1"},
		{"--expect-tool-count=3", "--expect-tool-count=3"},
	} {
		args := append([]string{"https://example.test/mcp"}, flags...)
		if options, err := parseTestOptions(args); err == nil {
			t.Errorf("accepted invalid args %#v: %#v", args, options)
		}
	}
	flags := []string{"https://example.test/mcp"}
	for i := 0; i <= interop.MaxExpectedToolNames; i++ {
		flags = append(flags, "--expect-tool", "tool_"+strings.Repeat("x", i))
	}
	if _, err := parseTestOptions(flags); err == nil {
		t.Fatal("excessive expected names accepted")
	}
}

func TestToolCheckAddsOptInOnlyJSONFieldAndHumanSummary(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "0.152.1", "https://example.test/mcp")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "real client passed")
	}
	if !result.SetObservedToolNames([]string{"ping", "read_tool"}) {
		t.Fatal("failed to set direct names")
	}
	n := 2
	check := interop.CheckToolExpectation(result, []string{"ping", "missing"}, &n)
	result.ToolExpectation = &check
	data, err := json.Marshal([]interop.Result{result})
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil || len(decoded) != 1 {
		t.Fatalf("array contract broke: %s, %v", data, err)
	}
	if _, ok := decoded[0]["tool_expectation"]; !ok {
		t.Fatalf("expectation not included: %s", data)
	}
	if strings.Contains(string(data), "read_tool") {
		t.Fatalf("private observed name leaked: %s", data)
	}
	var stdout bytes.Buffer
	if err := writeTestResults(&stdout, []interop.Result{result}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "EXPECTED TOOLS") || !strings.Contains(stdout.String(), "MISSING NAMES") {
		t.Fatalf("human assertion not rendered: %s", stdout.String())
	}
	result.ToolExpectation = nil
	baseline, err := json.Marshal([]interop.Result{result})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(baseline), "tool_expectation") {
		t.Fatalf("unrequested assertion changed legacy JSON: %s", baseline)
	}
}
