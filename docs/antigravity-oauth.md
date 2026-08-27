# Antigravity OAuth live-test boundary

[English](antigravity-oauth.md) | [日本語](antigravity-oauth.ja.md)

`mcp-interop test <url> --client antigravity --oauth` explicitly enables Antigravity's real MCP OAuth manager on macOS. The adapter runs the installed `agy` client under a PTY with an isolated temporary `HOME` and workspace, opens `/mcp`, selects the single isolated test server, and forwards the operator's authorization-code input directly to the real client. No model prompt is used.

## Credential isolation

For the tested `agy 1.1.11` baseline, Antigravity persists MCP OAuth state under:

```text
~/.gemini/antigravity/mcp_oauth_tokens.json
```

Because `HOME` is replaced with the temporary session home, this path remains isolated from the user's normal Antigravity state. Antigravity account authentication is a separate boundary from Remote-MCP OAuth: before launching `agy`, the adapter writes `modelProvider: "gemini"` to the isolated CLI settings, strips ambient Gemini model credentials/endpoint overrides, and injects a fixed non-secret `GEMINI_API_KEY` sentinel. [Antigravity documents](https://antigravity.google/docs/cli/install/) this Gemini API-key mode as not establishing an account session, so the isolated MCP test does not rely on a normal-user macOS Keychain session. No model prompt is sent, so the sentinel is never used to authorize a model request.

`mcp-interop` observes only file metadata (existence, regular-file type, and non-zero size); it never opens or parses the token file and never persists authorization URLs, authorization codes, access tokens, refresh tokens, cookies, or credential-file contents. The login Keychain before/after comparison remains a non-mutation gate; credential non-reuse is established by the documented no-account mode plus real-client E2E, not by an unchanged Keychain file alone.

## Result semantics

The OAuth path is deliberately conservative. If the isolated token file appears, `reach` and `auth` can be reported as `pass`. `init` and `tools` are reported as `pass` only when the client-side Antigravity tool cache is also observed.

On the tested `agy 1.1.11` OAuth path, the real client completes authenticated `initialize`, `notifications/initialized`, and `tools/list`, but does not materialize the same tool-cache files used by the no-auth adapter path. In that case the generic live result remains `init=unknown` and `tools=unknown` rather than inferring success from authentication alone.

The controlled localhost release E2E separately requires secret-free server-side evidence that the real Antigravity client performed authenticated `initialize`, `notifications/initialized`, and `tools/list`. This E2E evidence does not change the generic four-stage verdict for arbitrary Remote MCP targets.

## Safety gates

The real-client OAuth E2E verifies that normal Antigravity configuration, normal OAuth-token state, the macOS login Keychain, and the pre-existing `agy` process set are unchanged after the isolated test. Temporary PTY descendants are terminated during cleanup on both success and failure.
