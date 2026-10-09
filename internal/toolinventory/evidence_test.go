package toolinventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

const protectedURL = "https://example.test/private/access-token?key=secret-value"

func evidenceFixture(t *testing.T, version string, observed []string, expected []string, count *int) Evidence {
	t.Helper()
	result := interop.NewResult("codex", "Codex CLI", version, protectedURL)
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusPass, "real client")
	}
	if !result.SetObservedToolNames(observed) {
		t.Fatal("valid tool list rejected")
	}
	check := interop.CheckToolExpectation(result, expected, count)
	run, err := artifact.NewRunV2ProtectedPath(result, protectedURL, "dev", time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), "default", artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"}, "dev", "deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := New(run, check)
	if err != nil {
		t.Fatal(err)
	}
	return evidence
}

func intp(i int) *int { return &i }

func TestEvidencePrivateRoundTrip(t *testing.T) {
	e := evidenceFixture(t, "0.152.1", []string{"private_internal_tool", "ping", "read_tool"}, []string{"ping", "read_tool"}, intp(3))
	path := filepath.Join(t.TempDir(), "evidence.json")
	if err := WriteFile(path, e); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, e) {
		t.Fatalf("roundtrip mismatch: %#v != %#v", got, e)
	}
	data, _ := os.ReadFile(path)
	for _, forbidden := range []string{"private_internal_tool", "/private/access-token", "secret-value"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("unexpected observed name or secret disclosed: %s", forbidden)
		}
	}
	if !strings.Contains(string(data), "read_tool") {
		t.Fatal("explicitly expected name missing")
	}
	if err := WriteFile(path, e); err == nil {
		t.Fatal("overwriting existing evidence must fail")
	}
	if stat, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if runtime.GOOS != "windows" && stat.Mode().Perm() != 0o600 {
		t.Fatalf("non-private mode: %s", stat.Mode().Perm())
	}
}

func TestToolDriftDecisions(t *testing.T) {
	base := evidenceFixture(t, "0.152.1", []string{"ping", "read_tool"}, []string{"ping", "read_tool"}, nil)
	for _, tc := range []struct {
		name      string
		actual    []string
		version   string
		want      string
		missing   []string
		recovered []string
	}{
		{"clean", []string{"ping", "read_tool"}, "0.152.1", DecisionClean, nil, nil},
		{"count increased without name evidence", []string{"ping", "read_tool", "new_extra_tool"}, "0.152.1", DecisionDrift, nil, nil},
		{"count shrunk and missing", []string{"ping"}, "0.152.1", DecisionRegression, []string{"read_tool"}, nil},
		{"same count but expected name disappeared", []string{"ping", "new_extra_tool"}, "0.152.1", DecisionRegression, []string{"read_tool"}, nil},
		{"changed version but same inventory", []string{"ping", "read_tool"}, "0.153.0", DecisionClean, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := evidenceFixture(t, tc.version, tc.actual, []string{"ping", "read_tool"}, nil)
			d, err := Compare(base, current)
			if err != nil {
				t.Fatal(err)
			}
			if d.Decision != tc.want || !reflect.DeepEqual(d.NewlyMissing, tc.missing) || !reflect.DeepEqual(d.Recovered, tc.recovered) {
				t.Fatalf("unexpected diff: %#v", d)
			}
			if strings.Contains(stringify(d), "new_extra_tool") {
				t.Fatal("unexpected tool name leaked in diff")
			}
		})
	}
	failed := evidenceFixture(t, "0.152.1", []string{"ping"}, []string{"ping", "read_tool"}, nil)
	recovered, err := Compare(failed, base)
	if err != nil || recovered.Decision != DecisionRecovered || !reflect.DeepEqual(recovered.Recovered, []string{"read_tool"}) {
		t.Fatalf("recovery: %#v %v", recovered, err)
	}
	stuck, err := Compare(failed, failed)
	if err != nil || stuck.Decision != DecisionNonPass {
		t.Fatalf("consistent failure must not pass: %#v %v", stuck, err)
	}
}

func TestUnknownAndIncompatibleEvidenceFailClosed(t *testing.T) {
	base := evidenceFixture(t, "0.152.1", []string{"ping"}, []string{"ping"}, nil)
	unknown := base
	unknown.Expectation.Status = interop.StatusUnknown
	unknown.Expectation.ReasonCode = interop.ToolReasonUnobservable
	unknown.Expectation.ObservedCount = nil
	unknown.Expectation.MissingNames = nil
	d, err := Compare(base, unknown)
	if err != nil || d.Decision != DecisionUnknown {
		t.Fatalf("unknown promoted: %#v %v", d, err)
	}
	bad := base
	bad.Run.Endpoint.Identity = "other"
	if _, err := Compare(base, bad); err == nil {
		t.Fatal("different endpoint accepted")
	}
	bad = base
	bad.Expectation.ExpectedNames = []string{"other"}
	if _, err := Compare(base, bad); err == nil {
		t.Fatal("different expectation accepted")
	}
	bad = base
	bad.Run.Platform.OS = "intentionally-different-os"
	if _, err := Compare(base, bad); err == nil {
		t.Fatal("different platform accepted")
	}
	bad = base
	bad.Run.AuthMode = "oauth"
	if _, err := Compare(base, bad); err == nil {
		t.Fatal("different auth accepted")
	}
}

func TestEvidenceRejectsForgedOrUnknownFields(t *testing.T) {
	base := evidenceFixture(t, "0.152.1", []string{"ping"}, []string{"ping"}, nil)
	for _, tc := range []struct {
		name   string
		mutate func(*Evidence)
	}{
		{"unexpected names", func(v *Evidence) { v.Expectation.ExpectedNames = []string{"ping", "ping"} }},
		{"fake missing", func(v *Evidence) {
			v.Expectation.MissingNames = []string{"extra"}
			v.Expectation.Status = interop.StatusFail
			v.Expectation.ReasonCode = interop.ToolReasonMissing
		}},
		{"false fail", func(v *Evidence) { v.Expectation.Status = interop.StatusFail }},
		{"fake observed count", func(v *Evidence) { v.Expectation.ObservedCount = intp(0) }},
		{"fake provenance", func(v *Evidence) {
			v.Run.EvidenceProvenance.Kind = artifact.ProvenanceRunnerObservation
			v.Run.EvidenceProvenance.AdapterID = ""
		}},
		{"zero schema", func(v *Evidence) { v.SchemaVersion = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := base
			tc.mutate(&v)
			if err := Validate(v); err == nil {
				t.Fatalf("bad evidence accepted: %#v", v)
			}
		})
	}
	data, _ := json.Marshal(base)
	bad := strings.TrimSuffix(string(data), "}") + `,"unexpected":"secret"}`
	p := filepath.Join(t.TempDir(), "unknown.json")
	if err := os.WriteFile(p, []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(p); err == nil {
		t.Fatal("unknown JSON field allowed")
	}
	symlink := filepath.Join(t.TempDir(), "link.json")
	if err := os.Symlink(p, symlink); err != nil {
		t.Skip(err)
	}
	if _, err := ReadFile(symlink); err == nil {
		t.Fatal("symlink input accepted")
	}
}

func stringify(value any) string { b, _ := json.Marshal(value); return string(b) }
