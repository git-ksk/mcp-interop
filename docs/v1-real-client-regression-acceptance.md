# v1 representative real-client regression acceptance

[English](v1-real-client-regression-acceptance.md) | [日本語](v1-real-client-regression-acceptance.ja.md)

This record closes the representative real-client regression acceptance required by the v1 stable-contract exit criteria and tracked in [#162](https://github.com/git-ksk/mcp-interop/issues/162).

## Acceptance identity

- Date: 2026-09-03 (JST)
- Repository commit: `e18081ae30c9d32e526bb1d7bd15c6b5411392b6`
- Release-candidate build: `v1.0.0-rc.1`
- Build path: `scripts/build-release.sh`
- Darwin arm64 release archive SHA-256: `9cec0ebbebd63b88605db32647bad325ae1dcd334b460a9c916d8d37b7529ae2`
- Representative adapter: Codex CLI
- Exact real-client version: `codex-cli 0.133.0`
- Runner platform: macOS / darwin arm64
- Target: controlled localhost MCP fixture, no production tool invocation

## Pre-acceptance safety gate

Immediately before the regression workflow, `scripts/e2e-real-clients.sh` was run for Codex only on the same main commit. It passed all release gates:

- real-client protocol E2E: PASS;
- user configuration unchanged: PASS;
- login Keychain DB unchanged: PASS;
- no new real-client processes: PASS;
- no leaked `mcp-interop` session directories: PASS;
- `tools/call` avoided: PASS.

The controlled fixture observed the real Codex path and the result reported `reach/auth/init/tools=PASS`.

## Release-candidate contract smoke

The extracted Darwin arm64 RC binary was passed to `scripts/cli-regression-smoke.sh`; the complete smoke returned PASS. This includes the existing regression safety checks that preserve incomplete evidence and require a non-zero gate for `PASS_TO_UNKNOWN`, as well as the fixture-backed suite/baseline regression workflow.

This smoke is not real-client evidence by itself. It verifies that the same RC binary still enforces the retry/regression and portable-contract semantics before the real-client workflow below is accepted.

## Representative real-client regression workflow

Using the extracted RC binary and a `trusted_real_client` suite manifest, the same controlled localhost target was exercised twice through the real Codex adapter.

1. First `suite run`: PASS, `reach/auth/init/tools=PASS`.
2. `baseline create`: accepted the first real-client result set.
3. Second `suite run`: PASS, `reach/auth/init/tools=PASS`.
4. `baseline compare --fail-on-regression --json`: `decision=clean`, `has_regression=false`, `has_unstable=false`.
5. `compatibility matrix`: exact Codex `0.133.0` point classified as `tested`.

The fixture directly observed two complete client protocol exchanges: two `initialize`, two `notifications/initialized`, and two `tools/list` requests. No `tools/call` was required.

## Retained evidence identities

- Suite manifest fingerprint: `sha256:13746fb738ba300b667ed97446c955d8c1ec58feaa7499bdd5fb98ef1281e5ea`
- Accepted baseline fingerprint: `sha256:41addafc42f2fa3b18cc538529c96c10d7dc2589dd3204e3519b84ae3e411a0c`
- Baseline result-set digest: `sha256:3eaeff7b1802b544a189ef869ea4b655b4a7231d58961b71d45862db0b155348`
- Baseline `baseline.json` SHA-256: `7ade41c21cb6cc0d83554dad023ec59248c5e57b12cd38eb7efbf04c8325b047`
- First run artifact SHA-256: `a032262bc7d74bd782453072f6f6679049ea60149ae8852b704b2dc3773b1789`
- Second run artifact SHA-256: `d0e87930eacf6459654b7fa166988c3017b4ecaff39b36ddbb2cf0bf3dfb5e63`
- Suite index SHA-256 (both runs): `c25947fbaf4d42c7a0ba21b2f0821a50bf94ae5d870e2f1aebf4f95fac1270fa`

The portable schema-v2 evidence retained:

- `evidence_provenance.kind=real_client_adapter` and `adapter_id=codex`;
- exact client version `codex-cli 0.133.0`;
- `platform.os=darwin`, `platform.arch=arm64`;
- runtime `mcp_interop_version=v1.0.0-rc.1` and exact repository commit;
- all four stage statuses as PASS.

The raw protected endpoint path was absent from the retained result sets, baseline, comparison report, and compatibility matrix. The controlled localhost origin is non-secret and may remain observable. No acceptance fixture process or new acceptance session directory remained after execution.

## v1 decision impact

The operational v1 criterion added in PR #163 is satisfied by this acceptance. This does **not** promote Codex from `beta` to `stable`, broaden its advertised platform/version evidence, or change any adapter maturity decision. Project-level v1 contract stability and adapter maturity remain separate axes.
