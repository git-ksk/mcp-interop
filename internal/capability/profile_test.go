package capability

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestNewProfileSortsAndValidatesDistinctCapabilityStates(t *testing.T) {
	profile, err := NewProfile(testContext(), []Observation{
		{CapabilityID: "tasks", State: StateUntested, EvidenceKind: EvidenceNone},
		{CapabilityID: "resources", State: StatePass, EvidenceKind: EvidenceClientProtocol, EvidenceID: "resources.list.response"},
		{CapabilityID: "prompts", State: StateUnknown, EvidenceKind: EvidenceClientControlSurface, EvidenceID: "prompts.inventory.ambiguous"},
		{CapabilityID: "mrtr", State: StateUnsupported, EvidenceKind: EvidenceAdapterPolicy, EvidenceID: "adapter.no_mrtr_boundary"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(profile.Capabilities))
	for _, observation := range profile.Capabilities {
		got = append(got, observation.CapabilityID+":"+string(observation.State))
	}
	want := "mrtr:unsupported,prompts:unknown,resources:pass,tasks:untested"
	if strings.Join(got, ",") != want {
		t.Fatalf("capabilities=%v want %s", got, want)
	}
}

func TestCapabilityPassFailUnknownRequireDirectClientEvidence(t *testing.T) {
	for _, state := range []State{StatePass, StateFail, StateUnknown} {
		for _, direct := range []EvidenceKind{EvidenceClientProtocol, EvidenceClientControlSurface, EvidenceClientObservedState} {
			profile := testProfile(Observation{CapabilityID: "resources", State: state, EvidenceKind: direct, EvidenceID: "direct.resources"})
			if err := Validate(profile); err != nil {
				t.Fatalf("state=%s kind=%s rejected: %v", state, direct, err)
			}
		}
		for _, indirect := range []EvidenceKind{EvidenceAdapterPolicy, EvidenceNone, EvidenceKind("server_metadata"), EvidenceKind("client_config"), EvidenceKind("ui_presence")} {
			profile := testProfile(Observation{CapabilityID: "resources", State: state, EvidenceKind: indirect, EvidenceID: "indirect.resources"})
			if err := Validate(profile); err == nil {
				t.Fatalf("state=%s accepted indirect kind=%s", state, indirect)
			}
		}
	}
}

func TestCapabilityUnsupportedAndUntestedHaveDifferentEvidenceRules(t *testing.T) {
	unsupported := testProfile(Observation{
		CapabilityID: "tasks", State: StateUnsupported,
		EvidenceKind: EvidenceAdapterPolicy, EvidenceID: "adapter.tasks.unsupported",
	})
	if err := Validate(unsupported); err != nil {
		t.Fatalf("unsupported policy boundary rejected: %v", err)
	}

	untested := testProfile(Observation{CapabilityID: "tasks", State: StateUntested, EvidenceKind: EvidenceNone})
	if err := Validate(untested); err != nil {
		t.Fatalf("untested state rejected: %v", err)
	}

	wrongUnsupported := testProfile(Observation{
		CapabilityID: "tasks", State: StateUnsupported,
		EvidenceKind: EvidenceClientProtocol, EvidenceID: "tasks.failure",
	})
	if err := Validate(wrongUnsupported); err == nil {
		t.Fatal("unsupported accepted direct execution evidence instead of adapter policy")
	}
	wrongUntested := testProfile(Observation{
		CapabilityID: "tasks", State: StateUntested,
		EvidenceKind: EvidenceNone, EvidenceID: "pretend.evidence",
	})
	if err := Validate(wrongUntested); err == nil {
		t.Fatal("untested accepted an evidence id")
	}
}

func TestCapabilityContextRequiresExactRealClientIdentity(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Context)
	}{
		{name: "missing version", mutate: func(c *Context) { c.Client.Version = "" }},
		{name: "runner provenance", mutate: func(c *Context) {
			c.EvidenceProvenance = artifact.EvidenceProvenance{Kind: artifact.ProvenanceRunnerObservation}
		}},
		{name: "adapter mismatch", mutate: func(c *Context) { c.EvidenceProvenance.AdapterID = "cursor" }},
		{name: "bad deployment id", mutate: func(c *Context) { c.DeploymentID = "secret/path" }},
		{name: "bad fingerprint", mutate: func(c *Context) { c.DeploymentFingerprint = "sha256:nope" }},
		{name: "bad auth", mutate: func(c *Context) { c.AuthMode = "ambient" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			context := testContext()
			tc.mutate(&context)
			profile := testProfileWithContext(context, Observation{
				CapabilityID: "resources", State: StatePass,
				EvidenceKind: EvidenceClientProtocol, EvidenceID: "resources.list.response",
			})
			if err := Validate(profile); err == nil {
				t.Fatalf("invalid context accepted: %#v", context)
			}
		})
	}
}

