# Live interoperability result artifact schema v2

[English](live-result-schema-v2.md) | [日本語](live-result-schema-v2.ja.md)

schema v2は、URL pathを保存してはいけないRemote MCP endpoint向けに、operatorが明示的に指定するdeployment identityを追加します。

schema v1はdeployment identityとしてURL pathを意図的に保持するため、次のようなcredential-bearing pathを安全に表現できません。

```text
https://example.com/mcp/<opaque-capability>
```

schema v1は従来どおりサポートし、semanticsも変更しません。`mcp-interop test ... --output`は、`--deployment-id`を指定しない限りv1を書き出します。

## protected-path artifactを作る

```console
mcp-interop test 'https://example.com/mcp/<protected-path>' \
  --client codex \
  --output result.json \
  --deployment-id production-a
```

`--deployment-id`には`--output`が必要です。deployment IDはそのままartifactへ保存されるため、operatorが選んだ安定した**非secret**識別子でなければなりません。protected URL pathやcredentialをcopy、encode、truncateするなどして作ってはいけません。

現在のCLIでは1〜128 byte、ASCII英数字と`.`、`_`、`-`だけを許可します。

protected-path modeでは、通常のtext/JSON結果でもendpoint pathを再表示せず、canonical originだけを出力します。既存JSONのtop-level result array契約はそれ以外変更しません。

## document shape

```json
{
  "schema_version": 2,
  "artifact_type": "mcp-interop/live-results",
  "runs": [
    {
      "executed_at": "2026-08-27T12:00:00Z",
      "endpoint": {
        "identity": "production-a",
        "fingerprint": "sha256:<hex>",
        "identity_kind": "deployment_id",
        "origin": "https://example.com"
      },
      "client": {
        "id": "codex",
        "product": "Codex CLI",
        "version": "codex-cli 1.0.0"
      },
      "platform": {
        "os": "darwin",
        "arch": "arm64"
      },
      "runtime": {
        "mcp_interop_version": "v0.x.y",
        "mcp_interop_commit": "<commit>",
        "go_version": "go1.x.y"
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

run、client、platform、runtime、provenance、stage順序、status、reason codeのsemanticsはv1から変更しません。

## Endpoint identityとsecret safety

schema v2が現在サポートするendpoint identity modeは1つです。

```text
identity_kind = deployment_id
```

このmodeでは:

- `endpoint.identity`はoperator指定の非secret deployment ID
- `endpoint.origin`は`http(s)://lowercase-host[:explicit-port]`だけ
- endpoint pathは保存しない
- endpoint pathはhashしない
- query、userinfo、fragmentも保存・hashしない
- path shape、entropy、credentialらしい名前などのheuristicをsecurity boundaryにしない

`endpoint.fingerprint`はdomain separationした`deployment_id\0<origin>\0<identity>`のSHA-256です。これは、すでに非secretであるidentity materialに対するdeterministic convenienceでしかありません。**privacy mechanismではなく**、secretなdeployment IDを安全にする用途には使えません。

protected pathそのものは意図的にhashしません。低entropyのpath secretをplain stable hashにするとoffline guessing oracleになるためです。

### public originの境界

schema v2はcredential-bearing pathを除去しますが、canonical originは保存します。

```text
protected-path-safe != origin-private
```

hostname/origin自体が運用上privateなら、v2だからという理由だけでartifactを公開・commitしないでください。v0.6のdeployment privacyでは、このより広いsharing boundaryを引き続き明示します。

## Comparison identity

v2 runのpairing keyは次です。

```text
endpoint.origin + endpoint.identity
+ client.id
+ auth_mode
+ platform.os
+ platform.arch
```

exact client version、実行時刻、runner/runtime versionはpairing keyに含めません。canonical originはdeployment identityの一部ですが、protected pathの変化は意図的に無視します。同じdeployment IDを別originで再利用しても、異なるdeploymentを暗黙にpairしません。

同じartifact内でv2 comparison identityが重複した場合はrejectします。比較対象のartifact set内で一意になるdeployment IDを選んでください。

機械可読なcomparison outputはv2 artifact同士の場合comparison report `schema_version: 2`を使います。既存v1 comparisonは従来どおりcomparison report schema v1を出力します。

## v1 compatibilityとmigration

schema v1は既存のpath/fingerprint semanticsのままread/write/compareできます。

schema versionを暗黙に混在させません。

- v1 vs v1: 従来のv1 pairing semanticsで対応
- v2 vs v2: deployment ID pairing semanticsで対応
- v1 vs v2: input errorとしてreject（exit `2`）

v1のURL-path identityとv2のoperator identityには、安全に推測できる暗黙対応関係がありません。そのためcross-schema comparisonはguessしません。新しいv2 baselineを作る場合は、非secretな`--deployment-id`を選んで対象を再実行し、それ以後同じIDのv2 artifact同士を比較してください。

v2からv1への自動downgradeも行いません。credential-bearing pathをportable artifactへ再導入する可能性があるためです。

## Validationとfile safety

schema v2もv1のfile-safety ruleを維持します。

- JSON decode前にartifact input sizeを制限
- unknown JSON fieldをreject
- `artifact_type`は`mcp-interop/live-results`
- 少なくとも1 run必要
- comparison identityはartifact内でunique
- `executed_at`はUTC
- stageは`reach`, `auth`, `init`, `tools`の4つ
- statusは`pass`, `fail`, `skip`, `unknown`
- runner-only observationからPASSを生成しない
- output replacementは既存のprivate atomic-write pathを利用

schema v2が変更するのはendpoint identity表現だけです。real-client PASS boundaryを弱めず、新しいevidence sourceも追加しません。
