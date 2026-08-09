# Troubleshooting

[English](troubleshooting.md) | [日本語](troubleshooting.ja.md)

This guide covers common failure modes when running `mcp-interop` against real MCP clients.

## Start with client detection

Run:

```console
mcp-interop clients
mcp-interop clients --json
```

If a client is not detected, first confirm the executable is available in the current `PATH` and record its version independently. `mcp-interop` intentionally does not guess that a differently named binary is compatible unless the adapter explicitly supports that alias.

## Exit code is non-zero even though nothing obviously failed

A test exits `0` only when all four stages are `pass`:

```text
reach / auth / init / tools
```

`fail`, `skip`, and `unknown` all produce a non-zero result. This is intentional: CI should not accept an inconclusive interoperability result as success.

## What `unknown` means

`unknown` means the real client's available control/management surface did not provide enough evidence to prove success or failure.

Typical examples:

- a client exposes an empty tool inventory for both an unreachable server and a legitimate zero-tool server;
- a beta adapter cannot observe a stable machine-readable status surface;
- a client process exits before the adapter can prove the protocol stage.

Do not reinterpret `unknown` as `pass`. Capture the exact client version when reporting it.

## OAuth-required server

### Codex

Codex OAuth is supported only when explicitly requested:

```console
mcp-interop test https://example.com/mcp --client codex --oauth
```

The authorization URL is printed to stderr and is short-lived. Do not paste it into issue reports because it contains OAuth state.

When Codex fails before an authorization URL is available, inspect the `REASON` column or JSON `reason_code`. For example, `DCR_UNSUPPORTED` means the real Codex client explicitly reported that Dynamic Client Registration is not supported for that OAuth target. `mcp-interop` does not infer this code from a guessed registration URL returning `404`.

See [Reason codes](reason-codes.md) for the stable classification rules.

### Cursor

Cursor OAuth completion is not enabled in v0.1.x. The beta adapter can test no-auth endpoints, but an OAuth-required target remains incomplete until the authenticated path is verified and shipped.

### Antigravity

Automated OAuth completion is intentionally disabled in v0.1.x. OAuth discovery has been observed, but completion remains blocked until credential storage can be proven isolated from the normal macOS Keychain.

## Cursor callback-port conflict

The tested Cursor version has used a localhost callback during OAuth PoC work, but callback details are treated as version-specific behavior. Do not hard-code a permanent callback port in automation.

If a future OAuth flow reports a local callback conflict, record:

- exact Cursor version;
- callback address reported by the client;
- whether another process was already listening;
- sanitized client output with tokens/state removed.

## Antigravity returns `unknown`

The macOS beta adapter relies on a no-input PTY startup plus isolated machine-readable MCP tool-cache state.

If no valid cache appears before the run becomes inconclusive, the adapter returns `unknown` rather than claiming interoperability from configuration discovery alone.

Check:

- exact `agy` version;
- whether the target requires OAuth;
- whether the client created isolated `~/.gemini/antigravity-cli/mcp/...` state;
- whether the process exited early;
- whether the operating system is supported by the live adapter.

The shipped live implementation is macOS-only in v0.1.x.

## VS Code is detected but cannot be live-tested

VS Code remains research-only. Safe isolated MCP registration is possible, but the tested CLI does not expose a supported direct server-start/status/tool-discovery surface. Registration alone is not reported as interoperability success.

## Temporary state or process cleanup failure

A cleanup failure is a real test failure even if `reach/auth/init/tools` passed.

When reporting it, include:

- client and exact version;
- operating system/architecture;
- whether the leftover process belongs to the isolated test session;
- sanitized temporary path names;
- the cleanup error.

Do **not** kill every `codex`, `cursor-agent`, or `agy` process by name. The adapters are designed to terminate only processes they can prove belong to the current test session.

## User configuration changed unexpectedly

This should be treated as a bug and potentially a security issue.

Do not include credentials or raw credential files in a public issue. If the problem involves credential leakage, normal user config mutation, Keychain writes, or isolation failure, follow [SECURITY.md](../SECURITY.md) and use private vulnerability reporting.

## JSON output

Use:

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity --json
```

Multi-client JSON output is an array. Treat the stage values as the authoritative machine-readable result; do not infer success only from process exit text.

When a stage has a specific classified failure, JSON can also include a stable `reason_code`:

```json
{
  "stage": "auth",
  "status": "fail",
  "reason_code": "DCR_UNSUPPORTED",
  "message": "Codex reports that Dynamic Client Registration is not supported for this OAuth target"
}
```

Absence of `reason_code` does not imply success; it means the adapter did not have enough specific evidence to assign a stable classification.

## Real-client E2E harness

Maintainers can run the release-gate harness on macOS:

```console
bash scripts/e2e-real-clients.sh
```

It requires the real supported clients to be installed. The harness also checks protocol evidence, unexpected `tools/call`, selected user configuration metadata, Keychain database changes, leaked client processes, and leaked temporary sessions.

For release-candidate validation, do not bypass safety gates merely to obtain a green result.

## Reporting a reproducible bug

Include:

- `mcp-interop version` output;
- operating system and architecture;
- exact MCP client version;
- selected adapter;
- stage results and any `reason_code`;
- sanitized error/diagnostic output;
- whether the server requires OAuth;
- whether the issue reproduces against a localhost/synthetic fixture if applicable.

Never include bearer tokens, OAuth codes, client secrets, cookies, or credential files.
