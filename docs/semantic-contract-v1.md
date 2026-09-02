# Interoperability semantics v1

[English](semantic-contract-v1.md) | [日本語](semantic-contract-v1.ja.md)

This document defines the stable v1 meanings for adapter identity, maturity, core PASS, optional capabilities, and protocol-era normalization.

## Adapter identity

The shipped live-adapter IDs are currently:

```text
codex
cursor
antigravity
```

Within a stable `v1.x` line, an existing adapter ID is not renamed, reused for a different product, or silently aliased to a different measurement path. New adapter IDs may be added after the common graduation gate. Display names may improve without changing identity.

Detection is broader than shipped support. Seeing a research client in `clients` does not make it a live adapter.

## Tier, maturity, and graduation are separate

`tier` is roadmap/delivery placement. It is not evidence maturity.

Maturity states are:

- `research_only` — not a shipped evidence claim;
- `beta` — the beta evidence gate is met, but one or more stable criteria remain limited/missing;
- `stable` — every stable maturity criterion is met for the advertised scope.

Project-level v1 contract stability and adapter maturity are independent. A stable v1.x project release may ship adapters whose maturity is `beta`. Project v1 stability freezes the documented project contracts; it does not claim that every adapter satisfies the stable adapter gate. Each adapter must be promoted independently from retained evidence for its advertised scope.

Existing maturity-state names and meanings are stable. Criterion IDs are stable machine identifiers; new criteria may be added conservatively, but an existing criterion is not repurposed to make a stronger claim easier.

Research graduation is a separate gate. `eligible_for_beta` means every mandatory graduation criterion is met; it does not automatically ship an adapter. Shipping still requires implementation, review, a tier-v1 spec, and a valid maturity decision.

## Core Remote Tool Interoperability PASS

The public core remains exactly:

```text
reach -> auth -> init -> tools
```

A complete live PASS requires all four stages to be `pass` in that order. `fail`, `skip`, missing, or `unknown` is non-PASS.

- `reach` — the real client produced direct evidence that it reached enough of the target Remote MCP deployment to establish live interaction.
- `auth` — required client authentication completed, or direct live evidence proved that the tested path did not require it.
- `init` — protocol readiness was directly established through the real-client path. It does **not** permanently mean a literal legacy `initialize` request was observed.
- `tools` — the real client discovered the target server's tool inventory.

The core does not require `tools/call`. Arbitrary production tool execution is not a generic PASS prerequisite.

Diagnostics, configuration presence, registration success, metadata, fixture-only observations, or model output cannot substitute for a missing core stage.

## Protocol-era normalization

MCP wire behavior can change without changing the public meaning of protocol readiness.

`init=pass` may be projected from a supported real-client surface that directly proves usable protocol readiness, including tool-inventory evidence. A controlled fixture may verify the adapter implementation/release gate, but fixture-only evidence cannot create deployment-specific `init=pass`.

When the real-client surface does not expose the negotiated protocol revision, the deployment-specific revision remains unknown. Do not copy a revision observed only by a fixture into a production run.

Modern probe, legacy fallback, and future protocol revisions remain evidence details. They may change the internal observation model, but they must not weaken the public four-stage PASS meaning.

## Optional capability semantics

Capability profiles are separate from the core live result.

- `pass`, `fail`, and `unknown` require a documented direct real-client evidence surface for that capability;
- `unsupported` requires an explicit adapter-policy boundary;
- `untested` means no evidence exists for that exact context and carries no evidence ID.

A capability PASS does not upgrade core PASS, and core PASS does not imply Resources, Prompts, Tasks, MRTR, controlled tool calls, or any other optional capability.

## Exact-version and platform semantics

Compatibility is evidence for exact observed points, not a semantic-version range.

- client-version change alone is not a regression;
- an unobserved exact version remains `untested`;
- maturity does not automatically promote/demote when only the installed version string changes;
- platform fields in portable live artifacts describe the mcp-interop runner/process unless stronger client-executable architecture evidence is explicitly available;
- wrapper/script launchers do not inherit host architecture as client-binary evidence.

## Semantic change policy

Changing any of these requires explicit compatibility review:

- the four core stage names/order;
- the condition for aggregate PASS;
- existing adapter IDs;
- maturity/graduation state meanings;
- an existing capability state/evidence-kind meaning;
- whether fixture/metadata/configuration evidence can create deployment-specific PASS.

A change that weakens evidence requirements is not a compatible clarification.
