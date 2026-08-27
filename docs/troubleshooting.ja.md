# トラブルシューティング

[English](troubleshooting.md) | **日本語**

> この文書は英語版`troubleshooting.md`の日本語訳です。内容に差がある場合は英語版を正とします。

このページでは、`mcp-interop`を実クライアントで動かしたときによく起きる問題と、`diagnose`による事前診断の読み方をまとめます。

英語正本に記載されているstable releaseはv0.7.0です。以下のCursor / Antigravity OAuth対応はv0.4.0で導入されました。

## まずクライアントが検出されているか確認する

```console
mcp-interop clients
mcp-interop clients --json
```

対象クライアントが出てこない場合は、まず次を確認してください。

- 実行ファイルが現在の`PATH`から見えるか
- コマンドを直接実行してバージョンを取得できるか

`mcp-interop`は、未対応の別名binaryを「たぶん同じクライアント」と推測して使いません。

## FAILが見えないのにexit codeが0にならない

live testがexit code `0`になるのは、次の4段階が**すべて`pass`**の場合だけです。

```text
reach / auth / init / tools
```

`fail`だけでなく、`skip`と`unknown`もnon-zeroです。

これは、確認できていない相互運用性をCIが成功扱いしないための仕様です。

`diagnose`は別契約です。blockingな診断FAILはnon-zeroですが、WARNだけなら`PREFLIGHT PASS`になることがあります。

**PREFLIGHT PASSは、実クライアントのinterop PASSではありません。**

## `unknown`は何を意味するか

`unknown`は「成功とも失敗とも証明できなかった」という意味です。

典型例:

- 到達不能なサーバーと、正常だがtoolが0件のサーバーが同じempty inventoryに見える
- beta adapterで安定したmachine-readable statusを取得できない
- 必要なprotocol段階を確認する前にclient processが終了した
- OAuth認証は確認できたが、`init/tools`を直接確認するclient-side stateが生成されなかった

`unknown`を`pass`として読み替えないでください。

不具合報告では、必ず対象クライアントの正確なversionを含めてください。

## OAuthが必要なRemote MCP

### Codex

OAuthは`--oauth`を明示した場合だけ開始します。

```console
mcp-interop test https://example.com/mcp --client codex --oauth
```

authorization URLはstderrへ表示されます。短時間有効なOAuth `state`が含まれるため、Issueやログ共有へ貼らないでください。

認証開始前に失敗した場合は、`REASON`列またはJSONの`reason_code`を確認してください。

たとえば`DCR_UNSUPPORTED`は、**実Codexが対象OAuth経路でDynamic Client Registration非対応を明示した**場合に使われます。

推測したregistration URLが404だっただけでは、このreason codeにはしません。

詳しくは[Reason code](reason-codes.ja.md)を参照してください。

### Cursor

```console
mcp-interop test https://example.com/mcp --client cursor --oauth
```

一時`HOME`とworkspaceの中で、実Cursor MCP login経路を使います。認証後の`mcp list-tools`成功を、実クライアントのツール発見証拠として扱います。

controlled fixtureではDCR、Authorization Code + PKCE、token exchange、Bearer付きMCP request、認証後のtool discoveryまで検証しています。

callbackで失敗した場合は、固定portを前提にしないでください。記録すべきなのは次です。

- 正確なCursor version
- そのversionが表示したcallback address
- そのportが他processに使用されていたか
- token / stateを除去した出力

明示的なbind conflictを確認できた場合は`OAUTH_CALLBACK_PORT_CONFLICT`として分類されることがあります。

### Antigravity

```console
mcp-interop test https://example.com/mcp --client antigravity --oauth
```

一時`HOME`内のPTYで実Antigravity `/mcp`マネージャーを使います。

`agy`起動前に、一時profileで`modelProvider: "gemini"`を選択し、ambientなGemini model credential / endpoint overrideを除去して、固定の非秘密`GEMINI_API_KEY` sentinelを注入します。これはAntigravity公式ドキュメント上のno-account modeを選択するためのもので、通常ユーザーのAntigravity account / Keychain sessionへ依存しません。

Remote MCPのOAuth token stateは一時HOMEだけに保存され、`mcp-interop`はtokenファイルの内容を読みません。ファイルの存在などのメタデータだけを確認します。Keychainのbefore/after hashは非変更gateであり、それ単独を「Keychainを読んでいない」証拠にはしません。

検証済み`agy 1.1.11`では、認証完了後でも認証不要時と同じtool cacheが生成されない場合があります。

その場合は意図的に次のような結果を維持します。

```text
reach=pass
auth=pass
init=unknown
tools=unknown
```

認証できたという理由だけで、未観測の段階をPASSへしません。

詳細は[Antigravity OAuth](antigravity-oauth.ja.md)を参照してください。

## ChatGPTから接続できない

まず公開メタデータの事前診断を実行します。

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

blocking FAILでは、たとえば次を切り分けられます。

- Protected Resource Metadataを発見できない
- usableなAuthorization Server Metadataがない
- CIMD / DCRのどちらも利用できない
- CIMD時のtoken endpoint authentication methodが非互換
- PKCE `S256`が広告されていない

