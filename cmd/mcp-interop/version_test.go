package main

import (
	"strings"
	"testing"
)

func TestFormatVersionShortensCommit(t *testing.T) {
	got := formatVersion(versionInfo{
		Version:   "v0.1.0",
		Commit:    "0123456789abcdef",
		BuildDate: "2026-08-09T00:00:00Z",
	})
	want := "mcp-interop v0.1.0 (commit 0123456789ab, built 2026-08-09T00:00:00Z)"
	if got != want {
		t.Fatalf("formatVersion() = %q, want %q", got, want)
	}
}

func TestCurrentVersionInfoHasStableFallbacks(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate := version, commit, buildDate
	version, commit, buildDate = "", "", ""
	t.Cleanup(func() {
		version, commit, buildDate = oldVersion, oldCommit, oldBuildDate
	})

	got := currentVersionInfo()
	if strings.TrimSpace(got.Version) == "" || strings.TrimSpace(got.Commit) == "" || strings.TrimSpace(got.BuildDate) == "" {
		t.Fatalf("version info contains empty fields: %#v", got)
	}
}
