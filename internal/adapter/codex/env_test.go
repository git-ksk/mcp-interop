package codex

import (
	"strings"
	"testing"
)

func TestReplaceEnvRemovesCaseVariantDuplicates(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"CODEX_HOME=/old-upper",
		"Codex_Home=/old-mixed",
		"codex_home=/old-lower",
		"OTHER=value=with=equals",
	}
	got := replaceEnv(env, "CODEX_HOME", "/isolated")

	var matches []string
	for _, item := range got {
		key, _, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(key, "CODEX_HOME") {
			matches = append(matches, item)
		}
	}
	if len(matches) != 1 || matches[0] != "CODEX_HOME=/isolated" {
		t.Fatalf("CODEX_HOME variants were not isolated: %#v", got)
	}
	if !containsEnvEntry(got, "OTHER=value=with=equals") {
		t.Fatalf("unrelated environment entry was modified: %#v", got)
	}
}

func containsEnvEntry(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}
