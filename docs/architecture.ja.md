# アーキテクチャ

[English](architecture.md) | **日本語**

> この文書は英語版`architecture.md`の日本語訳です。内容に差がある場合は英語版を正とします。

`mcp-interop`は、Remote MCPのデプロイが**実際のMCPクライアント製品で使えるか**をブラックボックスで検証するランナーです。

MCP仕様そのものへの適合性と、実製品同士の相互運用性を意図的に分けています。`mcp-interop`を第二のMCP Conformance suiteにはしません。

## MCP Conformanceとの関係

`mcp-interop`は公式の[MCP Conformance Test Framework](https://github.com/modelcontextprotocol/conformance)を置き換えるものではなく、補完するものです。

違いは「実ソフトウェアか、synthetic testか」ではありません。公式Conformanceも実クライアントコマンドや実サーバーURLを使えます。

違うのは**何を正解として判定するか**です。

```text
MCP Conformance: implementation × specification
mcp-interop:      deployment × client product × client version
```

MCP Conformanceは、実装が仕様上の期待動作を満たすかを確認します。

`mcp-interop`は、特定のRemote MCPデプロイを、特定のクライアント製品・バージョンから実際に利用できるかを確認します。

そのため、ConformanceがPASSしても「すべての公開クライアントでそのデプロイが動く」とは限りません。逆に`mcp-interop`がPASSしても「MCP仕様へ完全適合している」とは限りません。

実運用では、次の順で使うのが分かりやすいです。

```text
1. MCP Conformanceで仕様適合性を確認
2. 実Remote MCPをデプロイ
3. mcp-interopで実クライアントとの相互運用性を確認
```

詳しくは[MCP Conformanceとmcp-interopの違い](conformance-vs-interop.ja.md)を参照してください。

## コアとなる結果モデル

各段階は次の4状態を返します。

- `pass` — 成功を実際に観測できた
- `fail` — 実行し、失敗を観測した
- `skip` — 前提条件を満たさない、またはアダプター未対応のため実行しなかった
- `unknown` — 利用できるクライアント側の観測手段だけでは成功・失敗を証明できなかった

現在の段階は次の4つです。

1. `reach` — 実クライアントが対象Remote MCPへ到達し、実通信を確認できた
2. `auth` — 必要な認証が完了した、またはツール発見により認証不要と確認できた
3. `init` — **MCP protocol readiness**の互換projection。実クライアントが利用可能なprotocol pathを証明したことを表し、modern MCPでliteralな`initialize` requestを必須にはしない
4. `tools` — クライアントがサーバーのツールを発見した

完全な相互運用PASSには4段階すべての`pass`が必要です。

証拠不足を成功扱いしないため、`skip`や`unknown`もnon-zero exitです。

### protocol-awareな`init`互換semantics

公開`init` fieldはJSON/CLI互換性のため維持しますが、stableな意味はwire-level initialization handshakeではなく**protocol readiness**です。内部では、real-client surfaceから直接観測できた場合だけera/revisionとreadiness evidenceを非serializeの`ProtocolObservation`として保持できます。

`init=pass`にはdirect real-client readiness evidenceが必要です。現在のCodex / Cursor / Antigravityは、実client自身のtool inventory / tool materialization成功を、usable MCP protocol pathが成立したことのより強い証拠として使います。fixture-only lifecycle、config存在、metadata compatibility、未観測protocol revisionだけではdeployment-specificな`init=pass`を作れません。

deployment-specific surfaceがnegotiated protocol revisionを返さない場合、protocol readinessとtool discoveryがPASSでもera/revisionは`unknown`のままです。legacy `initialize`/`initialized`、modern `server/discover`やself-describing request、将来のprotocol-era mechanismはいずれもreal clientから直接観測できた場合にreadiness evidenceになり得ますが、公開projectionは`init`を維持します。

## 品質改善で守る原則

現在は新しいクライアントを増やすことより、信頼性・再現性・退行検出を優先しています。

品質改善でも次を崩しません。

- 事前診断やメタデータだけでlive resultをPASSへ昇格させない
- Runtime Evidenceは秘密情報を含まない「存在したか」「一致したか」の観測だけを扱う
- 不完全・曖昧な観測は`WARN` / `unknown`のままにする
- 終了できるプロセスは、今回の隔離テストが所有していると証明できるものだけにする
- 固定sleepは、可能ならreadiness・process exit・state stabilityの条件へ置き換える
- release gateでも可能な範囲で通常CIと同じ品質条件を再確認する

## アダプターの境界

対応クライアントごとにアダプターを持ち、製品固有の処理を閉じ込めます。

概念的な流れは次のとおりです。

```text
検出
  -> 隔離profileを準備
  -> 対象endpointを登録
  -> 必要なら認証
  -> MCP状態・ツールを観測
  -> 一時状態を片付ける
```

アダプターは「設定ファイルへ書けた」ことから成功を推測せず、**実際にインストールされたクライアントが観測した事実**を結果にします。

## 隔離ポリシー

live adapterは、ユーザーが普段使っているMCP設定を黙って変更してはいけません。

優先順位は次のとおりです。

1. クライアントが公式に提供する一時profile / config overrideを使う
2. `HOME`基準で設定を解決するクライアントでは、挙動を確認できている場合に限り一時`HOME`を使う
3. 安全に隔離できない場合は、既存設定を変更せず`unknown` / `skip`を返す

テスト中に生成されたcredentialやOAuth tokenは、可能な限り隔離profile内へ閉じ込めます。

reportへBearer token、authorization code、client secret、cookie、credential fileの生データを含めません。

プロセスも同様です。実行ファイル名が同じという理由だけで、別のCodex / Cursor / Antigravityプロセスを終了してはいけません。

## 提供中のアダプター

英語正本で記載している現在のstable releaseはv0.7.0です。Cursor / Antigravity OAuth経路はv0.4.0で導入されました。

### Codex CLI

最も完成度の高いアダプターです。

- 一時`CODEX_HOME`
- 実`codex app-server`のMCP status
- 明示的な`--oauth`
- OAuth credentialを一時HOME内のファイルへ限定

を使います。

### Cursor CLI（beta）

一時`HOME`とworkspaceを使い、実Cursor CLIの`mcp enable`、`mcp list`、`mcp list-tools`で確認します。

`--oauth`時は実CursorのMCP login経路を使います。controlled fixtureではDCR、Authorization Code + PKCE、token exchange、Bearer付きMCP request、認証後の`mcp list-tools`まで検証しています。

callback addressはバージョン依存であり、固定portを仕様として決め打ちしません。

### Antigravity CLI（beta / macOS）

一時`HOME`とPTYを使う実クライアント経路です。

`agy`起動前に、一時`~/.gemini/antigravity-cli/settings.json`へ`modelProvider: "gemini"`を書き、ambientなGemini API key / base URL overrideを除去し、固定の非秘密`GEMINI_API_KEY` sentinelを注入します。これにより、Antigravity公式ドキュメント上のGemini API-key modeを選択し、Antigravity account sessionを成立させません。通常ユーザーのmacOS Keychain sessionへ依存せずにMCP discoveryを実行します。

login Keychainのbefore/after比較は「変更していない」ことのgateであり、それ単独では「読んでいない」ことの証明にはしません。credential非再利用の根拠は、上記のdocumented no-account modeと実クライアントrelease gateの組み合わせです。`agy 1.1.22`で、model prompt / `tools/call`なしの`initialize`、`notifications/initialized`、`tools/list`、通常ユーザーconfig / Keychain metadata不変、process / session leakなしを再検証しています。

認証不要の場合はクライアントが生成したmachine-readableなtool cacheを観測します。

OAuthでは実`/mcp`マネージャーを使います。Remote MCP OAuthとAntigravity account認証は別の境界で、クライアント自体は上記のno-account modeで起動したまま、tokenだけを一時`~/.gemini/antigravity/mcp_oauth_tokens.json`へ閉じ込めます。`mcp-interop`はtoken内容を読みません。

認証成功だけでは`init/tools`を推測しません。必要なtool cacheが観測できなければ`unknown`を維持します。

詳細は[Antigravity OAuth](antigravity-oauth.ja.md)を参照してください。

### VS Code（research）

隔離したuser-data directoryへMCP設定を登録することはできますが、安定したdirect lifecycle / tool-discovery経路はまだlive adapterへ昇格していません。

設定できたことだけをinterop成功とは扱いません。

### GitHub Copilot CLI（research）

現在のPoCでは、入力なしの実クライアント起動で`initialize` / `notifications/initialized`までは観測できましたが、認証済み/model backendなしで`tools/list`までは証明できていません。Issue #48を参照してください。

### ChatGPT（blocked）

現在はdiagnostics-onlyです。

公式にサポートされたdirect/headlessなChatGPT MCPアプリ管理インターフェースが利用可能になるまで、実クライアントアダプターはBLOCKEDです。

model prompt、DOM/UI automation、private endpoint、通常ユーザーのブラウザcredentialを、`reach/auth/init/tools`の代わりにはしません。

## 実クライアントE2Eの境界

リポジトリにはlocalhost限定のMCP fixtureと、macOSで実Codex / Cursor / Antigravityを確認する`scripts/e2e-real-clients.sh`があります。

release-gate harnessはprotocol-era-awareです。fixture readinessはcompleteなlegacy `initialize` / `notifications/initialized` / `tools/list` path、または明示的な`2026-07-28` protocol evidenceを持つmodern `tools/list`のどちらかを受け付けます。`server/discover` probeだけでは不足です。別のlegacy/modern/fallback matrixで両eraを検証し、core gateでは`tools/call`を引き続き禁止します。fixture protocol evidenceをdeployment-specific resultの昇格には使いません。

さらに、実行前後でユーザー設定のメタデータ、login Keychain DB、残存クライアントプロセス、一時セッションディレクトリを比較します。

Cursor / AntigravityのOAuth専用E2Eも、controlled loopback fixtureに対して実OAuth経路を検証します。authorization codeやtokenは保存証拠へ含めません。

このfixtureは**アダプター自身の測定経路を検証するrelease gate**であり、一般的なMCP Conformance suiteではありません。

GitHub-hosted CIには外部MCPクライアントをインストールしません。実クライアントE2Eはself-hosted macOS ARM64向けmanual workflowへ分離しています。

## このプロジェクトが検証しないもの

相互運用テストが成功しても、次は保証しません。

- 実装がMCP仕様へ完全適合していること
- MCPサーバーが安全であること
- ツール実装が正しいこと
- 破壊的操作が安全であること
- モデルが正しいツールを選ぶこと
- あらゆるOAuth identity / scopeで成功すること
- 実際にテストしていないクライアント製品・バージョンとの互換性

これらはConformance test、security scanner、agent evaluationなど別の層で扱います。

実行時の問題は[トラブルシューティング](troubleshooting.ja.md)を参照してください。
