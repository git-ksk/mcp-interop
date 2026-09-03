# Cursor stable acceptance

Cursor CLI is promoted to **stable** only for the explicitly evidenced **macOS arm64 non-OAuth core path**.

Current-main real-client evidence used for promotion:

- `2026.08.04-aaa8809`: `reach/auth/init/tools=PASS`
- `2026.09.02-c22c1a3`: `reach/auth/init/tools=PASS`
- runner: macOS 26.5 (25F71), arm64
- real client boundary: isolated Cursor CLI `mcp enable`, `mcp list`, `mcp list-tools`
- safety gates: user config unchanged, login Keychain DB unchanged, no new client processes, no leaked `mcp-interop` sessions, no `tools/call`

Historical retained core evidence at exact `2026.08.25-3e8eec8` is also recorded in PR #108.

Stable gate assessment:

- `repeat_path_version_coverage`: **met** — the same PASS-contributing core path passed on two exact current-main client versions.
- `advertised_platform_coverage`: **met** — stable scope is explicitly narrowed to macOS arm64.
- `measurement_surface_stability`: **met** — the supported Cursor MCP management surface is repeatedly evidenced across exact versions.
- all other stable criteria remain **met**.

This does **not** claim OAuth stability, Linux/Windows/macOS amd64 support, or a semantic-version range. Unobserved versions remain `untested` until measured.
