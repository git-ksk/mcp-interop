# Suite regression report v1

[English](suite-regression-report-v1.md) | [日本語](suite-regression-report-v1.ja.md)

Suite regression report v1 compares one validated baseline suite result set with one or more retained current attempts. It is evidence-derived: per-run stage and reason-code regression semantics come from the existing live-artifact comparator rather than a hand-written support matrix.

## CLI

```console
mcp-interop suite compare baseline-results attempt-1 [attempt-2 ...]
mcp-interop suite compare baseline-results attempt-1 [attempt-2 ...] --json
mcp-interop suite compare baseline-results attempt-1 [attempt-2 ...] --fail-on-regression
```

A directory argument resolves to its `index.json`; an index path may also be passed directly.

Exit behavior with `--fail-on-regression`:

- `0`: report decision is `clean`;
- `1`: regression and/or unstable retry evidence exists;
- `2`: invalid/unreadable/mismatched input or report-generation failure.

Without the gate flag, a valid report is emitted with exit `0` even when its decision is not clean, matching the existing report-only `compare` behavior.

## Retry invariant

Retries never replace earlier attempts. The report contains every supplied attempt in order.

For example:

```text
baseline: PASS
attempt 1: UNKNOWN
attempt 2: PASS
```

The first regression remains visible and the suite decision is `regression_and_unstable`. A later PASS cannot rewrite the evidence history into a clean result.

Repeated attempts with the same material evidence are not unstable merely because they were retried. Material attempt signatures include outcome, endpoint fingerprint, platform, and stage status/reason evidence. Client-version values remain separately visible in each attempt and version-only changes are not regressions by themselves.

## Decisions

- `clean`: no regression and no unstable/ambiguous attempt evidence;
- `regression`: regression evidence exists but retained attempts agree materially;
- `unstable`: attempts are mixed/ambiguous without a baseline regression;
- `regression_and_unstable`: both conditions exist.

Missing current evidence or an execution error is retained explicitly. When baseline evidence existed, those states are regression/evidence-loss signals rather than being silently omitted.

## Machine-readable evidence

The JSON artifact type is `mcp-interop/suite-regression-report`, schema version `1`. It preserves:

- baseline and current manifest fingerprints;
- attempt count and ordering;
- target/deployment/client/auth identity;
- baseline outcome/client version/platform/fingerprint/stages when available;
- every current attempt outcome/client version/platform/fingerprint/stages when available;
- direct stage/status/reason-code changes and regression kinds;
- client-version change markers;
- per-run regression/unstable flags and the suite decision.

The baseline and every current attempt must use the same manifest fingerprint. Attempts with different declarations are not treated as retries.

## Protocol evidence boundary

The current live-result artifact schema v2 intentionally does **not** serialize the internal `ProtocolObservation`. Therefore suite report v1 declares `protocol_evidence_status: "not_serialized_in_live_result_v2"` and does not invent protocol-era/revision changes from fixture or other indirect evidence.

If a future portable live-result schema carries directly observed protocol evidence, a future report revision may compare it with an explicit migration contract.

## Reader and privacy boundary

Report generation reads only validated suite result sets. Referenced artifacts must stay inside the result-set directory after symlink resolution and must match the index `deployment_id`, client, auth mode, and outcome.

Reports do not contain Remote MCP endpoint URLs, protected paths, endpoint environment-variable names/values, credentials, OAuth codes/tokens, or human diagnostic messages. Non-secret deployment IDs and schema-v2 endpoint fingerprints remain portable evidence.