ChatGPTはCIMDを利用できるため、`registration_endpoint`が無いだけではFAILにしません。

### PreflightはPASSするがChatGPT接続は失敗する

Authorization Serverのsanitized logから、秘密情報ではない実ChatGPTの`client_id` CIMD URLと`redirect_uri`が分かる場合は、次を実行します。

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

これにより次を追加確認できます。

- 実ChatGPT CIMD document
- redirect URI登録
- client/server間のtoken endpoint auth method
- `private_key_jwt`利用時のJWKS到達性

それでも失敗する場合は、server-sideで次のruntime境界を確認します。

1. authorization requestの`client_id`、`redirect_uri`、PKCE、scope、`resource`
2. Authorization ServerがCIMD documentを取得・検証できたか
3. `private_key_jwt`ならclient assertionをJWKSで検証できたか
4. token requestに`code_verifier`と一貫した`resource`が来ているか
5. token responseが受理されたか
6. MCP requestのBearer access tokenがResource Serverで受理されたか
7. MCP initialization / tool discoveryまで進んだか

`PREFLIGHT PASS`は、このruntime flowを実ChatGPTが完了した証明ではありません。

詳しくは[ChatGPT接続診断](chatgpt-diagnostics.ja.md)を参照してください。

OAuth `state`、authorization code、access/refresh token、cookie、private key、raw client assertionは共有しないでください。

## Antigravityが`unknown`になる

これは必ずしも異常ではありません。

Antigravity adapterは、クライアント自身から観測できる証拠だけで判定します。

認証不要時は、一時HOME内に生成されたmachine-readable MCP tool cacheを使います。設定ファイルを認識しただけではPASSにしません。

OAuth時はtoken persistenceで`reach/auth`を確認できても、tool cacheが生成されなければ`init/tools`は`unknown`です。

確認項目:

- `agy`の正確なversion
- OAuth targetなら`--oauth`を付けたか
- 一時`~/.gemini/antigravity-cli/mcp/...` stateが生成されたか
- OAuth時に一時`~/.gemini/antigravity/mcp_oauth_tokens.json`のメタデータを確認できたか
- client processが早期終了していないか
- OSが対応対象か

現在のlive implementationはmacOSのみです。

## VS Codeは検出されるのにlive testできない

VS Codeは現在research-onlyです。

隔離環境へMCP設定を登録することはできますが、安定したdirect lifecycle / tool-discovery automation経路はまだlive adapterへ昇格していません。

**設定できたことだけではinterop PASSにしません。**

## 一時状態やプロセスが残った

`reach/auth/init/tools`がすべてPASSでも、cleanupに失敗した場合はテスト失敗として扱ってください。

報告時には次を含めてください。

- client名と正確なversion
- OS / architecture
- 残存processが今回の隔離セッション由来か
- 秘密情報を含まないtemporary path
- cleanup error

`codex`、`cursor-agent`、`agy`という実行ファイル名だけを理由に、全processをkillしないでください。

## 普段使っているユーザー設定が変更された

これはbugです。credentialやKeychainが関係する場合はsecurity issueの可能性があります。

公開Issueへcredential値やcredential fileの内容を貼らないでください。

credential漏えい、通常設定の意図しない変更、Keychain write、隔離失敗が関係する場合は[セキュリティポリシー](../SECURITY.ja.md)に従ってPrivate Vulnerability Reportingを利用してください。

## JSON出力の読み方

複数クライアントのlive test:

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity --json
```

JSONは配列です。自動判定では表示メッセージではなくstageの`status`を正として扱ってください。

安全に分類できる失敗には`reason_code`が付く場合があります。

```json
{
  "stage": "auth",
  "status": "fail",
  "reason_code": "DCR_UNSUPPORTED",
  "message": "Codex reports that Dynamic Client Registration is not supported for this OAuth target"
}
```

`reason_code`が無いことは成功を意味しません。「安定した分類を付けられるだけの明示的証拠がない」という意味です。

事前診断のJSON:

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt --json
```

`checks`配列の`status`、`blocking`、秘密情報を除去したmessageを診断証拠として扱ってください。

## 実クライアントE2E harness

maintainer向けrelease gate:

```console
bash scripts/e2e-real-clients.sh
```

対象の実クライアントがインストールされている必要があります。

protocol evidenceだけでなく、予期しない`tools/call`、user config metadata、Keychain DB変更、残存client process、一時session漏れも確認します。

安全性gateを無効にしてrelease candidateをgreenにしないでください。

## 再現可能なbugを報告する場合

最低限、次を含めてください。

- `mcp-interop version`
- OS / architecture
- live adapterならMCP clientの正確なversion
- 使用したadapterまたはdiagnostic profile
- stage resultと`reason_code`、またはpreflight check
- 秘密情報を除去したerror / diagnostic output
- serverがOAuthを必要とするか
- 必要に応じてlocalhost / synthetic fixtureでも再現するか

Bearer token、OAuth code、client secret、cookie、credential file、OAuth `state`、private key、raw client assertionは絶対に含めないでください。
