# Antigravity OAuth live-test boundary

`mcp-interop test <url> --client antigravity --oauth` explicitly enables Antigravity's real MCP OAuth manager on macOS. The adapter runs the installed `agy` client under a PTY with an isolated temporary `HOME` and workspace, opens `/mcp`, selects the single isolated test server, and forwards the operator's authorization-code input directly to the real client. No model prompt is used.

## Credential isolation

For the tested `agy 1.1.11` baseline, Antigravity persists MCP OAuth state under:

```text
~/.gemini/antigravity/mcp_oauth_tokens.json
```

Because `HOME` is replaced with the temporary session home, this path remains isolated from the user's normal Antigravity state. `mcp-interop` observes only file metadata (existence, regular-file type, and non-zero size); it never opens or parses the token file and never persists authorization URLs, authorization codes, access tokens, refresh tokens, cookies, or credential-file contents.

## Result semantics

The OAuth path is deliberately conservative. If the isolated token file appears, `reach` and `auth` can be reported as `pass`. `init` and `tools` are reported as `pass` only when the client-side Antigravity tool cache is also observed.

On the tested `agy 1.1.11` OAuth path, the real client completes authenticated `initialize`, `notifications/initialized`, and `tools/list`, but does not materialize the same tool-cache files used by the no-auth adapter path. In that case the generic live result remains `init=unknown` and `tools=unknown` rather than inferring success from authentication alone.

The controlled localhost release E2E separately requires secret-free server-side evidence that the real Antigravity client performed authenticated `initialize`, `notifications/initialized`, and `tools/list`. This E2E evidence does not change the generic four-stage verdict for arbitrary Remote MCP targets.

## Safety gates

The real-client OAuth E2E verifies that normal Antigravity configuration, normal OAuth-token state, the macOS login Keychain, and the pre-existing `agy` process set are unchanged after the isolated test. Temporary PTY descendants are terminated during cleanup on both success and failure.
