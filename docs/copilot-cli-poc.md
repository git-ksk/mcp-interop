# GitHub Copilot CLI direct MCP inventory PoC

[English](copilot-cli-poc.md) | [日本語](copilot-cli-poc.ja.md)

This document records the research boundary for issue #48. It is a **PoC**, not a shipped adapter contract.

## Why this surface is interesting

GitHub's current Copilot CLI documentation exposes a supported non-interactive MCP management surface:

- `copilot mcp list [--json]`
- `copilot mcp get <name> [--json]`
- `copilot mcp add --transport http <name> <url>`

The CLI also supports isolated configuration/cache roots and explicit headless authentication inputs:

- `COPILOT_HOME`
- `COPILOT_CACHE_HOME`
- `--additional-mcp-config`
- `COPILOT_MCP_TOOL_CACHE=false`
- `COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, or `GITHUB_TOKEN` for non-interactive authentication

GitHub documents environment-variable authentication as the recommended method for CI/CD, containers, and other non-interactive environments. `COPILOT_GITHUB_TOKEN` has the highest precedence over stored credentials. Supported user-scoped token types include fine-grained PATs with the **Copilot Requests** account permission and supported OAuth/user tokens; classic `ghp_` PATs are not supported.

Official references:

- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
- https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/authenticate-copilot-cli
- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-programmatic-reference
- https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers

## Tested baselines

The original hosted macOS research used:

```text
GitHub Copilot CLI 1.0.79
macOS 26 arm64
Node.js 22
```

PR #69 repeated the decisive paths on 2026-08-23 with:

```text
GitHub Copilot CLI 1.0.80
GitHub-hosted macOS 15.7.7 arm64
```

Both runs used isolated temporary `COPILOT_HOME`, `COPILOT_CACHE_HOME`, and workspace state. Common GitHub/model token environment variables were removed. Ordinary outbound HTTP(S) was redirected to an unreachable loopback proxy while localhost remained available. MCP tool caching was disabled and no model prompt was used.

## Terminal MCP management result

`scripts/poc-copilot-cli-mcp.sh` exercises:

```console
copilot mcp list --json
copilot mcp get mcp-interop-fixture
copilot mcp get mcp-interop-fixture --json
```

The controlled configuration uses `deferTools: "never"` and the process sets `COPILOT_MCP_TOOL_CACHE=false`.

On both 1.0.79 and 1.0.80:

- all three terminal management commands exit successfully;
- text reports `Status: Enabled` and `Tools: * (all)`;
- JSON reports the configured `tools: ["*"]`, URL, source, and enabled state;
- the localhost fixture receives **no MCP requests**;
- no `initialize`, `notifications/initialized`, or `tools/list` is observed;
- the actual fixture tool `ping` is not returned by the management output.

Therefore the tested terminal `mcp list/get` path is **configuration/inventory management, not direct live MCP tool discovery**. Configuration registration or the configured `tools` allowlist must not be promoted to an interoperability PASS.

## No-input real-client startup result

`scripts/poc-copilot-cli-startup.sh` starts the real `copilot` TUI under a PTY with **no input and no model prompt**. It uses isolated state and a controlled localhost MCP target.

Both tested unauthenticated versions produced the same decisive wire boundary:

```text
initialize                  observed
notifications/initialized  observed
tools/list                  not observed
tools/call                  not observed
```

On 1.0.80, Copilot's debug log recorded the MCP service as initialized as a client and identified the fixture server's tools capability. The same isolated process reported `Login status unknown`, no available model backend, and no GitHub authentication token.

This proves a useful partial boundary: **real Copilot CLI startup can reach and initialize the configured Remote MCP server without a model prompt**, but the tested unauthenticated/no-model startup does not progress to observable tool discovery.

The most likely interpretation is that `tools/list` is gated behind a later authenticated/model-session lifecycle step. That is an inference from the observed wire trace and debug logs, not a documented Copilot contract.

## Copilot CLI 1.0.80 refresh — 2026-08-23

PR #69 installed the exact stable `@github/copilot` 1.0.80 package on GitHub-hosted macOS and re-ran both existing controlled PoCs.

The refresh confirmed rather than changed the boundary:

- terminal `mcp list/get` remains configuration-only and sends no MCP lifecycle traffic to the fixture;
- no-input startup reaches `initialize` and `notifications/initialized`;
- no-input startup still does **not** send `tools/list`;
- no `tools/call` occurs;
- the bounded PTY run leaves no Copilot process behind.

`copilot plugins list --kind mcp --json` was not used as evidence in this re-run. Configuration/resource inventory remains distinct from fixture-observed live discovery.

## Supported authenticated-isolation path

Current GitHub documentation resolves the earlier question of whether Copilot CLI has a supported headless authentication input: **it does**. For automated environments, a supported user-scoped token can be supplied through an environment variable, with `COPILOT_GITHUB_TOKEN` taking precedence over keychain and `gh` fallback credentials.

The startup PoC now has an explicit research-only authenticated mode. It is deliberately separate from ambient credentials:

```console
MCP_INTEROP_COPILOT_TEST_TOKEN='github_pat_...' \
MCP_INTEROP_COPILOT_ALLOW_NETWORK=1 \
bash scripts/poc-copilot-cli-startup.sh
```

Safety rules for this mode:

- the harness never consumes ambient `COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, or `GITHUB_TOKEN`; only `MCP_INTEROP_COPILOT_TEST_TOKEN` opts in;
- the dedicated token is injected into the child process as `COPILOT_GITHUB_TOKEN` and is never printed;
- authenticated mode requires the separate `MCP_INTEROP_COPILOT_ALLOW_NETWORK=1` opt-in because Copilot account/API access cannot be tested through the loopback-only network boundary;
- startup output and Copilot debug-log content are not printed in authenticated mode;
- authenticated temporary state cannot be retained with `MCP_INTEROP_KEEP_COPILOT_POC_TMP=1`;
- normal `~/.copilot` state and login-keychain file metadata/hash are compared before and after the run;
- `tools/call` remains forbidden; fixture-observed tool discovery is the only graduation signal.

