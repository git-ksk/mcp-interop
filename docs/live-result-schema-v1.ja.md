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

artifactには**完全なraw target URLを保存しません**。schema v1はcredentialが入りやすいURL要素を除外しますが、deployment identityとしてpathは意図的に保持します。

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

**schema v1はURL path自体が秘密情報ではないことを前提にします。** たとえば`https://example.com/mcp/<opaque-secret>`のようにAPI key、bearer相当token、signed capabilityなどをpathへ埋め込むdeploymentでは、そのendpointのv1 portable artifactをexportしないでください。credentialはpathではなく通常のMCP/OAuth認証で扱うことを推奨します。protected path対応は、明示的な非secret deployment identityを持つartifact v2としてIssue #87で設計します。

`endpoint.fingerprint`は、artifactへ保存するv1 identityのSHA-256です。完全なraw URLのうち、除外したquery/userinfo/fragmentはhashしません。一方、pathはv1 identityに含まれるため、fingerprintにも同じ「pathは非secret」という前提が適用されます。

その代わり、queryだけが違う2つのdeploymentをv1では区別できません。これはquery materialを持ち出さないことを優先した意図的なtrade-offです。

## Runに保存する情報

各runには次を保存します。

- UTCの`executed_at`
- 上記v1 path-safety前提に基づくendpoint identity / fingerprint
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

- JSON decode前にartifact入力を4 MiBへ制限
- `schema_version`は`1`
- `artifact_type`は`mcp-interop/live-results`
- 未知JSON fieldは拒否
- runは1件以上必要
- artifact内のcomparison identityは重複不可
- `executed_at`はUTC
- fingerprintはcanonicalなv1 identityと一致
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
- new stageがnon-PASSのまま、reason codeが追加・削除・変更された場合の`REASON_CODE_CHANGED`
- `NEW_EVIDENCE_MISSING`

stageがnon-PASSから`pass`へ回復した場合は、failure reason codeが消えたことも含めて`stage_changes`には表示しますが、その回復自体をregression扱いにはしません。new artifactにしか存在しないrunも表示しますが、それだけではregressionではありません。

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
