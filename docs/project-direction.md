# Project direction

[English](project-direction.md) | [日本語](project-direction.ja.md)

This document is the canonical product-direction contract for `mcp-interop`. It exists to keep the project focused as MCP conformance, inspectors, evaluation platforms, and client capability matrices expand around it.

The detailed category comparison with official MCP Conformance remains in [Conformance vs. mcp-interop](conformance-vs-interop.md). This document answers a different question: **what should this project build next, what should it refuse to become, and what evidence quality is non-negotiable?**

## Mission

`mcp-interop` should become the repeatable regression-testing layer for a concrete interoperability tuple:

```text
Remote MCP deployment
  × real shipping client product
  × exact client version
  × relevant platform/runtime context
```

The primary user question is:

> Does the Remote MCP endpoint I actually deploy still work, through the real MCP path, in the exact client products and versions my users run?

The durable value is not the number of client names in a matrix. It is trustworthy, repeatable evidence that a specific deployment/client pairing worked, failed, or regressed.

## Core product contract

A live interoperability result is scoped to the execution that produced it. It must not be generalized beyond the observed endpoint, client product/version, platform/runtime context, authentication mode, and evidence available in that run.

The currently shipped core live stages are:

```text
reach -> auth -> init -> tools
```

This is an existing output/evidence contract, not an assumption that every future MCP revision has a wire-level initialization phase. The [roadmap](roadmap.md) defines the protocol-aware work needed to preserve the semantic meaning of interoperability as MCP evolves.

A complete live PASS requires every required semantic stage for the observed/supported path to be `pass` from real-client evidence. `unknown`, `skip`, metadata compatibility, fixture-only success, or sanitized runtime observations must never be promoted to a deployment-specific live PASS.

The project should prefer a conservative false-negative or `unknown` over a convenient false-positive.

## Evidence hierarchy

Keep these layers distinct in implementation, output, documentation, and future schemas:

1. specification/conformance evidence;
2. direct server inspection/debugging;
3. product-profile preflight metadata;
4. sanitized Runtime Evidence from the deployment;
5. deterministic adapter/fixture evidence proving the measurement path;
6. **live deployment-specific evidence from the real shipping client**.

Only layer 6 can prove that the target deployment passed in the target client/version. Layer 5 proves that the adapter can measure what it claims; it does not prove a different production deployment passed.

## What counts as a real-client boundary

A live adapter should use a supported public automation surface when one exists. An observed product surface may be used when it is stable enough to reproduce and its limitations are documented, but the project must not weaken PASS semantics to increase client count.

Acceptable examples include:

- documented CLI MCP management commands;
- documented local app-server/control protocols;
- isolated client-owned state that directly reflects MCP lifecycle/tool discovery;
- deliberately controlled PTY interaction with the actual installed client when there is no safer supported machine interface, provided the path is deterministic and evidence remains conservative.

The following are not acceptable as the core PASS oracle:

- model prompts or an LLM choosing/calling a tool;
- brittle browser DOM automation;
- private/minified product endpoints or internal UI command identifiers used only because no supported boundary exists;
- copying or reusing the user's normal authentication credentials to make a test pass;
- configuration presence, enablement state, or an allowlist treated as discovered tools without live discovery evidence;
- server metadata or direct server inspection substituted for client-observed lifecycle evidence.

If a product cannot meet the boundary, keep it research-only or report `unknown`/`skip` rather than changing the meaning of PASS.

## Adapter lifecycle and graduation

Client support should have explicit maturity states.

### Research-only

Use this state while discovering whether a safe direct boundary exists. Research may prove partial stages, but the project must not present incomplete evidence as a supported live adapter.

Remain research-only when any required stage depends on unsafe credential reuse, model prompting, brittle UI scraping, private internals, or an unproven lifecycle/tool-discovery path.

### Beta

A beta adapter may ship when the real-client path is useful and repeatable but version/OS coverage or the observed product surface is still narrow.

Before beta, require all of the following for the supported path:

- isolated config and credential state;
- exact client version capture;
- relevant platform/runtime context capture;
- no-model core execution;
- deterministic reach/auth/init/tools interpretation;
- conservative `unknown` semantics where evidence is ambiguous;
- bounded timeout/cancellation behavior;
- owned-process/session cleanup;
- secret rejection/redaction;
- controlled fixture E2E proving the claimed real-client path;
- before/after checks showing normal user state is not mutated where practical.

### Stable

Promote beta only after the adapter has enough exact evidence to show that the measurement boundary is not a one-version or one-platform accident. Stable requires the full beta gate plus:

- repeated evidence for each advertised PASS path across at least two exact client versions, without inferring a continuous version range;
- retained real-client evidence for every advertised OS/architecture scope, or an explicitly narrower supported scope;
- a sufficiently supported or repeatedly evidenced client measurement surface;
- an exact-point compatibility/regression maintenance path for future client changes.

Stable does not mean the external client can never change. A newly installed but unobserved exact version is still `untested`; a version string changing by itself does not automatically promote, demote, or regress the adapter. Maturity changes require an explicit evidence review.

The canonical machine/human-readable criteria and current shipped-adapter decisions are defined in [Adapter maturity contract](adapter-maturity.md). `mcp-interop maturity` reports those decisions without detecting or executing a client.

Optional capability evidence uses the separate [Capability profile v1](capability-profile-v1.md) contract. Capability states do not broaden the core live PASS contract, and indirect metadata/configuration/UI presence cannot satisfy a capability PASS.

## Priority order

When roadmap items compete, use this order.

### P0 — Protect evidence correctness

Highest priority:

