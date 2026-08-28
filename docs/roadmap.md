# Roadmap to a stable interoperability contract

[English](roadmap.md) | [日本語](roadmap.ja.md)

This roadmap translates the product principles in [Project direction](project-direction.md) into milestones, exit criteria, and explicit non-goals.

It is a planning contract, not a date commitment. Minor-version numbers describe the intended order of maturity work, but they do not force the project to ship an incomplete feature or weaken an evidence boundary. In particular, `v1.0.0` is **not** automatically the release after `v0.9.0` or `v0.10.0`. The project may ship `v0.11.0`, `v0.12.0`, and later `v0.x` releases for as long as necessary. `v1.0.0` is earned only when the stable-contract exit criteria below are satisfied.

## How this document relates to other project documents

- [Project direction](project-direction.md) defines the mission, evidence hierarchy, competitive boundary, and priority order.
- This roadmap defines the planned maturity sequence and the evidence required to declare a milestone complete.
- [Architecture](architecture.md) describes the implementation and trust boundaries that exist today.
- [Live result artifact schema v1](live-result-schema-v1.md) and [schema v2](live-result-schema-v2.md) define the shipped portable result formats; this roadmap does not silently revise either schema.
- [Conformance vs. mcp-interop](conformance-vs-interop.md) defines the boundary with official MCP Conformance.
- [CONTRIBUTING](../CONTRIBUTING.md) defines how roadmap/contract changes are proposed and reviewed.

When these documents differ in purpose, do not treat a roadmap aspiration as already-shipped behavior. Current code, released schemas, and the release documentation remain the source of truth for what a published version actually does.

## GitHub execution tracking

The roadmap is the canonical planning/exit-criteria document. GitHub Milestones and roadmap tracking Issues are the operational source of truth for **progress**. Focused implementation/research Issues should carry the matching GitHub Milestone and be linked from the tracker; completed Issues remain attached as milestone history rather than being recreated.

