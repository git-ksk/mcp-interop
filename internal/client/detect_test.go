package client

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestDetectFindsFirstAvailableExecutable(t *testing.T) {
	detector := newSystemDetector(
		func(name string) (string, error) {
			if name == "cursor-agent" {
				return "", errors.New("not found")
			}
			if name == "agent" {
				return "/usr/local/bin/agent", nil
			}
			return "", errors.New("unexpected executable")
		},
		func(_ context.Context, path string, args []string) (string, error) {
			if path != "/usr/local/bin/agent" {
				t.Fatalf("unexpected path: %s", path)
			}
			if !reflect.DeepEqual(args, []string{"--version"}) {
				t.Fatalf("unexpected args: %#v", args)
			}
			return "Cursor Agent 2026.08.01\nextra output\n", nil
		},
	)

	result := detector.Detect(context.Background(), Spec{
		ID:          "cursor",
		DisplayName: "Cursor CLI",
		Tier:        TierV1,
		Executables: []string{"cursor-agent", "agent"},
		VersionArgs: []string{"--version"},
	})

	if !result.Installed {
		t.Fatal("expected client to be installed")
	}
	if result.Executable != "agent" {
		t.Fatalf("unexpected executable: %s", result.Executable)
	}
	if result.Version != "Cursor Agent 2026.08.01" {
		t.Fatalf("unexpected version: %q", result.Version)
	}
}

func TestDetectMissingClient(t *testing.T) {
	detector := newSystemDetector(
		func(string) (string, error) { return "", errors.New("not found") },
		func(context.Context, string, []string) (string, error) {
			t.Fatal("version should not run for a missing executable")
			return "", nil
		},
	)

	result := detector.Detect(context.Background(), Spec{
		ID:          "codex",
		DisplayName: "Codex CLI",
		Tier:        TierV1,
		Executables: []string{"codex"},
		VersionArgs: []string{"--version"},
	})

	if result.Installed {
		t.Fatal("expected client to be missing")
	}
}

func TestDetectKeepsInstalledStateWhenVersionFails(t *testing.T) {
	detector := newSystemDetector(
		func(string) (string, error) { return "/usr/bin/codex", nil },
		func(context.Context, string, []string) (string, error) {
			return "", errors.New("version command failed")
		},
	)

	result := detector.Detect(context.Background(), Spec{
		ID:          "codex",
		DisplayName: "Codex CLI",
		Tier:        TierV1,
		Executables: []string{"codex"},
		VersionArgs: []string{"--version"},
	})

	if !result.Installed {
		t.Fatal("expected client to remain installed")
	}
	if result.Error == "" {
		t.Fatal("expected version error")
	}
}

func TestNormalizeVersion(t *testing.T) {
	got := normalizeVersion("\n  first line  \nsecond line\n")
	if got != "first line" {
		t.Fatalf("unexpected normalized version: %q", got)
	}
}

func TestInspectExecutableArchitecturesFindsCurrentTestBinaryArchitecture(t *testing.T) {
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	architectures, err := inspectExecutableArchitectures(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsArchitecture(architectures, runtime.GOARCH) {
		t.Fatalf("architectures=%v, want runner arch %q", architectures, runtime.GOARCH)
	}
}

func TestInspectExecutableArchitecturesLeavesScriptUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client-wrapper")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	architectures, err := inspectExecutableArchitectures(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(architectures) != 0 {
		t.Fatalf("script architecture was inferred: %v", architectures)
	}
}

func TestDetectionJSONDoesNotExposeExecutableArchitectureEvidence(t *testing.T) {
	data, err := json.Marshal(Detection{
		ID: "codex", DisplayName: "Codex CLI", Tier: TierV1, Installed: true,
		Executable: "codex", Path: "/private/client", Version: "1.0.0",
		ExecutableArchitectures: []string{"arm64"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" || containsJSONKey(data, "executable_architectures") {
		t.Fatalf("internal architecture evidence changed clients JSON contract: %s", data)
	}
}

func containsJSONKey(data []byte, key string) bool {
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return false
	}
	_, ok := value[key]
	return ok
}
