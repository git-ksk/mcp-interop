# Cursor adapter

The Cursor adapter uses the installed Cursor CLI's dedicated MCP management commands and never sends a model prompt.

Current live path:

```text
isolated HOME + workspace
  -> .cursor/mcp.json
  -> mcp enable
  -> mcp list
  -> mcp list-tools
```

A successful `mcp list-tools` is treated as evidence that the real Cursor MCP client reached the target, passed any client authentication gate, initialized MCP, and completed tool discovery.

OAuth-required servers are detected and reported as incomplete. The adapter intentionally does not run `mcp login` yet because authenticated token exchange and cleanup still need a maintainer-verified end-to-end fixture before that path is enabled.

All generated Cursor state is redirected through a temporary HOME/workspace owned by the interoperability session and removed during cleanup.
