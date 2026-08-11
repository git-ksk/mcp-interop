# VS Code Agent Plugin MCP PoC

Status: experimental research for #6. This is not a release adapter yet.

## Goal

Test whether the real VS Code MCP client can reach a localhost Streamable HTTP MCP fixture and perform the direct lifecycle

```text
initialize -> notifications/initialized -> tools/list
```

without a model prompt, browser/DOM automation, or private workbench command IDs.

## Why this path is worth testing

Current VS Code Agent Plugin documentation exposes a supported local-plugin surface:

- `chat.pluginLocations` registers a local plugin directory;
- a plugin can include MCP server definitions in `.mcp.json`;
- plugin MCP servers start automatically when the plugin is enabled;
- plugin MCP servers are implicitly trusted and do not use the separate workspace-MCP trust prompt.

VS Code also documents `--user-data-dir` and `--extensions-dir` for isolated instances. Together these surfaces may provide the missing direct no-model lifecycle path for the no-auth portion of #6.

References:

- https://code.visualstudio.com/docs/agent-customization/agent-plugins
- https://code.visualstudio.com/docs/agent-customization/mcp-servers
- https://code.visualstudio.com/docs/configure/command-line

## PoC design

`scripts/poc-vscode-agent-plugin.sh`:

1. requires macOS and the installed `code` CLI;
2. builds the existing localhost `internal/e2e/fixture`;
3. creates a temporary VS Code `--user-data-dir`, temporary `--extensions-dir`, empty workspace, and local Agent Plugin;
4. enables Agent Plugins and registers the temporary plugin only inside the isolated profile;
5. points the plugin MCP server at the loopback fixture;
6. launches the real VS Code executable with common external-network proxy variables pointed at a closed loopback port while preserving loopback access;
7. treats server-side wire evidence as authoritative;
8. requires `initialize`, `notifications/initialized`, and `tools/list` on the dedicated fixture path;
9. fails if `tools/call` occurs;
10. stops only VS Code processes whose command line contains the unique temporary `--user-data-dir` path;
11. checks that the normal VS Code `settings.json` and user `mcp.json` metadata did not change.

Run locally:

```bash
bash scripts/poc-vscode-agent-plugin.sh
```

Keep diagnostics when needed:

```bash
MCP_INTEROP_KEEP_VSCODE_POC_TMP=1 bash scripts/poc-vscode-agent-plugin.sh
```

The branch also contains a temporary self-hosted macOS workflow so the same PoC can be exercised on the existing `mcp-interop-e2e` runner.

## PASS meaning

A PASS would prove, for the tested VS Code build and Agent Plugin feature state, that a supported public plugin configuration path can cause the real VS Code MCP client to perform direct no-auth initialization and tool discovery without model participation.

That is enough to move the no-auth portion of #6 out of the previous CLI-only BLOCKED state and justify implementing a real VS Code adapter around the same isolated-session pattern.

## What this does not prove

- OAuth completion or credential isolation;
- a stable client-side machine-readable MCP status API;
- compatibility when Agent Plugins are disabled by organization policy;
- stability of the Agent Plugin surface after Preview changes;
- that a workspace `.vscode/mcp.json` lifecycle can be driven headlessly;
- release readiness.

OAuth remains a separate gate. Do not infer authenticated interoperability from this PoC.

## Failure interpretation

No fixture traffic is not automatically a protocol failure. It can mean one of the following:

- `chat.plugins.enabled` is unavailable or disabled by policy;
- the installed VS Code build does not load Agent Plugins in the isolated profile;
- plugin MCP auto-start requires a product/session condition not present in the runner;
- the HTTP plugin MCP configuration shape changed;
- VS Code could not launch an isolated desktop instance from the execution context.

In those cases keep #6 open and capture the VS Code/fixture diagnostics before deciding whether a different supported surface exists.
