# mcp-interop

**Live interoperability testing for Remote MCP servers across real MCP clients.**

`mcp-interop` is an experimental, cross-client test runner for Remote Model Context Protocol (MCP) servers. It is designed to answer a practical question that protocol conformance alone cannot answer:

> Does this Remote MCP deployment actually connect, authenticate, initialize, and expose tools in the real clients my users run?

## Status

Early development.

Live adapters currently exist for:

- **Codex CLI** — live inventory and explicit opt-in OAuth flow.
- **Cursor CLI (beta)** — live no-auth inventory via dedicated MCP management commands; OAuth completion is still pending maintainer E2E verification.

The V1 roadmap also targets an Antigravity CLI beta adapter. VS Code remains research-only until a stable no-model server-start/tool-discovery surface is available.

GitHub Copilot CLI is a follow-up candidate. Claude Code support is intentionally deferred.

## What a test proves

A complete client test has four observable stages:

1. `reach` — the real client reached enough of the Remote MCP deployment to prove live interaction.
2. `auth` — required client authentication completed, or live tool discovery proved that client authentication was not required.
3. `init` — MCP initialization completed.
4. `tools` — the client discovered the server's tools.

A test exits with code `0` only when **all four stages are `pass`**. `fail`, `skip`, and `unknown` are non-zero results because CI should not silently accept an inconclusive interoperability test.

`mcp-interop` does **not** claim that a server is secure, that every tool behaves correctly, or that an AI model will choose the right tool.

## Current CLI

Detect known clients on the local machine:

```console
mcp-interop clients
mcp-interop clients --json
```

Run a live Codex test:

```console
mcp-interop test https://example.com/mcp
mcp-interop test https://example.com/mcp --client codex
mcp-interop test https://example.com/mcp --client codex --json
```

Run the Cursor beta adapter against a no-auth Remote MCP server:

```console
mcp-interop test https://example.com/mcp --client cursor
```

Run multiple implemented adapters sequentially:

```console
mcp-interop test https://example.com/mcp --client codex,cursor
```

If a Codex target requires OAuth, opt in explicitly:

```console
mcp-interop test https://example.com/mcp --client codex --oauth
```

`--oauth` does not silently open a browser. For Codex, `mcp-interop` prints the authorization URL to stderr and waits for the real Codex OAuth callback. Open that URL in a browser to continue. The URL contains short-lived OAuth state and should not be shared.

Cursor OAuth completion is not enabled yet. The beta adapter detects an authentication-required boundary and returns an incomplete result instead of starting an unverified credential flow.

Example successful result:

```text
CLIENT    Codex CLI
VERSION   codex-cli 0.133.0
ENDPOINT  https://example.com/mcp

STAGE  STATUS  DETAIL
reach  PASS    Codex returned live MCP inventory
auth   PASS    Codex reports an OAuth-authenticated MCP session
init   PASS    tool discovery proves MCP initialization completed
tools  PASS    Codex discovered 3 MCP tool(s)
```

JSON output is an array from the start so additional real-client adapters can be added without changing the top-level output contract.

## Codex adapter

The Codex adapter:

1. creates an isolated temporary `CODEX_HOME`;
2. forces MCP OAuth credential storage to a file inside that isolated home;
3. writes only the test Remote MCP endpoint into the isolated config;
4. starts the real `codex app-server` process;
5. initializes the app-server control connection;
6. asks Codex for `mcpServerStatus/list` with tool inventory;
7. optionally runs Codex's own OAuth flow when `--oauth` is explicitly enabled;
8. reports what Codex itself observed;
9. removes the temporary session, including OAuth credentials.

It does **not** send a model prompt and does not require model/API usage for the live MCP inventory or OAuth test.

### OAuth isolation

The temporary Codex config sets:

```toml
mcp_oauth_credentials_store = "file"
```

This prevents the test from using the normal automatic/keyring MCP OAuth storage mode. During an OAuth test, Codex stores credentials under the temporary `CODEX_HOME`; the whole test session is deleted during cleanup.

### Current Codex limitations

- OAuth is interactive and explicit. Without `--oauth`, a server that reports `notLoggedIn` remains an incomplete test and exits non-zero.
- Current Codex app-server versions can expose an unreachable server and a legitimate zero-tool server in the same empty-inventory shape. `mcp-interop` therefore reports those stages as `unknown` instead of inventing a pass/fail result.
- The adapter relies on the installed Codex app-server MCP status and OAuth surfaces. Older or future Codex versions that do not expose the required methods may return an inconclusive/error result until the adapter is updated.

## Cursor adapter (beta)

The Cursor adapter:

1. creates an isolated temporary `HOME` and workspace;
2. writes only the target Remote MCP endpoint to `<workspace>/.cursor/mcp.json`;
3. invokes the installed Cursor CLI's MCP management surface;
4. attempts `mcp enable`, then queries `mcp list` and `mcp list-tools`;
5. treats a successful `mcp list-tools` as direct evidence from Cursor's real MCP client;
6. removes all temporary Cursor state during shared session cleanup.

The adapter never sends a Cursor model prompt. A fresh isolated HOME prevents the test from reusing normal Cursor MCP auth/config state.

### Current Cursor limitations

- Maintainer PoC has verified OAuth discovery, DCR, PKCE flow start, and the local callback listener, but token exchange plus authenticated `tools/list` has not yet been completed with the localhost fixture.
- Until that verification is complete, an OAuth-required target returns an incomplete auth result rather than invoking `mcp login` automatically.
- MCP management output is human-readable rather than a dedicated JSON contract, so the adapter keeps interpretation deliberately conservative.
- Initial live validation is macOS-specific; additional client-version/OS evidence should be added as the adapter matures.

## Safety and isolation

- **Real clients, not emulators.** Client-specific checks invoke the installed client wherever practical.
- **No model benchmark required.** The core interoperability path does not ask a model to choose or call tools.
- **Do not mutate user configuration.** Live adapters must use isolated/temporary profiles or return `skip`/`unknown`.
- **Private temporary state.** Session directories are created with owner-only permissions where the OS supports them, and test configuration/credential files use owner-only permissions on POSIX systems where applicable.
- **Credential redaction.** Bearer/OAuth material and credential-like Remote MCP URL query parameters are redacted from reports.
- **OAuth is explicit.** Authorization only starts when the caller opts in and the selected adapter has a verified isolated OAuth implementation.
- **No hosted service required.** The core tool runs locally and in CI without a project-operated backend.

## V1 roadmap

- [x] Shared `pass` / `fail` / `skip` / `unknown` result model
- [x] Isolated test-session lifecycle and secret redaction
- [x] Codex CLI live inventory adapter
- [x] Codex OAuth live flow
- [x] Cursor CLI no-auth live adapter (beta)
- [ ] Cursor OAuth completion + authenticated tool discovery
- [ ] Antigravity CLI no-auth live adapter (beta)
- [ ] Safe Antigravity OAuth completion boundary
- [ ] Cross-client combined report
- [ ] Revisit VS Code when a supported direct lifecycle/tool-discovery surface exists

## Non-goals for V1

- MCP security scanning
- Tool quality or LLM-selection benchmarking
- Runtime sandboxing
- Permission/capability governance
- A new OAuth or MCP conformance specification
- Guaranteeing compatibility without running the corresponding client

## Contributing and security

Contributions are welcome; see `CONTRIBUTING.md` for the adapter and test requirements.

Please report suspected vulnerabilities privately according to `SECURITY.md` rather than opening a public issue with sensitive details.

## License

Licensed under the Apache License, Version 2.0. See `LICENSE`.
