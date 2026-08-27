# Suite manifest v1

[English](suite-manifest-v1.md) | [日本語](suite-manifest-v1.ja.md)

Suite manifest v1 is the declaration contract for the v0.7 repeatable regression workflow. The `suite` CLI validates the contract and can execute `trusted_real_client` manifests. Regression reporting and hosted-CI fixture orchestration remain later v0.7 work.

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
- A trusted real-client target requires a stable non-secret `deployment_id`. Suite execution uses the existing schema-v2 protected-path identity so the resolved path is not persisted or hashed into portable artifacts.
- Bearer tokens, client secrets, authorization codes, cookies, executable paths, shell hooks, environment overrides, and literal endpoint URLs are not fields in schema v1. Because decoding rejects unknown fields, adding them does not silently expand the trust boundary.

For target ID `production-a`, the only accepted endpoint variable is:

```text
MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A
```

The variable **value** is resolved only by trusted suite execution and is not part of the manifest or suite index.

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

`oauth` is explicit per client and is accepted only in `trusted_real_client` manifests. Validation does not authenticate, read endpoint-variable values, or launch clients. `suite run` resolves all declared endpoint variables before it launches the first client.

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

## Execution boundary

`mcp-interop suite run <manifest.json> --output-dir <dir>` currently executes only `trusted_real_client` manifests. It:

- resolves every target environment variable before launching any client;
- uses the existing direct live-test path for each selected client/auth mode;
- writes each run as a schema-v2 protected-path artifact;
- refuses to replace an existing output directory;
- preserves non-PASS and missing-client results instead of dropping them.

`hosted_fixture` declarations remain validation-only until #115 connects them to controlled localhost fixtures under the repository CI trust policy. Retry/flake semantics and baseline/regression reporting remain #114 work. The manifest intentionally contains no arbitrary command/hook mechanism for any later stage to execute.
