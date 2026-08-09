# ChatGPT接続診断

[English](chatgpt-diagnostics.md) | [日本語](chatgpt-diagnostics.ja.md)

`mcp-interop diagnose --profile chatgpt`は、Remote MCP serverが公開しているOAuth metadataを確認し、ChatGPTの現在のMCP認証仕様と既知のblocking mismatchがないかを事前診断します。

これは**preflight diagnostic**であり、ChatGPT live adapterではありません。ChatGPT appの作成、**Scan Tools**操作、ChatGPT内でのOAuth完遂は行わず、real-clientの`reach/auth/init/tools` PASSも主張しません。

## 基本的な使い方

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

JSON出力:

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt --json
```

現在のChatGPT profileでは主に次を確認します。

- Remote MCP endpointがHTTPSか
- `WWW-Authenticate`または標準well-known locationからOAuth 2.0 Protected Resource Metadataを発見できるか
- `authorization_servers`が広告されているか
- OAuth Authorization Server Metadata / OpenID Connect discoveryと`issuer`整合
- `authorization_endpoint` / `token_endpoint`
- client registration方式
  - Client ID Metadata Documents (CIMD)
  - またはDynamic Client Registration (DCR) fallback
- CIMD時のtoken endpoint authenticationがChatGPT対応の`none`または`private_key_jwt`か
- PKCE `S256`広告
- refresh token継続性の参考として`offline_access`広告
- Protected Resource Metadataの`resource`と入力したMCP endpointの関係

たとえばAuthorization Server Metadataが:

```json
{
  "client_id_metadata_document_supported": true,
  "token_endpoint_auth_methods_supported": ["none", "private_key_jwt"]
}
```

なら、`registration_endpoint`が無くてもChatGPTのregistration preflightはPASSできます。ChatGPTはCIMDを利用でき、このpathではDCRは必須ではありません。

## 実際のChatGPT client metadataまで照合する

Authorization Serverのlogで、ChatGPTから来たauthorization requestを**secret除去済みで**確認できる場合、`client_id`と`redirect_uri`は有用な非secret evidenceです。

`code`、`state`、token、cookie、client assertionなどは渡さないでください。

観測した値をそのまま指定します。

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

`--client-id`を指定すると、追加で:

- CIMD documentを取得
- ChatGPT側とAuthorization Server側のtoken endpoint auth methodを照合
- `--redirect-uri`指定時はCIMD documentの`redirect_uris`に完全一致するか確認
- `private_key_jwt`が互換methodならJWKSを取得してkeyが存在するか確認

まで行います。

OAuthがauthorization endpointまでは進むが、その後token exchange前後で失敗する場合に特に有効です。

## 結果の読み方

text outputは必ず:

```text
PREFLIGHT PASS
```

または:

```text
PREFLIGHT FAIL
```

と表示します。

WARNはpreflight failureにはしません。たとえば`offline_access`未広告は、初回OAuth自体は成功してもrefresh tokenが取得できず、access token失効後にChatGPTが再認証を必要とする可能性があるためのadvisoryです。

**`PREFLIGHT PASS`でも、実ChatGPT clientがOAuth/tool discoveryを完遂した証明にはなりません。** 現在確認可能な公開metadata上にknown blocking mismatchが見つからなかった、という意味です。

## preflight PASSなのにChatGPT接続が失敗する場合

Authorization Server / MCP Resource Serverのlogを使って、実flowがどこまで進んだか確認します。共有するlogは必ずsanitizeしてください。

確認ポイント:

1. authorization requestに期待する`client_id`、正確な`redirect_uri`、PKCE challenge、scope、`resource`が来ているか
2. CIMDの場合、Authorization ServerがChatGPTのclient metadata documentを取得・検証できているか
3. `private_key_jwt`の場合、token requestのclient assertionをChatGPT JWKSで検証できているか。またAS側が要求する`iss` / `sub` / `aud` / `exp` / `jti`条件を満たしているか
4. token exchangeにPKCE `code_verifier`と同じprotected-resource `resource`値が来ているか
5. token responseが受理され、継続接続が必要な構成ではrefresh tokenが発行されているか
6. その後のMCP requestに`Authorization: Bearer ...`が来て、MCP serverがsignature / issuer / audience(resource) / expiry / scopeを受理しているか
7. ChatGPTがMCP initialize / tool discoveryまで進んでいるか

Issueへauthorization code、access/refresh token、cookie、private key、raw client assertion、短時間有効なOAuth `state`を貼らないでください。

## 現在の境界

現状の`mcp-interop`には、Codex app-serverやCursorのMCP management commandに相当する、supportedなheadless ChatGPT app-management surfaceがありません。

そのため、ブラウザDOM automationで無理に「ChatGPT対応」を名乗らず、true real-client automationはresearch targetのまま維持します。

このprofileの基準にしている公式資料:

- OpenAI plugin authentication: `https://developers.openai.com/plugins/build/auth`
- OpenAI ChatGPT developer mode / MCP apps: `https://help.openai.com/en/articles/12584461-developer-mode-and-full-mcp-connectors-in-chatgpt`
- MCP authorization specification: `https://modelcontextprotocol.io/specification/draft/basic/authorization`
