# Architecture

[English](architecture.md) | [日本語](architecture.ja.md)

`mcp-interop` is a black-box interoperability runner for Remote MCP deployments. The project deliberately separates protocol-level observation from real-client execution so it does not become another MCP conformance suite.

## Core model

Each test produces stage results using four states:

- `pass` — the stage completed successfully and was observed.
- `fail` — the stage was attempted and failed.
- `skip` — the stage was not attempted because a prerequisite failed or the adapter does not support it.
- `unknown` — the available client interface cannot prove the outcome.

The current stages are:

1. `reach` — the real client reached enough of the Remote MCP deployment to prove live interaction.
2. `auth` — required client authentication completed, or live tool discovery proved authentication was not required.
3. `init` — the real client established an MCP session.
4. `tools` — the real client discovered the server's tools.

A complete interoperability pass requires all four stages to be `pass`. Inconclusive states intentionally remain non-zero so CI cannot silently treat missing evidence as compatibility.

## Adapter boundary

Every supported client has an adapter that owns client-specific behavior. The conceptual lifecycle is:

```text
Detect -> Prepare isolated profile -> Register endpoint -> Authenticate -> Discover -> Cleanup
```

Adapters report observations from the real installed client rather than infer success from configuration alone.

## Isolation policy

Live adapters must not silently alter the user's normal MCP configuration.

Preferred order:

1. Use an official temporary/profile/config override supported by the client.
2. If the client resolves configuration relative to the home directory, run it with an isolated temporary home when that behavior is supported and verified.
3. If safe isolation is not available, mark the adapter or stage `unknown`/`skip` rather than mutating the user's configuration.

Credentials and OAuth tokens created during a test must stay inside the isolated profile where possible. Reports must never include bearer tokens, authorization codes, client secrets, cookies, or raw credential files.

Process ownership follows the same rule: an adapter may reap only processes it can prove belong to the current isolated test session. It must not kill unrelated client processes by executable name.

## Shipped adapters

### Codex CLI

The Codex adapter is the most complete implementation in v0.1.0. It uses an isolated `CODEX_HOME`, the real `codex app-server` MCP status surface, and an explicit opt-in OAuth flow. OAuth credentials are forced into file storage inside the temporary home rather than the normal keyring path.

### Cursor CLI (beta)

The Cursor adapter uses an isolated temporary `HOME` and workspace plus the real CLI MCP management commands (`mcp enable`, `mcp list`, and `mcp list-tools`). It supports live no-auth interoperability testing without model prompts. OAuth completion remains planned for v0.2 after authenticated token exchange and tool discovery are fully verified.

### Antigravity CLI (beta, macOS)

The Antigravity adapter uses an isolated temporary `HOME`, the current `~/.gemini/config/mcp_config.json` format, and a no-input PTY startup. It observes machine-readable tool-cache state produced by the real client and reaps only descendants of the test PTY wrapper before session cleanup. Automated OAuth completion remains disabled until credential isolation from the macOS Keychain can be proven.

### VS Code (research)

VS Code can safely register MCP configuration in an isolated user-data directory, but the tested CLI does not expose a supported direct path for MCP server start/status/tool discovery. Registration alone is not treated as interoperability success. The adapter remains research-only until a stable no-model lifecycle surface exists.

### GitHub Copilot CLI (candidate)

GitHub Copilot CLI is a follow-up candidate for v0.3 if a stable automatable MCP inventory/lifecycle surface can provide the same black-box evidence without model prompts.

## Real-client E2E boundary

The repository includes a localhost-only MCP fixture and `scripts/e2e-real-clients.sh` for release-gate testing on a macOS machine with the real Codex, Cursor, and Antigravity clients installed.

The harness requires protocol evidence for:

```text
initialize
notifications/initialized
tools/list
```

and fails if `tools/call` occurs. It also checks user configuration metadata, the login Keychain database, leaked client processes, and temporary session directories before/after the run.

GitHub-hosted CI does not install external MCP clients. It validates adapter regression tests, fixture behavior, harness syntax/build paths, and release builds. A separate manual workflow targets a self-hosted macOS ARM64 runner for real-client E2E.

## What this project does not test

A successful interoperability result does not mean:

- the MCP server is secure;
- tool implementations are correct;
- destructive operations are safe;
- the model will choose the appropriate tool;
- every OAuth identity or scope combination will work.

Those concerns belong to separate security, conformance, or agent-evaluation tools.

For operational failure modes and result interpretation, see [Troubleshooting](troubleshooting.md).
