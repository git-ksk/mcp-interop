# Exact observed client coverage

[English](observed-coverage.md) | [日本語](observed-coverage.ja.md)

This page records only **exact real-client observations that are retained by the repository**. It is not a semantic-version support range and it is not a claim that every deployment works on the listed client.

## Claim boundary

A row may be called observed only when the exact client version, runner platform, real-client outcome, and evidence source are retained. Nearby versions and other operating systems remain untested unless they have their own evidence.

The current historical rows below come from the controlled localhost real-client release gate recorded in [Current real-client protocol-era observations](protocol-era-observations.md) and merged in [PR #108](https://github.com/git-ksk/mcp-interop/pull/108). That gate invoked the normal non-OAuth core path, required `reach/auth/init/tools=PASS`, observed protocol readiness on the fixture, prohibited `tools/call`, and required configuration, Keychain metadata, process, and temporary-session cleanup gates to pass.

The E2E harness deletes its temporary result directory after the run, so these 2026-08-27 rows are retained as a repository observation record, not as schema-v2 suite result sets. They therefore document exact historical coverage but are **not** valid inputs to `mcp-interop compatibility query` or `compatibility matrix`. New coverage should retain suite result sets/baselines when possible so the machine-readable compatibility model can preserve every attempt.

## Current retained exact observations

| Client | Exact version | Runner platform | Scope | Core result | Retained evidence |
| --- | --- | --- | --- | --- | --- |
| Codex CLI | `codex-cli 0.133.0` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/protocol-era-observations.md`, [PR #108](https://github.com/git-ksk/mcp-interop/pull/108) |
| Codex CLI | `codex-cli 0.152.1` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS`; user config/Keychain/process/session cleanup PASS; `tools/call` avoided | [Issue #170](https://github.com/git-ksk/mcp-interop/issues/170) acceptance run on current main |
| Cursor CLI | `2026.08.25-3e8eec8` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/protocol-era-observations.md`, [PR #108](https://github.com/git-ksk/mcp-interop/pull/108) |
| Antigravity CLI | `1.1.22` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/protocol-era-observations.md`, [PR #108](https://github.com/git-ksk/mcp-interop/pull/108) |

Those rows are exact points only. For example, the Cursor observation does not imply support for an earlier or later `2026.08.*` build.

Adapter-level beta/stable decisions are a separate review layer; see [Adapter maturity contract](adapter-maturity.md). Codex stable maturity is scoped to the retained macOS arm64 non-OAuth core path and does not create a version range. An exact version becoming untested does not by itself rewrite adapter maturity.

## OS/platform coverage state

| Client | macOS arm64 | macOS amd64 | Linux | Windows |
| --- | --- | --- | --- | --- |
| Codex CLI | observed exact points `0.133.0`, `0.152.1`; stable adapter scope is limited here | untested | untested | untested |
| Cursor CLI | observed exact point above | untested | untested | untested |
| Antigravity CLI | observed exact point above | untested | unsupported by the shipped live adapter | unsupported by the shipped live adapter |

`untested` means the project has no retained exact real-client evidence for that platform point. It is not a failure. `unsupported by the shipped live adapter` is an implementation boundary: the Antigravity non-macOS adapter deliberately returns skipped results until equivalent PTY/cache behavior is verified with the real client.

An unavailable client or execution error is also not converted into a tested failure. Machine-readable compatibility output retains such runs as `evidence_gaps` instead of manufacturing a `known_broken` point.

## Building a matrix from retained result sets

`compatibility matrix` lists every exact point from explicit accepted-baseline and suite-result-set inputs. It does not detect or execute a client and does not contact a Remote MCP endpoint:

```console
mcp-interop compatibility matrix \
  --baseline baselines/current \
  --observation attempt-1 \
  --observation attempt-2
```

Use `--json` to emit the complete compatibility-envelope v1 object. The JSON keeps every retained observation for each exact point, including source/attempt, outcome, stages, provenance, and regression information. Human output includes the observation sequence and attempt count so a failed/unknown attempt followed by a passing retry cannot appear as a clean single PASS.

Repeated `--observation` inputs are explicit oldest-to-newest collection order for `--stale-on-client-version-change`. Age staleness additionally requires `--max-age-seconds N --trust-executed-at-clock`.

## Rules for adding coverage

- Record the exact client version string; never convert observations into a version range.
- Record the runner/process OS and architecture separately from any client-binary architecture evidence.
- Retain FAIL, UNKNOWN, SKIP, execution-error, and retry evidence; do not publish only the successful retry.
- Keep controlled fixture evidence separate from deployment-specific claims.
- Do not use production tool calls, model prompts, normal-user credentials, or secret-bearing artifacts merely to increase coverage.
- Treat a newly detected but unobserved version/platform as `untested` until new real-client evidence exists.
