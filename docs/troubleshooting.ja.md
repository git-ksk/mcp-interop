# トラブルシューティング

[English](troubleshooting.md) | [日本語](troubleshooting.ja.md)

このページでは、`mcp-interop`を実際のMCP clientに対して実行したときの問題と、profileベースpreflight診断の読み方をまとめます。

## まずclient detectionを確認する

```console
mcp-interop clients
mcp-interop clients --json
```

clientが検出されない場合は、まず対象実行ファイルが現在の`PATH`から見えるか、versionを直接取得できるか確認してください。

`mcp-interop`は、adapterが明示的に対応していない別名binaryを「たぶん互換」と推測して使用しません。

## 明確なFAILがないのにexit codeがnon-zeroになる

live testがexit code `0`になるのは、次の4ステージが**すべて`pass`**の場合だけです。

```text
reach / auth / init / tools
```

`fail`だけでなく、`skip`と`unknown`もnon-zeroです。証拠不足のinterop結果をCIが成功として扱わないためです。

`diagnose` commandは別のpreflight contractを使います。blockingな診断FAILはnon-zeroですが、non-blocking WARNだけなら`PREFLIGHT PASS`になり得ます。preflight PASSはreal-client interoperability PASSではありません。

## `unknown`とは

`unknown`は、実クライアントが提供するcontrol/management surfaceだけでは、成功または失敗を証明できなかったことを意味します。

典型例:

- unreachable serverと「正常だがtoolが0件」のserverが同じempty inventoryに見える
- beta adapterでstableなmachine-readable statusを観測できない
- protocol stageを証明する前にclient processが終了した

`unknown`を`pass`として扱わないでください。報告時には必ず正確なclient versionを含めてください。

## OAuthが必要なserver

### Codex

Codex OAuthは明示的に`--oauth`を指定した場合だけ開始します。

```console
mcp-interop test https://example.com/mcp --client codex --oauth
```

authorization URLはstderrへ表示されます。短時間有効なOAuth stateを含むため、Issueなどへ貼らないでください。

authorization URLが得られる前にCodex OAuthが失敗した場合は、text outputの`REASON`列またはJSONの`reason_code`を確認してください。

たとえば`DCR_UNSUPPORTED`は、実Codex clientが対象OAuth flowについてDynamic Client Registration非対応を明示的に報告したことを意味します。推測したregistration URLが`404`になっただけでは、このreason codeを付けません。

stableな判定ルールは[Reason code](reason-codes.ja.md)を参照してください。

### Cursor

v0.2.0でもCursor OAuth完遂は未対応です。beta adapterでno-auth endpointは検証できますが、OAuth-required targetはauthenticated pathが正式実装されるまでincompleteになります。

### Antigravity

v0.2.0でも自動OAuth完遂を意図的に無効化しています。OAuth discoveryまでは観測済みですが、通常のmacOS Keychainからcredentialを安全に隔離できることが証明されるまでauthorization/token exchangeは実行しません。

## ChatGPT custom MCP appが接続できない

まずChatGPT preflight profileを実行します。

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

blocking FAILが出た場合、Protected Resource Metadataを発見できない、usableなAuthorization Server Metadataがない、CIMD/DCRの両方が無い、CIMD token endpoint auth methodがChatGPTと非互換、広告されたPKCE methodに`S256`が無い、といったpublic metadata上の問題を切り分けられます。

CIMD対応serverは、`registration_endpoint`が無いだけでは非互換扱いしません。ChatGPTはCIMD registration pathを利用できます。

最初の診断が`PREFLIGHT PASS`なのにChatGPT接続が失敗する場合は、Authorization Serverのsanitized logを確認します。実際のChatGPT authorization requestに非secretな`client_id` CIMD URLと`redirect_uri`があれば、次を実行します。

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

これにより、実ChatGPT CIMD document、redirect URI登録、client/server間のtoken endpoint auth method、`private_key_jwt`利用時のJWKS到達性まで照合します。

ここまでPASSしても失敗する場合、public metadataだけでは証明できないruntime boundaryをserver logで確認します。

1. authorization requestに期待する`client_id`、完全一致の`redirect_uri`、PKCE challenge、scope、protected-resource `resource`が来ているか
2. CIMD documentをAuthorization Serverが取得・検証できているか
3. `private_key_jwt`ならclient assertionをChatGPT JWKSで検証できているか
4. token requestにPKCE `code_verifier`と一貫した`resource`が来ているか
5. token responseが受理され、必要な構成ではrefresh tokenを取得できているか
6. 続くMCP requestのBearer access tokenをResource Serverが受理しているか
7. MCP initialize / tool discoveryまで進んでいるか

