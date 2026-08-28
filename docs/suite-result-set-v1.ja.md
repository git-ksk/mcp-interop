# Suite result set v1

[English](suite-result-set-v1.md) | **日本語**

Suite result set v1は、trusted `mcp-interop suite run`が生成するportable directory contractです。

## Layout

commit済みoutput directoryは次の形です。

```text
suite-results/
  index.json
  artifacts/
    production-a--codex--none.json
    production-a--cursor--none.json
```

private staging directoryでset全体を組み立て、index完成後にrenameします。`suite run`は既存output directoryを拒否するため、古いartifactが新しいsetへ黙って混ざりません。

`artifacts/`配下はそれぞれ、runを1件だけ含む通常の**live-result artifact schema v2**です。suite layer独自のstage/result modelは作りません。

## `index.json`

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/suite-results",
  "manifest_schema_version": 1,
  "manifest_fingerprint": "sha256:...",
  "execution_context": "trusted_real_client",
  "artifact_schema_version": 2,
  "runs": [
    {
      "target_id": "production-a",
      "deployment_id": "production-a",
      "client_id": "codex",
      "auth_mode": "none",
      "outcome": "pass",
      "exit_code": 0,
      "artifact": "artifacts/production-a--codex--none.json"
    }
  ]
}
```

manifest fingerprintはvalidation済みmanifest宣言だけから生成し、解決したendpoint値は入力にしません。`deployment_id`は参照先schema-v2 artifactと同じ安定した非secret operator labelで、readerはindexとartifactのidentity一致を検証します。

run entryはmanifest配列順に依存せず、`target_id`、`deployment_id`、`client_id`、`auth_mode`の順で固定します。

## Outcome semantics

- `pass` / exit `0`: 参照先direct live-result artifactの4stageがすべてPASS
- `non_pass` / exit `1`: valid artifactはあるがdirect live testが完全PASSではない。未インストールclientも参照artifact内の明示的`skip`としてここに残す
- `error` / exit `1`: valid per-run artifactをcommitできる前にexecution error。`artifact`参照は持たない

1件でも`non_pass`または`error`ならsuite commandはexit `1`です。manifest/input/preflight不正はclient起動前にexit `2`です。

## Secret / privacy境界

`index.json`には非secret `deployment_id`を保存しますが、次は保存しません。

- Remote MCP endpoint URL
- endpoint環境変数名や値
- protected URL path
- OAuth token / code / secret
- client executable pathやarbitrary環境変数override

trusted endpointはexecution前にmemory上で解決します。各run artifactはschema-v2 protected-path identityを使うためprotected pathを保存もhashもしません。ただしcanonical originはschema-v2 artifactに残るので、`credential-safe != deployment-public`の境界は引き続き有効です。

## Reader safety

result-set readerはregular fileの`index.json`、`artifacts/`配下のclean relative reference、regular artifact file、解決後もresult-set directory内に留まるartifact pathを要求します。symlink経由で外部pathへ逃げる参照はartifact内容を信頼する前にrejectします。

## 現在のscope

v0.7.0では、このschemaから`hosted_fixture` suite executionを有効化しません。repository CIではcontrolled localhost fixture executionと任意suite manifestを分離し、trusted / untrusted runner境界は[Self-hosted real-client CI security](self-hosted-ci-security.ja.md)で定義しています。suite regression / retry aggregationは[Suite regression report v1](suite-regression-report-v1.ja.md)で定義します。
