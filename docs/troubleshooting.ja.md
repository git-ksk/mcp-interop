# トラブルシューティング

[English](troubleshooting.md) | [日本語](troubleshooting.ja.md)

このページでは、`mcp-interop`を実際のMCPクライアントに対して実行したときに起こりやすい問題と、結果の読み方をまとめます。

## まずclient detectionを確認する

```console
mcp-interop clients
mcp-interop clients --json
```

clientが検出されない場合は、まず対象実行ファイルが現在の`PATH`から見えるか、versionを直接取得できるか確認してください。

`mcp-interop`は、adapterが明示的に対応していない別名binaryを「たぶん互換」と推測して使用しません。

## 明確なFAILがないのにexit codeがnon-zeroになる

テストがexit code `0`になるのは、次の4ステージが**すべて`pass`**の場合だけです。

```text
reach / auth / init / tools
```

`fail`だけでなく、`skip`と`unknown`もnon-zeroです。

これは意図した仕様です。証拠不足の相互運用結果をCIが成功として扱わないためです。

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

### Cursor

v0.1.xではCursor OAuth完遂は未対応です。beta adapterでno-auth endpointは検証できますが、OAuth-required targetはauthenticated pathが正式実装されるまでincompleteになります。

### Antigravity

v0.1.xでは自動OAuth完遂を意図的に無効化しています。OAuth discoveryまでは観測済みですが、通常のmacOS Keychainからcredentialを安全に隔離できることが証明されるまでauthorization/token exchangeは実行しません。

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

v0.1.xのAntigravity live adapterはmacOSのみ対応です。

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

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity --json
```

multi-client JSONはarrayです。machine-readableな判定ではstage valueを正として扱い、表示テキストだけから成功を推測しないでください。

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
- MCP clientの正確なversion
- 使用adapter
- stage result
- secretを除去したerror/diagnostic output
- serverがOAuthを必要とするか
- 必要に応じてlocalhost/synthetic fixtureでも再現するか

Bearer token、OAuth code、client secret、cookie、credential fileは絶対に含めないでください。
