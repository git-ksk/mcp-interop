# アーキテクチャ

[English](architecture.md) | [日本語](architecture.ja.md)

`mcp-interop` は、Remote MCP デプロイメントを**実際のMCPクライアントでブラックボックス検証する**相互運用性テストランナーです。プロトコル仕様への適合確認と、実クライアントで本当に動くかの確認を分離し、一般的なMCP conformance suiteとは異なる役割を持たせています。

## コアモデル

各テストステージは4種類の状態を返します。

- `pass` — ステージの成功を観測できた。
- `fail` — ステージを試行し、失敗を観測した。
- `skip` — 前提条件を満たさない、またはadapterが未対応のため実行しなかった。
- `unknown` — 利用できるクライアントの観測面だけでは結果を証明できない。

現在のステージは次の4つです。

1. `reach` — 実クライアントがRemote MCPへ到達し、live interactionを確認できた。
2. `auth` — 必要な認証が完了した、またはtool discoveryにより認証不要であることを確認できた。
3. `init` — 実クライアントがMCP sessionを確立した。
4. `tools` — 実クライアントがserverのtoolsを発見した。

完全な相互運用PASSには4ステージすべての`pass`が必要です。証拠が足りない状態をCIで互換性成功として扱わないため、`skip`や`unknown`もnon-zero exitになります。

## Adapter境界

対応クライアントごとにadapterを持ち、クライアント固有の処理を閉じ込めます。概念上のライフサイクルは次の通りです。

```text
Detect -> Prepare isolated profile -> Register endpoint -> Authenticate -> Discover -> Cleanup
```

adapterは「設定ファイルを書けた」ことを成功と推測せず、**実際にインストールされているクライアントが観測した事実**を報告します。

## Isolation方針

live adapterはユーザーが普段使っているMCP設定を黙って変更してはいけません。

優先順位は次の通りです。

1. クライアントが公式に提供するtemporary/profile/config overrideを使う。
2. HOME基準で設定を解決するクライアントでは、挙動を検証できている場合に限りtemporary HOMEを使う。
3. 安全な隔離ができない場合は、既存設定を変更せず`unknown`または`skip`を返す。

テスト中に作られるcredentialやOAuth tokenは、可能な限りisolated profile内に閉じ込めます。reportにはBearer token、authorization code、client secret、cookie、credential fileの生データを含めません。

processについても同じ原則を適用します。adapterが終了させてよいのは、現在のisolated test sessionに属すると証明できるprocessだけです。実行ファイル名だけを根拠に他のCodex/Cursor/Antigravity processをkillしてはいけません。

## v0.1.0で提供しているadapter

### Codex CLI

v0.1.0で最も完成度の高いadapterです。isolated `CODEX_HOME`、実`codex app-server`のMCP status surface、明示的なopt-in OAuth flowを使用します。OAuth credential storageはtemporary HOME内のfileへ強制し、通常のkeyring経路を使いません。

### Cursor CLI (beta)

isolated temporary `HOME`とworkspaceを作り、実Cursor CLIの`mcp enable`、`mcp list`、`mcp list-tools`を使います。model promptなしでno-auth Remote MCPのlive interoperabilityを確認できます。OAuth token exchangeとauthenticated tool discoveryはv0.2での完成を予定しています。

### Antigravity CLI (beta, macOS)

isolated temporary `HOME`、現在の`~/.gemini/config/mcp_config.json`形式、入力を送らないPTY startupを使います。実クライアントが生成するmachine-readable tool cacheを観測し、cleanup前にはtest PTY wrapperのdescendantだけを回収します。macOS Keychainから安全に隔離できることが証明されるまで、自動OAuth完遂は無効です。

### VS Code (research)

isolated user-data directoryへのMCP設定登録は安全にできますが、検証時点のCLIにはserver start/status/tool discoveryを直接観測できるsupported pathがありません。**登録できたことだけでは互換性PASSにしません。** no-modelで安定したlifecycle surfaceが利用可能になるまでresearch-onlyです。

### GitHub Copilot CLI (candidate)

model promptなしで同等のblack-box evidenceを得られる安定したMCP inventory/lifecycle surfaceが確認できれば、v0.3での候補になります。

## Real-client E2E境界

repoにはlocalhost限定のMCP fixtureと、macOSで実Codex/Cursor/Antigravityを検証する`scripts/e2e-real-clients.sh`があります。

harnessは各clientについて最低限、次のprotocol evidenceを要求します。

```text
initialize
notifications/initialized
tools/list
```

`tools/call`が発生した場合はFAILです。また、実行前後でuser config metadata、login Keychain DB、新規残存client process、temporary session directoryを比較します。

GitHub-hosted CIには外部MCP clientをインストールしません。通常CIではadapter regression test、fixture、harnessのsyntax/build path、release buildを検証します。実クライアントE2Eはself-hosted macOS ARM64 runner向けのmanual workflowとして分離しています。

## このプロジェクトが検証しないもの

interop testが成功しても、次を保証するものではありません。

- MCP serverが安全であること
- tool実装自体が正しいこと
- destructive operationが安全であること
- modelが適切なtoolを選択すること
- あらゆるOAuth identity/scope combinationが成功すること

これらはsecurity scanner、conformance test、agent evaluationなど別種類のツールが扱う領域です。

実行時の問題や結果の読み方は[トラブルシューティング](troubleshooting.ja.md)を参照してください。
