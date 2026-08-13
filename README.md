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

**v0.5.0 is the current published release.**

Release: [v0.5.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.5.0)

The live adapters in v0.5.0 are:

- **Codex CLI** — live inventory and explicit opt-in OAuth flow.
- **Cursor CLI (beta)** — live no-auth inventory plus explicit opt-in OAuth through the real Cursor MCP login path; authenticated `mcp list-tools` has been validated with the controlled fixture.
- **Antigravity CLI (beta, macOS)** — live no-auth inventory plus explicit opt-in OAuth through the real `/mcp` manager in an isolated PTY. Authentication can be proven independently of client-side tool-cache observation, so generic `init/tools` may conservatively remain `unknown` while controlled E2E proves the authenticated MCP exchange.

v0.5.0 adds **portable live-result artifact schema v1, `test --output`, artifact comparison, and `--fail-on-regression` CI gating** while preserving the existing `test --json` contract and real-client-only PASS boundary. v0.4.0 added Cursor OAuth completion, Antigravity OAuth completion on the tested macOS baseline, secret-free real-client OAuth capability enrichment, stricter deployment-specific live-evidence boundaries, and hardened release provenance gates.

Post-v0.5.0 work remains focused on quality/optimization rather than client expansion. The guarantees being hardened are:

- a live PASS still requires all four real-client stages to be `pass`;
- diagnostic metadata and Runtime Evidence remain separate from real-client PASS evidence;
- secret-bearing values are rejected or redacted before output;
- process cleanup is bounded and limited to temporary state or descendants owned by the current test session;
- exact client-version runs can be exported as secret-safe local artifacts and compared without weakening the existing live verdict;
- CI/release gates cover formatting, vet, unit tests, race tests, vulnerability scanning, fixture gates, shell syntax, and release archive smoke checks where practical.

VS Code remains research-only until its separate lifecycle/tool-discovery automation research is promoted into a stable live adapter.

GitHub Copilot CLI remains research-only: current testing proves real-client MCP initialization but has not yet proven `tools/list` under the project's no-model evidence contract ([#48](https://github.com/git-ksk/mcp-interop/issues/48)). Claude Code support is intentionally deferred.

