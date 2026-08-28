# Reason codes

[English](reason-codes.md) | [日本語](reason-codes.ja.md)

`mcp-interop` uses stable machine-readable `reason_code` values when a failure can be classified from explicit evidence.

There are now two evidence families:

- **real-client reason codes** — derived from a real MCP client's observable behavior, such as Codex reporting a DCR failure;
- **runtime-diagnostic reason codes** — derived from explicitly supplied, secret-free server observations used by `diagnose --profile chatgpt --runtime-evidence`.

A runtime-diagnostic code is not a real-client interoperability verdict. Preflight, Runtime Evidence, OpenAI Reference Pattern, and real-client execution remain separate evidence layers.

## Real-client OAuth codes

### `DCR_UNSUPPORTED`

The real client explicitly reports that Dynamic Client Registration is not supported for the OAuth target.

A guessed `/register` or `/oauth/register` returning `404` is **not** sufficient by itself.

### `DCR_FAILED`

The real client explicitly reports that it attempted DCR and the registration attempt failed for a reason other than unsupported.

For the Codex and Cursor adapters, `DCR_UNSUPPORTED` and `DCR_FAILED` also prove that the real client reached the MCP OAuth registration boundary. Those paths therefore report `reach=pass` while keeping `auth=fail` and later stages skipped. Generic OAuth startup failures do not get this reachability promotion.

### `OAUTH_CALLBACK_PORT_CONFLICT`

The real client explicitly reports that its loopback OAuth callback listener could not bind the callback address/port selected for that flow.

This code is based on client-observable bind-conflict evidence. It must not be inferred merely because a guessed or previously observed callback port is occupied. Callback addresses are client-version-specific and should not be hard-coded as a permanent contract.

## Runtime registration / token request codes

### `REGISTRATION_STRATEGY_UNSUPPORTED`

The explicitly observed registration strategy (`cimd` or `dcr`) is not advertised by the discovered authorization-server metadata. `predefined` is not failed solely from public metadata because pre-registration cannot generally be proven that way.

### `TOKEN_AUTH_METHOD_MISMATCH`

The observed token endpoint authentication does not match the method selected from available client/server metadata.

For the ChatGPT CIMD profile, if both the fetched ChatGPT CIMD and authorization server advertise `private_key_jwt` but the observed token request has no client assertion, the expected/observed pair can be:

```text
expected: private_key_jwt
observed: none
```

DCR and predefined registration do not inherit this CIMD expectation without registered-client evidence.

### `CLIENT_AUTH_REJECTED`

The token endpoint returned the sanitized OAuth error code `invalid_client`.

### `TOKEN_REQUEST_REJECTED`

The token endpoint returned a sanitized OAuth error that does not map to a narrower classification.

### `RESOURCE_MISMATCH`

A supplied runtime observation says the OAuth `resource` did not match the canonical protected MCP resource.

### `REDIRECT_URI_MISMATCH`

A supplied authorization-request observation says the redirect URI did not match the registered/client metadata value being diagnosed.

### `PKCE_S256_MISSING`

The observed authorization request did not use PKCE S256 when the ChatGPT profile expects it.

### `PKCE_VERIFIER_MISSING`

The observed token request did not contain a PKCE `code_verifier`. Only presence is accepted as evidence; the verifier value is never ingested.

## MCP resource-server codes

These codes describe secret-free observations made after OAuth token exchange. `mcp-interop` does not ingest the bearer token itself.

### `ACCESS_TOKEN_NOT_ATTACHED`

The subsequent MCP resource request was observed without a bearer token.

### `TOKEN_SIGNATURE_INVALID`

The resource server reported that bearer-token signature verification failed.

### `TOKEN_ISSUER_MISMATCH`

The resource server reported that the token issuer did not match its configured issuer.

### `TOKEN_AUDIENCE_MISMATCH`

The resource server reported that the token audience/resource did not match the protected MCP resource.

### `TOKEN_EXPIRED`

The resource server reported that the token was expired.

### `INSUFFICIENT_SCOPE`

The resource server reported that the token did not contain sufficient scopes for the MCP operation.

These checks align with the resource-server verification responsibilities in OpenAI's authentication documentation and the JWT verification pattern demonstrated by the authenticated Python MCP scaffold in `openai/openai-mcpkit`.

## Tool-level OAuth signal codes

These are only conclusive when the evidence explicitly says an OAuth challenge was expected. Missing observations remain `WARN / unknown`.

### `TOOL_OAUTH_METADATA_MISSING`

An auth-required tool was observed without the expected OAuth `securitySchemes` metadata.

### `TOOL_OAUTH_CHALLENGE_MISSING`

A tool-level authentication/reauthorization challenge was expected, but `_meta["mcp/www_authenticate"]` was observed as absent.

### `TOOL_OAUTH_CHALLENGE_INVALID`

The runtime challenge was present, but an explicitly observed required OAuth error/error-description signal was absent.

`mcp-interop` does not call tools just to obtain these signals. They must come from already available, sanitized server observations.

## Reason precedence

A Runtime Evidence report exposes the first conclusive failure reason in diagnostic evaluation order while retaining every individual check and its own `reason_code`. This keeps the top-level result compact without hiding secondary failures.

For example, a request may first fail with `TOKEN_AUTH_METHOD_MISMATCH` and also have `CLIENT_AUTH_REJECTED` from the token endpoint; both checks remain visible.


## Compatibility policy

Reason codes are an open string enum with stable existing values. Existing codes are not renamed, removed, or repurposed within a stable major line; new codes may be added for new direct evidence. Consumers must tolerate unknown non-empty codes and must not derive a replacement code from free-form messages. See [Public contract candidate](public-contract-v1-candidate.md).

## Security boundary

Runtime Evidence accepts only narrow presence/match booleans, stable non-secret metadata identifiers, registration strategy, and a short sanitized OAuth error code. Unknown JSON fields are rejected.

Never put these values into diagnostic evidence or reason-code details:

- access/refresh tokens;
- authorization codes or OAuth state;
- PKCE verifier/challenge values;
- raw client assertions or private keys;
- client secrets;
- cookies or credential files.

Raw real-client errors may also contain remote-generated text. Adapters keep such text in memory only when necessary for classification and expose project-authored messages plus stable codes instead of blindly persisting raw output.

## Capability correlation boundary

Capability evidence is discovered through MCP Protected Resource Metadata and authorization-server discovery. `mcp-interop` does not infer DCR support from guessed registration URLs.

Multiple authorization servers must be handled conservatively: if the relevant issuer selected by the real flow is not observable, auth-method-specific expectations should remain `unknown` rather than combining unrelated issuer capabilities.
