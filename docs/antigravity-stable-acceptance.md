# Antigravity stable acceptance

Antigravity CLI is promoted to **stable** only for the explicitly evidenced **macOS arm64 non-OAuth core path**.

Current-main real-client evidence used for promotion:

- `1.1.22`: `reach/auth/init/tools=PASS`
- `1.1.24`: `reach/auth/init/tools=PASS`
- runner: macOS 26.5 (25F71), arm64
- real client boundary: isolated PTY plus bounded observed live MCP tool-cache surface
- protocol evidence: `server/discover`, then fallback `initialize`, `notifications/initialized`, `tools/list`
- safety gates: user config unchanged, login Keychain DB unchanged, no new client processes, no leaked `mcp-interop` sessions, no `tools/call`

Stable gate assessment:

- `repeat_path_version_coverage`: **met** — the same PASS-contributing non-OAuth path passed on two exact client versions.
- `advertised_platform_coverage`: **met** — stable scope is explicitly narrowed to macOS arm64.
- `measurement_surface_stability`: **met** — the bounded observed tool-cache surface produced the same PASS boundary across repeated exact versions.
- all other stable criteria remain **met**.

This does **not** claim OAuth stability, non-macOS support, successful modern `server/discover` tool discovery, or a semantic-version range. Unobserved versions remain `untested` until measured.
