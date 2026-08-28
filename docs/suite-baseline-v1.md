# Suite baseline v1

[English](suite-baseline-v1.md) | [日本語](suite-baseline-v1.ja.md)

Suite baseline v1 is the immutable local baseline contract introduced for the v0.8 regression workflow.
It wraps an already validated suite result set; it does not create new interoperability evidence.

## Acceptance boundary

Creating a baseline is an explicit acceptance action:

```console
mcp-interop baseline create suite-results --output-dir baselines/codex-current
```

The source must contain complete schema-v2 **real-client adapter** evidence for every declared run,
including an exact client version. Execution errors, missing artifacts, runner-only observations, and
missing client versions are rejected as baseline sources.

A retry, client auto-update, or later `suite run` never mutates a baseline. The CLI reserves the output
directory with an exclusive directory creation operation and refuses any existing destination.
The copied result set is assembled privately inside that reserved directory and `baseline.json` is
written last, so a partial bundle never validates as an accepted baseline.

## Layout

```text
baseline/
  baseline.json
  results/
    index.json
    artifacts/
      production-a--codex--none.json
```

`results/` is a copied suite result-set v1 snapshot. It is not a pointer to the source directory.

## Descriptor

`baseline.json` uses:

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/suite-baseline",
  "created_at": "2026-08-28T08:00:00Z",
  "manifest_fingerprint": "sha256:...",
  "execution_context": "trusted_real_client",
  "result_set_digest": "sha256:...",
  "supersedes": "sha256:..."
}
```

`result_set_digest` deterministically binds the validated result index and every referenced live-result
artifact by logical JSON content. Readers recompute the digest and reject mutated snapshots.

The descriptor deliberately does not repeat endpoint origins, endpoint paths, executable paths, tokens,
OAuth credentials, or source filesystem paths. The copied schema-v2 artifacts retain their existing
canonical-origin privacy boundary: `credential-safe != deployment-public` still applies.

## Intentional replacement

Baselines are not updated in place. To accept a newer comparable result set, create a new directory and
name the prior baseline explicitly:

```console
mcp-interop baseline create suite-results-new \
  --output-dir baselines/codex-next \
  --supersedes baselines/codex-current
```

The new descriptor stores the prior baseline fingerprint in `supersedes`. The previous directory remains
unchanged and therefore remains auditable.

Superseding fails closed when the manifest fingerprint or execution context differs. It also rejects a
changed deployment fingerprint or platform for a run that has evidence on both sides. A version-only
client change is allowed; version changes are evidence context, not regressions by themselves.

There is intentionally no ambient mutable "current baseline" pointer. Selection is explicit by baseline
path, which prevents a retry or auto-updated client from silently becoming the comparison anchor.

## Comparison

Compare one selected immutable baseline with every retained attempt:

```console
mcp-interop baseline compare baselines/codex-current attempt-1 attempt-2
mcp-interop baseline compare baselines/codex-current attempt-1 attempt-2 --json
mcp-interop baseline compare baselines/codex-current attempt-1 attempt-2 --fail-on-regression
```

The output remains suite regression report v1. No migration is required for existing result-set v1,
live-result artifact v2, or suite regression report v1 consumers.

Before regression comparison, baseline-specific identity checks fail closed on manifest, execution
context, deployment fingerprint, or platform mismatch. Missing current evidence remains regression/
unstable evidence under the existing v0.7 report semantics, and all retries remain retained.

Exit behavior matches `suite compare`:

- `0` for a valid comparison when the gate is not triggered;
- `1` with `--fail-on-regression` when regression or unstable evidence is present;
- `2` for invalid, unreadable, mutated, or incomparable baseline/input evidence.

## Compatibility and migration

Suite baseline v1 is a new wrapper schema. It does **not** change:

- live-result artifact v1/v2;
- suite manifest v1;
- suite result-set v1;
- suite regression report v1;
- the real-client-only meaning of live PASS.

Unsupported future baseline schema versions are rejected rather than guessed or migrated implicitly.
