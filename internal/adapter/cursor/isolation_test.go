package cursor

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestIsolatedCursorEnvPinsConfigRootsAndDropsAccountCredential(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "tmp", "cursor-test-home")
	input := []string{
		"HOME=/real/home",
		"USERPROFILE=C:\\Users\\real",
		"CURSOR_CONFIG_DIR=/real/cursor-config",
		"XDG_CONFIG_HOME=/real/xdg-config",
		"XDG_CACHE_HOME=/real/xdg-cache",
		"XDG_DATA_HOME=/real/xdg-data",
		"XDG_STATE_HOME=/real/xdg-state",
		"APPDATA=C:\\Users\\real\\AppData\\Roaming",
		"LOCALAPPDATA=C:\\Users\\real\\AppData\\Local",
		"CURSOR_API_KEY=secret-value",
		"KEEP_ME=yes",
	}

	got := isolatedCursorEnv(input, home)
	want := map[string]string{
		"HOME":              home,
		"USERPROFILE":       home,
		"CURSOR_CONFIG_DIR": filepath.Join(home, "cursor-config"),
		"XDG_CONFIG_HOME":   filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":    filepath.Join(home, ".cache"),
		"XDG_DATA_HOME":     filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME":    filepath.Join(home, ".local", "state"),
		"APPDATA":           filepath.Join(home, "AppData", "Roaming"),
		"LOCALAPPDATA":      filepath.Join(home, "AppData", "Local"),
		"KEEP_ME":           "yes",
	}
	for key, value := range want {
		if actual, ok := envValue(got, key); !ok || actual != value {
			t.Fatalf("%s = %q, %v; want %q", key, actual, ok, value)
		}
	}
	if value, ok := envValue(got, "CURSOR_API_KEY"); ok {
		t.Fatalf("CURSOR_API_KEY leaked into isolated environment: %q", value)
	}
}

func TestIsolatedCursorEnvRemovesCaseVariantOverrides(t *testing.T) {
	got := isolatedCursorEnv([]string{
		"cursor_config_dir=/ambient",
		"Cursor_Api_Key=secret",
	}, "/isolated")
	if countEnvKey(got, "CURSOR_CONFIG_DIR") != 1 {
		t.Fatalf("expected exactly one isolated CURSOR_CONFIG_DIR: %#v", got)
	}
	if _, ok := envValue(got, "CURSOR_API_KEY"); ok {
		t.Fatalf("case-variant Cursor API key survived: %#v", got)
	}
}

func envValue(env []string, key string) (string, bool) {
	for _, item := range env {
		name, value, found := strings.Cut(item, "=")
		if found && strings.EqualFold(name, key) {
			return value, true
		}
	}
	return "", false
}

func countEnvKey(env []string, key string) int {
	count := 0
	for _, item := range env {
		name, _, found := strings.Cut(item, "=")
		if found && strings.EqualFold(name, key) {
			count++
		}
	}
	return count
}
