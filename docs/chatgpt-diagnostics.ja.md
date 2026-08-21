# ChatGPT接続診断

[English](chatgpt-diagnostics.md) | **日本語**

> この文書は英語版`chatgpt-diagnostics.md`の日本語訳です。診断契約の正確な定義は英語版を正とします。

`mcp-interop diagnose --profile chatgpt`は、Remote MCPの公開OAuth/MCP設定をChatGPTの公開仕様と照合し、必要に応じてserver-sideで観測した**秘密情報を含まないRuntime Evidence**も相関します。

これは**ChatGPTのlive adapterではありません**。

次を実行・証明する機能ではありません。

- ChatGPT appの作成
- ChatGPT UIの**Scan Tools**操作
- ChatGPT内でのOAuth完遂
- 実ChatGPTの`reach/auth/init/tools` PASS

## 診断で扱う3つの層

1. **Preflight** — Remote MCP / OAuthが公開しているメタデータの互換性
2. **Runtime Evidence** — Authorization Server / MCP Resource Serverで観測した、秘密情報を含まない「存在・一致」の情報
3. **OpenAI Reference Pattern** — OpenAIの認証ドキュメントと`openai-mcpkit`のauthenticated MCP例を基準にした比較

将来の実ChatGPT adapterは別の層です。安全に自動操作できる公式のChatGPT product surfaceが確認できるまで追加しません。

MCP仕様への一般的なConformanceと、この製品固有診断も分けます。

```text
MCP Conformance = implementation × specification
mcp-interop     = deployment × client product × client version
```

## 基本的な使い方

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

JSON出力:

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt --json
```

Preflightでは主に次を確認します。

- HTTPS Remote MCP endpoint
- Protected Resource Metadataの発見
- `authorization_servers`
- Authorization Server Metadata / OpenID Connect discoveryと`issuer`
- `authorization_endpoint` / `token_endpoint`
- CIMD / DCR
- CIMD時の`none` / `private_key_jwt`互換性
- PKCE `S256`
- `offline_access`
- protected resourceの`resource`整合性

ChatGPTはCIMDを利用できるため、CIMDが成立するなら`registration_endpoint`が無いだけではFAILにしません。

## 実ChatGPTのCIMD情報を照合する

Authorization Serverのsanitized logから、秘密情報ではない実ChatGPTの`client_id` metadata URLと`redirect_uri`が分かる場合:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

追加で次を確認します。

- CIMD documentを取得できるか
- documentの`client_id`とHTTPS URLが一致するか
- client / serverのtoken endpoint auth methodに共通方式があるか
- redirect URIが登録値と一致するか
- `private_key_jwt`利用時にJWKSへ到達できるか

**authorization code、OAuth state、token、cookie、client assertionは渡さないでください。**

## Runtime Evidence v3

Runtime Evidenceは、OAuthやResource Serverで観測した事実を、秘密情報そのものではなくbooleanなどへ変換した入力です。

v3ではtool-level情報を2種類へ分けます。

- `tool_metadata` — tool discovery時に見える静的なOAuth metadata
- `tool_challenge` — tool call時に追加認証・scopeが必要な場合のreauthorization signal

例:

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

`registration.strategy`は`cimd`、`dcr`、`predefined`を受け付けます。

registration方式によって、token endpoint authentication methodをどこまで推測できるかが異なります。

CIMDではChatGPT CIMDとAuthorization Server Metadataを比較できます。DCR / predefinedでは、実際に登録されたclient metadataが分からない限り同じ期待値を勝手に適用しません。

### v2 / legacy v1互換

schema v2はそのまま受け付けます。

従来の`tool_auth`は内部でv3と同じ評価境界へ正規化します。

v2 `tool_auth`とv3 `tool_metadata` / `tool_challenge`を同じobject内に混在させることはできません。

legacy v1のcompact JSONも互換目的で読み込めます。新しい連携ではv3を推奨します。

## Evidence utility

複数箇所から得た秘密情報を含まない断片を、診断前に検証・統合できます。

```console
mcp-interop evidence validate authorization.json
mcp-interop evidence summary authorization.json
mcp-interop evidence merge authorization.json resource.json tool.json -o runtime-evidence.json
```

- `validate` — `diagnose --runtime-evidence`と同じ厳密なdecoderを使う
- `summary` — schema、section名、入力field数だけを表示する。観測値そのものは表示しない
- `merge` — v1/v2/v3をschema v3へ正規化し、同じfieldに矛盾した観測があれば失敗する

## 秘密情報の境界

入力できるのは、限定されたboolean、stableなCIMD metadata URL、registration strategy、短いOAuth error codeだけです。

未知fieldは拒否します。

次は**絶対に入れないでください**。

- access / refresh token
- authorization code / OAuth `state`
- PKCE verifier / challengeの値
- raw client assertion / private key
- client secret
- cookie / credential file

## 実ChatGPTを使ったmanual dogfood

ChatGPT UI自動化へ依存せず、実ChatGPT productからRuntime Evidenceを得る場合は、実操作だけ人間が行い、証拠処理は決定的・secret-freeに保ちます。

1. 先にPreflightを実行する

   ```console
   mcp-interop diagnose https://example.com/mcp --profile chatgpt
   ```

2. 同じRemote MCPへ、ChatGPTの通常のsupported connection flowから接続する
3. 軽いread-only toolを2回呼ぶ
4. write-authorized経路を確認する必要がある場合は、既存データを変更しないno-op / dry-run / validation-onlyな手段を優先する
5. server-sideでRuntime Evidence schemaに許可された情報だけを記録する
6. fragmentをmerge / validate / summaryする
7. 同じdeploymentのPreflightとRuntime Evidenceを相関する
8. Preflight、Runtime Evidence、OpenAI Reference Patternを別々に確認する

この手順で「実ChatGPT session由来の観測」であることは確認できますが、それをgeneric MCP Conformanceやlive ChatGPT adapter PASSへ読み替えません。

### privacy boundary

Runtime Evidence schemaに存在しないユーザーID、resource ID、operation ID、request ID、raw OAuth/session artifactなどは保存・公開しません。

server logに含まれる場合も、evidence fileへ入れる前に許可されたbooleanや短いerror codeへ変換します。

## Runtimeで確認する代表的な境界

### Authorization request

- canonical `resource`が一致するか
- redirect URIが一致するか
- PKCE S256が使われているか

### Token request

- `resource`が一致するか
- `code_verifier`が存在するか
- token endpoint auth method
- `invalid_client`などのsanitized OAuth error

CIMDでChatGPTとAuthorization Serverの両方が`private_key_jwt`対応なのに、`client_assertion_present=false`なら`TOKEN_AUTH_METHOD_MISMATCH`として分類できます。

### MCP Resource request

Resource Server側から次のboolean observationを渡せます。

- Bearer tokenが届いたか
- signature validation
- issuer一致
- audience / resource一致
- expiry
- required scopes

### Tool-level OAuth signal

ChatGPTのtool-level OAuthでは少なくとも次の2種類を分けて扱います。

- tool metadataの`securitySchemes`に`oauth2`があるか
- 認証・再認証が必要なruntime errorに`_meta["mcp/www_authenticate"]`があるか

v3では前者を`tool_metadata`、後者を`tool_challenge`へ分離します。

**この証拠を得るために`mcp-interop`が勝手にtoolを実行することはありません。**

未観測ならFAILを推測せず`WARN / unknown`です。

## OpenAI Reference Pattern

Runtime Evidenceを渡すと、テキスト出力へ`OPENAI REFERENCE PATTERN`が追加されます。

主に次をまとめます。

- registration
- PKCE
- token endpoint auth
- Bearer delivery
- Resource Serverでのtoken verification
- tool-level OAuth signal

現在のprofile metadata:

```text
PROFILE_REVISION  2026-08-10.1
OBSERVED_DATE     2026-08-10
SOURCE            OpenAI authenticated MCP reference pattern
```

`profile_revision`は、`mcp-interop`が持つChatGPT/OpenAI向け相互運用前提のversionです。

MCP specification versionでもRuntime Evidence schema versionでもありません。

OpenAI側のproduct guidanceが実質的に変わった場合は、Runtime Evidence schemaを変えなくてもprofile revision/dateを更新します。

## 結果の読み方

PreflightがPASSでもRuntime EvidenceはFAILになり得ます。

```text
PREFLIGHT PASS

