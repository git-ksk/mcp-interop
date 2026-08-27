# Antigravity OAuth live-testの境界

[English](antigravity-oauth.md) | **日本語**

> この文書は英語版`antigravity-oauth.md`の日本語訳です。挙動の正確な定義は英語版を正とします。

`mcp-interop test <url> --client antigravity --oauth`は、macOS上で**実AntigravityクライアントのMCP OAuth経路**を明示的に有効化します。

アダプターは、インストール済みの`agy`を一時`HOME`とworkspaceを持つPTY内で起動し、実`/mcp`マネージャーを操作します。

モデルへのプロンプトは使いません。

## 認証情報の隔離

検証済み`agy 1.1.11`では、AntigravityのMCP OAuth stateは通常、次へ保存されます。

```text
~/.gemini/antigravity/mcp_oauth_tokens.json
```

テスト時は`HOME`そのものを一時directoryへ差し替えるため、このパスも通常ユーザーのAntigravity状態から隔離されます。

Antigravity account認証とRemote MCP OAuthは別の境界です。`agy`起動前に、一時CLI settingsへ`modelProvider: "gemini"`を書き、ambientなGemini model credential / endpoint overrideを除去して、固定の非秘密`GEMINI_API_KEY` sentinelを注入します。[Antigravity公式ドキュメント](https://antigravity.google/docs/cli/install/)では、このGemini API-key modeはaccount sessionを成立させないため、通常ユーザーのmacOS Keychain sessionへ依存しません。model promptは送らないため、このsentinelでmodel requestを認証することもありません。

`mcp-interop`が確認するのはファイルのメタデータだけです。

- ファイルが存在するか
- regular fileか
- サイズが0ではないか

**tokenファイルの内容は開きません。**

次も保存しません。

- authorization URL
- authorization code
- access token
- refresh token
- cookie
- credential fileの内容

login Keychainのbefore/after比較は非変更gateです。それ単独ではcredential非利用の証明にせず、documented no-account modeと実クライアントE2Eを組み合わせて通常ユーザーcredentialを再利用しない境界を成立させます。

## 結果をどう判定するか

OAuth経路の判定は意図的に保守的です。

一時tokenファイルの生成を確認できれば、`reach`と`auth`は`pass`にできます。

一方、`init`と`tools`を`pass`にするには、Antigravity自身が生成したtool cacheなど、クライアント側から直接確認できる証拠が必要です。

検証済み`agy 1.1.11`では、OAuth認証後に実クライアントが`initialize`、`notifications/initialized`、`tools/list`まで実行しても、認証不要経路と同じtool-cache fileを生成しない場合があります。

その場合、認証成功から後続段階を推測せず、generic live resultは次を維持します。

```text
reach=pass
auth=pass
init=unknown
tools=unknown
```

## controlled localhost E2Eとの違い

release向けのcontrolled localhost E2Eでは、server側で次を直接観測できることを要求します。

```text
initialize
notifications/initialized
tools/list
```

これは**Antigravityアダプターの測定経路が正しいことを検証する証拠**です。

任意のRemote MCPに対するgeneric four-stage verdictの意味を変更するものではありません。

## Safety gate

実クライアントOAuth E2Eでは、実行前後で次が変化していないことを確認します。

- 通常のAntigravity設定
- 通常のOAuth token state
- macOS login Keychain
- テスト前から存在していた`agy` process set

テスト用PTYから派生したprocessだけをcleanup対象にします。

成功・失敗にかかわらず、一時HOME / workspace / owned processを片付けます。
