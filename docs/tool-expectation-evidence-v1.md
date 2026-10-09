# Expected-tool evidence v1

[English](tool-expectation-evidence-v1.md) | [日本語](tool-expectation-evidence-v1.ja.md)

This independent, opt-in evidence family records **operator-declared expected names** and their directly proven presence/absence for one real client. It does not expose unknown observed names or extend the four core `reach/auth/init/tools` stages.

## Producer

`mcp-interop test <url> --client codex --expect-tool ping --output core.json --deployment-id safe-label --tool-evidence expected.json` writes an existing protected-path v2 core artifact **and** a separate private tool-evidence v1 artifact. A missing named tool exits `1` but retains the evidence. One client and an expectation are required; neither output may exist beforehand. `--deployment-id` is a public, non-secret operator label; never derive it from URL path or credentials. Expected names are explicit CLI arguments and may be visible in process listings.

## Document structure

- `schema_version: 1`
- `artifact_type: mcp-interop/tool-expectation-evidence`
- `run`: validated, protected-path **single** live-result v2-style run: exact client/version, runner platform, execution time, auth mode, real-client/runner provenance, four core stage statuses, origin and non-secret deployment fingerprint. No raw URL path, query or endpoint credential.
- `expectation`: separate operator-declared `expected_names` (unique sorted, up to 64, UTF-8 bound 128 bytes per name); optional `expected_count` (0..4096); `observed_count` only when the directly observed real-client name inventory was accepted; `missing_names` a sorted subset of explicitly expected names; status `pass`/`fail`/`unknown` and validated reason code.

Input is capped at 1 MiB, unknown fields and symlink inputs are rejected, and verdict/count/name consistency is checked. `unknown` has **no observed count and no inferred missing names**. Only complete core PASS and direct real-client provenance can support a known observation; an absent/ambiguous list cannot be turned into PASS. Evidence creation refuses existing destination files and attempts to use owner-private permissions (0600 where supported). File content is local evidence, **not an authenticated attestation**.

## Comparison

`mcp-interop tools compare baseline.json current.json --json --fail-on-drift` emits the derived report identity `mcp-interop/tool-expectation-diff` (schema v1). It checks strict compatibility for deployment/origin, client ID, platform, auth mode, expected names, and configured expected count. Differences in those fields fail as input errors. A client version change alone does not fail comparison.

The report includes old/new assertion statuses and observed counts, `newly_missing` and `recovered` names **only from the declared expectation set**, and a decision:

| Decision | Meaning | Gated exit |
|---|---|---|
| `clean` | Observed membership and count match | 0 |
| `regression` | Newly missing expected name or decreased count | 1 |
| `drift` | Count increased without a newly missing expected name | 1 |
| `recovered` | Previously missing or failed expectation now passes | 0 |
| `non_pass` | Current expectation is still failing | 1 |
| `unknown` | Either observation cannot establish exact membership/count | 1 |

Malformed/incompatible inputs exit `2`. Without `--fail-on-drift`, a valid report exits `0` regardless of its decision. No unexpected-name additions/deletions are inferred from a count-only change. **An unlisted tool replacement with the same number of tools is not detectable**: list every name whose identity matters using `--expect-tool`, and use `--expect-tool-count` to catch changes in inventory size. The strict suite/baseline/repeat schemas and their core-only verdict remain untouched.
