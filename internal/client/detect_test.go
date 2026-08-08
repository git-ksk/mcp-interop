package client

import (
	"context"
	"errors"
	"reflect"
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
