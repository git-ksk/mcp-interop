# Suite manifest v1

[English](suite-manifest-v1.md) | **日本語**

> この文書は英語版`suite-manifest-v1.md`の日本語訳です。契約の正本は英語版です。

Suite manifest v1は、v0.7のrepeatable regression workflowで使う宣言契約です。現在の`suite` CLIはこの契約の**validationだけ**を行います。実行とregression reportは後続v0.7 Issueで追加します。

## Validate

```console
mcp-interop suite validate suite.json
mcp-interop suite validate suite.json --json
```

validationはstrictです。unknown field、未対応schema version、unknown live-client ID、安全でないexecution contextの組み合わせ、任意のendpoint環境変数参照はexit code `2`でrejectします。

## Secret-safety境界

suite manifestにはRemote MCP endpoint URLそのものを保存しません。これは意図した設計です。

- `hosted_fixture` targetは`endpoint.source: "fixture"`のみで、OAuthを要求できません。
- `trusted_real_client` targetは`endpoint.source: "environment"`を使います。
- 環境変数名はtarget IDから決まり、manifest側で任意の変数名を選べません。
- trusted real-client targetには安定した非secret `deployment_id`が必須です。後続のsuite executionでは既存schema-v2 protected-path identityを使い、解決したpathをportable artifactへ保存もhashもしない設計にします。
- Bearer token、client secret、authorization code、cookie、実行ファイルpath、shell hook、環境変数override、literal endpoint URLはschema v1のfieldではありません。unknown fieldをrejectするため、これらを追加してtrust boundaryを暗黙拡張することもできません。

target IDが`production-a`の場合、受け付けるendpoint変数名は次だけです。

```text
MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A
```

変数の**値**は後続のtrusted executionでのみ解決し、manifestには含みません。

## Hosted fixture例

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

## Trusted real-client例

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

`oauth`はclientごとの明示指定で、`trusted_real_client` manifestだけが受け付けます。validation時には認証、endpoint変数値の読み込み、client起動を一切行いません。

## Stable v1 field

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

## #112の非目標

Manifest v1 validationではまだ次を行いません。

- client実行
- endpoint環境変数値の解決
- artifact set出力
- retry
- baseline compare
- privileged self-hosted workflow実行

これらは#113 / #114 / #115で扱います。後続処理で任意commandを実行できるようなcommand/hook機構もmanifestには設けません。