Use a dedicated, revocable test credential with the minimum required Copilot permission. Do not reuse a normal day-to-day token merely to make the PoC pass. Repository/Actions installation tokens are not treated as a substitute for the documented user-scoped Copilot authentication types.

This harness capability is **not** evidence that authenticated `tools/list` succeeds. The adapter remains blocked until an actual controlled authenticated run satisfies the evidence and state-isolation gates.

## Current verdict

**BLOCKED for a complete direct `mcp-interop` adapter.**

The project requires direct evidence for the full core path, including tool discovery. The 1.0.79 and 1.0.80 tested unauthenticated baselines currently provide:

- terminal management: safe configuration visibility, but no live server contact;
- no-input real-client startup: live `initialize` + `notifications/initialized`, but no `tools/list` without an authenticated/model backend.

Do not ship a Copilot adapter that reports `tools=pass` from `Tools: *`, configuration state, or successful MCP initialization alone.

## Next safe gate

The remaining gate is now an **actual controlled authenticated run**, not discovery of an isolation mechanism.

The run must:

1. use a dedicated supported user-scoped Copilot test token through the explicit harness opt-in;
2. retain temporary config/cache/workspace isolation and the no-model core path;
3. require fixture-observed `tools/list`;
4. continue to reject `tools/call`;
5. prove normal user configuration/credential state remains unchanged;
6. prove all owned processes are cleaned up.

If that run cannot produce `tools/list` without introducing a model prompt or weakening credential isolation, keep issue #48 research-only.

## PASS contract for a future adapter

A future PoC may graduate only when all of the following are true:

1. the real installed Copilot CLI/version is recorded;
2. config/cache/workspace and credential state are safely isolated;
3. no model prompt is used as the core evidence mechanism;
4. the fixture observes `initialize` or the relevant protocol-era readiness evidence;
5. the fixture observes the corresponding initialized/readiness progression where applicable;
6. the fixture observes `tools/list` or equivalent direct real-client tool-discovery evidence;
7. client-observable inventory identifies the controlled `ping` tool where a supported surface exists;
8. the fixture does **not** observe `tools/call`;
9. normal user configuration/credential state is unchanged;
10. no owned client process is leaked.

## OAuth

Copilot account authentication and Remote-MCP OAuth are separate concerns and must not be conflated. The documented Remote-MCP `client_credentials` option authenticates the client to an MCP server; it does not by itself solve the Copilot account/model-backend boundary.

## Running

Unauthenticated startup PoC:

```console
bash scripts/poc-copilot-cli-startup.sh
```

Explicit authenticated research PoC:

```console
MCP_INTEROP_COPILOT_TEST_TOKEN='github_pat_...' \
MCP_INTEROP_COPILOT_ALLOW_NETWORK=1 \
bash scripts/poc-copilot-cli-startup.sh
```

Set `MCP_INTEROP_KEEP_COPILOT_POC_TMP=1` only for unauthenticated sanitized diagnostics. Authenticated mode rejects retained temporary state.