| Roadmap milestone | Tracking Issue | Current focused Issues |
| --- | --- | --- |
| v0.6.x | [#102](https://github.com/git-ksk/mcp-interop/issues/102) | **Completed:** [#87](https://github.com/git-ksk/mcp-interop/issues/87), [#99](https://github.com/git-ksk/mcp-interop/issues/99), [#100](https://github.com/git-ksk/mcp-interop/issues/100), [#101](https://github.com/git-ksk/mcp-interop/issues/101) |
| v0.7.x | [#103](https://github.com/git-ksk/mcp-interop/issues/103) | **Completed:** [#112](https://github.com/git-ksk/mcp-interop/issues/112), [#113](https://github.com/git-ksk/mcp-interop/issues/113), [#114](https://github.com/git-ksk/mcp-interop/issues/114), [#115](https://github.com/git-ksk/mcp-interop/issues/115) |
| v0.8.x | [#104](https://github.com/git-ksk/mcp-interop/issues/104) | **Completed:** [#125](https://github.com/git-ksk/mcp-interop/issues/125), [#126](https://github.com/git-ksk/mcp-interop/issues/126), [#127](https://github.com/git-ksk/mcp-interop/issues/127) |
| v0.9.x | [#105](https://github.com/git-ksk/mcp-interop/issues/105) | Research candidates: [#6](https://github.com/git-ksk/mcp-interop/issues/6), [#20](https://github.com/git-ksk/mcp-interop/issues/20), [#48](https://github.com/git-ksk/mcp-interop/issues/48), [#68](https://github.com/git-ksk/mcp-interop/issues/68) |
| v0.10.x | [#106](https://github.com/git-ksk/mcp-interop/issues/106) | Split into focused audit/fix Issues when contract review starts |

This mapping is intentionally bidirectional: roadmap work should have a GitHub Issue before implementation begins, and an Issue that changes roadmap scope/exit criteria should update the English/Japanese roadmap pair in the same PR.

## Roadmap invariants

Every milestone must preserve these invariants:

1. **Evidence correctness before feature count.** A convenient green result is never worth a weaker PASS meaning.
2. **Real-client-only live PASS.** Fixture, configuration, metadata, direct-server inspection, or diagnostic evidence cannot substitute for live evidence from the real shipping client under test.
3. **Unknown stays unknown.** Missing or ambiguous evidence is not inferred into success merely to complete a compatibility matrix.
4. **Safe graduation.** A client or capability graduates only after a supported or deliberately accepted direct automation/evidence boundary is proven. A version milestone never justifies weaker criteria.
5. **Backward compatibility by evidence, not assumption.** Preserve public contracts where practical. Introduce a new schema or incompatible contract only after a concrete requirement demonstrates that the existing one is insufficient.
6. **Local-first core.** A hosted backend, account, dashboard, or SaaS service is not required to obtain the core value of the project.
7. **No normal-user credential reuse.** A test must not copy normal browser/client credentials, tokens, Keychain entries, or persistent user state merely to make an adapter pass.

## Core interoperability profile

Before expanding capability count, `mcp-interop` should make the minimum claim behind a core live PASS explicit.

The intended core profile is **Remote Tool Interoperability**:

> The real client reached the target Remote MCP deployment, satisfied any required authentication boundary, established a usable MCP protocol path for the observed protocol era, and obtained real tool-inventory evidence through that client path.

The core profile deliberately does **not** require invoking an arbitrary production tool. Tool calls can have side effects and must not become a generic PASS prerequisite merely to prove discovery.

Resources, prompts, tasks, Multi Round-Trip Requests (MRTR), safe controlled-fixture tool calls, or future MCP extensions may become separate capability profiles. Adding such a profile must not silently broaden the meaning of the core profile.

The currently shipped public result model remains:

```text
reach -> auth -> init -> tools
```

That is an existing compatibility contract, not a declaration that `initialize` is a permanent wire-level phase in every future MCP protocol revision. Protocol-aware normalization must preserve the meaning of old output while avoiding false assumptions about new protocol eras.

## Protocol-era policy

The MCP protocol is evolving. The official `2026-07-28` revision removes the legacy `initialize` / `notifications/initialized` handshake and protocol-level session model. Requests in that revision are self-describing; servers implement `server/discover`, while clients may use discovery but are not required to do so.

`mcp-interop` must therefore separate **observed evidence** from the **normalized interoperability meaning**:

```text
real client execution
  -> observed protocol/client evidence, when available
  -> protocol-era-specific interpretation
  -> normalized core interoperability verdict
```

Protocol version or era is evidence, not a guessed property. A controlled fixture may observe the negotiated protocol revision directly while a production real-client surface may not expose it. In the latter case the protocol revision remains `unknown`. Fixture-observed protocol information must never be copied into a production run as if the production client proved it.

A future internal semantic state such as `protocol_ready` may normalize legacy initialization and modern request readiness. Any replacement or reinterpretation of the existing public `init` field requires an explicit compatibility design and migration plan.

### Remote transport boundary

The core project remains focused on **Remote MCP deployments**.

- Streamable HTTP is the primary modern remote transport.
- Legacy remote HTTP/SSE behavior may remain observable while real shipping clients still use it.
- Supporting every historical transport is not a goal.
- `stdio` interoperability remains outside the core deployment-specific Remote MCP scope unless the project direction is explicitly revised.

## v0.6.x — Protocol-aware core and deployment privacy

**GitHub tracking:** [#102](https://github.com/git-ksk/mcp-interop/issues/102), with focused work in [#99](https://github.com/git-ksk/mcp-interop/issues/99), [#100](https://github.com/git-ksk/mcp-interop/issues/100), and [#101](https://github.com/git-ksk/mcp-interop/issues/101). Protected-path deployment identity work [#87](https://github.com/git-ksk/mcp-interop/issues/87) is complete.

**Status:** completed. #87, #99, #100, and #101 satisfy the v0.6.x exit criteria. v0.7.x has also completed; the current next milestone is v0.8.x / #104.

### Goal

Make the core evidence model correct across legacy and modern MCP protocol eras without weakening the existing live-PASS invariant.

### Required work

- Re-observe current Codex, Cursor, and Antigravity real-client paths and record what protocol-era/version evidence is actually available from their supported or accepted automation surfaces.
- Define protocol-era-aware interpretation for the core Remote Tool Interoperability profile.
- Preserve `unknown` when the production real-client path cannot prove the negotiated protocol revision.
- Keep the existing public `init` stage as the compatibility projection for protocol readiness; a literal legacy `initialize` handshake is not required by the stable semantic meaning.
- Test legacy and modern protocol behavior against controlled fixtures, including fallback/unsupported-era behavior where practical.
- Re-run existing isolation, timeout, cancellation, process-cleanup, state-cleanup, and secret-redaction gates while protocol-aware changes are made.
- Maintain the documented deployment-identity privacy boundary before portable artifacts become routine baseline/CI inputs; schema v2 now removes credential-bearing paths through an explicit non-secret deployment ID, while private-origin sharing remains a separate concern.
- Preserve artifact schema v1 semantics for existing users. Schema v2 was introduced only after #87 demonstrated the protected-path limitation; any further schema revision still requires a concrete need plus explicit compatibility/migration rationale.

### Deployment identity privacy

The current portable artifact deliberately removes raw query values and credential-bearing URL material. That makes an artifact credential-safe, but a secret-safe hostname/path may still be operationally private.

```text
credential-safe != deployment-public
```

Schema v2 now provides an opaque user-defined target identity such as `production-a` for protected-path endpoints. It never hashes the protected path and pairs on the canonical public origin plus that non-secret identity. The canonical origin is still persisted, so this closes the credential-bearing-path gap but does not make a private hostname safe to publish. A deterministic hash alone must not be treated as sufficient privacy when likely deployment identities can be guessed and compared.

### Exit criteria

`v0.6.x` is complete when:

- lifecycle-model assumptions cannot create a false live PASS across the legacy/modern protocol behaviors covered by the project;
- unobserved protocol information remains explicitly unknown;
- the existing three adapters have protocol-aware controlled-fixture coverage appropriate to their actual observable surfaces;
- deployment identity has a documented secret/privacy model suitable for later baseline workflows;
- any artifact schema change has a demonstrated need and an explicit compatibility/migration rationale.

### Non-goals

- no new client is required to graduate;
- no hosted history service;
- no generic MCP conformance replacement;
- no production tool-call requirement.

## v0.7.x — Repeatable regression workflow

**GitHub tracking:** [#103](https://github.com/git-ksk/mcp-interop/issues/103), with completed focused work in [#112](https://github.com/git-ksk/mcp-interop/issues/112), [#113](https://github.com/git-ksk/mcp-interop/issues/113), [#114](https://github.com/git-ksk/mcp-interop/issues/114), and [#115](https://github.com/git-ksk/mcp-interop/issues/115).

**Status:** completed. #112 / #113 / #114 / #115 satisfy the v0.7.x exit criteria; the next active roadmap milestone is v0.8.x / #104.

### Goal

Turn the shipped artifact/compare primitives into an operational workflow that can execute multiple real clients, produce comparable evidence, and make a deterministic CI decision without repository-specific glue for every user.

### Required work

- Define a secret-safe suite/manifest model for selecting targets, clients, authentication modes, and allowed execution contexts.
- Execute multiple selected clients and produce a coherent portable artifact set.
- Generate compatibility/regression reports from evidence rather than maintaining a hand-written support matrix.
- Preserve stage/reason-code changes, client-version changes, protocol-evidence changes, and missing evidence in machine-readable output.
- Define retry and flake semantics. A retry must not erase the first failure. Mixed attempts must be represented as unstable/ambiguous evidence rather than silently converted to PASS.
- Define and enforce the CI trust boundary for real-client execution.

### CI trust boundary

Untrusted pull-request content must not turn a self-hosted real-client runner into an arbitrary network or credential execution surface.

Default policy:

- ordinary/untrusted pull-request CI uses hosted runners and controlled localhost fixtures only;
- self-hosted real-client execution is limited to trusted branches, explicit manual dispatch, or another deliberately approved execution path;
- suite/manifest content from an untrusted PR cannot redirect a privileged runner to private hosts or production-equivalent credential state;
- OAuth remains explicit opt-in and preserves the existing isolation and secret-safety guarantees.

### Exit criteria

`v0.7.x` is complete when a declared test suite can produce real-client artifacts, comparisons, evidence-derived compatibility reporting, and a deterministic CI exit decision while preserving the CI trust boundary.

## v0.8.x — Baselines and compatibility envelopes

**GitHub tracking:** [#104](https://github.com/git-ksk/mcp-interop/issues/104), with focused work in [#125](https://github.com/git-ksk/mcp-interop/issues/125), [#126](https://github.com/git-ksk/mcp-interop/issues/126), and [#127](https://github.com/git-ksk/mcp-interop/issues/127).

**Status:** completed. #125 / #126 / #127 satisfy the v0.8.x exit criteria; the next roadmap milestone is v0.9.x / #105.

### Goal

Make repeated regression testing maintainable across client auto-updates and multiple tested client versions/platforms.

### Required work

- Define a baseline lifecycle: create/select, compare, intentionally accept, and retire/supersede.
- Prevent accidental baseline replacement from hiding a regression.
- Detect stale or missing baseline evidence when it affects confidence.
- Define adapter compatibility envelopes using **observed tested points**, not inferred continuous version ranges.
- Distinguish at least `tested`, `untested`, `stale`, and `known-broken` where those states improve decisions.
- Surface useful adapter/client compatibility information from `mcp-interop clients` or an equivalent machine-readable command without claiming an untested version is compatible.
- Carry exact client version, runner platform/architecture, auth mode, test time/context, and relevant evidence provenance into compatibility reporting.

Example principle:

```text
Tested:
  Cursor X on runner macOS arm64 -> PASS
  Cursor Y on runner macOS arm64 -> PASS

Does not imply:
  every version between X and Y is supported
```

### Exit criteria

`v0.8.x` is complete when baseline changes are intentional/auditable and a client auto-update can be classified as tested, untested, stale, or regressed without fabricating a support range.

## v0.9.x — Coverage, capability profiles, and safe graduation

**GitHub tracking:** [#105](https://github.com/git-ksk/mcp-interop/issues/105). Existing research candidates [#6](https://github.com/git-ksk/mcp-interop/issues/6), [#20](https://github.com/git-ksk/mcp-interop/issues/20), [#48](https://github.com/git-ksk/mcp-interop/issues/48), and [#68](https://github.com/git-ksk/mcp-interop/issues/68) are assigned here for eventual graduation decisions; research may continue earlier without implying shipped support.

### Goal

Deepen confidence in existing adapters and graduate additional product/capability support only where the evidence boundary is strong enough.

### Focused implementation issues

1. [#133](https://github.com/git-ksk/mcp-interop/issues/133) — cross-runner clock-skew chronology hardening
2. [#134](https://github.com/git-ksk/mcp-interop/issues/134) — runner platform vs real client executable architecture
3. [#136](https://github.com/git-ksk/mcp-interop/issues/136) — exact observed version/OS coverage matrix for shipped clients
4. [#135](https://github.com/git-ksk/mcp-interop/issues/135) — baseline authenticity / acceptance provenance boundary
5. [#137](https://github.com/git-ksk/mcp-interop/issues/137) — evidence-based adapter maturity and beta-to-stable criteria
6. [#138](https://github.com/git-ksk/mcp-interop/issues/138) — capability-profile evidence contract and precise PASS semantics
7. [#139](https://github.com/git-ksk/mcp-interop/issues/139) — equal-evidence graduation gate for new real clients

Dependency order: #133 -> #134 -> #136 -> #135 -> #137 -> #138 -> #139.

### Priority order

1. strengthen Codex, Cursor, and Antigravity across realistic current client versions;
2. broaden OS/platform coverage where the real client supports it and safe execution is practical;
3. decide beta-to-stable promotion from observed evidence rather than project age or product popularity;
4. add capability profiles such as resources, prompts, tasks, or MRTR only when their PASS claim can be stated precisely;
5. graduate a new real client only after it satisfies the same established adapter criteria.

Current research candidates such as GitHub Copilot CLI, VS Code, and ChatGPT remain research-only until a supported or deliberately accepted direct boundary can prove the required real-client evidence safely.

A `v0.9.x` release with **zero new clients** is a successful release if evidence quality, tested coverage, and adapter maturity improve materially.

### Exit criteria

Existing adapters have documented tested envelopes and maturity status. Any newly shipped client or capability meets the same evidence/isolation/cleanup standard instead of relying on a weaker special case.

## v0.10.x — Public contract candidate

**GitHub tracking:** [#106](https://github.com/git-ksk/mcp-interop/issues/106). Focused audit/fix Issues are created when contract review starts.

### Goal

Prepare the public contracts that the project may eventually promise to keep stable across `v1.x`.

### Required work

Review and, where necessary, stabilize:

- CLI command and flag semantics;
- primary JSON output compatibility;
- portable artifact schema evolution and cross-schema comparison/migration policy;
- reason-code naming and compatibility policy;
- exit-code contract;
- adapter identity and maturity-state semantics;
- core/capability-profile meanings;
- protocol-era compatibility policy;
- baseline and compatibility-report semantics;
- deprecation/removal policy;
- release, security, privacy, and cleanup guarantees.

Portable artifacts are evidence records, not cryptographic attestations by default. If provenance signing, attestations, or tamper evidence are later added, their trust model must be explicit. A v1 contract must not imply that an unsigned local artifact proves who executed it.

### Exit criteria

The public surface can be reviewed and the maintainers can reasonably say: **we are prepared to preserve these meanings across v1.x, or deliberately version/migrate them when evolution is necessary.**

If a fundamental CLI, schema, or evidence-model redesign still appears likely, the project remains in `v0.x`.

## v0.11.x and later — Stabilization buffer

No feature set is reserved for `v0.11.0` or any later pre-1.0 minor release.

Use additional `v0.x` milestones for issues discovered by real use, including:

- unexpected modern/legacy client differences;
- artifact schema migration gaps;
- cross-platform lifecycle or cleanup problems;
- baseline/manifest usability problems;
- new OAuth behavior;
- a real-client automation surface changing or disappearing;
- protocol revisions/extensions that require evidence-model adaptation.

There is no penalty for shipping `v0.12.0`, `v0.13.0`, or later instead of declaring `v1.0.0` prematurely.

## v1.0.0 — Stable-contract exit criteria

`v1.0.0` ships only when all of the following categories are satisfied.

### Evidence correctness

- The core live PASS has a precise documented meaning.
- PASS requires evidence from the real client under test; fixture/configuration/metadata/diagnostic evidence cannot substitute for deployment-specific real-client evidence.
- Protocol-era differences are normalized without pretending unobserved protocol details are known.
- Exact client version and relevant platform/runtime/auth context are retained wherever the client surface permits safe observation.
- Ambiguous evidence remains fail-closed or `unknown`.

### Stable real-client adapters

For every adapter declared stable:

- isolation from normal client configuration/credential state is documented and tested;
- exact version capture is bounded and deterministic;
- timeout/cancellation and owned-process cleanup are bounded;
- controlled fixtures prove the claimed measurement path;
- compatibility evidence exists across enough realistic versions/platforms to justify the documented envelope;
- client-surface changes fail conservatively rather than creating false PASS.

The number of supported clients is **not** itself a v1 exit criterion.

### Regression operation

The project provides a coherent local-first path for:

- portable versioned artifacts;
- suite/multi-client execution;
- run/baseline comparison;
- intentional baseline lifecycle;
- evidence-derived compatibility reports/matrices;
- CI regression gating;
- explicit retry/flake semantics;
- trusted execution policy for self-hosted real-client runs.

### Public stability

The project is prepared to preserve or deliberately version/migrate:

- CLI behavior;
- primary JSON contracts;
- artifact schemas;
- exit codes;
- stable reason codes;
- adapter IDs and maturity semantics;
- core/capability-profile meanings.

### Security and privacy

- secret-bearing values are rejected/redacted from output and portable evidence;
- normal-user credentials are not copied into test profiles merely to make tests pass;
- OAuth authorization material is not persisted or exposed outside the explicit safe flow;
- deployment identity privacy has a documented sharing/baseline model;
- self-hosted CI cannot be driven by untrusted changes into arbitrary privileged execution;
- cleanup targets only temporary state/processes owned by the test session;
- release provenance and security gates remain active.

### Scope boundary

A v1 release still does not claim to be:

- a replacement for official MCP Conformance;
- a security certification or scanner;
- an LLM tool-selection benchmark;
- a brittle GUI/DOM automation framework;
- a hosted SaaS requirement;
- proof that every MCP feature or every MCP client works;
- proof that arbitrary production tools are safe to invoke.

## Decision gate before every minor release

Before assigning a planned capability to the next minor release, ask:

1. Does it improve the trustworthiness or operational usefulness of deployment-specific real-client evidence?
2. Can it preserve the real-client-only PASS boundary?
3. Can it be implemented without copying normal-user credentials or creating an unsafe automation surface?
4. Does it belong here rather than in official MCP Conformance, an inspector, a security tool, or a model benchmark?
5. Is the evidence surface actually observed and stable enough to implement, or is the work still research?

If those answers are weak, defer the feature even if a roadmap milestone appears to have room for it.

## Protocol references informing the roadmap

The protocol-aware milestones are informed by the official MCP `2026-07-28` release and schema:

- [`2026-07-28` specification release](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/blog/content/posts/2026-07-28-spec-ga/index.md)
- [`2026-07-28` schema](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/schema/2026-07-28/schema.ts)

These references explain why this roadmap does not make the legacy initialization handshake a permanent universal interoperability stage.
