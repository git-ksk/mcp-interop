package antigravity

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteConfigUsesIsolatedGlobalConfig(t *testing.T) {
	home := t.TempDir()
	endpoint := "https://example.com/mcp?tenant=acme&value=%22quoted%22"
	if err := writeConfig(home, endpoint); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, testServerName) || !strings.Contains(text, `"serverUrl": "`+endpoint+`"`) {
		t.Fatalf("unexpected config: %s", text)
	}
	if strings.Contains(text, `"url":`) || strings.Contains(text, `"httpUrl":`) {
		t.Fatalf("legacy remote URL key present: %s", text)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("config permissions = %o, want 600", got)
		}
	}

	settingsPath := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	settings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(settings), `"modelProvider": "gemini"`) {
		t.Fatalf("isolated account-bypass setting missing: %s", settings)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("settings permissions = %o, want 600", got)
		}
	}
}

func TestReplaceEnvForHomeForcesNoAccountSessionMode(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/Users/example",
		"GEMINI_API_KEY=normal-user-secret",
		"GOOGLE_GEMINI_BASE_URL=https://private.example.invalid",
		"GOOGLE_API_KEY=legacy-secret",
		"GOOGLE_GENERATIVE_AI_API_KEY=legacy-generative-secret",
		"OTHER=value",
	}
	got := replaceEnv(env, "HOME", "/tmp/isolated-home")

	values := map[string][]string{}
	for _, item := range got {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		values[strings.ToUpper(key)] = append(values[strings.ToUpper(key)], value)
	}
	if home := values["HOME"]; len(home) != 1 || home[0] != "/tmp/isolated-home" {
		t.Fatalf("HOME values = %#v", home)
	}
	if keys := values["GEMINI_API_KEY"]; len(keys) != 1 || keys[0] != isolatedGeminiAPIKey {
		t.Fatalf("GEMINI_API_KEY values = %#v", keys)
	}
	for _, key := range []string{"GOOGLE_GEMINI_BASE_URL", "GOOGLE_API_KEY", "GOOGLE_GENERATIVE_AI_API_KEY"} {
		if values[key] != nil {
			t.Fatalf("ambient %s unexpectedly reached isolated environment: %#v", key, values[key])
		}
	}
	for _, item := range got {
		for _, forbidden := range []string{"normal-user-secret", "private.example.invalid", "legacy-secret", "legacy-generative-secret"} {
			if strings.Contains(item, forbidden) {
				t.Fatalf("ambient Antigravity model state leaked into isolated environment: %q", item)
			}
		}
	}
}

func TestCountValidToolCacheFiles(t *testing.T) {
	home := t.TempDir()
	root := toolCacheRoot(home)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ping.json"), []byte(`{"name":"ping"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.json"), []byte(`not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte(`ignored`), 0o600); err != nil {
		t.Fatal(err)
	}

	count, err := countValidToolCacheFiles(home)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("cache count = %d, want 1", count)
	}
}

func TestCountValidToolCacheFilesMissingRoot(t *testing.T) {
	count, err := countValidToolCacheFiles(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cache count = %d, want 0", count)
	}
}

func TestCopyCompletedOnboardingStateCopiesOnlyValidatedFlags(t *testing.T) {
	sourceHome := t.TempDir()
	isolatedHome := t.TempDir()
	sourceDir := filepath.Join(sourceHome, ".gemini", "antigravity-cli", "cache")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "onboarding.json")
	if err := os.WriteFile(source, []byte(`{"consumerOnboardingComplete":true,"enterpriseOnboardingComplete":false,"onboardingComplete":true,"unknownSecret":"must-not-copy"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyCompletedOnboardingState(sourceHome, isolatedHome); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(isolatedHome, ".gemini", "antigravity-cli", "cache", "onboarding.json")
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, `"onboardingComplete": true`) || strings.Contains(text, "unknownSecret") {
		t.Fatalf("unexpected sanitized onboarding state: %s", text)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(destination)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("onboarding permissions = %o, want 600", got)
		}
	}
}

func TestCopyCompletedOnboardingStateDoesNotSynthesizeCompletion(t *testing.T) {
	sourceHome := t.TempDir()
	isolatedHome := t.TempDir()
	sourceDir := filepath.Join(sourceHome, ".gemini", "antigravity-cli", "cache")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "onboarding.json"), []byte(`{"onboardingComplete":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyCompletedOnboardingState(sourceHome, isolatedHome); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(isolatedHome, ".gemini", "antigravity-cli", "cache", "onboarding.json")); !os.IsNotExist(err) {
		t.Fatalf("incomplete onboarding state was synthesized: %v", err)
	}
}

func TestCopyCompletedOnboardingStateRefusesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	sourceHome := t.TempDir()
	isolatedHome := t.TempDir()
	sourceDir := filepath.Join(sourceHome, ".gemini", "antigravity-cli", "cache")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(target, []byte(`{"onboardingComplete":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(sourceDir, "onboarding.json")); err != nil {
		t.Fatal(err)
	}
	if err := copyCompletedOnboardingState(sourceHome, isolatedHome); err == nil {
		t.Fatal("symlink onboarding state was accepted")
	}
}

func TestCopyCompletedOnboardingStateInputBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name      string
		content   string
		missing   bool
		wantError bool
	}{
		{name: "missing", missing: true},
		{name: "empty", wantError: true},
		{name: "malformed", content: `{`, wantError: true},
		{name: "wrong flag type", content: `{"onboardingComplete":"true"}`, wantError: true},
		{name: "oversize", content: strings.Repeat(" ", (16<<10)+1), wantError: true},
		{name: "null", content: `null`},
		{name: "no completion flag", content: `{"consumerOnboardingComplete":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourceHome, isolatedHome := t.TempDir(), t.TempDir()
			relative := filepath.Join(".gemini", "antigravity-cli", "cache", "onboarding.json")
			source := filepath.Join(sourceHome, relative)
			if !tc.missing {
				if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(source, []byte(tc.content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := copyCompletedOnboardingState(sourceHome, isolatedHome); (err != nil) != tc.wantError {
				t.Fatalf("error = %v, want error %v", err, tc.wantError)
			}
			if _, err := os.Stat(filepath.Join(isolatedHome, relative)); !os.IsNotExist(err) {
				t.Fatalf("unexpected destination state: %v", err)
			}
		})
	}
}
