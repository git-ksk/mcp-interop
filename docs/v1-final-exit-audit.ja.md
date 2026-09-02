# v1.0.0 最終exit audit

[English](v1-final-exit-audit.md) | [日本語](v1-final-exit-audit.ja.md)

このauditは、代表的real-client regression acceptance完了後のmainに対して、`docs/roadmap.ja.md`で確定した`v1.0.0` exit criteriaを適用します。

## Decision

**PASS — project-level v1 exit blockerは残っていません。**

shipped Codex / Cursor / Antigravity adapterは引き続き`beta`です。これは意図した状態で、project contract stabilityとadapter maturityは別axisのためproject-level v1 blockerではありません。このauditでadapterをpromoteしません。

## PASS / GAP / N/A

| Category | Result | Retained basis |
| --- | --- | --- |
| Evidence correctness | PASS | `semantic-contract-v1.ja.md`、live-result schema、protocol-era observation、conservative non-PASS / unknown test |
| Stable real-client adapters | N/A | shipped adapterに`stable`宣言なし。`adapter-maturity.ja.md`はstable criterionが1つでもnon-`met`ならpromotionをfail-close |
| Regression operation | PASS | suite/result/baseline/compare/compatibility contractと`v1-real-client-regression-acceptance.ja.md` |
| Public stability | PASS | `public-contract-v1.ja.md`、`schema-evolution-v1.ja.md`、`semantic-contract-v1.ja.md`、stable reason-code policy、contract regression test |
| Security / privacy | PASS | `security-contract-v1.ja.md`、security contract gate、self-hosted trust guard、owned cleanup、release provenance workflow |
| Scope boundary | PASS | roadmap / project-direction / conformance-vs-interopで非目標を明示し、PASS意味を拡張していない |

## 最終operational verification

main `686d75aaf61b9ad4da70df9ea5c0fc2b5cec4c15`で以下を確認しました。

- `go test ./...` — PASS
- `go vet ./...` — PASS
- `bash scripts/test-security-contract.sh` — PASS
- `bash scripts/test-real-client-e2e-guard.sh` — PASS
- 通常の`scripts/build-release.sh`によるrelease-candidate audit build — Darwin / Linux / Windows × amd64 / arm64でPASS
- 代表的real-client regression acceptance — PASS。`v1-real-client-regression-acceptance.ja.md`に保持

release workflowはさらにformat、race、vulnerability、OAuth fixture、CLI regression、provenance、artifact attestation、tag-to-main ancestry gateを維持します。

## Adapter maturity note

現在のshipped adapter maturityは`beta`です。

- Codex: repeated exact-version coverage / advertised-platform coverageがstable blocker。
- Cursor: 上記2点に加えmeasurement-surface stabilityがlimited。
- Antigravity: 上記2点に加えmeasurement-surface stabilityがlimited。

これらはadapter-level `stable` promotionだけをblockします。stable project v1 contractの意味は弱めません。

## Release boundary

このauditが許可するのは**release decision**までで、自動tagではありません。`v1.0.0` tag作成は明示的release actionとして通常のtagged-release workflow / provenance gateを必ず通します。