ChatGPT real-client support remains intentionally blocked ([#20](https://github.com/git-ksk/mcp-interop/issues/20)) until an officially supported direct/headless ChatGPT MCP app-management surface exists. Model prompts, brittle DOM/UI automation, private endpoints, and normal-user browser credentials are not acceptable evidence for a real-client PASS.

## Install

With Go 1.24 or newer, install the current stable release explicitly:

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@v0.5.0
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

The [v0.5.0 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.5.0) provides checksummed archives for macOS, Linux, and Windows on both amd64 and arm64.

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

### Portable regression artifacts

Export the same live run into a separate versioned, secret-safe local artifact without changing stdout:

```console
mcp-interop test https://example.com/mcp --client codex --output result.json
```

The artifact records the exact detected client version, OS/architecture, runner/runtime context, invocation auth mode, evidence provenance, and the existing four stage status/reason results. The raw endpoint URL is not persisted; query values are excluded before deriving the endpoint fingerprint. Human stage messages and diagnostic payloads are also excluded from artifact v1.

Compare two artifacts across client versions or repeated runs:

```console
mcp-interop compare old.json new.json
mcp-interop compare old.json new.json --json
mcp-interop compare old.json new.json --fail-on-regression
```

The comparison explicitly reports `PASS_TO_FAIL`, `PASS_TO_UNKNOWN`, `PASS_TO_SKIP`, reason-code changes, and missing baseline evidence. A client-version change by itself is not a regression. `--fail-on-regression` exits `1` only when one of those regression/evidence-loss conditions is present; malformed or unsupported artifacts are usage/input errors and exit `2`.

See [Live interoperability result artifact schema v1](docs/live-result-schema-v1.md) ([日本語](docs/live-result-schema-v1.ja.md)) for the exact compatibility, secret-safety, pairing, and exit-code contract.

OAuth flows are always explicit opt-in:

```console
mcp-interop test https://example.com/mcp --client codex --oauth
mcp-interop test https://example.com/mcp --client cursor --oauth
mcp-interop test https://example.com/mcp --client antigravity --oauth
```

For Codex, `mcp-interop` prints the authorization URL to stderr and waits for the real Codex OAuth callback. The URL contains short-lived OAuth state and should not be shared.

Cursor uses the real Cursor MCP login path inside an isolated temporary HOME/workspace and proves authenticated discovery with `mcp list-tools`. Callback details are version-specific and are not hard-coded.

Antigravity enters the real `/mcp` manager inside an isolated PTY. OAuth token persistence is confined to the isolated temporary HOME; authorization codes and token contents are not persisted in `mcp-interop` evidence. See [Antigravity OAuth live-test boundary](docs/antigravity-oauth.md).

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

Minimal v3 evidence:

```json
{
  "schema_version": 3,
  "registration": {
    "strategy": "cimd",
    "client_metadata_url": "https://chatgpt.com/oauth/.../client.json"
  },
  "token_request": {
    "resource_matches": true,
    "client_assertion_present": false
  },
  "tool_metadata": {
    "oauth2_security_scheme_present": true
  },
  "tool_challenge": {
    "expected": false
  }
}
```

Additional authorization/token/resource/tool observations are optional. Schema v3 keeps `tool_metadata` and `tool_challenge` independent; schema v2 `tool_auth` and legacy v1 input remain accepted for compatibility. Missing observations remain `WARN / unknown`; they are never inferred. Unknown JSON fields are rejected, so tokens, authorization codes, PKCE verifier values, raw client assertions, cookies, and credentials are not accepted.

Preflight, Runtime Evidence, and real-client interoperability remain separate evidence layers. A server can therefore show `PREFLIGHT PASS` and Runtime Evidence `FAIL` with `TOKEN_AUTH_METHOD_MISMATCH`.

### Evidence utilities

Use the same strict secret-free decoder outside `diagnose` to validate, summarize, or combine evidence fragments:

```console
mcp-interop evidence validate runtime-evidence.json
mcp-interop evidence summary runtime-evidence.json
mcp-interop evidence merge authorization.json resource.json tool.json -o runtime-evidence.json
```

`summary` prints only structural coverage (section names and supplied-field counts), never observed values or metadata URLs. `merge` fails on conflicting observations instead of using last-write-wins and emits canonical schema v3 JSON. Unknown fields remain rejected, so these commands do not create a second path for ingesting tokens, authorization codes, raw client assertions, cookies, or other credentials.

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
5. when `--oauth` is explicit, invokes the real Cursor MCP login path inside the isolated session;
6. treats successful authenticated `mcp list-tools` as direct evidence from Cursor's real MCP client;
7. removes all temporary Cursor state during shared session cleanup.

The adapter never sends a Cursor model prompt. A fresh isolated HOME prevents the test from reusing normal Cursor MCP auth/config state.

### Current Cursor limitations

- OAuth is explicit opt-in; the adapter does not silently start login for an OAuth-required target without `--oauth`.
- Callback addresses are version-specific and must not be treated as a permanent fixed port.
- MCP management output is human-readable rather than a dedicated JSON contract, so the adapter keeps interpretation deliberately conservative.
- Real OAuth validation has been completed on macOS for the tested Cursor CLI version; additional client-version/OS evidence should be added as the adapter matures.

## Antigravity adapter (beta)

The Antigravity adapter currently has a live implementation for macOS only:

1. creates an isolated temporary `HOME` and workspace;
2. writes the target Remote MCP endpoint to the temporary `~/.gemini/config/mcp_config.json` using the current `serverUrl` field;
3. starts the installed `agy` process under a PTY without a model prompt;
4. in no-auth mode, observes machine-readable tool schema state under the isolated `~/.gemini/antigravity-cli/mcp/<server>/` cache;
5. when `--oauth` is explicit, enters the real Antigravity `/mcp` manager and forwards authorization-code input directly to the isolated PTY;
6. observes OAuth token persistence only through metadata for isolated `~/.gemini/antigravity/mcp_oauth_tokens.json`, never by opening the token file;
7. captures and reaps only descendants of the test PTY wrapper before shared session cleanup, then removes the temporary HOME/workspace.

### Current Antigravity limitations

- The live adapter remains macOS-only until equivalent real-client evidence exists on other operating systems.
- OAuth is explicit opt-in and still depends on the tested Antigravity interactive `/mcp` surface.
- On the tested `agy 1.1.11` OAuth path, authenticated `initialize` and `tools/list` can complete without materializing the same client-side tool cache used by no-auth mode. The generic result therefore keeps `init/tools=unknown` rather than inferring pass from authentication alone.
- The controlled localhost OAuth E2E independently requires authenticated `initialize`, `notifications/initialized`, and `tools/list` server-side evidence. See [Antigravity OAuth live-test boundary](docs/antigravity-oauth.md).
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

OAuth-specific Cursor and Antigravity harnesses additionally exercise their real OAuth paths against controlled loopback fixtures and keep authorization codes/tokens out of persisted evidence.

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
- [Live result artifact schema v1](docs/live-result-schema-v1.md) ([日本語](docs/live-result-schema-v1.ja.md))
- [Troubleshooting](docs/troubleshooting.md) ([日本語](docs/troubleshooting.ja.md))
- [Reason codes](docs/reason-codes.md) ([日本語](docs/reason-codes.ja.md))
- [ChatGPT connection diagnostics](docs/chatgpt-diagnostics.md) ([日本語](docs/chatgpt-diagnostics.ja.md))
- [Contributing](CONTRIBUTING.md) ([日本語](CONTRIBUTING.ja.md))
- [Security Policy](SECURITY.md) ([日本語](SECURITY.ja.md))
- [CHANGELOG](CHANGELOG.md)

## Release process

Release archives are built by `scripts/build-release.sh`. A `v*` tag triggers the release workflow, which validates the tag, runs source quality gates, embeds version/commit/build-time metadata, builds six platform/architecture archives, generates `checksums.txt`, verifies the Linux artifact's embedded version, and publishes the files to GitHub Releases.

Normal pull requests smoke-test the same release build path on Ubuntu before a tag is ever created. Cross-platform jobs keep the Go 1.24-compatible regular test/build path on Linux, macOS, and Windows; Ubuntu additionally runs the race detector, then switches to the pinned release/security Go toolchain for `govulncheck`. Tagged release artifacts are built with that pinned patched toolchain rather than the minimum module version.

Published release history is summarized in [CHANGELOG.md](CHANGELOG.md).

## Roadmap

### Shipped in v0.2.0

- [x] Structured OAuth reason codes for explicit real-client and Runtime Evidence failures.
- [x] ChatGPT OAuth/server preflight with PRM, CIMD/DCR, PKCE, and token-auth diagnostics.
- [x] Exact ChatGPT CIMD / redirect URI / JWKS validation from observed non-secret metadata.
- [x] Runtime Evidence v2 with `cimd` / `dcr` / `predefined` registration strategy and legacy v1 compatibility.
- [x] Token/resource request correlation, Bearer delivery, and resource-server signature/issuer/audience/expiry/scope diagnostics.
- [x] OpenAI authenticated MCP reference-pattern and tool-level OAuth signal diagnostics.
- [x] Conservative multiple-authorization-server handling.
- [x] Expanded English/Japanese diagnostics and troubleshooting documentation.

### Shipped in v0.3.0

- [x] Runtime Evidence v3 with independent `tool_metadata` / `tool_challenge` sections and v1/v2 input compatibility.
- [x] Explicit Runtime Evidence coverage counters and `N/A` semantics distinct from `WARN / unknown`.
- [x] Secret-free `evidence validate`, `summary`, and conflict-safe `merge` utilities with canonical v3 output.
- [x] Controlled insufficient-scope OAuth fixture and release gate for tool-level `securitySchemes` / `mcp/www_authenticate` behavior.
- [x] Partial tool OAuth aggregation that keeps unobserved static metadata as `WARN` rather than over-reporting `N/A`.
- [x] Versioned OpenAI reference profile metadata and a documented manual real-ChatGPT secret-free dogfood workflow.

### Shipped in v0.4.0

- [x] Correlate explicit real-client DCR failures with discovered CIMD/DCR server capability evidence while keeping the four-stage verdict separate ([#19](https://github.com/git-ksk/mcp-interop/issues/19)).
- [x] Complete Cursor explicit opt-in OAuth, token exchange, and authenticated `mcp list-tools` validation with isolated state ([#3](https://github.com/git-ksk/mcp-interop/issues/3)).
- [x] Complete Antigravity explicit opt-in OAuth with isolated token persistence, conservative generic stage semantics, and controlled authenticated wire-evidence E2E ([#5](https://github.com/git-ksk/mcp-interop/issues/5)).

### Open after v0.4.0

- [ ] Research a supported headless ChatGPT MCP/app-management surface before any real ChatGPT adapter; do not use brittle DOM scraping as the compatibility contract ([#20](https://github.com/git-ksk/mcp-interop/issues/20)).
- [ ] Revisit VS Code when a supported direct lifecycle/tool-discovery surface can satisfy the project's no-model evidence contract ([#6](https://github.com/git-ksk/mcp-interop/issues/6)).
- [ ] Complete GitHub Copilot CLI tool-discovery/auth-isolation research before any live adapter ([#48](https://github.com/git-ksk/mcp-interop/issues/48)).
- [ ] Evaluate additional real MCP clients when they expose stable automatable lifecycle/tool-discovery surfaces.

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
