# Suite baseline v1

[English](suite-baseline-v1.md) | [日本語](suite-baseline-v1.ja.md)

Suite baseline v1は、v0.8の退行テスト向けに追加するimmutableなlocal baseline contractです。
すでに検証済みのsuite result setを固定するだけで、新しい相互運用証拠を作りません。

## accept境界

baseline作成そのものを明示的なaccept操作にします。

```console
mcp-interop baseline create suite-results --output-dir baselines/codex-current
```

sourceには、宣言された全runについてschema v2の**real-client adapter**証拠とexact client versionが
必要です。execution error、artifact欠落、runner-only observation、client version欠落はbaselineに
acceptしません。

retry、client auto-update、後続の`suite run`がbaselineを書き換えることはありません。CLIは
output directoryを排他的なdirectory作成で先に確保し、既存destinationを必ず拒否します。
コピーしたresult setはその予約済みdirectory内でprivate stagingし、最後に`baseline.json`を
書くため、途中状態はvalidなaccepted baselineとして読めません。

## layout

```text
baseline/
  baseline.json
  results/
    index.json
    artifacts/
      production-a--codex--none.json
```

`results/`はsuite result-set v1のコピーです。source directoryへのpointerではありません。

## descriptor

`baseline.json`は次の形です。

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/suite-baseline",
  "created_at": "2026-08-28T08:00:00Z",
  "manifest_fingerprint": "sha256:...",
  "execution_context": "trusted_real_client",
  "result_set_digest": "sha256:...",
  "supersedes": "sha256:..."
}
```

`result_set_digest`は、validated result indexと参照される全live-result artifactのlogical JSON
contentをdeterministicに固定します。readerは再計算し、snapshotの変更を拒否します。

descriptorにはendpoint origin/path、executable path、token、OAuth credential、source filesystem
pathを重複保存しません。コピーされたschema v2 artifactには既存のcanonical origin privacy境界が
そのまま残るため、`credential-safe != deployment-public`は引き続き成立します。

## 意図的な置換

baselineはin-place更新しません。新しいcomparable result setをacceptするときは別directoryを作り、
旧baselineを明示します。

```console
mcp-interop baseline create suite-results-new \
  --output-dir baselines/codex-next \
  --supersedes baselines/codex-current
```

新descriptorの`supersedes`には旧baseline fingerprintが入り、旧directoryは変更されません。

manifest fingerprintまたはexecution contextが違う場合、supersedeはfail-closedです。両側に証拠が
あるrunでdeployment fingerprintまたはplatformが変わった場合も拒否します。client versionだけの
変更は許可します。version changeだけではregressionではありません。

暗黙のmutableな"current baseline" pointerは持ちません。baseline pathを毎回明示して選択するため、
retryやauto-update後のclientが勝手に比較基準になることを防ぎます。

## 比較

選択したimmutable baselineと、保持した全attemptを比較します。

```console
mcp-interop baseline compare baselines/codex-current attempt-1 attempt-2
mcp-interop baseline compare baselines/codex-current attempt-1 attempt-2 --json
mcp-interop baseline compare baselines/codex-current attempt-1 attempt-2 --fail-on-regression
```

出力は既存のsuite regression report v1のままです。result-set v1、live-result artifact v2、
suite regression report v1の既存consumerにmigrationは不要です。

比較前にbaseline専用のidentity checkを行い、manifest、execution context、deployment fingerprint、
platform mismatchはfail-closedにします。current evidence欠落はv0.7の既存semanticsどおり
regression/unstable evidenceとして残り、retryも全件保持します。

exit behaviorも`suite compare`と同じです。

- valid comparisonでgate非発火なら`0`
- `--fail-on-regression`指定時にregressionまたはunstable evidenceがあれば`1`
- invalid/unreadable/mutated/incomparableなbaseline/inputは`2`

## compatibility / migration

Suite baseline v1は新しいwrapper schemaで、次の既存contractは変更しません。

- live-result artifact v1/v2
- suite manifest v1
- suite result-set v1
- suite regression report v1
- real-client-only live PASSの意味

将来の未対応baseline schema versionは推測や暗黙migrationをせず拒否します。
