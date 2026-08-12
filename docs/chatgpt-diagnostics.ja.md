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

`mcp-interop`はproduct interoperability診断であり、genericなMCP conformance suiteではありません。境界は次のように考えます。

- **MCP conformance** = implementation × specification
- **mcp-interop** = deployment × client product × client version

そのためOpenAI Reference Patternのproduct依存前提は、Runtime Evidence schemaやMCP specification自体とは別にversion管理します。

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

## Runtime Evidence v3

v3ではv2のregistration、authorization request、token request、resource-server observationを維持しつつ、tool-level evidenceを独立した2セクションに分けます。

- `tool_metadata` — tool discovery時に観測する静的per-tool OAuth metadata
- `tool_challenge` — tool callで認証/追加scopeが必要になった場合だけ観測するruntime reauthorization signal

```json
{
  "schema_version": 3,
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
    "client_assertion_type_present": false
  },
  "resource_request": {
    "bearer_present": true,
    "signature_valid": true,
    "issuer_matches": true,
    "audience_matches": true,
    "expiry_valid": true,
    "scopes_sufficient": true
  },
  "tool_metadata": {
    "oauth2_security_scheme_present": true
  },
  "tool_challenge": {
    "expected": false
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

registration方式ごとにtoken endpoint authenticationの推論可能範囲が違います。CIMDではChatGPT CIMDとAS metadataを比較して、双方が`private_key_jwt`を広告していればexpectedとして扱えます。一方DCR/predefinedでは、registered clientの実metadataが分からない限りtoken auth methodを推測しません。

### v2互換

schema v2は変更なしで引き続き受け付けます。従来のcombined `tool_auth`は内部でv3と同じ評価境界へ正規化されるため、既存evidence producerの診断結果は維持されます。

```json
{
  "schema_version": 2,
  "tool_auth": {
    "challenge_expected": true,
    "oauth2_security_scheme_present": true,
    "www_authenticate_present": true,
    "www_authenticate_has_error": true,
    "www_authenticate_has_error_description": true
  }
}
```

1つのevidence object内でv2 `tool_auth`とv3 `tool_metadata` / `tool_challenge`を混在させることはできません。

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

内部ではlegacy v1のCIMD/token-request evidenceとして正規化します。新規連携はv3推奨です。

### Evidence utility

独立したsecret-free fragmentを`diagnose`前にvalidate/mergeできます。

```console
mcp-interop evidence validate authorization.json
mcp-interop evidence summary authorization.json
mcp-interop evidence merge authorization.json resource.json tool.json -o runtime-evidence.json
```

- `validate`は`diagnose --runtime-evidence`と同じstrict decoderを使います。
- `summary`はinput schema、section名、supplied field数だけを表示し、boolean値、OAuth error、client metadata URLなどのobservation値は表示しません。
- `merge`はv1/v2/v3をcanonical schema v3へ正規化し、同じfieldに競合observationがあれば失敗します。

### Secret boundary

入力できるのはboolean、stable CIMD metadata URL、registration strategy、短いOAuth error codeだけです。未知fieldは拒否します。

次は絶対に入れないでください。

- access / refresh token
- authorization code / OAuth `state`
- PKCE verifier/challengeの値
- raw client assertion / private key
- client secret
- cookie / credential file

## 実ChatGPT manual dogfood workflow

ChatGPT UI automationへ依存せず、実際のChatGPT productからRuntime Evidenceを得たい場合は次の手順を使います。real-client操作だけmanualで行い、その後のevidence処理はdeterministicかつsecret-freeに保ちます。

1. 先にpublic preflightを実行します。

   ```console
   mcp-interop diagnose https://example.com/mcp --profile chatgpt
   ```

2. 同じRemote MCP deploymentへ、実ChatGPT productの通常のsupported connection flowから接続します。
3. 軽いread-only toolを2回呼びます。customer dataの露出を避け、可能な限り小さいrequestを使います。
4. 既存のpersistent dataを変更しない、明示的にsafeなwrite-authorized pathを1回だけ呼びます。専用no-op、dry-run、validation-only、またはstate-preservingなwrite fixtureを優先します。OAuth検証だけのためにproduction recordを更新・削除しません。
5. server-sideでsecret-free Runtime Evidence fragmentを取得します。`mcp-interop`が受け付けるschema fieldだけを記録し、raw HTTP header、request body、credential、application固有identifierをevidence fileへコピーしません。
6. fragmentをmerge → validate → summaryします。

   ```console
   mcp-interop evidence merge authorization.json resource.json tool.json -o runtime-evidence.json
   mcp-interop evidence validate runtime-evidence.json
   mcp-interop evidence summary runtime-evidence.json
   ```

7. 同じdeploymentのpreflightとsanitized observationを相関します。

   ```console
   mcp-interop diagnose https://example.com/mcp \
     --profile chatgpt \
     --runtime-evidence runtime-evidence.json
   ```

8. public Preflight、Runtime Evidence、versioned OpenAI Reference Patternを別々に確認します。manualなChatGPT操作により「実product session由来のevidence」であることは確認できますが、それをgeneric MCP conformanceへ読み替えません。

### Dogfood privacy boundary

Runtime Evidence schemaに存在しないidentifierやcredentialは保存・公開しません。unique user/resource/operation/request identifierやraw OAuth/session artifactは除外し、server logに含まれる場合もevidence workflowへ入れる前に許可されたboolean / short error codeへ変換します。

Monokuraはこのworkflowのreal-world validation対象として利用実績があります。公開docsには「product-levelでこの手順を実施した」という事実だけを残し、unique user/resource/operation identifierやcredentialは例へ含めません。

現時点では、このmanual workflowをChatGPT DOM automationより優先します。UI/DOM依存はproduct changeで壊れやすく、credential isolationが難しく、CI再現性が低く、private implementationへ依存しやすいためです。

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

v3ではpresence/shapeを`tool_metadata`と`tool_challenge`へ分離して渡します。v2 `tool_auth`も引き続き読み込み、内部で同じ評価境界へ正規化します。**この証拠を取るためにmcp-interopが勝手にtoolをcallすることはありません。**

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

OpenAI Reference Pattern結果はself-versionedです。現在のprofile metadataは次です。

```text
PROFILE_REVISION  2026-08-10.1
OBSERVED_DATE     2026-08-10
SOURCE            OpenAI authenticated MCP reference pattern
```

JSONでは同じ情報を`runtime_evidence.openai_reference_pattern`配下の`profile_revision`、`observed_date`、`source`へ出します。`profile_revision`はmcp-interopが持つChatGPT/OpenAI interoperability前提のversionであり、**MCP specification versionでもRuntime Evidence schema versionでもありません**。`observed_date`は、そのprofile revisionについてOpenAIのproduct guidanceを最後に確認した日です。product guidanceが実質的に変われば、Runtime Evidence schema v3を変更しない場合でもreference profile revision/dateを更新します。

このprofile metadata追加は、受け付けるRuntime Evidence v1/v2/v3 input contractを変更しません。diagnostic interpretation layer自体をversion管理するための情報です。

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

Issue #20はそのためBLOCKEDのままです。model prompt、scripted ChatGPT UI/DOM interaction、private endpoint、通常ユーザーのbrowser credentialをPASS evidenceに使いません。manual product observationはdocumented secret-free Runtime Evidenceへ変換できますが、それはdiagnostic layerのままであり、live ChatGPT adapter resultにはなりません。

公式/reference source:

- OpenAI authentication: `https://developers.openai.com/plugins/build/auth`
- OpenAI ChatGPT developer mode / MCP apps: `https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt`
- OpenAI authenticated Python MCP scaffold: `https://github.com/openai/openai-mcpkit/tree/main/python-authenticated-mcp-server-scaffold`
- MCP authorization specification: `https://modelcontextprotocol.io/specification/draft/basic/authorization`
