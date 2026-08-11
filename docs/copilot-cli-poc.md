# GitHub Copilot CLI direct MCP inventory PoC

This document records the research boundary for issue #48. It is a **PoC**, not a shipped adapter contract.

## Why this surface is interesting

GitHub's current Copilot CLI documentation exposes a supported non-interactive MCP management surface:

- `copilot mcp list [--json]`
- `copilot mcp get <name> [--json]`
- `copilot mcp add --transport http <name> <url>`

The command reference describes `copilot mcp` as usable from the command line without starting an interactive session, and `mcp get` as showing a server's configuration and tools.

The CLI also supports:

- `COPILOT_HOME` to replace the normal `~/.copilot` configuration/state directory;
- `COPILOT_CACHE_HOME` to separately redirect the cache directory;
- `--additional-mcp-config` for session-only MCP definitions;
- `COPILOT_MCP_TOOL_CACHE=false` to disable MCP tool snapshot caching.

Official references:

- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference
- https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers

## Tested baseline

Hosted macOS research used the official npm package:

```text
GitHub Copilot CLI 1.0.79
macOS 26 arm64
Node.js 22
```

The CLI was run with an isolated temporary `COPILOT_HOME`, `COPILOT_CACHE_HOME`, and workspace. Common GitHub/model token environment variables were removed. Ordinary outbound HTTP(S) was redirected to an unreachable loopback proxy while localhost remained available.

## Terminal MCP management result

`scripts/poc-copilot-cli-mcp.sh` exercises:

```console
copilot mcp list --json
copilot mcp get mcp-interop-fixture
copilot mcp get mcp-interop-fixture --json
```

The controlled configuration uses `deferTools: "never"` and the process sets `COPILOT_MCP_TOOL_CACHE=false`.

Observed on Copilot CLI 1.0.79:

- all three terminal management commands exit successfully;
- text reports `Status: Enabled` and `Tools: * (all)`;
- JSON reports the configured `tools: ["*"]`, URL, source, and enabled state;
- the localhost fixture receives **no MCP requests**;
- no `initialize`, `notifications/initialized`, or `tools/list` is observed;
- the actual fixture tool `ping` is not returned by the management output.

Therefore the tested terminal `mcp list/get` path is **configuration/inventory management, not direct live MCP tool discovery**. Configuration registration or the configured `tools` allowlist must not be promoted to an interoperability PASS.

## No-input real-client startup result

`scripts/poc-copilot-cli-startup.sh` starts the real `copilot` TUI under a PTY with **no input and no model prompt**. It uses the same isolated state and loopback-only MCP target.

A 30-second hosted-macOS observation showed:

```text
initialize                  observed
notifications/initialized  observed
tools/list                  not observed
tools/call                  not observed
```

Copilot's debug log also records that the MCP service initialized as a client and recognized the fixture server's `tools` capability. At the same time, no model backend/account authentication was available in the isolated environment (`Login status unknown`; no GitHub auth token was available).

This proves a useful partial boundary: **real Copilot CLI startup can reach and initialize the configured Remote MCP server without a model prompt**, but the tested unauthenticated/no-model startup does not progress to observable tool discovery.

The most likely interpretation is that `tools/list` is gated behind a later authenticated/model-session lifecycle step. That is an inference from the observed wire trace and debug logs, not a documented Copilot contract.

## Current verdict

**BLOCKED for a complete direct `mcp-interop` adapter.**

The project requires direct evidence for the full core path, including tool discovery. Copilot CLI 1.0.79 currently provides:

- terminal management: safe configuration visibility, but no live server contact;
- no-input real-client startup: live `initialize` + `notifications/initialized`, but no `tools/list` without an authenticated/model backend.

Do not ship a Copilot adapter that reports `tools=pass` from `Tools: *`, configuration state, or successful MCP initialization alone.

## Next safe gate

The remaining research question is whether an **isolated authenticated Copilot session** can reach `tools/list` without copying or mutating the user's normal credential state and without requiring a model prompt as the interoperability oracle.

Do not copy normal user tokens, Keychain entries, or `~/.copilot` credential state into the PoC. If Copilot does not expose a supported isolated authentication mechanism suitable for this test, keep issue #48 research-only.

## PASS contract for a future adapter

A future PoC may graduate only when all of the following are true:

1. the real installed Copilot CLI/version is recorded;
2. config/cache/workspace and credential state are safely isolated;
3. no model prompt is used as the core evidence mechanism;
4. the fixture observes `initialize`;
5. the fixture observes `notifications/initialized`;
6. the fixture observes `tools/list`;
7. client-observable inventory identifies the controlled `ping` tool where a supported surface exists;
8. the fixture does **not** observe `tools/call`;
9. normal user configuration/credential state is unchanged;
10. no owned client process is leaked.

## OAuth

Remote-MCP OAuth is out of scope until the no-auth/tool-discovery boundary is solved. Copilot account authentication and Remote-MCP OAuth are separate concerns and must not be conflated.

## Running

Terminal management PoC:

```console
bash scripts/poc-copilot-cli-mcp.sh
```

No-input startup PoC:

```console
bash scripts/poc-copilot-cli-startup.sh
```

Set `MCP_INTEROP_KEEP_COPILOT_POC_TMP=1` only when sanitized diagnostics need to be retained. Kept directories may contain client logs, so review them before sharing.
