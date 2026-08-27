# Suite manifest v1

[English](suite-manifest-v1.md) | [日本語](suite-manifest-v1.ja.md)

Suite manifest v1 is the declaration contract for the v0.7 repeatable regression workflow. The current `suite` CLI only **validates** this contract; execution and regression reporting are added by later v0.7 issues.

## Validate

```console
mcp-interop suite validate suite.json
mcp-interop suite validate suite.json --json
```

Validation is strict. Unknown fields, unsupported schema versions, unknown live-client IDs, unsafe execution-context combinations, and arbitrary endpoint environment-variable references fail with exit code `2`.

## Secret-safety boundary

A suite manifest never contains the Remote MCP endpoint URL itself. This is intentional.

- `hosted_fixture` targets use `endpoint.source: "fixture"` and cannot request OAuth.
- `trusted_real_client` targets use `endpoint.source: "environment"`.
- The environment-variable name is derived from the target ID and cannot be chosen arbitrarily by the manifest.
- A trusted real-client target requires a stable non-secret `deployment_id`. Later suite execution uses the existing schema-v2 protected-path identity so the resolved path is not persisted or hashed into portable artifacts.
- Bearer tokens, client secrets, authorization codes, cookies, executable paths, shell hooks, environment overrides, and literal endpoint URLs are not fields in schema v1. Because decoding rejects unknown fields, adding them does not silently expand the trust boundary.

For target ID `production-a`, the only accepted endpoint variable is:

```text
MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A
```

The variable **value** is resolved only by later trusted execution code and is not part of the manifest.

## Hosted fixture example

```json
{
  "schema_version": 1,
  "execution_context": "hosted_fixture",
  "targets": [
    {
      "id": "fixture-a",
      "endpoint": {"source": "fixture"},
      "clients": [
        {"id": "codex", "auth": "none"},
        {"id": "cursor", "auth": "none"}
      ]
    }
  ]
}
```

`hosted_fixture` is the unprivileged declaration shape. It cannot select a network endpoint or opt into OAuth.

## Trusted real-client example

```json
{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [
    {
      "id": "production-a",
      "endpoint": {
        "source": "environment",
        "variable": "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"
      },
      "deployment_id": "production-a",
      "clients": [
        {"id": "codex", "auth": "none"},
        {"id": "cursor", "auth": "oauth"}
      ]
    }
  ]
}
```

`oauth` is explicit per client and is accepted only in `trusted_real_client` manifests. Validation does not authenticate, read endpoint-variable values, or launch clients.

## Stable v1 fields

Top level:

- `schema_version`: must be `1`.
- `execution_context`: `hosted_fixture` or `trusted_real_client`.
- `targets`: one or more target declarations.

Target:

- `id`: 1-63 lowercase ASCII letters/digits with internal `-`; also determines the endpoint environment-variable name.
- `endpoint.source`: `fixture` or `environment`, constrained by the execution context.
- `endpoint.variable`: required only for trusted real-client targets and must exactly match the derived name.
- `deployment_id`: required only for trusted real-client targets; uses the existing non-secret deployment-ID syntax.
- `clients`: one or more unique shipped live adapters: `codex`, `cursor`, or `antigravity`.

Client selection:

- `id`: shipped live-adapter ID.
- `auth`: `none` or `oauth`; `oauth` requires `trusted_real_client`.

## Non-goals for #112

Manifest v1 validation does not yet:

- execute clients;
- resolve endpoint environment-variable values;
- write artifact sets;
- retry failed attempts;
- compare baselines;
- run privileged self-hosted workflows.

Those behaviors remain tracked by #113, #114, and #115. The manifest intentionally contains no arbitrary command/hook mechanism for those later stages to execute.