- no false live PASS;
- stage/result invariants;
- secret safety;
- process/session cleanup;
- deterministic timeout/cancellation;
- reproducible fixture gates;
- CI/release provenance and regression coverage.

A change that increases adapter count but weakens these invariants should not merge.

### P1 — Make regressions first-class

The next major product layer should make version-to-version and run-to-run changes easy to detect.

Target capabilities:

- versioned interoperability result/evidence schema;
- stable run identity containing endpoint identity/safe fingerprint, client product, exact version, platform/runtime, auth mode, and stage results;
- optional secret-free evidence bundle export;
- compare/diff output between two runs;
- CI-friendly regression gates such as `PASS -> AUTH FAIL` or `TOOLS PASS -> UNKNOWN`;
- machine-readable reason changes, not only aggregate PASS/FAIL;
- local-file-first history so the core project does not require a hosted backend.

Example target workflow:

```text
production endpoint + Cursor 2026.08.04 -> PASS
production endpoint + Cursor 2026.08.11 -> AUTH FAIL
                                      ^ regression
```

A future hosted/dashboard layer may consume these artifacts, but it is not required for the core value proposition.

### P2 — Harden existing adapters across versions and platforms

Before aggressively adding clients, improve confidence in existing shipping adapters:

- test multiple recent client versions where feasible;
- expand OS/platform coverage when the client itself supports it;
- record known-good/known-bad compatibility envelopes;
- detect output/control-surface changes conservatively;
- keep fixture and cleanup gates aligned with each supported platform.

One deeply trustworthy adapter is more valuable than several adapters whose PASS semantics differ.

### P3 — Add new real clients selectively

Add a client only when it exposes a credible automation/evidence boundary. Popularity alone is not enough.

Candidate evaluation should ask:

1. Can normal user state be isolated?
2. Can the real client be made to reach the target without a model prompt?
3. Can auth be completed or classified safely?
4. Can initialization be observed directly or conservatively inferred from a documented product surface?
5. Can actual tool discovery be proven?
6. Can exact version/platform context be captured?
7. Can the session be deterministically cleaned up?
8. Can a controlled fixture prove the measurement path?

If the answer to a required item is no, keep the client in research.

### P4 — Product-specific diagnostics only where they explain live failures

`diagnose --profile <product>` remains useful when a product has a documented compatibility pattern that explains a real deployment/client mismatch.

Do not grow diagnostics into a second generic OAuth/MCP conformance suite. Prefer official MCP Conformance for specification questions.

## Competitive boundary

The project should deliberately complement, not duplicate, adjacent tools.

### Official MCP Conformance

Owns specification correctness and conformance scenarios. `mcp-interop` should not reimplement generic protocol requirements merely to create its own certification layer.

### Inspectors and MCP testing/evaluation platforms

Own interactive server debugging, emulation, playgrounds, model evaluations, and broad developer UX. `mcp-interop` should only overlap where direct real-client execution is necessary to answer the deployment-specific interoperability question.

### Static client capability matrices

Own the general question "what features does client X support?" They are useful inputs for deciding what to test, but static capability data is not live evidence for a deployment.

### Security scanners and governance products

Own vulnerability scanning, policy, permissions, sandboxing, and security certification. A connectivity PASS must not be marketed as a security result.

### LLM/tool-selection benchmarks

Own whether a model chooses or uses tools well. Model behavior may be a separate downstream test layer, never a prerequisite for the core interoperability PASS.

## Re-evaluate the category when the ecosystem changes

The current direction is valid only while the project continues to own a distinct deployment-specific real-client evidence boundary.

Revisit positioning if a major adjacent project begins to provide all of the following as a first-class, reproducible workflow:

- executes the actual released client products rather than only emulating/configuring them;
- targets an arbitrary user-supplied Remote MCP deployment;
- isolates client config/credentials;
- records exact client version/platform context;
- proves reach/auth/init/tool discovery without relying on a model prompt;
- exports machine-readable per-run evidence suitable for regression comparison.

If that happens, compare maintenance cost, adoption, evidence quality, and extension points before duplicating the same layer. Integrating with or contributing to the stronger upstream project may be better than defending overlap for its own sake.

## Decisions under pressure

When there is tension between growth and evidence quality, choose in this order:

1. protect the meaning of PASS;
2. protect user credentials and local state;
3. preserve reproducibility and cleanup;
4. preserve exact client/version/platform provenance;
5. improve regression detection;
6. broaden version/OS coverage;
7. add another client;
8. add convenience UX.

This ordering is intentional. `mcp-interop` loses its reason to exist if a green result becomes easier to obtain but less trustworthy.

## Current strategic direction

The milestone-level plan is maintained in [Roadmap to a stable interoperability contract](roadmap.md). This document remains the higher-level product-direction contract; the roadmap may sequence work, but it must not override the evidence and safety priorities defined here.

Near-term work should therefore stay focused on:

1. making the core interoperability meaning protocol-era-aware without weakening existing real-client-only PASS semantics;
2. preserving deployment privacy as portable artifacts become routine regression/baseline inputs;
3. turning the shipped artifact/compare primitives into a safe repeatable regression workflow;
4. strengthening Codex/Cursor/Antigravity across observed client versions and supported platforms;
5. keeping ChatGPT, VS Code, GitHub Copilot CLI, and other candidates research-only until their direct automation boundaries satisfy the graduation criteria;
6. stabilizing public contracts only after they have been exercised by real regression workflows.

The intended destination is not "the biggest MCP client list." It is **the most trustworthy answer to whether a real Remote MCP deployment still works in the client versions users actually run.**