RUNTIME EVIDENCE
VERDICT  FAIL
REASON   TOKEN_AUTH_METHOD_MISMATCH
```

Runtime EvidenceでconclusiveなFAILがあればCLI exit codeはnon-zeroです。

非FAIL状態は次のように分けます。

- `PASS` — 観測結果が期待動作と明確に一致
- `WARN / unknown` — 未観測または判断不能
- `N/A / not_applicable` — 今回のflowではそのsignalが発生する条件自体が成立していない

coverageも表示します。

```text
COVERAGE  observed=10 passed=10 failed=0 unknown=1 not_applicable=1
```

**Preflight PASS + Runtime Evidence PASSでも、実ChatGPTがOAuth、MCP initialization、tool discoveryを完遂した証明にはなりません。**

## mTLSの境界

英語正本で参照しているOpenAIの説明では、ChatGPTがMCP serverへTLS接続する際にOpenAI-managed client certificateを提示する場合があります。

これはtransport-levelのclient identificationであり、end-userのauthentication / authorizationは引き続きOAuthの役割です。

現在はmTLS証拠が無いことだけでFAILにはしません。

## 実ChatGPT adapterを作らない理由

現在、Codex app-serverやCursor MCP management commandに相当する、公式にサポートされたheadless ChatGPT app-management interfaceは確認できていません。

そのためIssue #20はBLOCKEDです。

次をlive PASS evidenceには使いません。

- model prompt
- scripted ChatGPT UI / DOM automation
- private endpoint
- 通常ユーザーのbrowser credential

manual product observationはRuntime Evidenceへ変換できますが、あくまで診断層の証拠です。

## 参照先

- OpenAI authentication: `https://developers.openai.com/plugins/build/auth`
- OpenAI ChatGPT developer mode / MCP apps: `https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt`
- OpenAI authenticated Python MCP scaffold: `https://github.com/openai/openai-mcpkit/tree/main/python-authenticated-mcp-server-scaffold`
- MCP authorization specification: `https://modelcontextprotocol.io/specification/draft/basic/authorization`
