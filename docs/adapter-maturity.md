# Adapter maturity contract

[English](adapter-maturity.md) | [日本語](adapter-maturity.ja.md)

Adapter maturity is an **evidence-reviewed lifecycle decision**. It is separate from the existing client `tier` field: `tier=v1` means a client belongs to the shipped/live-adapter delivery track, not that its evidence maturity is `stable`.

Use the current machine/human-readable decision report without detecting or executing any client:

```console
mcp-interop maturity
mcp-interop maturity --json
```

The JSON report uses `schema_version: 1` and `artifact_type: "mcp-interop/adapter-maturity"`. It contains no installed-client version, executable path, endpoint, credential, or user-state data.

## Report contract

Each decision contains:

- `client_id` / `display_name` — the shipped adapter identity;
- `tier` — the existing roadmap/delivery tier, intentionally separate from maturity;
- `maturity` — `research_only`, `beta`, or `stable`;
- `criteria` — the complete known criterion set, each with `met`, `limited`, or `missing`;
- `blockers` — every non-`met` stable criterion for a beta decision; none may be hidden;
- `evidence_refs` — retained repository evidence references such as `docs/...` or `github:pr/<number>`. They are references only; the command does not fetch them.

Unknown criteria, duplicate evidence references, undocumented beta blockers, and a `stable` decision with any non-`met` criterion fail validation. The report is a reviewed project decision, not a live probe.

## States

- `research_only` — a safe direct measurement boundary is not yet complete enough to ship as a live adapter.
- `beta` — the supported path meets the minimum evidence/safety gate, but one or more stable criteria are still limited or missing.
- `stable` — every beta and stable criterion below is explicitly met for the adapter's advertised scope.

Project age, popularity, release count, and nearby client versions are never maturity evidence.

## Project v1 stability and adapter maturity

Project-level contract stability and adapter maturity are separate release axes. A stable `v1.x` project release may ship one or more `beta` adapters. Publishing v1 does not auto-promote an adapter, broaden tested version/platform coverage, or turn `tier=v1` into stable maturity.

Each adapter is promoted only after every stable criterion is met for its advertised scope and the evidence-backed maturity decision is explicitly reviewed.

## Beta gate

Every beta adapter must have all of these criteria at `met`:

| Criterion ID | Requirement |
| --- | --- |
| `direct_real_client_boundary` | PASS semantics come from a real client through a direct supported/observed boundary, not a model prompt or server-only substitute. |
| `isolated_state` | Test configuration/auth state is isolated from normal user state. |
| `owned_cleanup` | Processes/sessions/temp state owned by the test are bounded and cleaned up. |
| `secret_safety` | Secret-bearing values are rejected/redacted and are not retained as interoperability evidence. |
| `conservative_failure_semantics` | Ambiguous evidence stays `unknown`/`skip`; PASS is not broadened to increase coverage. |
| `controlled_fixture_e2e` | A repeatable controlled real-client fixture gate proves the claimed core path. |
| `exact_version_platform_evidence` | Real-client evidence records the exact client version and runner platform context. |

If any beta criterion becomes unproven, a stronger maturity claim is blocked until explicit review. A client version string changing by itself does not prove that this happened.

## Stable gate

Stable requires every beta criterion plus all of these at `met`:

| Criterion ID | Requirement |
| --- | --- |
| `repeat_path_version_coverage` | Each advertised path that contributes to a PASS claim has retained real-client evidence across at least two exact client versions; the points are not interpolated into a version range. |
| `advertised_platform_coverage` | Every OS/architecture scope advertised as supported has retained exact real-client evidence, or the advertised scope is explicitly narrowed. Unit/hosted build success alone is not real-client platform evidence. |
| `measurement_surface_stability` | The client observation/control surface is sufficiently supported or repeatedly evidenced that the PASS boundary is not a one-build accident. |
| `regression_maintenance_path` | Exact-point compatibility, regression gates, failure semantics, and a documented revalidation path exist when the external client changes. |

`limited` means relevant evidence exists but is not strong enough for stable promotion. `missing` means the required evidence is absent. Either state blocks `stable`.

Stable does **not** create a semantic-version support range. A stable adapter can have an installed client version classified `untested` until that exact version is measured.

## Current shipped-adapter decisions

The pre-v1 review promotes **Codex CLI to stable for the explicitly documented macOS arm64 core path**. Cursor and Antigravity remain beta. Adapter maturity is an evidence review, not a regression result and not an automatic consequence of a version change.

| Adapter | Decision | Evidence supporting beta | Stable blockers |
| --- | --- | --- | --- |
| Codex CLI | `stable` | isolated app-server real-client boundary; retained controlled non-OAuth core PASS evidence at exact `0.133.0` and `0.152.1` on macOS arm64; cleanup/secret/failure gates; regression maintenance path | none for the advertised stable scope. The stable claim is intentionally limited to the documented macOS arm64 non-OAuth core path and does not imply Linux/Windows/macOS amd64 or OAuth stability. |
| Cursor CLI | `beta` | isolated supported MCP management/login path; OAuth evidence at exact `2026.08.04-aaa8809` (PR #39); controlled core evidence at exact `2026.08.25-3e8eec8` (PR #108) | `repeat_path_version_coverage`, `advertised_platform_coverage`, `measurement_surface_stability`: the two exact versions cover different path modes, real-client OS evidence is narrow, and MCP management output remains human-readable rather than a dedicated machine contract |
| Antigravity CLI | `beta` | isolated PTY/no-account boundary; OAuth evidence at exact `agy 1.1.11` (PR #40); controlled core evidence at exact `agy 1.1.22` (PR #108) | `repeat_path_version_coverage`, `advertised_platform_coverage`, `measurement_surface_stability`: OAuth/core versions differ, retained macOS evidence is arm64-only, and no-auth tool discovery relies on a bounded observed client cache surface |

The current exact coverage table is maintained in [Exact observed client coverage](observed-coverage.md). The Codex stable promotion evidence is retained in [Codex stable-adapter acceptance](codex-stable-acceptance.md). Historical PR evidence is retained only for the exact path/version claim stated by that PR; it does not turn nearby versions into tested points.

## Client version changes

Maturity is reviewed separately from exact compatibility:

1. an installed client version change alone does not promote, demote, or regress an adapter;
2. `compatibility query`/`compatibility matrix` classify the new exact point from retained evidence;
3. an unobserved new version is `untested` even if the adapter remains beta/stable;
4. new retained failures, cleanup/isolation failures, or repeated evidence gaps can trigger an explicit maturity review;
5. a maturity change is committed as a reviewed project decision with updated evidence rationale, never inferred from SemVer ordering.

Research candidates such as VS Code, GitHub Copilot CLI, ChatGPT, and Claude are not emitted by `mcp-interop maturity` because the command reports **shipped live adapters** only. Research does not become beta merely because detection or a partial PoC exists.
