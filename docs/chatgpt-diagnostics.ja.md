# ChatGPT接続診断

[English](chatgpt-diagnostics.md) | [日本語](chatgpt-diagnostics.ja.md)

`mcp-interop diagnose --profile chatgpt`は、Remote MCP deploymentをChatGPTの公開OAuth/MCP仕様と照合し、さらに明示的に渡されたsecret-freeなruntime observationを相関できます。

これは**ChatGPT live adapterではありません**。ChatGPT appの作成、**Scan Tools**、ChatGPT内OAuthの完遂、real-clientの`reach/auth/init/tools` PASSは主張しません。

## 証拠レイヤー

診断結果は次を分離します。

1. **Preflight** — 公開されているRemote MCP / OAuth metadataの互換性
2. **Runtime Evidence** — Authorization Server / MCP Resource Serverで観測したsecret-freeなpresence/match情報
3. **OpenAI Reference Pattern** — OpenAIの認証ドキュメントと`openai-mcpkit`のauthenticated MCP scaffoldを基準にしたruntime evidenceの相関

将来のreal ChatGPT adapterは別レイヤーです。supportedでautomatableなChatGPT product surfaceが確認できるまで追加しません。

## 基本

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

JSON:

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt --json
```

Preflightでは主に次を確認します。

- HTTPS Remote MCP endpoint
- `WWW-Authenticate`または標準well-known locationからProtected Resource Metadataを発見できるか
- `authorization_servers`
- Authorization Server Metadata / OpenID Connect discoveryと`issuer`整合
- `authorization_endpoint` / `token_endpoint`
- CIMD / DCR広告
- CIMD時の`none` / `private_key_jwt`互換性
- PKCE `S256`
- `offline_access`
- Protected Resource Metadataの`resource`整合

CIMDが利用できるなら、`registration_endpoint`が無くてもChatGPT registration preflightはPASSできます。ChatGPTはCIMDを優先しますが、DCRも選択・fallback可能な方式です。

## 実ChatGPT CIMD metadataを照合

sanitized authorization requestから非secretな`client_id` metadata URLと`redirect_uri`が分かる場合:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

追加で:

- CIMD document取得
- documentの`client_id`とstable HTTPS URLの一致
- client/server token endpoint auth method intersection
- redirect URI一致
- `private_key_jwt`利用時のJWKS到達性

を確認します。

authorization code、OAuth state、token、cookie、client assertionは渡さないでください。

## Runtime Evidence v2

v2ではregistration、authorization request、token request、resource-server verification、tool-level OAuth signalを分離します。

```json
{
  "schema_version": 2,
  "registration": {
    "strategy": "cimd",
    "client_metadata_url": "https://chatgpt.com/oauth/.../client.json"
  },
  "authorization_request": {
    "resource_matches": true,
    "redirect_uri_matches": true,
    "pkce_s256": true
  },
  "token_request": {
    "resource_matches": true,
    "code_verifier_present": true,
    "client_assertion_present": false,
    "client_assertion_type_present": false,
    "oauth_error": "invalid_client"
  },
  "resource_request": {
    "bearer_present": false
  },
  "tool_auth": {
    "challenge_expected": true,
    "oauth2_security_scheme_present": true,
    "www_authenticate_present": true,
    "www_authenticate_has_error": true,
    "www_authenticate_has_error_description": true
  }
}
```

実行:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --runtime-evidence runtime-evidence.json
```

`registration.strategy`は次を受け付けます。

- `cimd`
- `dcr`
- `predefined`

ここを明示する理由は、registration方式ごとにtoken endpoint authenticationの推論可能範囲が違うためです。CIMDではChatGPT CIMDとAS metadataを比較して、双方が`private_key_jwt`を広告していればそれをexpectedとして扱えます。一方DCR/predefinedでは、registered clientの実metadataが分からない限りtoken auth methodを勝手に推測しません。

### legacy v1互換

従来のcompact JSONも引き続き読めます。

```json
{
  "client_id": "https://chatgpt.com/oauth/.../client.json",
  "resource_matches": true,
  "code_verifier_present": true,
  "client_assertion_present": false
}
```

内部ではlegacy v1のCIMD/token-request evidenceとして正規化します。新規連携はv2推奨です。

### Secret boundary

入力できるのはboolean、stable CIMD metadata URL、registration strategy、短いOAuth error codeだけです。未知fieldは拒否します。

次は絶対に入れないでください。

- access / refresh token
- authorization code / OAuth `state`
- PKCE verifier/challengeの値
- raw client assertion / private key
- client secret
- cookie / credential file

## Runtimeで見られる境界

### Authorization request

- canonical `resource`一致
- redirect URI一致
- PKCE S256

### Token request

