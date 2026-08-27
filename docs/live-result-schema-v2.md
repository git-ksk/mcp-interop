# Live interoperability result artifact schema v2

[English](live-result-schema-v2.md) | [日本語](live-result-schema-v2.ja.md)

Schema v2 adds an explicit operator-supplied deployment identity for Remote MCP endpoints whose URL path must not be persisted. It exists because schema v1 intentionally keeps the URL path as deployment identity and therefore cannot safely represent a credential-bearing path such as:

```text
https://example.com/mcp/<opaque-capability>
```

Schema v1 remains supported with unchanged semantics. `mcp-interop test ... --output` still writes v1 unless `--deployment-id` is supplied.

## Creating a protected-path artifact

```console
mcp-interop test 'https://example.com/mcp/<protected-path>' \
  --client codex \
  --output result.json \
  --deployment-id production-a
```

`--deployment-id` requires `--output`. The deployment ID is persisted verbatim and must be a stable **non-secret** identifier chosen by the operator. It must not be copied, encoded, truncated, or otherwise derived from the protected URL path or any credential.

The current CLI accepts 1 to 128 bytes using only ASCII letters, digits, `.`, `_`, and `-`.

When protected-path mode is active, ordinary text/JSON result output prints only the canonical origin rather than echoing the endpoint path. The existing JSON top-level result-array contract is otherwise unchanged.

## Document shape

```json
{
  "schema_version": 2,
  "artifact_type": "mcp-interop/live-results",
  "runs": [
    {
      "executed_at": "2026-08-27T12:00:00Z",
      "endpoint": {
        "identity": "production-a",
        "fingerprint": "sha256:<hex>",
        "identity_kind": "deployment_id",
        "origin": "https://example.com"
      },
      "client": {
        "id": "codex",
        "product": "Codex CLI",
        "version": "codex-cli 1.0.0"
      },
      "platform": {
        "os": "darwin",
        "arch": "arm64"
      },
      "runtime": {
        "mcp_interop_version": "v0.x.y",
        "mcp_interop_commit": "<commit>",
        "go_version": "go1.x.y"
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

The run, client, platform, runtime, provenance, stage ordering, status, and reason-code semantics are unchanged from v1.

## Endpoint identity and secret safety

Schema v2 currently supports one endpoint identity mode:

```text
identity_kind = deployment_id
```

For this mode:

- `endpoint.identity` is the operator-supplied non-secret deployment ID;
- `endpoint.origin` is only `http(s)://lowercase-host[:explicit-port]`;
- the endpoint path is not persisted;
- the endpoint path is not hashed;
- query, userinfo, and fragment material are not persisted or hashed;
- no path-shape, entropy, or credential-name heuristic is used as a security boundary.

`endpoint.fingerprint` is SHA-256 of the domain-separated string `deployment_id\0<origin>\0<identity>`. This fingerprint is only a deterministic convenience over already non-secret identity material. It is **not** a privacy mechanism and must never be used to make a secret deployment ID safe.

The implementation deliberately does not hash the protected path. A plain stable hash of a low-entropy path secret would create an offline guessing oracle.

### Public-origin boundary

Schema v2 removes credential-bearing path material, but it still persists the canonical origin. Therefore:

```text
protected-path-safe != origin-private
```

If even the hostname/origin is operationally sensitive, do not publish or commit the artifact merely because it is v2. The v0.6 deployment-privacy work keeps that broader sharing boundary explicit.

## Comparison identity

Two v2 runs pair on:

```text
endpoint.origin + endpoint.identity
+ client.id
+ auth_mode
+ platform.os
+ platform.arch
```

Exact client version, execution time, and runner/runtime versions are not pairing keys. The canonical origin is part of deployment identity, while changes to the protected path are deliberately ignored. Reusing a deployment ID on a different origin therefore does not silently pair two distinct deployments.

A duplicate v2 comparison identity inside one artifact is rejected. Operators must choose deployment IDs that are unambiguous within the artifact set they intend to compare.

Machine-readable comparison output uses comparison report `schema_version: 2` for v2 artifacts. Existing v1 comparisons continue to emit comparison report schema v1 unchanged.

## v1 compatibility and migration

Schema v1 remains readable, writable, and comparable with its existing path/fingerprint semantics.

Schema versions are not silently mixed:

- v1 vs v1: supported with unchanged v1 pairing semantics;
- v2 vs v2: supported with deployment-ID pairing semantics;
- v1 vs v2: rejected as an input error (exit `2`).

Cross-schema comparison is intentionally not guessed because a v1 URL-path identity and a v2 operator identity have no safe implicit equivalence. To establish a new v2 baseline, rerun the target with a chosen non-secret `--deployment-id`, then compare subsequent v2 artifacts using that same ID.

There is no automatic downgrade from v2 to v1. Doing so could reintroduce a credential-bearing path into a portable artifact.

## Validation and file safety

Schema v2 keeps the v1 file-safety rules:

- artifact input is bounded before JSON decoding;
- unknown JSON fields are rejected;
- `artifact_type` must be `mcp-interop/live-results`;
- at least one run is required;
- each comparison identity must be unique;
- `executed_at` must use UTC;
- stages remain exactly `reach`, `auth`, `init`, `tools`;
- statuses remain `pass`, `fail`, `skip`, `unknown`;
- runner-only observations cannot create PASS evidence;
- output replacement uses the existing private atomic-write path.

Schema v2 changes endpoint identity representation only. It does not weaken the real-client PASS boundary or add a new evidence source.
