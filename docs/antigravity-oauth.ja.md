# Antigravity OAuth live-test boundary

[English](antigravity-oauth.md) | [日本語](antigravity-oauth.ja.md)

`mcp-interop test <url> --client antigravity --oauth`は、macOS上でAntigravityの実MCP OAuth managerを明示的に有効化します。adapterはインストール済みの`agy` clientをisolated temporary `HOME`とworkspaceを持つPTY内で実行し、`/mcp`を開き、単一のisolated test serverを選択し、operatorが入力したauthorization codeを実clientへ直接転送します。model promptは使用しません。

## Credential isolation

検証済み`agy 1.1.11` baselineでは、AntigravityはMCP OAuth stateを次へ保存します。

```text
~/.gemini/antigravity/mcp_oauth_tokens.json
```

`HOME`をtemporary session homeへ置き換えるため、このpathはユーザーが通常利用するAntigravity stateから隔離されます。`mcp-interop`が観測するのはfile metadata（存在、regular-file type、non-zero size）のみです。token fileを開いたりparseしたりせず、authorization URL、authorization code、access token、refresh token、cookie、credential-file contentsをpersistしません。

## Result semantics

OAuth pathは意図的にconservativeです。isolated token fileの生成を確認できた場合、`reach`と`auth`は`pass`として報告できます。`init`と`tools`は、client-side Antigravity tool cacheも観測できた場合にのみ`pass`として報告します。

検証済み`agy 1.1.11`のOAuth pathでは、実clientがauthenticated `initialize`、`notifications/initialized`、`tools/list`を完了しても、no-auth adapter pathで使うものと同じtool-cache fileを生成しません。この場合、authenticationだけから成功を推測せず、generic live resultは`init=unknown`、`tools=unknown`を維持します。

controlled localhost release E2Eでは別途、実Antigravity clientがauthenticated `initialize`、`notifications/initialized`、`tools/list`を実行したことを示すsecret-free server-side evidenceを必須にします。このE2E evidenceによって、任意のRemote MCP targetに対するgeneric four-stage verdictの意味を変更することはありません。

## Safety gates

real-client OAuth E2Eでは、通常のAntigravity configuration、通常のOAuth-token state、macOS login Keychain、事前に存在していた`agy` process setがisolated test後も変化していないことを検証します。temporary PTY descendantはsuccess/failureのどちらでもcleanup中に終了させます。
