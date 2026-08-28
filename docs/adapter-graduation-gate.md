# Real-client adapter graduation gate

[English](adapter-graduation-gate.md) | [日本語](adapter-graduation-gate.ja.md)

The graduation gate is the **single minimum evidence policy for moving a research client into the shipped live-adapter set**. A candidate does not receive a weaker path because it is popular, difficult to automate, UI-oriented, cloud-hosted, or already partially characterized.

Use the current reviewed candidate report without detecting or executing a client:

```console
mcp-interop graduation
mcp-interop graduation --json
```

The JSON report uses:

```text
schema_version: 1
artifact_type: mcp-interop/adapter-graduation
```

It contains reviewed policy/evidence references only. It does not contain installed versions, executable paths, Remote MCP endpoints, credentials, cookies, or user-state data, and it performs no network access.

## What graduation means

This gate controls **research -> eligible for beta implementation/shipping**. It is not the same as adapter beta -> stable maturity.

```text
research candidate
  -> every graduation criterion met
  -> eligible_for_beta
  -> explicit adapter implementation/review
  -> shipped beta maturity decision
  -> later stable maturity gate
```

`eligible_for_beta` does not automatically add an adapter to `mcp-interop test`. Shipping still requires an explicit implementation and a valid shipped maturity decision. Conversely, adding a parser branch or adapter switch case does not bypass the policy: the live `--client` parser derives its allowed client IDs from validated shipped maturity decisions.

The existing shipped set remains Codex, Cursor, and Antigravity. Research candidates are not runnable merely because they appear in client detection, diagnostics, a PoC, or this report.

## Mandatory criteria

Every criterion is mandatory for graduation. `limited` and `missing` both block eligibility.

| Criterion ID | Requirement |
| --- | --- |
| `direct_real_client_boundary` | The real client can prove the complete core path through a supported or deliberately accepted direct boundary. Configuration/registration presence, server metadata, model prompts, browser DOM scraping, private/minified internals, and fixture-only observations are not substitutes. |
| `isolated_state` | The complete claimed path uses isolated client/config/auth state and does not depend on copying or reusing normal-user credentials. |
| `owned_cleanup` | Processes, sessions, temporary configuration, and test-created state have bounded ownership and cleanup semantics. |
| `secret_safety` | Credentials, authorization codes, cookies, tokens, protected endpoint paths, and raw secret-bearing client output are not persisted or logged as interoperability evidence. |
| `conservative_failure_semantics` | Missing/ambiguous evidence remains non-PASS; configuration success or partial lifecycle evidence cannot be promoted into complete `reach/auth/init/tools` PASS. |
| `controlled_fixture_e2e` | A repeatable controlled real-client E2E proves the complete path claimed by the proposed adapter, including direct tool discovery where the core profile requires it. |
| `exact_version_platform_evidence` | The tested real-client version and runner/platform context are exact and retained. Nearby versions/platforms are not inferred. |
| `supported_platform_scope` | The proposed shipped OS/architecture scope is explicit and limited to what can be supported and revalidated safely. A single observed platform may justify a narrow beta scope, but it must be named rather than generalized. |

The gate deliberately has **no candidate-specific exception criterion**. Adding an unknown criterion is invalid. A research-only decision must list every `limited` or `missing` criterion as a blocker; blockers cannot be hidden.

## Machine semantics

Each candidate decision contains:

- `client_id` / `display_name`;
- `research_issue`;
- `status`: `research_only` or `eligible_for_beta`;
- `eligible`: boolean derived from the complete mandatory criterion set;
- every mandatory criterion with `met`, `limited`, or `missing`;
- every non-`met` criterion in `blockers`;
- retained repository `evidence_refs`.

Validation requires:

```text
all criteria met
  <=> eligible = true
  <=> status = eligible_for_beta
  <=> blockers is empty
```

Any incomplete criterion requires `research_only`, `eligible=false`, and an exact blocker entry. Duplicate evidence references, unknown criteria, hidden blockers, or inconsistent eligibility are rejected.

## Current v0.9 candidate review

No new real client currently passes the common graduation gate. **Zero graduated clients is an acceptable v0.9 outcome.** The release is successful if existing evidence quality and policy enforcement improve without weakening PASS.

| Candidate | Issue | Current state | Key blockers |
| --- | ---: | --- | --- |
| GitHub Copilot CLI | #48 | `research_only` | Direct complete boundary remains limited: exact 1.0.80 no-input startup reaches/initializes but does not prove `tools/list`; isolated authenticated account/session state is not yet proven; therefore the complete fixture, secret/auth boundary, and shipped platform scope remain incomplete. |
| VS Code | #6 | `research_only` | Isolated registration is proven, but the supported external CLI does not provide a direct start/status/tool-discovery path; no-input isolated startup produced zero fixture MCP requests. Complete E2E, cleanup/auth boundary, and shipped platform scope remain incomplete. |
| ChatGPT | #20 | `research_only` | Current supported custom-app workflow remains product/UI-driven with no public isolated headless app-management/tool-scan contract suitable for the core live adapter. Exact automatable client version/platform, cleanup, auth isolation, and controlled real-client E2E remain missing. |
| Claude web/Desktop | #68 | `research_only` | Remote custom connectors are a legitimate target, but no supported isolated automatable real-client control/tool-discovery boundary has been proven. Exact version/platform, cleanup, auth isolation, and controlled E2E remain missing. |

These rows summarize the linked research evidence. They are not compatibility claims and do not infer support from neighboring versions or product documentation.

## How this connects to shipped adapters

The live-test parser no longer owns an independent handwritten allowlist as the policy source. `ShippedLiveAdapterIDs` derives the permitted set from validated shipped `MaturityDecision` records and cross-checks it with `TierV1` client specs.

That means a future client cannot ship by only:

- adding a detection `Spec`;
- changing a research tier to `v1`;
- adding a `switch` branch;
- documenting a partial PoC.

A `TierV1` spec without a valid shipped maturity decision fails the catalog policy. A shipped maturity decision without a matching `TierV1` spec also fails. The proposed adapter still needs implementation/tests, but the policy entry points cannot silently disagree.

This is intentionally fail-closed policy wiring; it does not claim cryptographic approval or replace human review.

## Relationship to capability profiles

Optional capability graduation follows the same evidence philosophy but uses the separate [Capability profile v1](capability-profile-v1.md) contract. A new client cannot use an optional-capability PASS to compensate for a missing core real-client boundary, and a core adapter graduation does not imply Resources/Prompts/Tasks/MRTR support.

## Research is still useful

`research_only` is not a failure verdict on the product. Research can retain partial evidence and safely improve the next experiment. The gate prevents partial observations from changing shipped compatibility claims before the full boundary exists.
