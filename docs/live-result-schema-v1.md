# Live interoperability result artifact schema v1

[English](live-result-schema-v1.md) | [日本語](live-result-schema-v1.ja.md)

This document defines the first portable result artifact for deployment-specific real-client interoperability runs. It is intentionally separate from the existing `mcp-interop test --json` output contract.

The schema exists to support local-file-first regression comparison across exact client versions without weakening the meaning of `reach -> auth -> init -> tools`.

## Compatibility boundary

The existing command remains backward-compatible:

```console
mcp-interop test https://example.com/mcp --client codex --json
```

Its stdout remains the existing JSON array of live `Result` objects. Schema v1 does not add `schema_version`, timestamps, platform fields, or artifact metadata to that legacy payload.

Portable output is opt-in and written to a separate file:

```console
mcp-interop test https://example.com/mcp --client codex --output result.json
```

`--json` and `--output` may be combined. `--output` never redirects or replaces stdout.

## Top-level shape

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/live-results",
  "runs": [
    {
      "executed_at": "2026-08-12T15:00:00Z",
      "endpoint": {
        "identity": "https://example.com/mcp",
        "fingerprint": "sha256:..."
      },
      "client": {
        "id": "codex",
        "product": "Codex CLI",
        "version": "codex-cli 0.147.0"
      },
      "platform": {
        "os": "darwin",
        "arch": "arm64"
      },
      "runtime": {
        "mcp_interop_version": "v0.5.0",
        "mcp_interop_commit": "...",
        "go_version": "go1.24.x"
      },
      "auth_mode": "default",
      "evidence_provenance": {
        "kind": "real_client_adapter",
        "adapter_id": "codex"
      },
      "stages": [
        {"stage": "reach", "status": "pass"},
        {"stage": "auth", "status": "pass"},
        {"stage": "init", "status": "pass"},
        {"stage": "tools", "status": "pass"}
      ]
    }
  ]
}
```

A stage may additionally contain an existing stable `reason_code`. Human stage messages and diagnostic payloads are deliberately excluded from v1 artifacts; the regression layer does not need them and excluding them reduces secret-bearing output surface.

## Endpoint identity and secret safety

Portable artifacts never persist the raw target URL.

`endpoint.identity` contains only:

```text
http(s)://lowercase-host[:explicit-port]/path
```

User information, query parameters, query values, and fragments are excluded. Query values are excluded even when their parameter names are not recognized as credential-like by legacy redaction.

`endpoint.fingerprint` is SHA-256 of that already secret-safe identity, prefixed with `sha256:`. The raw URL is never hashed, so a portable artifact does not retain a secret-derived fingerprint.

The consequence is intentional: v1 cannot distinguish two deployment targets whose only identity difference is in query parameters. Secret safety takes priority over that distinction.

## Run context

Each run records:

- UTC `executed_at`;
- secret-safe endpoint identity and fingerprint;
- client ID, product name, and exact detected client version for a real-client adapter run;
- operating system and architecture;
- `mcp-interop` version/commit and Go runtime version;
- invocation auth mode (`default` or explicit `oauth` today);
- evidence provenance;
- exactly the ordered `reach`, `auth`, `init`, and `tools` stage results.

`auth_mode` describes how the runner was invoked. It does not infer the server's authentication requirements from metadata.

## Evidence provenance and PASS

`evidence_provenance.kind` is one of:

- `real_client_adapter` — the installed real client was executed; `adapter_id` and exact client version are required;
- `runner_observation` — the runner itself observed a condition such as a missing client before any real-client adapter could run.

A `runner_observation` is never allowed to contain a `pass` stage. The artifact layer does not create new PASS evidence and does not reinterpret adapter results. A complete live PASS still requires all four existing stages to be `pass` from real-client evidence.

## Strict validation

Schema v1 comparison input is decoded strictly:

- `schema_version` must be `1`;
- `artifact_type` must be `mcp-interop/live-results`;
- unknown JSON fields are rejected;
- at least one run is required;
- each comparison identity must be unique within an artifact;
- `executed_at` must use UTC;
- endpoint fingerprints must match their canonical secret-safe identity;
- stages must appear exactly once and in `reach`, `auth`, `init`, `tools` order;
- stage statuses remain limited to `pass`, `fail`, `skip`, and `unknown`.

Unknown or incomplete evidence is rejected or preserved as non-PASS; it is never normalized into success.

## Comparison identity

`mcp-interop compare` pairs baseline and new runs using:

```text
endpoint fingerprint
+ client.id
+ auth_mode
+ platform.os
+ platform.arch
```

Exact client version, execution time, and runner/runtime versions are recorded context but are deliberately not pairing keys. This allows a client upgrade to be compared against the previous client version rather than being treated as an unrelated run.

A version-only change is not a regression.

## Regression semantics

```console
mcp-interop compare old.json new.json
mcp-interop compare old.json new.json --json
mcp-interop compare old.json new.json --fail-on-regression
```

The comparison explicitly classifies:

- `PASS_TO_FAIL`;
- `PASS_TO_UNKNOWN`;
- `PASS_TO_SKIP`;
- `REASON_CODE_CHANGED` when a reason code is added, removed, or changed;
- `NEW_EVIDENCE_MISSING` when a baseline run has no paired run in the new artifact.

A new-only run is reported but is not itself a regression. A client-version change with stable stage/reason evidence is reported as context and does not fail the gate.

## Compare exit codes

`mcp-interop compare` uses the following contract:

- `0` — valid comparison completed; without `--fail-on-regression`, this also covers reports that contain regressions;
- `1` — `--fail-on-regression` was requested and at least one regression or baseline evidence-loss regression exists;
- `2` — usage error, unreadable input, unsupported/invalid artifact schema, or comparison output failure.

The ordinary `mcp-interop test` exit contract is unchanged. Portable artifact creation/write failures are execution failures (`1`), while `test` itself still returns success only when every required real-client stage passes.

## Scope

Schema v1 is local-file-first. It does not introduce a hosted service, database, SQLite history, dashboard, new client adapter, generic MCP conformance suite, security scanner, or LLM evaluation layer.

Future artifact revisions must use a new `schema_version` rather than silently changing v1 semantics.