# mcp-interopへのコントリビューション

[English](CONTRIBUTING.md) | [日本語](CONTRIBUTING.ja.md)

real-client MCP interoperability testingの改善に協力いただきありがとうございます。

## Pull Requestを開く前に

1. 関連する既存Issue/PRがないか確認してください。
2. 新しいclient adapterを追加する場合は、そのclientが提供するMCP management surfaceとisolation戦略を記録したIssueを作成または参照してください。
3. ユーザーが普段使うclient設定を黙って変更するlive adapterは追加しないでください。

## 開発

必要条件:

- `go.mod`で宣言されているGo version

CIと同じ基本checkを実行します。

```console
gofmt -w .
git diff --exit-code
go vet ./...
go test ./...
go build ./cmd/mcp-interop
```

process lifecycle、OAuth、shared stateに関係する変更では、追加で次も実行してください。

```console
go test -race ./...
```

## Adapter要件

live client adapterは次を満たすべきです。

- client挙動をemulateせず、実際にインストールされたclientを呼び出す
- client management/control surfaceで結果を証明できる場合、model promptを使わない
- config/credentialをtemporary profile/HOME/config directoryへ隔離する
- client surfaceだけではstageを証明できない場合、成功を推測せず`unknown`を返す
- `reach`、`auth`、`init`、`tools`を別々の観測として扱う
- success/failure両方でtemporary stateをcleanupする
- report/errorからBearer/OAuth credential等のsecretをredactする
- 検証したclient versionを記録する
- successおよび主要なfailure/inconclusive pathのtestを追加する

安全なisolationを確立できないclientは、既存ユーザー設定を変更して無理に対応せず、experimental/research-onlyのままにしてください。

## OAuth変更

OAuthは特に慎重に扱います。

- user interactionを発生させる可能性があるauthenticationは明示的opt-inにする
- CLI contractで明示されていないauthorization URLを勝手にopenしない
- test credentialを通常のOS Keychain/client credential storeへ保存しない
- authorization URLやcallback stateをmachine-readable resultへ含めない
- automated testではproduction credentialではなくlocal/synthetic OAuth fixtureを使用する

## Pull Request

PRは焦点を絞ってください。最低限、次を含めます。

- 検証したclient/version
- interoperabilityを証明するために使った具体的なobservable surface
- isolation/cleanupの挙動
- local test結果
- 意図的に`unknown`として残す制限

security vulnerabilityはpublic Issueではなく、`SECURITY.md` / `SECURITY.ja.md`に従ってprivateに報告してください。
