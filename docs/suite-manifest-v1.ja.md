# Suite manifest v1

[English](suite-manifest-v1.md) | **日本語**

> この文書は英語版`suite-manifest-v1.md`の日本語訳です。契約の正本は英語版です。

Suite manifest v1は、v0.7の繰り返し検証で使う宣言形式です。`suite` CLIは内容を検証し、`trusted_real_client` manifestを実行できます。結果の比較方法は[Suite regression report v1](suite-regression-report-v1.ja.md)で定義済みです。`hosted_fixture`宣言はv0.7.0ではvalidation-onlyで、repository CIのcontrolled localhost fixtureは別の信頼済み経路で実行します。

## 内容を検証する

```console
mcp-interop suite validate suite.json
mcp-interop suite validate suite.json --json
```

validationはstrictです。unknown field、未対応schema version、unknown live-client ID、安全でないexecution contextの組み合わせ、任意のendpoint環境変数参照はexit code `2`でrejectします。

## 秘密情報をmanifestへ入れない

suite manifestにはRemote MCP endpoint URLそのものを保存しません。これは意図した設計です。

- `hosted_fixture` targetは`endpoint.source: "fixture"`のみで、OAuthを要求できません。
- `trusted_real_client` targetは`endpoint.source: "environment"`を使います。
- 環境変数名はtarget IDから決まり、manifest側で任意の変数名を選べません。
- trusted real-client targetには安定した非secret `deployment_id`が必須です。suite executionでは既存schema-v2 protected-path identityを使い、解決したpathをportable artifactへ保存もhashもしません。
- Bearer token、client secret、authorization code、cookie、実行ファイルpath、shell hook、環境変数override、literal endpoint URLはschema v1のfieldではありません。unknown fieldをrejectするため、これらを追加してtrust boundaryを暗黙拡張することもできません。

target IDが`production-a`の場合、受け付けるendpoint変数名は次だけです。

```text
MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A
```

変数の**値**はtrusted suite executionでのみ解決し、manifestやsuite indexには含みません。

## hosted fixtureの例

```json
{
  "schema_version": 1,
  "execution_context": "hosted_fixture",
  "targets": [
    {
      "id": "fixture-a",
      "endpoint": {"source": "fixture"},
      "clients": [
        {"id": "codex", "auth": "none"},
        {"id": "cursor", "auth": "none"}
      ]
    }
  ]
}
```

`hosted_fixture`はunprivilegedな宣言形です。任意network endpointを選べず、OAuthにもopt-inできません。

## trusted real-clientの例

```json
{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [
    {
      "id": "production-a",
      "endpoint": {
        "source": "environment",
        "variable": "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"
      },
      "deployment_id": "production-a",
      "clients": [
        {"id": "codex", "auth": "none"},
        {"id": "cursor", "auth": "oauth"}
      ]
    }
  ]
}
```

`oauth`はclientごとの明示指定で、`trusted_real_client` manifestだけが受け付けます。validation時には認証、endpoint変数値の読み込み、client起動を一切行いません。`suite run`は最初のclient起動前に宣言済みendpoint変数を全件解決・検証します。

## v1で使えるフィールド

Top level:

- `schema_version`: `1`のみ。
- `execution_context`: `hosted_fixture`または`trusted_real_client`。
- `targets`: 1件以上のtarget宣言。

Target:

- `id`: 1-63文字のlowercase ASCII letter/digit + 内部`-`。endpoint環境変数名の決定にも使います。
- `endpoint.source`: `fixture`または`environment`。execution contextで利用可能なsourceを制限します。
- `endpoint.variable`: trusted real-client targetだけ必須で、target IDから導出した名前と完全一致が必要です。
- `deployment_id`: trusted real-client targetだけ必須。既存の非secret deployment-ID構文を使います。
- `clients`: shipped live adapterの`codex` / `cursor` / `antigravity`を1件以上、重複なしで指定します。

Client selection:

- `id`: shipped live-adapter ID。
- `auth`: `none`または`oauth`。`oauth`は`trusted_real_client`のみ。

## 実行境界

`mcp-interop suite run <manifest.json> --output-dir <dir>`は現在`trusted_real_client` manifestだけを実行します。

- client起動前に全target環境変数を解決する
- 選択したclient/authごとに既存direct live-test経路を使う
- 各runをschema-v2 protected-path artifactとして保存する
- 既存output directoryを置換しない
- non-PASSや未インストールclientもdropせず保存する

`hosted_fixture`宣言はv0.7.0ではvalidation-onlyです。repositoryのPull Request CIはcontrolled localhost fixture gateを別経路で使い、privileged runnerで任意suite manifestを実行しません。retry / flake / regression semanticsは[Suite regression report v1](suite-regression-report-v1.ja.md)で定義済みです。manifestには後続処理が実行できるarbitrary command/hook機構を追加しません。
