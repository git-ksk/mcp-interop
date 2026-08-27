# Current real-client protocol-era observations

[English](protocol-era-observations.md) | [日本語](protocol-era-observations.ja.md)

This document records the v0.6 issue #99 re-observation of the currently shipped Codex, Cursor, and Antigravity adapters. It is an evidence record for the protocol-aware design work in #100 and the controlled-fixture matrix in #101; it does not change the existing public `reach -> auth -> init -> tools` contract by itself.

## Evidence boundary

Two evidence layers must remain separate:

1. **deployment-specific adapter evidence** — what `mcp-interop test` can prove from the supported or deliberately accepted real-client surface used against the target deployment;
2. **controlled-fixture wire evidence** — what the localhost fixture can observe directly on the wire while the same real client is exercised by the adapter.

A protocol revision observed only by the fixture is not copied into a production run. If the real-client adapter surface does not expose the negotiated protocol revision, the deployment-specific protocol revision remains `unknown`.

The official MCP `2026-07-28` release defines two relevant protocol eras for this work: legacy revisions use `initialize` / `notifications/initialized`; the modern `2026-07-28` revision removes that handshake and may use `server/discover`. See the [MCP 2026-07-28 specification release](https://blog.modelcontextprotocol.io/posts/2026-07-28/).

## Re-observation environment

Observed on 2026-08-27:

```text
macOS 26.5 (25F71)
arm64
mcp-interop main d848de52d74e6c7857adaf75802108fa4d05b5c2
```

Real clients:

```text
Codex CLI        codex-cli 0.133.0
Cursor CLI       2026.08.25-3e8eec8
Antigravity CLI  1.1.22
```

Cursor was installed through its current documented CLI installer. Its supported MCP management surface includes `cursor-agent mcp list` and `cursor-agent mcp list-tools <identifier>`; see the [Cursor CLI parameter reference](https://docs.cursor.com/en/cli/reference/parameters).

Antigravity's documented interactive MCP management surface remains `/mcp`; see the [Antigravity CLI reference](https://antigravity.google/docs/cli/reference/).

## Results

| Client | Adapter surface used for deployment-specific evidence | Protocol revision visible from that surface? | Controlled-fixture wire observation | Fixture-era conclusion |
| --- | --- | --- | --- | --- |
| Codex CLI 0.133.0 | isolated `codex app-server`, then the app-server MCP inventory/status surface | No. The adapter receives live inventory/auth/tool state, not the MCP wire revision. | `initialize` proposed `2025-06-18`, followed by `notifications/initialized` and `tools/list`. | Legacy path observed. |
| Cursor CLI 2026.08.25-3e8eec8 | isolated workspace/HOME with `cursor-agent mcp enable`, `mcp list`, and `mcp list-tools` | No. The supported command output proves live tool discovery but does not expose the negotiated MCP revision. | Separate management invocations opened legacy sessions using `initialize` `2025-11-25`; the `list-tools` path reached `tools/list`. | Legacy path observed. |
| Antigravity CLI 1.1.22 | isolated PTY plus bounded live MCP tool-cache observation; `/mcp` is reserved for explicit OAuth management | No. The cache proves live tool materialization but does not expose the negotiated MCP revision. | First sent `server/discover` with `2026-07-28`, then fell back to `initialize` `2025-11-25`, `notifications/initialized`, and `tools/list`. | Modern probe plus successful legacy fallback observed; successful modern tool discovery is not yet proven. |

All three controlled runs completed the existing real-client release gate:

- real-client adapter result: PASS for `reach`, `auth`, `init`, and `tools`;
- fixture-observed `tools/list`;
- no `tools/call`;
- normal user configuration unchanged;
- login Keychain database unchanged;
- no new client process left running;
- no leaked `mcp-interop` temporary session.

No model prompt was used as the interoperability oracle.

## What the observations prove

The observations prove that the current adapters can still exercise their existing real-client evidence paths against a controlled localhost deployment, and that the fixture can distinguish the protocol era actually used on that controlled run.

They do **not** prove that a production deployment used the same revision. Today, none of the three deployment-specific adapter surfaces returns the negotiated MCP protocol revision in a form that `mcp-interop` can safely attribute to the production run. Therefore:

```text
fixture protocol revision != production-run protocol revision
```

The production-run revision remains `unknown` unless a future real-client surface exposes it directly.

## Input to #100

These observations constrain the protocol-aware core design:

1. the public `init=pass` field cannot permanently mean "a literal `initialize` request was observed" because modern MCP removes that wire phase;
2. the internal semantic model needs a protocol-readiness concept that can project to the existing public `init` field without manufacturing protocol details;
3. protocol revision/era should remain optional evidence and default to `unknown` for deployment runs where the client surface does not expose it;
4. fixture-only protocol evidence must stay tagged as fixture evidence and must never upgrade a production run;
5. Antigravity's observed `2026-07-28` probe followed by legacy fallback must be covered explicitly by the #101 fixture matrix before modern-era semantics are considered complete.

## Controlled protocol-era fixture matrix

Issue #101 adds three explicit fixture modes so protocol-aware behavior is tested without turning the fixture into deployment-specific evidence:

| Mode | Controlled behavior | Purpose |
| --- | --- | --- |
| `legacy` | `server/discover` is rejected; handshake revisions can use `initialize`, `notifications/initialized`, and `tools/list`. | Prove the legacy readiness projection remains supported. |
| `modern` | legacy `initialize` is rejected; `server/discover` and `tools/list` require explicit `2026-07-28` request version evidence. Discovery and list responses include conservative cache hints. | Prove stateless modern readiness without inventing a handshake. |
| `fallback` | `server/discover` returns a deliberately non-definitive response, then legacy initialization remains available. | Reproduce a safe modern-probe-to-legacy fallback such as the current Antigravity observation. |

The matrix never needs `tools/call` for the core interoperability proof. Unsupported/missing modern protocol versions are rejected instead of being silently normalized to modern success.

The real-client release gate is also protocol-era-aware. Controlled fixture readiness passes only when either:

- a complete legacy `initialize -> notifications/initialized -> tools/list` path is observed; or
- `tools/list` itself carries explicit `2026-07-28` protocol evidence.

A modern `server/discover` probe by itself is not enough. The adapter must independently PASS from its deployment-specific real-client surface, and fixture evidence remains release-gate/self-test evidence only.

Historical remote HTTP/SSE variants that have no current product relevance are not added merely for coverage count; the matrix targets the current Remote MCP scope and the protocol eras observed or expected for shipping clients.

## Non-claims

This re-observation does not:

- add a new adapter;
- claim native modern `2026-07-28` tool discovery for any shipping adapter;
- require a production tool call;
- change OAuth behavior;
- weaken the existing live-PASS invariant.
