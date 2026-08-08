# mcp-interop

**Live interoperability testing for Remote MCP servers across real MCP clients.**

`mcp-interop` is an experimental, cross-client test runner for Remote Model Context Protocol (MCP) servers. It is designed to answer a practical question that protocol conformance alone cannot answer:

> Does this Remote MCP deployment actually connect, authenticate, initialize, and expose tools in the real clients my users run?

## Status

Early development.

The first live adapter is implemented for **Codex CLI**, including an explicit opt-in OAuth flow. The V1 roadmap also targets:

- Cursor CLI
- Antigravity CLI
- VS Code (beta adapter)

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

Run the current live Codex test:

```console
mcp-interop test https://example.com/mcp
mcp-interop test https://example.com/mcp --client codex
mcp-interop test https://example.com/mcp --client codex --json
```

If the server requires OAuth, opt in explicitly:

```console
mcp-interop test https://example.com/mcp --client codex --oauth
```

`--oauth` does not silently open a browser. `mcp-interop` prints the authorization URL to stderr and waits for the real Codex OAuth callback. Open that URL in a browser to continue. The URL contains short-lived OAuth state and should not be shared.

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

## Safety and isolation

- **Real clients, not emulators.** Client-specific checks invoke the installed client wherever practical.
- **No model benchmark required.** The core interoperability path does not ask a model to choose or call tools.
- **Do not mutate user configuration.** Live adapters must use isolated/temporary profiles or return `skip`/`unknown`.
- **Private temporary state.** Session directories are created with owner-only permissions where the OS supports them, and Codex test configuration/credential files use owner-only permissions on POSIX systems.
- **Credential redaction.** Bearer/OAuth material and credential-like Remote MCP URL query parameters are redacted from reports.
- **OAuth is explicit.** Authorization only starts when the caller opts in with `--oauth`.
- **No hosted service required.** The core tool runs locally and in CI without a project-operated backend.

## V1 roadmap

- [x] Shared `pass` / `fail` / `skip` / `unknown` result model
- [x] Isolated test-session lifecycle and secret redaction
- [x] Codex CLI live inventory adapter
- [x] Codex OAuth live flow
- [ ] Cursor CLI live adapter
- [ ] Establish a safe non-interactive Antigravity CLI automation boundary
- [ ] VS Code beta adapter
- [ ] Cross-client combined report

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
