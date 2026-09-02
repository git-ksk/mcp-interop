# v1代表real-client regression acceptance

[English](v1-real-client-regression-acceptance.md) | [日本語](v1-real-client-regression-acceptance.ja.md)

この記録は、v1 stable-contract exit criteriaで要求し、[#162](https://github.com/git-ksk/mcp-interop/issues/162)で追跡した代表real-client regression acceptanceを完了するものです。

## Acceptance identity

- 実施日: 2026-09-03 (JST)
- Repository commit: `e18081ae30c9d32e526bb1d7bd15c6b5411392b6`
- Release-candidate build: `v1.0.0-rc.1`
- Build path: `scripts/build-release.sh`
- Darwin arm64 release archive SHA-256: `9cec0ebbebd63b88605db32647bad325ae1dcd334b460a9c916d8d37b7529ae2`
- 代表adapter: Codex CLI
- Exact real-client version: `codex-cli 0.133.0`
- Runner platform: macOS / darwin arm64
- Target: controlled localhost MCP fixture。production toolは呼び出していません。

## Acceptance前のsafety gate

regression workflow直前に、同じmain commitでCodexだけを対象に`scripts/e2e-real-clients.sh`を実行し、次のrelease gateがすべてPASSしました。

- real-client protocol E2E: PASS
- user configuration unchanged: PASS
- login Keychain DB unchanged: PASS
- no new real-client processes: PASS
- no leaked `mcp-interop` session directories: PASS
- `tools/call` avoided: PASS

controlled fixtureは実Codex経路を観測し、resultは`reach/auth/init/tools=PASS`でした。

## Release-candidate contract smoke

抽出したDarwin arm64 RC binaryを`scripts/cli-regression-smoke.sh`へ渡し、全smokeがPASSしました。ここには、不完全なevidenceを保持し`PASS_TO_UNKNOWN`でnon-zero gateを要求する既存regression safety checkと、fixture-backed suite/baseline regression workflowが含まれます。

このsmoke単体をreal-client evidenceとは扱いません。real-client workflowをacceptする前に、同じRC binaryがretry/regressionとportable contract semanticsを維持していることを確認するものです。

## 代表real-client regression workflow

抽出したRC binaryと`trusted_real_client` suite manifestを使い、同一controlled localhost targetを実Codex adapter経由で2回実行しました。

1. 1回目の`suite run`: PASS、`reach/auth/init/tools=PASS`
2. `baseline create`: 1回目のreal-client result setをaccept
3. 2回目の`suite run`: PASS、`reach/auth/init/tools=PASS`
4. `baseline compare --fail-on-regression --json`: `decision=clean`、`has_regression=false`、`has_unstable=false`
5. `compatibility matrix`: exact Codex `0.133.0` pointを`tested`と分類

fixtureは2回のcomplete client protocol exchangeを直接観測し、`initialize`、`notifications/initialized`、`tools/list`をそれぞれ2回記録しました。`tools/call`は不要でした。

## Retained evidence identity

- Suite manifest fingerprint: `sha256:13746fb738ba300b667ed97446c955d8c1ec58feaa7499bdd5fb98ef1281e5ea`
- Accepted baseline fingerprint: `sha256:41addafc42f2fa3b18cc538529c96c10d7dc2589dd3204e3519b84ae3e411a0c`
- Baseline result-set digest: `sha256:3eaeff7b1802b544a189ef869ea4b655b4a7231d58961b71d45862db0b155348`
- Baseline `baseline.json` SHA-256: `7ade41c21cb6cc0d83554dad023ec59248c5e57b12cd38eb7efbf04c8325b047`
- 1回目run artifact SHA-256: `a032262bc7d74bd782453072f6f6679049ea60149ae8852b704b2dc3773b1789`
- 2回目run artifact SHA-256: `d0e87930eacf6459654b7fa166988c3017b4ecaff39b36ddbb2cf0bf3dfb5e63`
- Suite index SHA-256（両run）: `c25947fbaf4d42c7a0ba21b2f0821a50bf94ae5d870e2f1aebf4f95fac1270fa`

portable schema-v2 evidenceは次を保持しました。

- `evidence_provenance.kind=real_client_adapter` / `adapter_id=codex`
- exact client version `codex-cli 0.133.0`
- `platform.os=darwin` / `platform.arch=arm64`
- runtime `mcp_interop_version=v1.0.0-rc.1`とexact repository commit
- 4 stageすべてPASS

raw protected endpoint pathはretained result set、baseline、comparison report、compatibility matrixに含まれていませんでした。controlled localhost originはnon-secretなので観測可能です。acceptance終了後にfixture processや新しいacceptance session directoryは残っていません。

## v1 decisionへの影響

PR #163で追加したv1 operational criterionはこのacceptanceで満たされます。ただし、Codexを`beta`から`stable`へpromoteするものではなく、advertised platform/version evidenceを広げるものでも、adapter maturity decisionを変更するものでもありません。project-level v1 contract stabilityとadapter maturityは引き続き別axisです。
