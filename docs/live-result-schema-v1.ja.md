# Live interoperability result artifact schema v1

[English](live-result-schema-v1.md) | **日本語**

> この文書は英語版`live-result-schema-v1.md`の日本語訳です。schemaの正確な契約は英語版を正とします。

このschemaは、**特定のRemote MCPを実クライアントで検証した結果を保存し、あとから比較するためのportable artifact**を定義します。

既存の`mcp-interop test --json`出力とは意図的に別契約です。

目的は、`reach -> auth -> init -> tools`の意味を弱めずに、クライアントのバージョン更新前後で退行をローカルファイルだけで比較できるようにすることです。

## 既存JSON出力は変えない

従来のコマンド:

```console
mcp-interop test https://example.com/mcp --client codex --json
```

は、これまでどおりlive `Result`のJSON配列を標準出力へ返します。

artifact schema v1を導入しても、この既存payloadへ`schema_version`やtimestampなどを追加しません。

artifactは明示した場合だけ別ファイルへ保存します。

```console
mcp-interop test https://example.com/mcp --client codex --output result.json
```

`--json`と`--output`は併用できます。`--output`は標準出力を置き換えません。

## 全体構造

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/live-results",
  "runs": [
    {
      "executed_at": "2026-08-12T15:00:00Z",
      "endpoint": {
        "identity": "https://example.com/mcp",
        "fingerprint": "sha256:..."
      },
      "client": {
        "id": "codex",
        "product": "Codex CLI",
        "version": "codex-cli 0.147.0"
      },
      "platform": {
        "os": "darwin",
        "arch": "arm64"
      },
      "runtime": {
        "mcp_interop_version": "v0.5.0",
        "mcp_interop_commit": "...",
        "go_version": "go1.24.x"
      },
      "auth_mode": "default",
      "evidence_provenance": {
        "kind": "real_client_adapter",
        "adapter_id": "codex"
      },
      "stages": [
        {"stage": "reach", "status": "pass"},
        {"stage": "auth", "status": "pass"},
        {"stage": "init", "status": "pass"},
        {"stage": "tools", "status": "pass"}
      ]
    }
  ]
}
```

各stageには、必要に応じて既存のstable `reason_code`を含められます。

一方、人間向けの詳細messageやdiagnostic payloadはv1 artifactへ保存しません。

理由は2つです。

1. regression比較に必須ではない
2. portable fileへ秘密情報を持ち出す面を減らせる

## Endpoint identityと秘密情報

artifactには**raw target URLを保存しません**。

`endpoint.identity`へ残すのは次の形だけです。

```text
http(s)://lowercase-host[:explicit-port]/path
```

次は除外します。

- userinfo
- query parameter
- query value
- fragment

query parameter名が秘密情報らしく見えなくても、query値は例外なく削除します。

`endpoint.fingerprint`は、この安全化したidentityのSHA-256です。

raw URLをhashしないため、secretそのものから派生したfingerprintを残しません。

その代わり、queryだけが違う2つのdeploymentをv1では区別できません。これは秘密情報保護を優先した意図的なtrade-offです。

## Runに保存する情報

各runには次を保存します。

- UTCの`executed_at`
- 安全化したendpoint identity / fingerprint
- 実クライアントrunではclient ID / product / 正確なversion
- OS / architecture
- `mcp-interop` version / commit / Go runtime version
- 実行時のauth mode（`default`または明示的`oauth`）
- 証拠の出所
- 順序固定の`reach` / `auth` / `init` / `tools`

`auth_mode`は「runnerをどう起動したか」を表します。server metadataから認証要否を推測した値ではありません。

## 証拠の出所とPASS

`evidence_provenance.kind`は次の2種類です。

- `real_client_adapter` — 実際にインストールされたクライアントを実行したrun。`adapter_id`と正確なclient versionが必須
- `runner_observation` — client未installなど、アダプター実行前にrunner自身が観測した状態

`runner_observation`へ`pass` stageを入れることは禁止します。

artifact層は新しいPASS証拠を作らず、アダプター結果を都合よく再解釈しません。

完全なlive PASSは従来どおり、4stageすべてが実クライアントの証拠で`pass`である場合だけです。

## Strict validation

比較入力として読むときは厳密に検証します。

- `schema_version`は`1`
- `artifact_type`は`mcp-interop/live-results`
- 未知JSON fieldは拒否
- runは1件以上必要
- artifact内のcomparison identityは重複不可
- `executed_at`はUTC
- fingerprintはcanonical identityと一致
- stageは`reach`, `auth`, `init`, `tools`をこの順序で1回ずつ
- statusは`pass`, `fail`, `skip`, `unknown`だけ

不足した証拠をsuccessへ補正しません。invalidなartifactは拒否し、validだが不確定な結果はnon-PASSのまま保持します。

## 比較時に同じrunとみなす条件

`mcp-interop compare`は次の組み合わせでbaselineとnew runを対応付けます。

```text
endpoint fingerprint
+ client.id
+ auth_mode
+ platform.os
+ platform.arch
```

client version、実行時刻、runner versionはcontextとして保存しますが、pairing keyには含めません。

これにより、クライアント更新後も前versionと比較できます。

versionだけ変わり、stage / reason evidenceが同じならregressionではありません。

## 退行の分類

```console
mcp-interop compare old.json new.json
mcp-interop compare old.json new.json --json
mcp-interop compare old.json new.json --fail-on-regression
```

明示的に分類するもの:

- `PASS_TO_FAIL`
- `PASS_TO_UNKNOWN`
- `PASS_TO_SKIP`
- `REASON_CODE_CHANGED`
- `NEW_EVIDENCE_MISSING`

new artifactにしか存在しないrunは表示しますが、それだけではregression扱いにしません。

## compareのexit code

- `0` — validな比較が完了。`--fail-on-regression`なしならregressionがあっても0
- `1` — `--fail-on-regression`指定時にregressionまたはbaseline evidence lossを検出
- `2` — usage error、入力file読込失敗、invalid/unsupported schema、出力失敗

通常の`mcp-interop test`のexit contractは変えません。

## Scope

schema v1はlocal-file-firstです。

次は導入しません。

- hosted service
- database / SQLite history
- dashboard
- 新client adapter
- generic MCP Conformance suite
- security scanner
- LLM evaluation layer

将来artifact contractを変更する場合、v1の意味を黙って変更せず、新しい`schema_version`を使います。
