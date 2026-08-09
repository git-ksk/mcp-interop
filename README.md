# mcp-interop

[![CI](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml/badge.svg)](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/git-ksk/mcp-interop)](https://github.com/git-ksk/mcp-interop/releases/latest)
[![License](https://img.shields.io/github/license/git-ksk/mcp-interop)](LICENSE)

[English](README.md) | [日本語](README.ja.md)

**Live interoperability testing for Remote MCP servers across real MCP clients.**

`mcp-interop` is an experimental, cross-client test runner for Remote Model Context Protocol (MCP) servers. It is designed to answer a practical question that protocol conformance alone cannot answer:

> Does this Remote MCP deployment actually connect, authenticate, initialize, and expose tools in the real clients my users run?

It also includes profile-based **preflight diagnostics** for client surfaces that do not yet expose a safe headless real-client automation boundary. Preflight results are deliberately kept separate from live interoperability PASS results.

## Status

**v0.1.0 is released.** This is the first public version of `mcp-interop`.

Release: [v0.1.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.1.0)

Live adapters currently exist for:

- **Codex CLI** — live inventory and explicit opt-in OAuth flow.
- **Cursor CLI (beta)** — live no-auth inventory via dedicated MCP management commands; OAuth completion is still pending maintainer E2E verification.
- **Antigravity CLI (beta, macOS)** — live no-auth inventory through an isolated no-prompt PTY startup and machine-readable MCP tool cache; automated OAuth completion is intentionally disabled.

The development branch also includes a **ChatGPT OAuth/server preflight profile**. It validates published MCP/OAuth metadata against ChatGPT's documented authentication behavior without claiming that the real ChatGPT client ran.

VS Code remains research-only until a stable no-model server-start/tool-discovery surface is available.

GitHub Copilot CLI is a follow-up candidate. Claude Code support is intentionally deferred.

## Install

With Go 1.24 or newer, install the current stable release explicitly:

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@v0.1.0
```

To track the newest published module version instead:

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@latest
```

Check the installed build:

```console
mcp-interop version
# or
mcp-interop --version
```

The [v0.1.0 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.1.0) also provides checksummed archives for macOS, Linux, and Windows on both amd64 and arm64.

## What a test proves

A complete client test has four observable stages:

1. `reach` — the real client reached enough of the Remote MCP deployment to prove live interaction.
2. `auth` — required client authentication completed, or live tool discovery proved that client authentication was not required.
3. `init` — MCP initialization completed.
4. `tools` — the client discovered the server's tools.

A test exits with code `0` only when **all four stages are `pass`**. `fail`, `skip`, and `unknown` are non-zero results because CI should not silently accept an inconclusive interoperability test.

`mcp-interop` does **not** claim that a server is secure, that every tool behaves correctly, or that an AI model will choose the right tool.

A `diagnose` command has a different contract: it produces `PREFLIGHT PASS` / `PREFLIGHT FAIL` from published server/client metadata and never substitutes that result for a real-client interoperability PASS.

## Current CLI

Detect known clients on the local machine:

```console
mcp-interop clients
mcp-interop clients --json
```

Run one live adapter:

```console
mcp-interop test https://example.com/mcp --client codex
mcp-interop test https://example.com/mcp --client cursor
mcp-interop test https://example.com/mcp --client antigravity
```

Run multiple implemented adapters sequentially:

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity
```

When more than one client is selected, the text output starts with a cross-client summary and then prints the detailed result for each client:

```text
SUMMARY
CLIENT           REACH  AUTH  INIT  TOOLS  VERSION
Codex CLI        PASS   PASS  PASS  PASS   codex-cli 0.133.0
Cursor CLI       PASS   PASS  PASS  PASS   2026.08.04-aaa8809
Antigravity CLI  PASS   PASS  PASS  PASS   1.1.11
```

JSON output remains an array, preserving the existing machine-readable contract.

If a Codex target requires OAuth, opt in explicitly:

```console
mcp-interop test https://example.com/mcp --client codex --oauth
```

`--oauth` does not silently open a browser. For Codex, `mcp-interop` prints the authorization URL to stderr and waits for the real Codex OAuth callback. Open that URL in a browser to continue. The URL contains short-lived OAuth state and should not be shared.

Cursor and Antigravity OAuth completion are not enabled yet. Their beta adapters return incomplete/inconclusive authentication results rather than starting an unverified credential flow.

### ChatGPT OAuth/server preflight

Check whether a Remote MCP server's published OAuth metadata has a known blocking mismatch with ChatGPT's documented MCP authentication path:

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

The profile follows Protected Resource Metadata and authorization-server discovery, then checks CIMD/DCR registration strategy, token endpoint authentication methods, PKCE S256, and refresh-token related metadata.

A server with `client_id_metadata_document_supported: true` can pass registration preflight without a `registration_endpoint`: ChatGPT can use CIMD and does not require DCR for that path.

If sanitized authorization-server logs expose the exact non-secret ChatGPT `client_id` CIMD URL and `redirect_uri`, they can be checked too:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

That extended check validates the CIMD document, redirect URI, client/server token auth method intersection, and JWKS when `private_key_jwt` is available.

This command does **not** operate the ChatGPT UI, complete OAuth, or claim a real ChatGPT client PASS. See [ChatGPT connection diagnostics](docs/chatgpt-diagnostics.md) ([日本語](docs/chatgpt-diagnostics.ja.md)).

### Secret-free ChatGPT Runtime Evidence

When sanitized Authorization Server logs can observe token-request **presence/match signals rather than values**, `diagnose` can correlate those observations as a separate Runtime Evidence layer:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --runtime-evidence runtime-evidence.json
```

Minimal evidence:

```json
{
  "client_id": "https://chatgpt.com/oauth/.../client.json",
  "resource_matches": true,
  "client_assertion_present": false
}
```

`code_verifier_present` and `client_assertion_type_present` are optional booleans. Missing observations remain `WARN / unknown`; they are never inferred. Unknown JSON fields are rejected, so tokens, authorization codes, PKCE verifier values, raw client assertions, cookies, and credentials are not accepted.

Preflight, Runtime Evidence, and real-client interoperability remain separate evidence layers. A server can therefore show `PREFLIGHT PASS` and Runtime Evidence `FAIL` with `TOKEN_AUTH_METHOD_MISMATCH`.

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

## Antigravity adapter (beta)

The Antigravity adapter currently ships a live implementation for macOS only:

1. creates an isolated temporary `HOME` and workspace;
2. writes the target Remote MCP endpoint to the temporary `~/.gemini/config/mcp_config.json` using the current `serverUrl` field;
3. starts the installed `agy` process under a PTY without sending TUI input or a model prompt;
4. observes machine-readable tool schema state under the isolated `~/.gemini/antigravity-cli/mcp/<server>/` cache;
5. treats valid tool schema files as evidence that the real client reached the server, initialized MCP, and completed tool discovery;
6. captures and reaps only descendants of the test PTY wrapper before shared session cleanup, then removes the temporary HOME/workspace.

### Current Antigravity limitations

- The no-prompt PTY/tool-cache path has been maintainer-validated on macOS and is deliberately `skip` on other operating systems until equivalent real-client evidence exists.
- OAuth-required discovery and DCR have been observed in the maintainer fixture, but automated authorization/token exchange is disabled because credential storage has not yet been proven to remain fully isolated from the macOS Keychain.
- If no isolated tool cache appears, the adapter returns `unknown` rather than treating configuration discovery as interoperability success.
- The tool cache is an observed Antigravity client surface rather than a stable cross-vendor protocol API, so version information must remain part of every result.

## Safety and isolation

- **Real clients, not emulators.** Client-specific checks invoke the installed client wherever practical.
- **No model benchmark required.** The core interoperability path does not ask a model to choose or call tools.
- **Do not mutate user configuration.** Live adapters must use isolated/temporary profiles or return `skip`/`unknown`.
- **Private temporary state.** Session directories are created with owner-only permissions where the OS supports them, and test configuration/credential files use owner-only permissions on POSIX systems where applicable.
- **Credential redaction.** Bearer/OAuth material and credential-like Remote MCP URL query parameters are redacted from reports.
- **OAuth is explicit.** Authorization only starts when the caller opts in and the selected adapter has a verified isolated OAuth implementation.
- **Preflight is not live evidence.** Profile diagnostics may fetch public metadata, but they never promote server metadata compatibility into a real-client `reach/auth/init/tools` PASS.
- **No hosted service required.** The core tool runs locally and in CI without a project-operated backend.

## Real-client E2E on macOS

The repository includes a deterministic localhost MCP fixture plus a release-gate runner for the installed real clients:

```console
bash scripts/e2e-real-clients.sh
```

By default it tests Codex, Cursor, and Antigravity. A subset can be selected explicitly:

```console
MCP_INTEROP_CLIENTS=codex,cursor bash scripts/e2e-real-clients.sh
```

The harness:

- builds and tests the current checkout before E2E;
- starts a Go fixture bound only to `127.0.0.1` with one deterministic `ping` tool;
- runs each selected real client against a distinct fixture path;
- requires the fixture to observe `initialize`, `notifications/initialized`, and `tools/list` for every selected client;
- fails if any `tools/call` occurs;
- removes common model/API key environment variables and points normal outbound HTTP(S) proxy variables at an unreachable loopback port while exempting localhost;
- compares curated user MCP/config/credential metadata before and after the run, including the login Keychain database by default;
- detects newly leaked `codex`, `cursor-agent`, or `agy` processes without killing processes by name;
- detects newly leaked `mcp-interop-*` temporary session directories.

The network controls are defense in depth, not a packet-capture proof that a client implementation cannot bypass proxy settings. The core adapter paths used by this harness do not submit model prompts.

For a shared development Mac where unrelated Keychain writes would make the database hash noisy, the Keychain check can be explicitly skipped:

```console
MCP_INTEROP_SKIP_KEYCHAIN=1 bash scripts/e2e-real-clients.sh
```

Do not skip that gate for a release-candidate validation unless the equivalent Keychain comparison is performed separately.

A manual GitHub Actions workflow is also included at `.github/workflows/e2e-real-macos.yml`. It intentionally targets a self-hosted runner labeled:

```text
self-hosted, macOS, ARM64, mcp-interop-e2e
```

The runner is expected to have the real Codex, Cursor, and Antigravity CLIs installed. GitHub-hosted CI does **not** install those external clients; normal CI validates the fixture, the harness syntax/build path, adapter regression tests, and release build path without making external client availability a pull-request dependency.

## Documentation

- [Architecture](docs/architecture.md) ([日本語](docs/architecture.ja.md))
- [Troubleshooting](docs/troubleshooting.md) ([日本語](docs/troubleshooting.ja.md))
- [Reason codes](docs/reason-codes.md) ([日本語](docs/reason-codes.ja.md))
- [ChatGPT connection diagnostics](docs/chatgpt-diagnostics.md) ([日本語](docs/chatgpt-diagnostics.ja.md))
- [Contributing](CONTRIBUTING.md) ([日本語](CONTRIBUTING.ja.md))
- [Security Policy](SECURITY.md) ([日本語](SECURITY.ja.md))
- [CHANGELOG](CHANGELOG.md)

## Release process

Release archives are built by `scripts/build-release.sh`. A `v*` tag triggers the release workflow, which validates the tag, embeds version/commit/build-time metadata, builds six platform/architecture archives, generates `checksums.txt`, verifies the Linux artifact's embedded version, and publishes the files to GitHub Releases.

Normal pull requests smoke-test the same release build path on Ubuntu before a tag is ever created.

Published release history is summarized in [CHANGELOG.md](CHANGELOG.md).

## Roadmap

### v0.2 — authentication completeness

- [x] Add structured OAuth reason codes for client-observed DCR failures.
- [x] Add ChatGPT-oriented Protected Resource Metadata / CIMD / DCR / PKCE / token-auth preflight diagnostics while keeping them separate from live-client verdicts.
- [ ] Correlate additional real-client OAuth failures with the profile diagnostic evidence.
- [ ] Complete Cursor OAuth token exchange + authenticated tool discovery.
- [ ] Establish a safe Antigravity OAuth completion boundary before enabling authorization/token exchange.
- [ ] Add sanitized verbose traces for remaining `unknown` / incomplete results.

### v0.3 — client coverage

- [ ] Research a supported headless ChatGPT MCP/app-management surface before any real ChatGPT adapter; do not use brittle DOM scraping as the compatibility contract.
- [ ] Revisit VS Code when a supported direct lifecycle/tool-discovery surface exists.
- [ ] Evaluate GitHub Copilot CLI when a stable automatable MCP inventory surface is available.
- [ ] Add additional OS/client-version evidence for beta adapters where the real client supports it.

### Shipped in v0.1.0

- [x] Shared `pass` / `fail` / `skip` / `unknown` result model.
- [x] Isolated test-session lifecycle and secret redaction.
- [x] Codex CLI live inventory adapter.
- [x] Codex OAuth live flow.
- [x] Cursor CLI no-auth live adapter (beta).
- [x] Antigravity CLI no-auth live adapter (beta, macOS).
- [x] Cross-client combined text report.
- [x] Repeatable real-client macOS E2E harness.
- [x] Versioned release build/release automation.

## Current non-goals

- MCP security scanning
- Tool quality or LLM-selection benchmarking
- Runtime sandboxing
- Permission/capability governance
- A new OAuth or MCP conformance specification
- Guaranteeing compatibility without running the corresponding client

## Contributing and security

Contributions are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md) or [CONTRIBUTING.ja.md](CONTRIBUTING.ja.md) for the adapter and test requirements.

Please report suspected vulnerabilities privately according to [SECURITY.md](SECURITY.md). A Japanese reference translation is available at [SECURITY.ja.md](SECURITY.ja.md); the English policy is canonical.

## License

Licensed under the Apache License, Version 2.0. See `LICENSE`.
