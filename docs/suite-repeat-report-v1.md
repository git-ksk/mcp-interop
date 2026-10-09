# Suite repeat report v1

[English](suite-repeat-report-v1.md) | [日本語](suite-repeat-report-v1.ja.md)

The `suite repeat` workflow executes the **same trusted manifest** two to five times and retains every attempt as an independently readable suite result set. It does not select a baseline automatically, call MCP tools, or decide MCP protocol conformance.

## Command

```console
export MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A='https://example.com/mcp/<protected-path>'
mcp-interop suite repeat suite.json \
  --output-dir repeat-results \
  --attempts 3 \
  --timeout 45s \
  --json
```

`--attempts` is required, integer **2..5**; `--timeout` is required, bounded **1s..10m** per client. The aggregate declared active-client timeout budget must be no more than **45 minutes**. The output directory must not exist. The suite is checked and every endpoint is validated before creating it. No untrusted remote manifest or endpoint can enter a hosted GitHub PR job through this option.

The output directory is owner-private (0700 where supported) and contains:

```text
repeat-results/
  manifest.json             # frozen validated declaration, never endpoint values
  attempt-01/index.json     # suite result-set schema v1
  attempt-01/artifacts/...  # strict protected-path live-result schema v2
  attempt-02/index.json
  attempt-02/artifacts/...
  attempt-03/index.json
  attempt-03/artifacts/...
  repeat-report.json        # separately versioned *derived* summary
```

All endpoint variables are resolved once; each attempt uses the same in-memory endpoint values and frozen manifest. The attempt directories are atomically committed by the existing `suite run` implementation. Completed attempts are **never overwritten or removed** if a later attempt fails or the caller cancels. An incomplete current attempt may be recorded with an execution error or fail before its index commits. In that case the summary has `decision=incomplete` and the completed attempt directories are retained.

## Derived report contract

`repeat-report.json` is a **derived, informational schema v1 report**, not a replacement for the strictly validated suite index, live-result artifacts, or an authenticated attestation.

- `schema_version`: integer `1`
- `artifact_type`: `mcp-interop/suite-repeat-report`
- `manifest_fingerprint`: the canonical manifest declaration fingerprint, **never** a hash of raw endpoint values
- `requested_attempts` / `completed_attempts`: integers
- `complete`: boolean; `false` after cancellation, missing result-set output, or a failed attempt recording process
- `decision`: `clean`, `non_pass`, `unstable`, `non_pass_and_unstable`, or `incomplete`
- `has_non_pass` / `has_unstable`: booleans computed over all completed attempts
- `attempt_indexes`: ordered relative paths, e.g. `attempt-01/index.json`
- `runs[]`: target/deployment/client/auth identity, `all_pass`, `unstable`, and **all** observed attempts
- `runs[].attempts[]`: 1-based `attempt` and an `evidence` object containing the validated outcome, exit code, relative artifact reference, exact client version, runner platform, endpoint fingerprint and four stage status/reason values. An execution error has no fabricated live-result artifact or stage evidence.

A `clean` decision requires every intended attempt to complete and pass and all observed signatures to agree. The comparison signature incorporates exact client version, per-stage statuses/reasons, observed runner platform, protected-path endpoint fingerprint and suite outcome. Changed client versions or endpoints across attempts are **unstable**, not silently comparable. A consistently failing set is `non_pass`, not a passing stable set. Failed-then-passing and passing-then-failed cases are `non_pass_and_unstable`. Cancellation is always `incomplete`, even when completed attempts all pass.

Exit codes: **0** only for complete, consistently passing attempts; **1** for `non_pass`, `unstable`, `non_pass_and_unstable`, `incomplete`, or execution/report errors; **2** for invalid invocation/manifest, unresolved endpoints, excessive budget, or existing output directory, all rejected before real-client execution.

The report does not infer success from a later retry and does not include full raw server/client logs, raw endpoint paths or queries, OAuth credentials, model prompts, or unexpected tool names. Do not use the result report alone as a security or authenticity guarantee. Save and verify the referenced suite result sets as your underlying evidence.

## Comparison with an accepted baseline

A repeat report answers **"were these repeated observations consistent and passing?"**. A regression report answers **"did the observed run regress relative to this independently accepted baseline?"**. To answer the second question, explicitly compare *all* attempts:

```console
mcp-interop suite compare baseline-results \
  repeat-results/attempt-01 \
  repeat-results/attempt-02 \
  repeat-results/attempt-03 \
  --json --fail-on-regression
```

Do not automatically promote attempt-01 to an accepted baseline. `suite compare` and `baseline compare` keep their existing schemas and semantics. Optional expected-tool name assertions are not included in live-result v2 artifacts; a passing suite repeat does **not** establish that every expected tool exists.