- canonical `resource`一致
- `code_verifier`存在
- token endpoint auth method
- `invalid_client`などsanitized OAuth error

CIMDでChatGPTとAS双方が`private_key_jwt`対応なのに`client_assertion_present=false`なら:

```text
TOKEN_AUTH_METHOD_MISMATCH
```

を返せます。

### MCP Resource request

token exchange後について、Resource Server側で次のboolean observationを渡せます。

- Bearer token到着
- signature validation
- issuer一致
- audience/resource一致
- expiry
- required scopes

これはOpenAI認証ドキュメントでresource serverに求められているtoken verificationと、公式authenticated Python MCP scaffoldのJWT verification patternに対応します。

### Tool-level OAuth signal

OpenAIの現在の仕様ではChatGPTのtool-level OAuth linking UIに、少なくとも次の2側面があります。

- tool metadataの`securitySchemes`に`oauth2`
- 認証/再認証が必要なruntime errorの`_meta["mcp/www_authenticate"]`

`tool_auth`ではpresence/shapeだけを渡します。**この証拠を取るためにmcp-interopが勝手にtoolをcallすることはありません。**

2つのtool-level境界は独立して評価します。

- `oauth2_security_scheme_present`は静的なper-tool OAuth metadataです。現在のgrantが十分なscopeを持つ場合でも適用対象で、明示的に欠落していれば`TOOL_OAUTH_METADATA_MISSING`です。
- `www_authenticate_present`はruntime reauthorization challengeです。`challenge_expected=false`ならこのcheckは`N/A`です。`challenge_expected=true`なのに明示的にchallengeが無ければ`TOOL_OAUTH_CHALLENGE_MISSING`です。

signal自体を観測できていない場合はFAILを推測せず`WARN / unknown`です。

## OpenAI Reference Pattern

Runtime Evidenceを渡すとtext outputに:

```text
OPENAI REFERENCE PATTERN
```

が追加されます。

主に次をまとめます。

- registration
- PKCE
- token endpoint auth
- bearer delivery
- resource-server token verification
- tool-level OAuth signal

基準はOpenAIの現在のauthentication docsと、`openai/openai-mcpkit/python-authenticated-mcp-server-scaffold`が示すauthenticated MCP構成です。

ただしこれは**Auth0固有設定を全MCPの必須条件にする機能ではありません**。scaffold内のprovider-specific運用手順はprotocol requirementとして扱いません。

## Result semantics

たとえば:

```text
PREFLIGHT PASS
```

でも、同時に:

```text
RUNTIME EVIDENCE
VERDICT  FAIL
REASON   TOKEN_AUTH_METHOD_MISMATCH
```

になれます。

Runtime EvidenceでconclusiveなFAILがあればCLI exit codeはnon-zeroです。未観測signalは原則`WARN / unknown`です。

非FAIL状態は意味を分けます。

- `PASS` — observationが期待動作と明確に一致した。
- `WARN / unknown` — signalが未観測、または結論を出せない。
- `N/A / not_applicable` — signalには発生条件があり、その条件が今回のflowでは明示的に発生していない。

Runtime Evidence reportにはcoverageも表示します。

```text
COVERAGE  observed=10 passed=10 failed=0 unknown=1 not_applicable=1
```

JSONでも同じ値を`runtime_evidence.coverage`に出力します。`observed`は結論が出た`PASS`と`FAIL`の合計で、WARNとN/Aは別集計です。

**Preflight PASS + Runtime Evidence PASSでも、実ChatGPT productがOAuth、MCP initialize、tool discoveryを完遂した証明にはなりません。**

## mTLS boundary

OpenAIは現在、ChatGPTがMCP serverへTLS接続する際にOpenAI-managed client certificateを提示すると説明しています。これはtransport-levelのclient identificationに使えますが、end user認証/authorizationには引き続きOAuthを使います。

Runtime Evidence v2ではmTLS certificate observationのreason-code化はまだ行いません。mTLS observationが無いだけでFAILにはしません。

## real-client boundary

現在もCodex app-serverやCursor MCP management commandに相当するsupported headless ChatGPT app-management surfaceは確認できていません。browser DOM automation/private UI internalsはstable real-client adapterの依存にしません。

公式/reference source:

- OpenAI authentication: `https://developers.openai.com/plugins/build/auth`
- OpenAI ChatGPT developer mode / MCP apps: `https://help.openai.com/en/articles/12584461-developer-mode-and-full-mcp-connectors-in-chatgpt`
- OpenAI authenticated Python MCP scaffold: `https://github.com/openai/openai-mcpkit/tree/main/python-authenticated-mcp-server-scaffold`
- MCP authorization specification: `https://modelcontextprotocol.io/specification/draft/basic/authorization`
