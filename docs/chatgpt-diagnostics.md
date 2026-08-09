# ChatGPT connection diagnostics

[English](chatgpt-diagnostics.md) | [日本語](chatgpt-diagnostics.ja.md)

`mcp-interop diagnose --profile chatgpt` checks a Remote MCP deployment against ChatGPT's documented OAuth/MCP flow and can correlate explicitly supplied, secret-free runtime observations.

It is **not** a ChatGPT live adapter. It does not create a ChatGPT app, press **Scan Tools**, complete OAuth inside ChatGPT, or claim a real-client `reach/auth/init/tools` PASS.

## Evidence layers

The diagnostic keeps three distinct evidence layers:

1. **Preflight** — public Remote MCP / OAuth metadata compatibility.
2. **Runtime Evidence** — sanitized presence/match observations from the authorization server or MCP resource server.
3. **OpenAI Reference Pattern** — a compact correlation of the supplied runtime evidence against the authenticated-MCP flow documented by OpenAI and demonstrated by `openai-mcpkit`.

A future real ChatGPT adapter remains a separate fourth layer and requires a supported, automatable ChatGPT product surface.

## Basic usage

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

Machine-readable output:

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt --json
```

The Preflight layer checks:

- HTTPS Remote MCP endpoint;
- OAuth 2.0 Protected Resource Metadata discovery from `WWW-Authenticate` or standardized well-known locations;
- advertised `authorization_servers`;
- OAuth Authorization Server Metadata / OpenID Connect discovery with matching `issuer`;
- `authorization_endpoint` and `token_endpoint`;
- Client ID Metadata Documents (CIMD) and Dynamic Client Registration (DCR) advertising;
- CIMD token endpoint authentication compatibility (`none` / `private_key_jwt`);
- PKCE `S256` advertising;
- `offline_access` advertising as a refresh-token/connectivity advisory;
- Protected Resource Metadata `resource` consistency.

A server can pass registration preflight with CIMD and no `registration_endpoint`. ChatGPT prioritizes CIMD when available, but DCR remains a valid alternative when selected/needed.

## Verify the exact ChatGPT CIMD metadata

When a sanitized authorization request exposes the non-secret `client_id` metadata URL and `redirect_uri`, pass them directly:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

The diagnostic then verifies:

- the CIMD document is fetchable;
- the document's `client_id` matches its stable HTTPS URL;
- client/server token endpoint auth methods intersect;
- the observed redirect URI is listed when supplied;
- the advertised JWKS is reachable when `private_key_jwt` is compatible.

Do not supply authorization codes, OAuth state, tokens, cookies, or client assertions.

## Runtime Evidence v2

Runtime Evidence v2 separates registration, authorization request, token request, resource-server verification, and tool-level OAuth signals.

Example:

```json
{
  "schema_version": 2,
  "registration": {
    "strategy": "cimd",
    "client_metadata_url": "https://chatgpt.com/oauth/.../client.json"
  },
  "authorization_request": {
    "resource_matches": true,
    "redirect_uri_matches": true,
    "pkce_s256": true
  },
  "token_request": {
    "resource_matches": true,
    "code_verifier_present": true,
    "client_assertion_present": false,
    "client_assertion_type_present": false,
    "oauth_error": "invalid_client"
  },
  "resource_request": {
    "bearer_present": false
  },
  "tool_auth": {
    "challenge_expected": true,
    "oauth2_security_scheme_present": true,
    "www_authenticate_present": true,
    "www_authenticate_has_error": true,
    "www_authenticate_has_error_description": true
  }
}
```

Run it with:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --runtime-evidence runtime-evidence.json
```

`registration.strategy` accepts:

- `cimd`
- `dcr`
- `predefined`

This matters because token endpoint authentication cannot be inferred the same way for every registration strategy. For CIMD, the diagnostic can compare ChatGPT's fetched client metadata with authorization-server metadata and prefer `private_key_jwt` when both advertise it. For DCR or predefined clients, it does **not** assume a token auth method that was not observed from the registered client.

### Legacy v1 compatibility

The original compact evidence shape remains accepted:

```json
{
  "client_id": "https://chatgpt.com/oauth/.../client.json",
  "resource_matches": true,
  "code_verifier_present": true,
  "client_assertion_present": false
}
```

