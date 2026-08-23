package cursor

import (
	"path/filepath"
	"strings"
)

// isolatedCursorEnv prevents ambient Cursor configuration/account state from
// bypassing the adapter's temporary HOME. Cursor documents explicit config
// overrides on Unix and USERPROFILE-based config on Windows, so all supported
// config roots are pinned inside the session-owned home.
func isolatedCursorEnv(env []string, home string) []string {
	assignments := [][2]string{
		{"HOME", home},
		{"USERPROFILE", home},
		{"CURSOR_CONFIG_DIR", filepath.Join(home, "cursor-config")},
		{"XDG_CONFIG_HOME", filepath.Join(home, ".config")},
		{"XDG_CACHE_HOME", filepath.Join(home, ".cache")},
		{"XDG_DATA_HOME", filepath.Join(home, ".local", "share")},
		{"XDG_STATE_HOME", filepath.Join(home, ".local", "state")},
		{"APPDATA", filepath.Join(home, "AppData", "Roaming")},
		{"LOCALAPPDATA", filepath.Join(home, "AppData", "Local")},
	}
	for _, assignment := range assignments {
		env = replaceEnvFold(env, assignment[0], assignment[1])
	}
	// Account/API authentication is unrelated to the direct MCP management
	// boundary and must not silently reuse a normal user's Cursor credential.
	return removeEnvFold(env, "CURSOR_API_KEY")
}

func replaceEnvFold(env []string, key, value string) []string {
	out := removeEnvFold(env, key)
	return append(out, key+"="+value)
}

func removeEnvFold(env []string, key string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		name, _, found := strings.Cut(item, "=")
		if found && strings.EqualFold(name, key) {
			continue
		}
		out = append(out, item)
	}
	return out
}
