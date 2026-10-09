package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestParseBoundedTimeout(t *testing.T) {
	for _, value := range []string{"1s", "45s", "2m", "1m30s", "10m"} {
		got, err := parseBoundedTimeout(value)
		want, _ := time.ParseDuration(value)
		if err != nil || got != want {
			t.Errorf("valid %q: got %v, error %v", value, got, err)
		}
	}
	for _, value := range []string{"", "0", "0s", "-1s", "999ms", "10m1s", "100h", "inf", "abc", "99999999999999999999s"} {
		if _, err := parseBoundedTimeout(value); err == nil {
			t.Errorf("unsafe timeout accepted: %q", value)
		}
	}
}

func TestParseTestTimeoutAndDefault(t *testing.T) {
	a, err := parseTestOptions([]string{"https://example.test/mcp"})
	if err != nil || a.timeout != nil {
		t.Fatalf("default must remain unchanged: %#v, %v", a, err)
	}
	for _, args := range [][]string{
		{"https://example.test/mcp", "--timeout", "45s", "--client", "codex"},
		{"--timeout=45s", "https://example.test/mcp"},
	} {
		got, err := parseTestOptions(args)
		if err != nil || got.timeout == nil || *got.timeout != 45*time.Second {
			t.Fatalf("got %#v, err=%v", got, err)
		}
	}
	for _, flags := range [][]string{
		{"--timeout"}, {"--timeout="}, {"--timeout", "0s"}, {"--timeout=-5s"},
		{"--timeout=11m"}, {"--timeout=1ms"}, {"--timeout=1m", "--timeout=2m"},
	} {
		args := append([]string{"https://example.test/mcp"}, flags...)
		if _, err := parseTestOptions(args); err == nil {
			t.Fatalf("invalid timeout accepted: %#v", args)
		}
	}
}

func TestParseSuiteRunTimeoutAndDefault(t *testing.T) {
	a, err := parseSuiteRunOptions([]string{"manifest.json", "--output-dir", "results"})
	if err != nil || a.timeout != nil {
		t.Fatalf("suite default changed: %#v %v", a, err)
	}
	b, err := parseSuiteRunOptions([]string{"manifest.json", "--output-dir", "results", "--timeout=2m"})
	if err != nil || b.timeout == nil || *b.timeout != 2*time.Minute {
		t.Fatalf("suite timeout: %#v %v", b, err)
	}
	for _, flags := range [][]string{
		{"--timeout"}, {"--timeout=0s"}, {"--timeout=11m"},
		{"--timeout=1s", "--timeout=2s"},
	} {
		args := append([]string{"manifest.json", "--output-dir", "results"}, flags...)
		if _, err := parseSuiteRunOptions(args); err == nil {
			t.Fatalf("invalid suite timeout accepted: %#v", args)
		}
	}
}

func TestSuiteRunForwardsTimeoutToCoreRunner(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "suite.json")
	outputDir := filepath.Join(root, "result-set")
	manifest := `{"schema_version":1,"execution_context":"trusted_real_client","targets":[{"id":"dev","endpoint":{"source":"environment","variable":"MCP_INTEROP_SUITE_ENDPOINT_DEV"},"deployment_id":"dev","clients":[{"id":"codex","auth":"none"}]}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	endpoint := "https://example.test/very-private-path"
	called := 0
	fake := func(_ context.Context, args []string, _ io.Writer, _ io.Writer) int {
		called++
		if got := optionValue(t, args, "--timeout"); got != "45s" {
			t.Fatalf("timeout not forwarded: %q", got)
		}
		if args[0] != endpoint {
			t.Fatal("suite lost the protected endpoint")
		}
		result := interop.NewResult("codex", "Codex", "1.0", endpoint)
		for _, stage := range interop.OrderedStages {
			result.Set(stage, interop.StatusPass, "ok")
		}
		run, err := artifact.NewRunV2ProtectedPath(result, endpoint, "dev", time.Unix(1, 0).UTC(), "default", artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"}, "test", "deadbeef")
		if err != nil {
			t.Fatal(err)
		}
		if err := artifact.WriteFile(optionValue(t, args, "--output"), artifact.NewArtifactV2([]artifact.Run{run})); err != nil {
			t.Fatal(err)
		}
		return 0
	}
	var out, errs bytes.Buffer
	rc := runSuiteRunWith(context.Background(), []string{manifestPath, "--output-dir", outputDir, "--timeout", "45s"}, &out, &errs, func(name string) (string, bool) { return endpoint, name == "MCP_INTEROP_SUITE_ENDPOINT_DEV" }, fake)
	if rc != 0 || called != 1 {
		t.Fatalf("rc=%d calls=%d stderr=%s", rc, called, errs.String())
	}
	if _, err := os.Stat(filepath.Join(outputDir, "index.json")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "very-private-path") {
		t.Fatalf("suite summary leaked endpoint: %s", out.String())
	}
}

func TestSuiteRunRejectsExcessiveAggregateTimeoutBudgetBeforeExecution(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "suite.json")
	outputDir := filepath.Join(root, "result-set")
	manifest := `{"schema_version":1,"execution_context":"trusted_real_client","targets":[{"id":"dev","endpoint":{"source":"environment","variable":"MCP_INTEROP_SUITE_ENDPOINT_DEV"},"deployment_id":"dev","clients":[{"id":"codex","auth":"none"},{"id":"cursor","auth":"none"},{"id":"antigravity","auth":"none"}]},{"id":"prod","endpoint":{"source":"environment","variable":"MCP_INTEROP_SUITE_ENDPOINT_PROD"},"deployment_id":"prod","clients":[{"id":"codex","auth":"none"},{"id":"cursor","auth":"none"},{"id":"antigravity","auth":"none"}]}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	called := 0
	fake := func(_ context.Context, _ []string, _ io.Writer, _ io.Writer) int { called++; return 0 }
	var out, errs bytes.Buffer
	rc := runSuiteRunWith(context.Background(), []string{manifestPath, "--output-dir", outputDir, "--timeout", "10m"}, &out, &errs, func(string) (string, bool) { return "https://example.test/secret-path", true }, fake)
	if rc != 2 || called != 0 || !strings.Contains(errs.String(), "45 minutes") {
		t.Fatalf("budget was not rejected before execution: rc=%d called=%d stderr=%s", rc, called, errs.String())
	}
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Fatalf("invalid suite created output: %v", err)
	}
}
