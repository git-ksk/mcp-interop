# Reason codes

[English](reason-codes.md) | [日本語](reason-codes.ja.md)

`mcp-interop` stage results can include an optional `reason_code` in addition to `status` and the human-readable message.

Reason codes are deliberately conservative. They are emitted only when the adapter can distinguish a specific interoperability failure from evidence returned by the real client. They are not inferred from guessed endpoints or a single HTTP status in isolation.

## Initial OAuth reason codes

### `DCR_UNSUPPORTED`

The real client explicitly reports that Dynamic Client Registration (DCR) is not supported for the OAuth target.

Example shape:

```text
STAGE  STATUS  REASON           DETAIL
auth   FAIL    DCR_UNSUPPORTED  Codex reports that Dynamic Client Registration is not supported for this OAuth target
```

JSON:

```json
{
  "stage": "auth",
  "status": "fail",
  "reason_code": "DCR_UNSUPPORTED",
  "message": "Codex reports that Dynamic Client Registration is not supported for this OAuth target"
}
```

The initial Codex classifier recognizes explicit client errors equivalent to `Dynamic client registration not supported`. A guessed `/register` or `/oauth/register` returning `404` is **not** sufficient by itself.

### `TOKEN_AUTH_METHOD_MISMATCH`

Emitted when explicitly supplied, secret-free Runtime Evidence disagrees with the token endpoint auth method selected from published client/server metadata. For example, if both sides share `private_key_jwt` but the observed token request has `client_assertion_present=false`, this reason code is reported.

This is not a Preflight failure: `PREFLIGHT PASS` and Runtime Evidence `FAIL` can coexist. Raw assertions and token request bodies are never accepted or stored.

### `DCR_FAILED`

The real client explicitly reports that it attempted Dynamic Client Registration and that the registration attempt failed for a reason other than unsupported.

This distinction leaves room for later diagnostics to retain safe details such as an observed policy rejection or server-side registration failure without collapsing those cases into `DCR_UNSUPPORTED`.

## Security boundary

Raw app-server error messages may contain remote or client-generated text. The Codex adapter retains those details only in memory for classification. Ordinary error strings, text reports, and JSON results expose the stable reason code plus a project-authored message rather than the raw client error.

Authorization URLs, tokens, codes, client secrets, cookies, and credential files must never be added to reason-code details.

## Server capability correlation

The first reason-code implementation classifies explicit real-client evidence. It does **not** yet independently claim that an authorization server advertises CIMD or DCR.

A follow-up v0.2 capability diagnostic will correlate client-observed failures with authorization-server metadata such as:

- `client_id_metadata_document_supported`
- `registration_endpoint`

That correlation must follow MCP Protected Resource Metadata and authorization-server discovery rather than guessing registration URLs. It is supporting diagnostic evidence only; real-client execution remains the source of interoperability pass/fail results.
