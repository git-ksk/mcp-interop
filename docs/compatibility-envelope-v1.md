# Compatibility envelope v1

[English](compatibility-envelope-v1.md) | [日本語](compatibility-envelope-v1.ja.md)

Compatibility envelope v1 represents interoperability as a set of **exact observed client-version points**. It never turns two successful observations into an inferred continuous version range.

## Evidence model

An observed point is identified by:

- target ID and non-secret deployment ID;
- deployment fingerprint from the validated schema-v2 live result;
- client adapter ID and exact client version string;
- platform OS and architecture;
- auth mode.

Every point retains each contributing observation, including `executed_at`, suite result-set digest, `mcp-interop` runtime identity, real-client adapter provenance, stage/reason evidence, and any regression relationship to an accepted baseline. `executed_at` is wall-clock evidence, not a portable total-order proof across independent runners.

The envelope is scoped to one exact suite manifest fingerprint and execution context. A different manifest, execution context, logical run identity, or deployment fingerprint is rejected rather than silently combined.

## States

The states have deliberately different meanings:

- `tested` — the exact observed point has consistent PASS evidence.
- `untested` — query-only state: the requested exact version/platform point is absent from a known context. It is never serialized as an observed point.
- `stale` — the exact point has otherwise-tested PASS evidence, but an explicit freshness policy marks that evidence stale.
- `known_broken` — the exact point has consistent observed FAIL evidence.
- `regressed` — an observed current point has a real regression against a comparable accepted baseline.
- `unknown` — the evidence for an observed exact point is uncertain or unstable, for example UNKNOWN/SKIP evidence or conflicting retained retries.

`unknown` is therefore an evidence state. `untested` is a coverage state. They are not interchangeable.

A client-version-only change is not a regression. If a locally detected auto-updated version has no exact observed point, classification is `untested` until that version is measured.

## Staleness

Staleness is policy-driven and never implies support for versions that were not observed. v1 supports two explicit signals:

- `max_age_seconds`: mark otherwise-tested evidence stale after the configured age. New age-policy construction requires `trust_executed_at_clock=true`; the CLI expresses this with `--trust-executed-at-clock`.
- `stale_on_client_version_change`: mark an older otherwise-tested point stale when a later collected observation in the same target/deployment/client/auth/platform context has a different exact client-version string.

Version-change chronology does **not** use wall-clock timestamps. The accepted baseline, when present, is ordered before current observations, and repeated `--observation` inputs are ordered exactly as supplied. When this policy is enabled, callers must therefore supply observation result sets from oldest to newest. The rule never parses, compares, or interpolates semantic version numbers. For example, observations of `1.2.0` and `1.8.0` do **not** imply that `1.5.0` was tested or supported.

Age-based freshness is different: it inherently depends on `executed_at`. The explicit clock-trust flag is an operator assertion that the runner clocks are synchronized closely enough for the chosen age threshold. Without that assertion, new `max_age_seconds` envelopes are rejected instead of silently making a stronger freshness claim. Even with trusted clocks, an observation timestamp later than `evaluated_at` is conservatively `stale` with `observation_after_evaluation` rather than being treated as fresh.

## Evidence gaps

Some suite executions cannot become exact compatibility points. They are retained separately as `evidence_gaps`, including:

- execution error with no live-result artifact;
- missing logical-run evidence;
- non-real-client provenance;
- missing exact client version.

A gap may still carry regression/evidence-loss information. The implementation does not invent a client version merely to place that gap on the compatibility envelope.

## Baseline relationship

When an accepted immutable baseline is supplied, current observations are compared against that baseline only where the evidence is safely comparable. Deployment mismatch fails closed. Platform differences remain distinct observed points rather than being coerced into one point.

`regressed` is derived only from actual baseline comparison evidence such as PASS-to-FAIL/UNKNOWN/SKIP, reason-code regression, or equivalent retained regression semantics. Merely observing a different client version never produces `regressed`.

## Schema compatibility

Compatibility envelope v1 is a new reporting model layered on top of the existing contracts. It does not change:

- live-result artifact schema v2;
- suite manifest v1;
- suite result-set v1;
- suite regression report v1;
- the meaning of real-client live PASS.

No migration is required for those existing artifacts. The envelope consumes their validated evidence and adds an exact-point compatibility view. Compatibility-envelope artifacts emitted by v0.8 without `trust_executed_at_clock` remain structurally readable; only construction of a new age-based policy now requires the explicit clock-trust assertion.

## Secret/privacy boundary

The envelope contains non-secret deployment identity and the deployment fingerprint already present in validated portable evidence. It does not add raw Remote MCP endpoint URLs, protected endpoint paths, OAuth credentials, bearer tokens, executable paths, or human diagnostic payloads.

As with schema-v2 live results, credential-safe evidence is not automatically deployment-public: an origin or operator-chosen deployment ID can still be operationally private and should be shared accordingly.

## Query the installed client

`mcp-interop compatibility query` detects the currently installed exact version of one shipped live client and classifies that version against explicit local evidence:

```console
mcp-interop compatibility query \
  --client codex \
  --target production-a \
  --deployment-id production-a \
  --baseline baselines/current \
  --observation suite-results-latest \
  --stale-on-client-version-change
```

Use `--json` for the versioned `mcp-interop/compatibility-query` machine-readable output. The output includes the manifest/execution context, stale policy, optional accepted-baseline fingerprint, detected exact version, exact query, classification, retained point evidence, and relevant evidence gaps. It deliberately omits the detected executable path and all input filesystem paths.

`--max-age-seconds N` enables age-based staleness only together with `--trust-executed-at-clock`. `--observation` may be repeated up to 128 times; its argument order is the explicit collection order used by `--stale-on-client-version-change`, so supply oldest to newest. The bound prevents an accidental command invocation from producing an unbounded retained-observation report. At least one `--baseline` or `--observation` is required.

The command does not change the existing `mcp-interop clients` or `clients --json` contract. Detection alone still makes no compatibility claim. A newly auto-updated installed version that is absent from the supplied exact observed points is reported as `untested`.
