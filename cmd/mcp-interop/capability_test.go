package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/capability"
)

func TestCapabilityValidateHumanDistinguishesAllStates(t *testing.T) {
	path := writeCLICapabilityProfile(t)
	var stdout, stderr bytes.Buffer
	if rc := runCapability([]string{"validate", path}, &stdout, &stderr); rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	for _, want := range []string{
		"VALID_CAPABILITY_PROFILE", "Codex CLI", "codex-cli 0.133.0",
		"resources", "pass", "prompts", "unknown", "mrtr", "unsupported", "tasks", "untested",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestCapabilityValidateJSONReemitsOnlyValidatedProfileWithoutInputPath(t *testing.T) {
	path := writeCLICapabilityProfile(t)
	var stdout, stderr bytes.Buffer
	if rc := runCapability([]string{"validate", "--json", path}, &stdout, &stderr); rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, stderr.String())
	}
	var profile capability.Profile
	if err := json.Unmarshal(stdout.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.ArtifactType != capability.ArtifactType || len(profile.Capabilities) != 4 {
		t.Fatalf("unexpected profile: %#v", profile)
	}
	if strings.Contains(stdout.String(), path) {
		t.Fatalf("JSON leaked input path: %s", stdout.String())
	}
	for _, forbidden := range []string{"https://", "/mcp/secret", "bearer", "authorization_code"} {
		if strings.Contains(strings.ToLower(stdout.String()), forbidden) {
			t.Fatalf("JSON contains secret-bearing surface %q: %s", forbidden, stdout.String())
		}
	}
}

func TestCapabilityValidateRejectsInvalidInputAndOptions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if rc := runCapability([]string{"validate"}, &stdout, &stderr); rc != 2 {
		t.Fatalf("missing path rc=%d stderr=%s", rc, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if rc := runCapability([]string{"validate", "a", "b"}, &stdout, &stderr); rc != 2 {
		t.Fatalf("multiple paths rc=%d stderr=%s", rc, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if rc := runCapability([]string{"validate", "file", "--json", "--json"}, &stdout, &stderr); rc != 2 {
		t.Fatalf("duplicate flag rc=%d stderr=%s", rc, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if rc := runCapability([]string{"other"}, &stdout, &stderr); rc != 2 {
		t.Fatalf("unknown subcommand rc=%d stderr=%s", rc, stderr.String())
	}
}

func writeCLICapabilityProfile(t *testing.T) string {
	t.Helper()
	context := capability.Context{
		ObservedAt:            time.Date(2026, 8, 28, 11, 30, 0, 0, time.UTC),
		DeploymentID:          "fixture-a",
		DeploymentFingerprint: "sha256:" + strings.Repeat("b", 64),
		Client: artifact.ClientIdentity{
			ID: "codex", Product: "Codex CLI", Version: "codex-cli 0.133.0",
		},
		Platform: artifact.Platform{OS: "darwin", Arch: "arm64"},
		Runtime: artifact.Runtime{
			MCPInteropVersion: "dev", MCPInteropCommit: "deadbeef", GoVersion: "go1.26.6",
		},
		AuthMode: "default",
		EvidenceProvenance: artifact.EvidenceProvenance{
			Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex",
		},
	}
	profile, err := capability.NewProfile(context, []capability.Observation{
		{CapabilityID: "resources", State: capability.StatePass, EvidenceKind: capability.EvidenceClientProtocol, EvidenceID: "resources.list.response"},
		{CapabilityID: "prompts", State: capability.StateUnknown, EvidenceKind: capability.EvidenceClientControlSurface, EvidenceID: "prompts.inventory.ambiguous"},
		{CapabilityID: "mrtr", State: capability.StateUnsupported, EvidenceKind: capability.EvidenceAdapterPolicy, EvidenceID: "adapter.no_mrtr_boundary"},
		{CapabilityID: "tasks", State: capability.StateUntested, EvidenceKind: capability.EvidenceNone},
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "capability-profile.json")
	if err := capability.WriteFile(path, profile); err != nil {
		t.Fatal(err)
	}
	return path
}
