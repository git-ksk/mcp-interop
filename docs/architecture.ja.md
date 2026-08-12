# アーキテクチャ

[English](architecture.md) | [日本語](architecture.ja.md)

`mcp-interop` は、Remote MCP デプロイメントを**実際のMCPクライアントでブラックボックス検証する**相互運用性テストランナーです。プロトコル仕様へのconformanceと、実client productとのinteroperabilityを明確に分離し、一般的なMCP conformance suiteとは異なる役割を持たせています。

## MCP Conformanceとの関係

`mcp-interop` は、公式の [MCP Conformance Test Framework](https://github.com/modelcontextprotocol/conformance) を置き換えるものではなく、補完するテストレイヤーです。

違いは「real softwareかsynthetic testか」ではありません。公式Conformanceは実client commandを起動でき、実server URLも直接テストできます。公式側のoracleはMCP specificationであり、scenario-controlledなinteractionをexpected protocol behaviorと比較します。

一方`mcp-interop`は、**特定のRemote MCP deploymentを、特定のreleased client product/versionで実際に使えるか**を、その製品自身のMCP surfaceから観測します。比較軸は次の通りです。

```text
MCP Conformance: implementation x specification
mcp-interop:      deployment x client product x client version
```

Conformance PASSだけでは、特定deploymentがすべてのreleased clientで動くことまでは証明しません。逆に`mcp-interop` PASSだけでも、MCP仕様への完全な適合性は証明しません。release pipelineでは、まずConformanceを通し、実endpointをdeployし、その後usersが実際に使うclientで`mcp-interop`を実行する二段階構成が適しています。

Product-specificな`diagnose` profileも同じ境界を守ります。GenericなMCP/OAuth conformanceは公式Conformanceの担当です。`diagnose`は特定client productとの互換性を確認できますが、metadata compatibilityをgeneric conformanceやreal-client interoperability PASSとして扱ってはいけません。

詳細は[MCP Conformance と mcp-interop の違い](conformance-vs-interop.ja.md)を参照してください。

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

## 品質フェーズのinvariant

現在の開発は新client追加より、reliability、testability、reproducibility、measured regression detectionを優先しています。品質改善でも次のinvariantは維持します。

- diagnostic/preflight metadataはfailure説明には使えるが、live adapter resultをPASSへ昇格させない
- Runtime Evidenceはsecret-freeなpresence/match observationだけを扱い、不完全・曖昧な観測は`WARN` / `unknown`のままにする
- 終了させてよいprocessはcurrent isolated test sessionが所有すると証明できるもの、またはharness自身が直接起動したものだけ
- fixed sleepは、flakeを減らせる場合にreadiness、process exit、state stabilityの条件へ置き換える
- release gateは可能な範囲で通常CIと同じsource-quality invariantを再検証する

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

## 提供中のadapter

現在のstable releaseはv0.4.0です。以下のCursor/Antigravity OAuth pathはv0.4.0に含まれます。

### Codex CLI

現在最も完成度の高いadapterです。isolated `CODEX_HOME`、実`codex app-server`のMCP status surface、明示的なopt-in OAuth flowを使用します。OAuth credential storageはtemporary HOME内のfileへ強制し、通常のkeyring経路を使いません。

### Cursor CLI (beta)

isolated temporary `HOME`とworkspaceを作り、実Cursor CLIの`mcp enable`、`mcp list`、`mcp list-tools`を使います。model promptなしでno-auth Remote MCPのlive interoperabilityを確認できます。

v0.4.0では、明示的な`--oauth`によりisolated session内で実Cursor MCP login pathを起動します。controlled OAuth fixtureでDCR、Authorization Code + PKCE、token exchange、Bearer付きMCP request、authenticated `mcp list-tools`まで検証済みです。authenticated `mcp list-tools`成功を、tested Cursor CLI surfaceにおける`reach/auth/init/tools`の直接evidenceとして扱います。callback addressはversion-specificとして扱い、固定portをhard-codeしません。

### Antigravity CLI (beta, macOS)

isolated temporary `HOME`、現在の`~/.gemini/config/mcp_config.json`形式、PTYを使ったreal-client pathを使用します。no-auth modeでは実クライアントが生成するmachine-readable tool cacheを観測し、cleanup前にはtest PTY wrapperのdescendantだけを回収します。

v0.4.0では、明示的な`--oauth`によりisolated PTY内で実Antigravity `/mcp` managerへ入ります。OAuth tokenはisolated `~/.gemini/antigravity/mcp_oauth_tokens.json`に閉じ込め、`mcp-interop`はfile metadataだけを観測し、token内容を開いたり保存したりしません。generic resultはconservativeなままで、authenticationを証明できてもOAuth pathでno-auth時と同じtool cacheが生成されなければ`init/tools=unknown`を維持します。controlled localhost E2Eでは別途、authenticated `initialize`、`notifications/initialized`、`tools/list`のserver-side evidenceを必須にします。詳細は[Antigravity OAuth live-test boundary](antigravity-oauth.md)を参照してください。

### VS Code (research)

isolated user-data directoryへのMCP設定登録は安全にできますが、stableなsupported direct lifecycle/tool-discovery automation boundaryはまだlive adapterへ昇格していません。**登録できたことだけでは互換性PASSにしません。** researchはshipped adapter contractとは分離して継続します。

### GitHub Copilot CLI (research)

GitHub Copilot CLIはresearch-onlyです。現在のPoCではno-input startupで実clientの`initialize` / `notifications/initialized`までは確認できましたが、authenticated/model backendなしの`tools/list`は未証明です。詳細は#48を参照してください。

### ChatGPT (blocked)

ChatGPTは引き続きdiagnostics-only profileです。officially supportedなdirect/headless ChatGPT MCP app-management surfaceが利用可能になるまでreal-client adapterはBLOCKEDです。model prompt、DOM/UI automation、private endpoint、通常ユーザーのbrowser credentialを、このprojectのreal-client `reach/auth/init/tools` evidence contractの代替にはしません。詳細は#20を参照してください。

## Real-client E2E境界

repoにはlocalhost限定のMCP fixtureと、macOSで実Codex/Cursor/Antigravityを検証する`scripts/e2e-real-clients.sh`があります。

harnessは各clientについて最低限、次のprotocol evidenceを要求します。

```text
initialize
notifications/initialized
tools/list
```

`tools/call`が発生した場合はFAILです。また、実行前後でuser config metadata、login Keychain DB、新規残存client process、temporary session directoryを比較します。

Cursor/AntigravityのOAuth専用E2E harnessも同じisolation原則を使い、controlled loopback fixtureに対して実OAuth client pathを検証します。authorization codeやtokenなどのsecret-bearing materialはpersisted evidenceへ含めません。

このfixtureは**adapterのself-test / release gate**であり、一般的なMCP conformance suiteではありません。目的は、`mcp-interop`のmeasurement pathが本当に実clientを観測でき、isolation guaranteeを維持していることを確認することです。

GitHub-hosted CIには外部MCP clientをインストールしません。通常CIではadapter regression test、fixture、harnessのsyntax/build path、release buildを検証します。実クライアントE2Eはself-hosted macOS ARM64 runner向けのmanual workflowとして分離しています。

## このプロジェクトが検証しないもの

interop testが成功しても、次を保証するものではありません。

- implementationが完全にMCP conformantであること
- MCP serverが安全であること
- tool実装自体が正しいこと
- destructive operationが安全であること
- modelが適切なtoolを選択すること
- あらゆるOAuth identity/scope combinationが成功すること
- 実際にテストしていないclient product/versionとの互換性

これらはsecurity scanner、conformance test、agent evaluationなど別種類のツールが扱う領域です。

実行時の問題や結果の読み方は[トラブルシューティング](troubleshooting.ja.md)を参照してください。