It is normalized internally as legacy v1 CIMD/token-request evidence. New integrations should prefer schema v2.

### Secret boundary

Only booleans, the stable CIMD metadata URL, registration strategy, and a short OAuth error code are accepted. Unknown fields are rejected. Do **not** put any of the following into Runtime Evidence:

- access or refresh tokens;
- authorization codes or OAuth `state`;
- PKCE verifier/challenge values;
- raw client assertions or private keys;
- client secrets;
- cookies or credential files.

## Runtime checks

When supplied, v2 can correlate:

### Authorization request

- canonical `resource` match;
- redirect URI match;
- PKCE S256 usage.

### Token request

- canonical `resource` match;
- `code_verifier` presence;
- observed token endpoint authentication;
- sanitized OAuth errors such as `invalid_client`.

For example, a CIMD path where both ChatGPT and the authorization server advertise `private_key_jwt`, but the observed token request omits `client_assertion`, produces:

```text
TOKEN_AUTH_METHOD_MISMATCH
```

### MCP resource request

After token exchange, the resource server can supply presence/result booleans for:

- bearer token attached;
- signature validation;
- issuer match;
- audience/resource match;
- expiry validation;
- required scopes.

This mirrors the resource-server responsibilities in OpenAI's authentication documentation and the JWT verification pattern in the official authenticated Python MCP scaffold.

### Tool-level OAuth signals

OpenAI documents two halves for ChatGPT's tool-level OAuth linking UI:

- tool authentication metadata such as an `oauth2` entry in `securitySchemes`;
- runtime `_meta["mcp/www_authenticate"]` challenges when authentication/reauthorization is required.

`tool_auth` evidence can record only the presence/shape signals. `mcp-interop` does **not** call tools merely to obtain this evidence.

If `challenge_expected=true`, an explicitly missing OAuth security scheme or runtime challenge can be classified. If the observation was not captured, the result stays `WARN / unknown` rather than guessing.

## OpenAI Reference Pattern

When Runtime Evidence is supplied, text output also includes:

```text
OPENAI REFERENCE PATTERN
```

The reference summary covers the observed boundaries for:

- registration;
- PKCE;
- token endpoint auth;
- bearer delivery;
- resource-server token verification;
- tool-level OAuth signals.

The reference source is the current OpenAI authentication documentation plus the structural authenticated-MCP pattern demonstrated by `openai/openai-mcpkit/python-authenticated-mcp-server-scaffold`.

This is a **reference-pattern comparison**, not a statement that Auth0-specific setup is universally required. Provider-specific operational steps from the scaffold are not treated as protocol requirements.

## Result semantics

The Preflight verdict remains independent:

```text
PREFLIGHT PASS
```

can coexist with:

```text
RUNTIME EVIDENCE
VERDICT  FAIL
REASON   TOKEN_AUTH_METHOD_MISMATCH
```

or a failing OpenAI Reference Pattern summary.

The CLI exits non-zero when supplied Runtime Evidence contains a conclusive failure. Missing observations generally become `WARN / unknown`.

Even `PREFLIGHT PASS` + Runtime Evidence `PASS` does **not** prove the real ChatGPT product completed OAuth, MCP initialization, or tool discovery.

## Mutual TLS boundary

OpenAI currently documents that ChatGPT presents an OpenAI-managed client certificate when establishing TLS connections to MCP servers. That transport-level signal is useful for client identification, while OAuth continues to authenticate/authorize the end user.

`mcp-interop` does not yet classify mTLS certificate observations in Runtime Evidence v2. Do not treat absence of an mTLS observation as a diagnostic failure.

## Current real-client boundary

`mcp-interop` still does not have a supported headless ChatGPT app-management surface equivalent to Codex app-server or Cursor MCP management commands. Browser DOM automation and private UI internals remain out of scope for a stable real-client adapter.

Official/reference sources:

- OpenAI authentication: `https://developers.openai.com/plugins/build/auth`
- OpenAI ChatGPT developer mode / MCP apps: `https://help.openai.com/en/articles/12584461-developer-mode-and-full-mcp-connectors-in-chatgpt`
- OpenAI authenticated Python MCP scaffold: `https://github.com/openai/openai-mcpkit/tree/main/python-authenticated-mcp-server-scaffold`
- MCP authorization specification: `https://modelcontextprotocol.io/specification/draft/basic/authorization`
