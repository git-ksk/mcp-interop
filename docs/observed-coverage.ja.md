# Exact observed client coverage

[English](observed-coverage.md) | **日本語**

このページは、repositoryに保持されている**exactなreal-client実測**だけを記録します。semantic-versionのsupport rangeではなく、記載clientであらゆるdeploymentが動くという主張でもありません。

## 主張の境界

exact client version、runner platform、real-client outcome、evidence sourceが保持されている場合だけobserved rowとして扱います。近いversionや別OSは、それぞれ固有の証拠がない限り`untested`です。

現在のhistorical rowは、[現行real-clientのprotocol-era観測](protocol-era-observations.ja.md)と[PR #108](https://github.com/git-ksk/mcp-interop/pull/108)に保持されたcontrolled localhost real-client release gate由来です。このgateは通常のnon-OAuth core pathを使い、`reach/auth/init/tools=PASS`、fixture上のprotocol readiness、`tools/call`なし、normal configuration / Keychain metadata / process / temporary-session cleanup gateのPASSを要求しました。

E2E harnessは実行後にtemporary result directoryを削除するため、2026-08-27のrowはschema-v2 suite result setではなくrepository上のobservation recordとして保持されています。したがってexact historical coverageの記録には使えますが、`mcp-interop compatibility query`や`compatibility matrix`の入力にはできません。今後のcoverageは可能な限りsuite result set / baselineを保持し、machine-readable compatibility modelで全attemptを残します。

## 現在保持しているexact observation

| Client | Exact version | Runner platform | Scope | Core result | Retained evidence |
| --- | --- | --- | --- | --- | --- |
| Codex CLI | `codex-cli 0.133.0` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/protocol-era-observations.ja.md`, [PR #108](https://github.com/git-ksk/mcp-interop/pull/108) |
| Codex CLI | `codex-cli 0.152.1` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS`、user config/Keychain/process/session cleanup PASS、`tools/call` avoided | current mainでの[Issue #170](https://github.com/git-ksk/mcp-interop/issues/170) acceptance run |
| Cursor CLI | `2026.08.04-aaa8809` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/cursor-stable-acceptance.ja.md`, issue #172 |
| Cursor CLI | `2026.08.25-3e8eec8` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/protocol-era-observations.ja.md`, [PR #108](https://github.com/git-ksk/mcp-interop/pull/108) |
| Cursor CLI | `2026.09.02-c22c1a3` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/cursor-stable-acceptance.ja.md`, issue #172 |
| Antigravity CLI | `1.1.22` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/protocol-era-observations.ja.md`, [PR #108](https://github.com/git-ksk/mcp-interop/pull/108), `docs/antigravity-stable-acceptance.ja.md` |
| Antigravity CLI | `1.1.24` | macOS 26.5 (25F71), arm64 | controlled localhost, non-OAuth core path | `reach/auth/init/tools=PASS` | `docs/antigravity-stable-acceptance.ja.md`, issue #173 |

この表はexact pointだけです。たとえばCursorの1行から、前後の`2026.08.*` buildまでsupportedとは推測しません。

Adapter-levelのbeta/stable decisionは別のreview layerです。[Adapter maturity contract](adapter-maturity.ja.md)を参照してください。exact versionがuntestedになっただけでadapter maturityを自動変更しません。

## OS/platform coverage state

| Client | macOS arm64 | macOS amd64 | Linux | Windows |
| --- | --- | --- | --- | --- |
| Codex CLI | exact point `0.133.0`、`0.152.1`をobserved。stable adapter scopeはここに限定 | untested | untested | untested |
| Cursor CLI | 上記exact pointをobserved | untested | untested | untested |
| Antigravity CLI | 上記exact pointをobserved | untested | shipped live adapterではunsupported | shipped live adapterではunsupported |

`untested`は、そのplatform pointについて保持されたexact real-client evidenceが無いという意味で、FAILではありません。`shipped live adapterではunsupported`はimplementation boundaryです。Antigravityのnon-macOS adapterは、real clientで同等のPTY/cache behaviorを検証できるまで意図的にSKIPを返します。

client unavailableやexecution errorもtested failureへ変換しません。machine-readable compatibility outputでは`known_broken` pointを捏造せず、`evidence_gaps`として保持します。

## 保持result setからmatrixを作る

`compatibility matrix`は、明示したaccept済みbaseline / suite result setから全exact pointを列挙します。client検出・client実行・Remote MCP endpointへの接続は行いません。

```console
mcp-interop compatibility matrix \
  --baseline baselines/current \
  --observation attempt-1 \
  --observation attempt-2
```

`--json`ではcompleteなcompatibility-envelope v1 objectを出力します。各exact pointについてsource/attempt、outcome、stage、provenance、regression情報を含む全observationを保持します。human outputにもobservation sequenceとattempt countを出すため、FAIL/UNKNOWN後のretry PASSをcleanな単発PASSへ見せかけません。

`--stale-on-client-version-change`を使う場合、繰り返した`--observation`はoldest -> newestの明示collection orderです。age stale判定は追加で`--max-age-seconds N --trust-executed-at-clock`が必要です。

## coverage追加ルール

- exact client version文字列を記録し、version rangeへ一般化しない
- runner/process OS・architectureとclient-binary architecture evidenceを分離する
- FAIL / UNKNOWN / SKIP / execution error / retryを保持し、成功したretryだけを公開しない
- controlled fixture evidenceとdeployment-specific claimを混ぜない
- coverage数を増やすためにproduction tool call、model prompt、normal-user credential、secret-bearing artifactを使わない
- 新しく検出したversion/platformでも、未観測ならnew real-client evidenceが出るまで`untested`とする
