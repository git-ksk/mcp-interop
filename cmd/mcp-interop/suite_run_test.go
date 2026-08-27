package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

func TestRunSuiteRunEmitsDeterministicArtifactSetAndKeepsNonPass(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "suite.json")
	outputDir := filepath.Join(root, "results")
	manifest := `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [
    {
      "id": "production-a",
      "endpoint": {"source": "environment", "variable": "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"},
      "deployment_id": "production-a",
      "clients": [
        {"id": "cursor", "auth": "none"},
        {"id": "codex", "auth": "none"}
      ]
    }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	const protectedEndpoint = "https://example.com/mcp/very-secret-path"
	lookup := func(name string) (string, bool) {
		if name != "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A" {
			t.Fatalf("unexpected endpoint variable %q", name)
		}
		return protectedEndpoint, true
	}
	calls := make([][]string, 0, 2)
	fake := func(_ context.Context, args []string, _ io.Writer, _ io.Writer) int {
		calls = append(calls, append([]string(nil), args...))
		clientID := optionValue(t, args, "--client")
		outputPath := optionValue(t, args, "--output")
		deploymentID := optionValue(t, args, "--deployment-id")
		if args[0] != protectedEndpoint || deploymentID != "production-a" {
			t.Fatalf("unexpected protected execution args: %#v", args)
		}
		result := interop.NewResult(clientID, clientID+" test", "1.0.0", protectedEndpoint)
		rc := 0
		for _, stage := range interop.OrderedStages {
			status := interop.StatusPass
			message := "ok"
			if clientID == "cursor" {
				status = interop.StatusSkip
				message = "client missing"
				rc = 1
			}
			result.Set(stage, status, message)
		}
		run, err := artifact.NewRunV2ProtectedPath(
			result,
			protectedEndpoint,
			deploymentID,
			time.Unix(1, 0).UTC(),
			"default",
			artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: clientID},
			"test",
			"deadbeef",
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := artifact.WriteFile(outputPath, artifact.NewArtifactV2([]artifact.Run{run})); err != nil {
			t.Fatal(err)
		}
		return rc
	}

	var stdout, stderr bytes.Buffer
	rc := runSuiteRunWith(
		context.Background(),
		[]string{manifestPath, "--output-dir", outputDir, "--json"},
		&stdout,
		&stderr,
		lookup,
		fake,
	)
	if rc != 1 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", rc, stdout.String(), stderr.String())
	}
	if len(calls) != 2 {
		t.Fatalf("run calls = %d", len(calls))
	}
	if got := optionValue(t, calls[0], "--client"); got != "codex" {
		t.Fatalf("first deterministic client = %q", got)
	}
	if got := optionValue(t, calls[1], "--client"); got != "cursor" {
		t.Fatalf("second deterministic client = %q", got)
	}

	var index suite.ResultIndex
	if err := json.Unmarshal(stdout.Bytes(), &index); err != nil {
		t.Fatalf("decode suite JSON: %v\n%s", err, stdout.String())
	}
	if len(index.Runs) != 2 || index.Runs[0].Outcome != suite.OutcomePass || index.Runs[1].Outcome != suite.OutcomeNonPass {
		t.Fatalf("unexpected suite outcomes: %#v", index.Runs)
	}
	for _, data := range []string{stdout.String(), readFileString(t, filepath.Join(outputDir, "index.json"))} {
		if strings.Contains(data, "very-secret-path") || strings.Contains(data, "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A") {
			t.Fatalf("suite index output leaked endpoint material: %s", data)
		}
	}
	for _, entry := range index.Runs {
		if entry.Artifact == "" {
			t.Fatalf("run missing artifact reference: %#v", entry)
		}
		artifactData := readFileString(t, filepath.Join(outputDir, filepath.FromSlash(entry.Artifact)))
		if strings.Contains(artifactData, "very-secret-path") {
			t.Fatalf("artifact leaked protected path: %s", artifactData)
		}
	}
}

func TestRunSuiteRunResolvesAllEndpointsBeforeLaunchingClients(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "suite.json")
	outputDir := filepath.Join(root, "results")
	manifest := `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [
    {
      "id": "alpha",
      "endpoint": {"source": "environment", "variable": "MCP_INTEROP_SUITE_ENDPOINT_ALPHA"},
      "deployment_id": "alpha",
      "clients": [{"id": "codex", "auth": "none"}]
    },
    {
      "id": "beta",
      "endpoint": {"source": "environment", "variable": "MCP_INTEROP_SUITE_ENDPOINT_BETA"},
      "deployment_id": "beta",
      "clients": [{"id": "cursor", "auth": "none"}]
    }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	launched := 0
	lookup := func(name string) (string, bool) {
		if name == "MCP_INTEROP_SUITE_ENDPOINT_ALPHA" {
			return "https://example.com/mcp/secret-alpha", true
		}
		return "", false
	}
	var stdout, stderr bytes.Buffer
	rc := runSuiteRunWith(context.Background(), []string{manifestPath, "--output-dir", outputDir}, &stdout, &stderr, lookup,
		func(context.Context, []string, io.Writer, io.Writer) int {
			launched++
			return 0
		})
	if rc != 2 || launched != 0 {
		t.Fatalf("exit=%d launched=%d stderr=%s", rc, launched, stderr.String())
	}
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Fatalf("output directory should not exist after preflight failure: %v", err)
	}
	if strings.Contains(stderr.String(), "secret-alpha") {
		t.Fatalf("preflight error leaked resolved endpoint: %s", stderr.String())
	}
}

func TestRunSuiteRunRefusesExistingOutputDirectoryBeforeLaunch(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "suite.json")
	outputDir := filepath.Join(root, "results")
	if err := os.Mkdir(outputDir, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [{
    "id": "production-a",
    "endpoint": {"source": "environment", "variable": "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"},
    "deployment_id": "production-a",
    "clients": [{"id": "codex", "auth": "none"}]
  }]
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	launched := 0
	var stdout, stderr bytes.Buffer
	rc := runSuiteRunWith(context.Background(), []string{manifestPath, "--output-dir", outputDir}, &stdout, &stderr,
		func(string) (string, bool) { return "https://example.com/mcp/secret", true },
		func(context.Context, []string, io.Writer, io.Writer) int {
			launched++
			return 0
		})
	if rc != 2 || launched != 0 {
		t.Fatalf("exit=%d launched=%d stderr=%s", rc, launched, stderr.String())
	}
}

func TestWriteLiveTestErrorHidesProtectedPathFailureDetail(t *testing.T) {
	var output bytes.Buffer
	writeLiveTestError(&output, "Codex", io.ErrUnexpectedEOF, true)
	if strings.Contains(output.String(), "unexpected EOF") {
		t.Fatalf("protected-path error detail leaked: %s", output.String())
	}
	if !strings.Contains(output.String(), "protected-path execution failed") {
		t.Fatalf("missing protected-path failure marker: %s", output.String())
	}
}

func optionValue(t *testing.T, args []string, name string) string {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	t.Fatalf("option %s missing from %#v", name, args)
	return ""
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
