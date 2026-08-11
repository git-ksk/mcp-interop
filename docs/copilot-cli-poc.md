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
- `--additional-mcp-config` for session-only MCP definitions.

Official references:

- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference
- https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers

## PoC topology

`scripts/poc-copilot-cli-mcp.sh` uses the repository's existing loopback MCP fixture and a real installed `copilot` executable.

```text
isolated temporary workspace
        |
        +-- COPILOT_HOME
        |     +-- settings.json (auto-update disabled)
        |     +-- mcp-config.json -> localhost fixture
        |
        +-- COPILOT_CACHE_HOME
        |
        +-- copilot mcp get mcp-interop-fixture --json
                         |
                         v
              127.0.0.1 MCP fixture
```

The PoC does not send a model prompt and blocks ordinary outbound HTTP(S) through an unreachable loopback proxy while exempting localhost.

## PASS contract

The PoC passes only when all of the following are true:

1. the real installed Copilot CLI returns exit code 0;
2. `copilot mcp get ... --json` emits valid JSON containing the controlled `ping` tool;
3. the fixture observes `initialize`;
4. the fixture observes `notifications/initialized`;
5. the fixture observes `tools/list`;
6. the fixture does **not** observe `tools/call`;
7. selected normal `~/.copilot` metadata and the login Keychain database are unchanged;
8. no new `copilot` process remains after the run.

Configuration registration alone is not a PASS.

## Isolation boundary

The PoC intentionally does not copy the user's normal Copilot authentication/configuration state into the temporary profile. `COPILOT_HOME` and `COPILOT_CACHE_HOME` both point inside the PoC directory.

Common GitHub/model API token environment variables are removed from the Copilot child process. This is intended to answer a narrow question: can the supported `copilot mcp get` management surface perform direct MCP inventory without a model/API turn or reuse of normal credential state?

If the command requires Copilot account authentication before it can perform MCP inventory, the PoC should fail and issue #48 should remain research-only until a safe credential-isolation design is proven.

## OAuth

OAuth is explicitly out of scope for this first PoC. A no-auth direct inventory PASS is required before any OAuth adapter work.

## Running

On the self-hosted macOS real-client runner:

```console
bash scripts/poc-copilot-cli-mcp.sh
```

Set `MCP_INTEROP_KEEP_COPILOT_POC_TMP=1` only when sanitized diagnostics need to be retained. The kept directory may contain client logs, so review it before sharing.
