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

## Startup hardening follow-up (2026-09-08)

On `fix/antigravity-stability`, the installed `1.1.25` passed the non-OAuth
four-stage path on macOS 26.5 arm64 using `MCP_INTEROP_CLIENTS=antigravity bash
scripts/e2e-real-clients.sh`. The fixture observed `server/discover` followed by
legacy `initialize`, `notifications/initialized`, and `tools/list`. User state,
login Keychain DB, process cleanup, session cleanup, and no `tools/call` gates
passed. This is branch evidence, not evidence of a published release.

Startup now copies only three boolean onboarding flags into the isolated HOME
when the normal CLI state explicitly says onboarding is complete. Missing or
incomplete state is not treated as completion; invalid, oversized, or symlinked
state is rejected. No account/token state is copied. The PTY starts at 40 rows
by 120 columns. Regression tests exercise these settings at the child-process
boundary and cover invalid onboarding input. Fresh-user onboarding and OAuth
remain outside this additional stable evidence.
