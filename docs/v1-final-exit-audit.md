# v1.0.0 final exit audit

[English](v1-final-exit-audit.md) | [日本語](v1-final-exit-audit.ja.md)

This audit applies the finalized `v1.0.0` exit criteria in `docs/roadmap.md` to main after the representative real-client regression acceptance.

## Decision

**PASS — no unresolved project-level v1 exit blocker was found.**

The shipped Codex, Cursor, and Antigravity adapters remain `beta`. That is intentional and is not a project-level v1 blocker: project contract stability and adapter maturity are separate axes. No adapter is promoted by this audit.

## PASS / GAP / N/A

| Category | Result | Retained basis |
| --- | --- | --- |
| Evidence correctness | PASS | `semantic-contract-v1.md`, live-result schemas, protocol-era observations, conservative non-PASS/unknown tests |
| Stable real-client adapters | N/A | No shipped adapter is declared `stable`; `adapter-maturity.md` fail-closes stable promotion when any stable criterion is non-`met` |
| Regression operation | PASS | Suite/result/baseline/compare/compatibility contracts plus `v1-real-client-regression-acceptance.md` |
| Public stability | PASS | `public-contract-v1.md`, `schema-evolution-v1.md`, `semantic-contract-v1.md`, stable reason-code policy and contract regression tests |
| Security and privacy | PASS | `security-contract-v1.md`, security contract gate, self-hosted trust guard, owned cleanup and release provenance workflow |
| Scope boundary | PASS | Roadmap/project-direction/conformance-vs-interop retain explicit non-goals and do not broaden PASS |

## Final operational verification

On main `686d75aaf61b9ad4da70df9ea5c0fc2b5cec4c15`:

- `go test ./...` — PASS
- `go vet ./...` — PASS
- `bash scripts/test-security-contract.sh` — PASS
- `bash scripts/test-real-client-e2e-guard.sh` — PASS
- normal `scripts/build-release.sh` release-candidate audit build — PASS for Darwin/Linux/Windows on amd64/arm64
- representative real-client regression acceptance — PASS, retained in `v1-real-client-regression-acceptance.md`

The release workflow additionally preserves format, race, vulnerability, OAuth fixture, CLI regression, provenance, artifact attestation, and tag-to-main ancestry gates.

## Adapter maturity note

Current shipped adapter maturity is `beta`:

- Codex: stable blockers remain repeated exact-version coverage and advertised-platform coverage.
- Cursor: those two blockers plus measurement-surface stability remain limited.
- Antigravity: those two blockers plus measurement-surface stability remain limited.

These block adapter-level `stable` promotion only. They do not weaken or change the stable project v1 contract.

## Release boundary

This audit authorizes a **release decision**, not an automatic tag. Creating `v1.0.0` remains an explicit release action and must run through the normal tagged-release workflow and provenance gates.
