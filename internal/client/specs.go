package client

// Specs returns the client set currently tracked by mcp-interop.
//
// Detection is intentionally broader than the V1 live-adapter set so users can
// see which clients are present before a corresponding live adapter lands.
func Specs() []Spec {
	return []Spec{
		{
			ID:          "codex",
			DisplayName: "Codex CLI",
			Tier:        TierV1,
			Executables: []string{"codex"},
			VersionArgs: []string{"--version"},
		},
		{
			ID:          "cursor",
			DisplayName: "Cursor CLI",
			Tier:        TierV1,
			Executables: []string{"cursor-agent", "agent"},
			VersionArgs: []string{"--version"},
		},
		{
			ID:          "antigravity",
			DisplayName: "Antigravity CLI",
			Tier:        TierV1,
			Executables: []string{"agy"},
			VersionArgs: []string{"--version"},
		},
		{
			ID:          "vscode",
			DisplayName: "VS Code",
			Tier:        TierBeta,
			Executables: []string{"code"},
			VersionArgs: []string{"--version"},
		},
		{
			ID:          "copilot",
			DisplayName: "GitHub Copilot CLI",
			Tier:        TierLater,
			Executables: []string{"copilot"},
			VersionArgs: []string{"--version"},
		},
	}
}
