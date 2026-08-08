# mcp-interop

**Live interoperability testing for Remote MCP servers across real MCP clients.**

`mcp-interop` is an experimental, cross-client test runner for Remote Model Context Protocol (MCP) servers. It is designed to answer a practical question that protocol conformance alone cannot answer:

> Does this Remote MCP deployment actually connect, authenticate, initialize, and expose tools in the real clients my users run?

## Status

Early development. The first milestone targets live, black-box interoperability checks with:

- Codex CLI
- Cursor CLI
- Antigravity CLI
- VS Code (beta adapter)

Claude Code support is intentionally deferred.

## V1 scope

A successful client test means that the tested client can complete the applicable stages below:

1. Remote endpoint registration
2. Authentication / OAuth when required
3. MCP initialization
4. Tool discovery

`mcp-interop` does **not** claim that a server is secure, that every tool behaves correctly, or that an AI model will choose the right tool.

## Design principles

- **Real clients, not emulators.** Client-specific checks should invoke the installed client wherever practical.
- **No model benchmark required.** Connection testing should not require sending prompts to a model.
- **Do not mutate user configuration.** Test adapters should use isolated or temporary profiles/configuration whenever the client permits it.
- **Deterministic reports.** Prefer machine-readable states such as `pass`, `fail`, `skip`, and `unknown` over subjective scoring.
- **No hosted service required.** The core tool should run locally and in CI without a project-operated backend.

## Planned CLI

```console
mcp-interop clients
mcp-interop probe https://example.com/mcp
mcp-interop test https://example.com/mcp
mcp-interop test https://example.com/mcp --client codex,cursor
mcp-interop test https://example.com/mcp --json
```

Example target output:

```text
Remote MCP: https://example.com/mcp

                 Reach   Auth   Init   Tools
Protocol probe     PASS   PASS   PASS   PASS
Codex CLI          PASS   PASS   PASS   PASS
Cursor CLI         PASS   FAIL   SKIP   SKIP
Antigravity CLI    PASS   PASS   PASS   PASS
VS Code (beta)     PASS   PASS   PASS   PASS

Cursor CLI
└─ FAIL: OAuth flow did not complete successfully
```

## Non-goals for V1

- MCP security scanning
- Tool quality or LLM-selection benchmarking
- Runtime sandboxing
- Permission/capability governance
- A new OAuth or MCP conformance specification
- Guaranteeing compatibility without running the corresponding client

## License

License will be finalized before the first public release.
