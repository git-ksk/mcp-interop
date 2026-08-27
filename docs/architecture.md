# Architecture

[English](architecture.md) | [日本語](architecture.ja.md)

`mcp-interop` is a black-box interoperability runner for Remote MCP deployments. The project deliberately separates protocol conformance from real-client product interoperability so it does not become another MCP conformance suite.

## Relationship to MCP Conformance

`mcp-interop` is complementary to the official [MCP Conformance Test Framework](https://github.com/modelcontextprotocol/conformance), not a replacement for it.

The distinction is not “real software versus synthetic tests.” The official framework can launch a real client command and can test a real server URL. Its oracle is the MCP specification: scenario-controlled interactions are evaluated against expected protocol behavior.

`mcp-interop` instead evaluates a **specific deployed Remote MCP endpoint against specific released client products and versions** through those products' own MCP surfaces. Its core comparison axis is therefore:

```text
MCP Conformance: implementation x specification
mcp-interop:      deployment x client product x client version
```

A conformance PASS does not prove that every released client will interoperate with a particular deployment. An `mcp-interop` PASS does not prove full specification conformance. A useful release pipeline can run Conformance first, deploy the real endpoint, then run `mcp-interop` against the clients users actually use.

Product-specific `diagnose` profiles must preserve the same boundary: generic MCP/OAuth conformance belongs to the official suite; profile diagnostics may test compatibility with a named client product but must never present metadata compatibility as generic conformance or as a real-client interoperability PASS.

See [MCP Conformance vs. mcp-interop](conformance-vs-interop.md) for the detailed boundary and test topologies.

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

## Quality-phase invariants

Current development is focused on reliability, testability, reproducibility, and measured regression detection rather than adding clients. Quality work must preserve these invariants:

- diagnostic/preflight metadata can explain failures but cannot promote a live adapter result to PASS;
- Runtime Evidence can only use secret-free presence/match observations, and incomplete or ambiguous observations remain `WARN` / `unknown`;
- a process may be terminated only when it is owned by the current isolated test session or launched directly by the harness;
- fixed sleeps should be replaced with readiness, process-exit, or state-stability conditions where that makes the test less flaky;
- release gates should exercise the same source-quality invariants as normal CI where practical.

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

The current stable release is v0.5.1. The Cursor and Antigravity OAuth paths were introduced in v0.4.0.

### Codex CLI

The Codex adapter is currently the most complete implementation. It uses an isolated `CODEX_HOME`, the real `codex app-server` MCP status surface, and an explicit opt-in OAuth flow. OAuth credentials are forced into file storage inside the temporary home rather than the normal keyring path.

### Cursor CLI (beta)

The Cursor adapter uses an isolated temporary `HOME` and workspace plus the real CLI MCP management commands (`mcp enable`, `mcp list`, and `mcp list-tools`). It supports live no-auth interoperability testing without model prompts.

In v0.4.0, explicit `--oauth` invokes the real Cursor MCP login path inside the isolated session. The controlled OAuth fixture verifies DCR, Authorization Code + PKCE, token exchange, bearer-authenticated MCP, and authenticated `mcp list-tools`. A successful authenticated `mcp list-tools` directly proves `reach/auth/init/tools` for the tested Cursor CLI surface. Callback addresses remain version-specific and are not hard-coded.

### Antigravity CLI (beta, macOS)

The Antigravity adapter uses an isolated temporary `HOME`, the current `~/.gemini/config/mcp_config.json` format, and a PTY-based real-client path. Before launching `agy`, it also writes `modelProvider: "gemini"` into the isolated `~/.gemini/antigravity-cli/settings.json`, removes ambient Gemini API-key/base-URL overrides, and injects a fixed non-secret `GEMINI_API_KEY` sentinel. This selects Antigravity's documented Gemini API-key mode, which does not establish an Antigravity account session, so the adapter does not depend on a normal-user macOS Keychain session. The no-auth mode then observes machine-readable tool-cache state produced by the real client and reaps only descendants of the test PTY wrapper before session cleanup.

The login Keychain before/after comparison remains a non-mutation gate; by itself it does not prove that a client never read the Keychain. Credential non-reuse instead rests on the documented no-account startup mode above plus the real-client release gate. That gate was revalidated with `agy 1.1.22`, requiring `initialize`, `notifications/initialized`, and `tools/list` with no model prompt or `tools/call`, unchanged normal-user config/Keychain metadata, and no leaked client process/session.

In v0.4.0, explicit `--oauth` entered the real Antigravity `/mcp` manager inside the isolated PTY. Remote-MCP OAuth is separate from Antigravity account authentication: the client still starts in the no-account mode above, while MCP OAuth token persistence is confined to the isolated `~/.gemini/antigravity/mcp_oauth_tokens.json`. `mcp-interop` observes only file metadata and never opens or persists token contents. The generic result remains conservative: authentication can be proven while `init/tools` stay `unknown` if the OAuth path does not materialize the same client-side tool cache. The controlled localhost E2E separately requires authenticated `initialize`, `notifications/initialized`, and `tools/list` server-side evidence. See [Antigravity OAuth live-test boundary](antigravity-oauth.md).

### VS Code (research)

VS Code can safely register MCP configuration in an isolated user-data directory, but a stable supported direct lifecycle/tool-discovery automation boundary has not yet been promoted into a live adapter. Registration alone is not treated as interoperability success. Research continues separately from the shipped adapter contract.

### GitHub Copilot CLI (research)

GitHub Copilot CLI remains research-only. Current PoC evidence shows real-client `initialize` / `notifications/initialized` on no-input startup, but not `tools/list` without an authenticated/model backend; see #48.

### ChatGPT (blocked)

ChatGPT remains a diagnostics-only profile. A real-client adapter is blocked until an officially supported direct/headless ChatGPT MCP app-management surface is available. Model prompts, DOM/UI automation, private endpoints, and normal-user browser credentials are not acceptable substitutes for the project's real-client `reach/auth/init/tools` evidence contract; see #20.

## Real-client E2E boundary

The repository includes a localhost-only MCP fixture and `scripts/e2e-real-clients.sh` for release-gate testing on a macOS machine with the real Codex, Cursor, and Antigravity clients installed.

The harness requires protocol evidence for:

```text
initialize
notifications/initialized
tools/list
```

and fails if `tools/call` occurs. It also checks user configuration metadata, the login Keychain database, leaked client processes, and temporary session directories before/after the run.

OAuth-specific Cursor and Antigravity E2E harnesses use the same isolation principle but additionally exercise their real OAuth client paths against controlled loopback fixtures. Secret-bearing authorization codes and tokens are excluded from persisted evidence.

The fixture is an **adapter self-test and release gate**, not a general MCP conformance suite. Its job is to prove that the `mcp-interop` measurement path actually observes the real clients and preserves the project's isolation guarantees.

GitHub-hosted CI does not install external MCP clients. It validates adapter regression tests, fixture behavior, harness syntax/build paths, and release builds. A separate manual workflow targets a self-hosted macOS ARM64 runner for real-client E2E.

## What this project does not test

A successful interoperability result does not mean:

- the implementation is fully MCP-conformant;
- the MCP server is secure;
- tool implementations are correct;
- destructive operations are safe;
- the model will choose the appropriate tool;
- every OAuth identity or scope combination will work;
- untested client products or versions are compatible.

Those concerns belong to separate security, conformance, or agent-evaluation tools.

For operational failure modes and result interpretation, see [Troubleshooting](troubleshooting.md).