`PREFLIGHT PASS`は、実ChatGPT clientがこのruntime flowを完遂したことを意味しません。詳細は[ChatGPT接続診断](chatgpt-diagnostics.ja.md)を参照してください。

OAuth `state`、authorization code、access/refresh token、cookie、private key、raw client assertionは共有しないでください。

## CursorのOAuth callback port conflict

検証済みversionではlocalhost callbackが観測されていますが、callback addressはversion-specificな挙動として扱います。固定portを永久仕様としてhard-codeしないでください。

将来OAuth flowでcallback conflictが起きた場合は、以下を記録してください。

- Cursorの正確なversion
- clientが表示したcallback address
- そのportを別processが使用していたか
- token/stateを除去したsanitized output

## Antigravityが`unknown`になる

macOS beta adapterは、入力を送らないPTY startupと、isolated HOME内のmachine-readable MCP tool cacheを使ってtool discoveryを観測します。

有効なcacheを確認できないまま結果が不確定になった場合、設定ファイルを認識しただけで互換性成功とせず`unknown`を返します。

確認項目:

- `agy`の正確なversion
- targetがOAuth必須か
- isolated `~/.gemini/antigravity-cli/mcp/...` stateが生成されたか
- client processが早期終了していないか
- OSがlive adapter対応対象か

v0.2.0のAntigravity live adapterもmacOSのみ対応です。

## VS Codeは検出されるのにlive testできない

VS Codeは現在research-onlyです。

isolated環境へのMCP設定登録は安全にできますが、検証済みCLIにはsupportedなserver start/status/tool discoveryのdirect surfaceがありません。

したがって、**設定を登録できたことだけではinterop PASSにしません。**

## temporary state / process cleanup failure

`reach/auth/init/tools`がすべてPASSでも、cleanupに失敗した場合はテスト失敗として扱ってください。

報告時には以下を含めてください。

- client名と正確なversion
- OS / architecture
- 残存processがisolated test session由来か
- secretを含まないtemporary path
- cleanup error

`codex`、`cursor-agent`、`agy`という名前だけを根拠に全processをkillしないでください。adapterは現在のtest session由来と証明できるprocessだけを終了する設計です。

## user configが変更された

これはbugとして扱うべきで、credentialやKeychainが関係する場合はsecurity issueの可能性もあります。

public Issueへcredential値やraw credential fileを貼らないでください。credential leakage、通常設定の意図しない変更、Keychain write、isolation failureが関係する場合は[SECURITY.ja.md](../SECURITY.ja.md)に従ってprivate vulnerability reportingを使用してください。

## JSON output

live multi-client test:

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity --json
```

multi-client JSONはarrayです。machine-readableな判定ではstage valueを正として扱い、表示テキストだけから成功を推測しないでください。

特定のfailureを安全に分類できたstageでは、stableな`reason_code`も含まれます。

```json
{
  "stage": "auth",
  "status": "fail",
  "reason_code": "DCR_UNSUPPORTED",
  "message": "Codex reports that Dynamic Client Registration is not supported for this OAuth target"
}
```

`reason_code`が無いからといって成功という意味ではありません。adapterがstableな分類を付けるだけの具体的証拠を持っていないことを意味します。

preflight JSON:

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt --json
```

`checks` arrayの`status`、`blocking`、sanitized messageをpreflight evidenceとして扱ってください。

## Real-client E2E harness

maintainer向けrelease gateはmacOSで次を実行できます。

```console
bash scripts/e2e-real-clients.sh
```

対応する実クライアントがインストールされている必要があります。

harnessはprotocol evidenceだけでなく、予期しない`tools/call`、user config metadata、Keychain DB変更、新規残存client process、temporary session漏れも確認します。

release candidate検証では、greenにするためだけに安全性gateを無効化しないでください。

## 再現可能なbugを報告する場合

以下を含めてください。

- `mcp-interop version`
- OS / architecture
- live adapterの場合はMCP clientの正確なversion
- 使用adapterまたはdiagnostic profile
- stage resultと`reason_code`、またはpreflight checks
- secretを除去したerror/diagnostic output
- serverがOAuthを必要とするか
- 必要に応じてlocalhost/synthetic fixtureでも再現するか

Bearer token、OAuth code、client secret、cookie、credential file、OAuth `state`、private key、raw client assertionは絶対に含めないでください。