func TestCapabilityProfileRejectsDuplicatesAndNonDeterministicOrder(t *testing.T) {
	profile := testProfile(
		Observation{CapabilityID: "resources", State: StateUntested, EvidenceKind: EvidenceNone},
		Observation{CapabilityID: "resources", State: StateUntested, EvidenceKind: EvidenceNone},
	)
	if err := Validate(profile); err == nil {
		t.Fatal("duplicate capability id accepted")
	}
	profile = testProfile(
		Observation{CapabilityID: "tasks", State: StateUntested, EvidenceKind: EvidenceNone},
		Observation{CapabilityID: "resources", State: StateUntested, EvidenceKind: EvidenceNone},
	)
	if err := Validate(profile); err == nil {
		t.Fatal("non-deterministic capability order accepted")
	}
}

func TestCapabilityReadFileRejectsUnknownSecretBearingFields(t *testing.T) {
	profile, err := NewProfile(testContext(), []Observation{{
		CapabilityID: "resources", State: StatePass,
		EvidenceKind: EvidenceClientProtocol, EvidenceID: "resources.list.response",
	}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["endpoint"] = "https://example.com/mcp/secret-path"
	context := raw["context"].(map[string]any)
	context["token"] = "must-not-be-accepted"
	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(path); err == nil {
		t.Fatal("unknown endpoint/token fields were accepted")
	}
}

func TestCapabilityWriteReadRoundTripIsSecretSafe(t *testing.T) {
	profile, err := NewProfile(testContext(), []Observation{{
		CapabilityID: "resources", State: StatePass,
		EvidenceKind: EvidenceClientProtocol, EvidenceID: "resources.list.response",
	}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := WriteFile(path, profile); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Context.Client.Version != profile.Context.Client.Version || len(got.Capabilities) != 1 {
		t.Fatalf("round trip mismatch: %#v", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"https://", "/mcp/", "token", "password", "authorization"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("profile contains forbidden secret-bearing surface %q: %s", forbidden, data)
		}
	}
}

func TestCapabilityReadFileBoundsInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxProfileBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(path); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected bounded input error, got %v", err)
	}
}

func testProfile(observations ...Observation) Profile {
	return testProfileWithContext(testContext(), observations...)
}

func testProfileWithContext(context Context, observations ...Observation) Profile {
	return Profile{
		SchemaVersion: SchemaVersion,
		ArtifactType:  ArtifactType,
		Context:       context,
		Capabilities:  observations,
	}
}

func testContext() Context {
	return Context{
		ObservedAt:            time.Date(2026, 8, 28, 11, 0, 0, 0, time.UTC),
		DeploymentID:          "fixture-a",
		DeploymentFingerprint: "sha256:" + strings.Repeat("a", 64),
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
}

func TestContextFromLiveRunV2CopiesValidatedSecretSafeIdentity(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "codex-cli 0.133.0", "https://example.com/mcp/opaque-secret?token=hidden")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "test")
	}
	run, err := artifact.NewRunV2ProtectedPath(
		result,
		"https://example.com/mcp/opaque-secret?token=hidden",
		"production-a",
		time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
		"default",
		artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"},
		"dev",
		"deadbeef",
	)
	if err != nil {
		t.Fatal(err)
	}
	context, err := ContextFromLiveRunV2(run)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(context)
	if err != nil {
		t.Fatal(err)
	}
	if context.DeploymentID != "production-a" || context.DeploymentFingerprint != run.Endpoint.Fingerprint {
		t.Fatalf("context did not preserve v2 identity: %#v", context)
	}
	if strings.Contains(string(data), "opaque-secret") || strings.Contains(string(data), "token=hidden") || strings.Contains(string(data), "https://") {
		t.Fatalf("context leaked endpoint material: %s", data)
	}
}

func TestContextFromLiveRunV2RejectsV1Run(t *testing.T) {
	result := interop.NewResult("codex", "Codex CLI", "codex-cli 0.133.0", "https://example.com/mcp")
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "test")
	}
	run, err := artifact.NewRun(
		result,
		time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
		"default",
		artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"},
		"dev",
		"deadbeef",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ContextFromLiveRunV2(run); err == nil {
		t.Fatal("v1 run was treated as protected-path v2 capability context")
	}
}

func TestCapabilityContextRejectsControlCharactersAndOversizedDisplayFields(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Context)
	}{
		{name: "product newline", mutate: func(c *Context) { c.Client.Product = "Codex\nInjected" }},
		{name: "version escape", mutate: func(c *Context) { c.Client.Version = "1.0\x1b[31m" }},
		{name: "runtime tab", mutate: func(c *Context) { c.Runtime.MCPInteropVersion = "dev\tbad" }},
		{name: "oversized version", mutate: func(c *Context) { c.Client.Version = strings.Repeat("x", maxDisplayFieldBytes+1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := testContext()
			tc.mutate(&context)
			profile := testProfileWithContext(context, Observation{
				CapabilityID: "resources", State: StateUntested, EvidenceKind: EvidenceNone,
			})
			if err := Validate(profile); err == nil {
				t.Fatal("unsafe display field accepted")
			}
		})
	}
}
