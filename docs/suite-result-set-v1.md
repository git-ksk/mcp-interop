# Suite result set v1

[English](suite-result-set-v1.md) | [日本語](suite-result-set-v1.ja.md)

Suite result set v1 is the portable directory contract produced by trusted `mcp-interop suite run` execution.

## Layout

A successful commit of the output directory has this shape:

```text
suite-results/
  index.json
  artifacts/
    production-a--codex--none.json
    production-a--cursor--none.json
```

The directory is assembled in a private staging directory and renamed into place after the index is complete. `suite run` refuses an already-existing output directory so stale files are not silently mixed with a new set.

Each file below `artifacts/` is an ordinary **live-result artifact schema v2** containing exactly one run. The suite layer does not invent a second stage/result model.

## `index.json`

The index uses:

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/suite-results",
  "manifest_schema_version": 1,
  "manifest_fingerprint": "sha256:...",
  "execution_context": "trusted_real_client",
  "artifact_schema_version": 2,
  "runs": [
    {
      "target_id": "production-a",
      "deployment_id": "production-a",
      "client_id": "codex",
      "auth_mode": "none",
      "outcome": "pass",
      "exit_code": 0,
      "artifact": "artifacts/production-a--codex--none.json"
    }
  ]
}
```

The manifest fingerprint is derived from the validated manifest declaration only. Resolved endpoint values are not fingerprint inputs. `deployment_id` is the same stable non-secret operator label used by the referenced schema-v2 artifact; readers verify that the index and artifact identities match.

Run entries are ordered by `target_id`, `deployment_id`, `client_id`, then `auth_mode`, independent of declaration-array order.

## Outcome semantics

- `pass` / exit `0`: the referenced direct live-result artifact passed all four stages.
- `non_pass` / exit `1`: a valid artifact exists but the direct live test did not fully pass. Missing/uninstalled clients are represented this way with explicit `skip` stages in the referenced artifact.
- `error` / exit `1`: execution failed before a valid per-run artifact could be committed. The entry has no `artifact` reference.

The suite command exits `1` if any entry is `non_pass` or `error`. Invalid manifest/input/preflight conditions exit `2` before client execution begins.

## Secret/privacy boundary

`index.json` persists the non-secret `deployment_id`, but never contains:

- Remote MCP endpoint URLs;
- endpoint environment-variable names or values;
- protected URL paths;
- OAuth tokens/codes/secrets;
- client executable paths or arbitrary environment overrides.

Trusted endpoints are resolved in memory before execution. Per-run artifacts use schema-v2 protected-path identity, so the protected path is neither persisted nor hashed. The canonical origin is still present in each schema-v2 artifact; `credential-safe != deployment-public` continues to apply.

## Reader safety

Result-set readers require a regular `index.json`, clean relative artifact references below `artifacts/`, regular artifact files, and resolved artifact paths that remain inside the result-set directory. Symlink-based escapes are rejected before artifact content is trusted.

## Current scope

`hosted_fixture` suite execution remains disabled by this schema. Repository CI keeps controlled localhost fixture execution separate from arbitrary suite manifests, while the trusted/untrusted runner boundary is documented in [Self-hosted real-client CI security](self-hosted-ci-security.md). Suite regression/retry aggregation is defined by [Suite regression report v1](suite-regression-report-v1.md).
