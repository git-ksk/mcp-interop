# Reason code

[English](reason-codes.md) | [日本語](reason-codes.ja.md)

`mcp-interop`の各stage resultは、`status`と人間向けmessageに加えて、任意の`reason_code`を持てます。

reason codeは保守的に付与します。実クライアントが返した証拠から特定のinterop failureを区別できる場合だけ出力し、推測したendpointや単独のHTTP statusだけから判定しません。

## 初期OAuth reason code

### `DCR_UNSUPPORTED`

実クライアントが、対象OAuth flowではDynamic Client Registration (DCR) がサポートされていないと明示的に報告した場合です。

例:

```text
STAGE  STATUS  REASON           DETAIL
auth   FAIL    DCR_UNSUPPORTED  Codex reports that Dynamic Client Registration is not supported for this OAuth target
```

JSON:

```json
{
  "stage": "auth",
  "status": "fail",
  "reason_code": "DCR_UNSUPPORTED",
  "message": "Codex reports that Dynamic Client Registration is not supported for this OAuth target"
}
```

初期のCodex classifierは、`Dynamic client registration not supported`と同等の明示的client errorを認識します。

推測した`/register`や`/oauth/register`が`404`になっただけでは、**`DCR_UNSUPPORTED`とは判定しません。**

### `TOKEN_AUTH_METHOD_MISMATCH`

明示的に渡されたsecret-free Runtime Evidenceが、公開client/server metadataから選択されるtoken endpoint auth methodと一致しない場合です。たとえば双方が`private_key_jwt`を共有しているのに、観測token requestで`client_assertion_present=false`ならこのcodeを返します。

これはPreflight failureではありません。`PREFLIGHT PASS`とRuntime Evidence `FAIL`は同時に成立できます。raw assertionやtoken request bodyは入力・保存しません。

### `DCR_FAILED`

実クライアントがDynamic Client Registrationを試み、そのregistration attemptが「unsupported以外の理由」で失敗したと明示的に報告した場合です。

将来的にはpolicy rejectionやserver-side registration failureなど、secret-safeなdetailを保持しつつ、`DCR_UNSUPPORTED`と混同しないための分類として使います。

## Security boundary

app-serverのraw errorには、remote serverまたはclient生成の文字列が含まれる可能性があります。

Codex adapterは分類に必要なraw detailをprocess内memoryにだけ保持し、通常のerror string、text report、JSON resultへはそのまま出しません。外部へ出すのはstableなreason codeと、`mcp-interop`側で定義したmessageです。

authorization URL、token、authorization code、client secret、cookie、credential fileをreason-code detailへ含めてはいけません。

## Server capabilityとの相関

最初のreason-code実装は、実クライアントが明示した証拠を分類します。authorization serverがCIMDやDCRを広告しているかを、`mcp-interop`自身が独立して断定する機能はまだ含みません。

v0.2のfollow-upとして、次のauthorization-server metadataとclient failureを相関させるcapability diagnosticを追加予定です。

- `client_id_metadata_document_supported`
- `registration_endpoint`

この診断は、registration URLを推測するのではなく、MCP Protected Resource Metadataとauthorization-server discoveryに従う必要があります。

また、metadata診断は補助証拠です。interopのpass/fail判定の中心は引き続き**実クライアント実行**です。
