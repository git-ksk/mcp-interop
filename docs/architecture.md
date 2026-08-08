# Architecture

`mcp-interop` is a black-box interoperability runner for Remote MCP deployments. The project deliberately separates protocol-level observation from real-client execution so it does not become another MCP conformance suite.

## Core model

Each test produces stage results using four states:

- `pass` — the stage completed successfully and was observed.
- `fail` — the stage was attempted and failed.
- `skip` — the stage was not attempted because a prerequisite failed or the adapter does not support it.
- `unknown` — the available client interface cannot prove the outcome.

The initial stages are:

1. `reach` — the remote endpoint is reachable by the adapter.
2. `auth` — required client authentication completes or an existing client session is accepted.
3. `init` — the client establishes an MCP session.
4. `tools` — the client can discover the server's tools.

## Adapter boundary

Every supported client will have an adapter that owns client-specific behavior. The intended interface is conceptually:

```text
Detect -> Prepare isolated profile -> Register endpoint -> Authenticate -> Discover -> Cleanup
```

Adapters must report observations rather than infer success from configuration alone.

## Isolation policy

Live adapters must not silently alter the user's normal MCP configuration.

Preferred order:

1. Use an official temporary/profile/config override supported by the client.
2. If the client resolves configuration relative to the home directory, run it with an isolated temporary home when that behavior is supported and verified.
3. If safe isolation is not available, mark the adapter or stage `unknown`/`skip` rather than mutating the user's configuration.

Credentials and OAuth tokens created during a test must stay inside the isolated profile where possible. Reports must never include bearer tokens, authorization codes, client secrets, cookies, or raw credential files.

## Initial adapters

### Codex CLI — V1

Target observations:

- client present/version
- remote MCP registration in an isolated configuration
- MCP OAuth login when required
- server initialization/status
- tool discovery when exposed through a stable client management surface

### Cursor CLI — V1

Cursor exposes dedicated MCP management commands including login, list, and list-tools. The adapter should rely on those management commands instead of asking the model to call a tool.

### Antigravity CLI — V1

Antigravity supports local and remote MCP configuration and exposes MCP management through its CLI/TUI. The first adapter should remain conservative until a stable non-interactive management path is verified.

### VS Code — V1 beta

VS Code can install MCP server configuration from its CLI, but end-to-end status/tool discovery is more UI/command-palette oriented. Keep this adapter beta until a reliable black-box automation path is established.

### GitHub Copilot CLI — later

Copilot CLI has explicit MCP add commands for HTTP servers and is a strong follow-up adapter after the first three live adapters are stable.

## What this project does not test

A successful interoperability result does not mean:

- the MCP server is secure;
- tool implementations are correct;
- destructive operations are safe;
- the model will choose the appropriate tool;
- every OAuth identity or scope combination will work.

Those concerns belong to separate security, conformance, or agent-evaluation tools.
