# MCP Conformanceとmcp-interopの違い

[English](conformance-vs-interop.md) | **日本語**

> この文書は英語版`conformance-vs-interop.md`の日本語訳です。内容に差がある場合は英語版を正とします。

`mcp-interop`は、公式の[MCP Conformance Test Framework](https://github.com/modelcontextprotocol/conformance)を置き換えるものではありません。

両者は**別の問いに答えるテスト**で、併用するのが適切です。

## 何が違うのか

| | MCP Conformance | `mcp-interop` |
| --- | --- | --- |
| 主な問い | このMCP実装は仕様どおりか | このRemote MCPは、このクライアント製品・バージョンで実際に使えるか |
| 比較するもの | implementation × specification | deployment × client product × client version |
| 判定基準 | MCP仕様とscenarioの期待動作 | 実際にインストールされたクライアントから観測した事実 |
| 主な結果 | 仕様適合のPASS / FAIL | `reach` / `auth` / `init` / `tools` |
| 製品固有の挙動 | 仕様違反なら問題 | 相互運用性を左右する重要な観測対象 |

「MCP Conformanceはsynthetic、mcp-interopはreal software」という区別ではありません。

公式Conformanceも実クライアントコマンドを起動でき、実サーバーURLを直接テストできます。

違いは、**仕様を基準に採点するのか、特定製品との実相互運用を観測するのか**です。

## MCP Conformanceが検証するもの

公式Conformanceは主に次を検証します。

- MCP wire / lifecycle要件を満たすか
- 特定のMCP specification revisionへ適合するか
- OAuth / MCP authorization scenarioを仕様どおり実装しているか
- JSON-RPC messageやprotocol behaviorが仕様に適合するか

`mcp-interop`は、これらのgenericな仕様適合テストを再実装しません。

## mcp-interopが検証するもの

`mcp-interop test`は、ユーザーが実際に利用しようとしているRemote MCPデプロイを入力にします。

同じRemote MCPを、各製品自身のMCP機能から実クライアントへ登録し、どこまで成立するかを確認します。

```text
同じRemote MCP
   |
   +--> Codex CLI version X
   +--> Cursor CLI version Y
   +--> Antigravity CLI version Z
```

各クライアントで次を観測します。

```text
reach -> auth -> init -> tools
```

サーバーとクライアントが個別には仕様適合していても、次のような製品固有差で特定の組み合わせだけ失敗することがあります。

- 設定形式
- OAuth discovery順序
- credential storage
- callback処理
- registration方式
- クライアントバージョンの退行

そのため、**製品名と正確なバージョン自体が相互運用性の証拠の一部**です。

## 静的な互換性一覧とは違う

「Cursorは一般に機能Xをサポートする」といったcapability matrixは便利ですが、それだけでは特定のRemote MCPが動くことを証明できません。

`mcp-interop`が強く証明できるのは、次のような実行単位の結果です。

```text
endpoint A + client X version 1 -> PASS
endpoint A + client X version 2 -> AUTH FAIL
endpoint A + client Y version 7 -> PASS
```

つまり、**バージョン間の退行検出**を重要な用途として扱います。

結果を、テストしていない製品・バージョンまで一般化しません。

## 証拠を混ぜない

このプロジェクトでは次の証拠を区別します。

1. 仕様・Conformanceの証拠
2. サーバーを直接調査したデバッグ情報
3. 製品向け事前診断（Preflight）の公開メタデータ
4. サーバー側で取得した秘密情報を含まないRuntime Evidence
5. **対象Remote MCPを実クライアントで動かして得たlive evidence**

対象デプロイについて`reach/auth/init/tools`のPASSを出せるのは5だけです。

localhost fixtureの成功は「アダプターが正しく測定できる」ことを示しますが、別の本番デプロイがPASSしたことまでは示しません。

## アダプターを対応済みにする条件

クライアント名を増やすことより、PASSの信頼性を優先します。

少なくとも次を確認します。

- **隔離** — 通常ユーザーの設定・credentialを再利用・変更しない
- **安全に観測できるクライアント経路** — private/minifiedな内部UIへ安易に依存しない
- **モデル非依存** — コアのinterop証拠をLLMのツール選択に依存させない
- **保守的な判定** — 証拠不足は`unknown`にする
- **終了処理** — 一時credential / config / process / stateを片付ける
- **バージョン情報** — 実際に検証した製品バージョンとplatformを記録する
- **fixtureによる検証** — controlled E2Eで、アダプターが本当に実クライアント経路を測定していると確認する

この条件を満たせないクライアントは、PASSの意味を弱めるよりresearch-onlyのままにします。 machine-readableな共通policyと現在のcandidate blockerは[Real-client adapter graduation gate](adapter-graduation-gate.ja.md)で定義します。

## 併用する場合

推奨するrelease pipelineは次のとおりです。

```text
1. MCP仕様への適合性
   -> modelcontextprotocol/conformance

2. 実Remote MCPをデプロイ

3. 実製品との相互運用性
   -> mcp-interop
```

Conformance PASSとmcp-interop PASSは、どちらか一方で他方を代替できません。

## OAuth / diagnoseの境界

genericなMCP/OAuth仕様適合性は公式Conformanceの担当です。

`mcp-interop diagnose --profile <product>`は、特定製品向けの**互換性診断**です。

たとえばChatGPT profileでは、公開メタデータや秘密情報を含まないRuntime Evidenceを、ChatGPTの公開されている認証パターンと照合します。

ルールは次のとおりです。

- 製品固有の期待動作を明示する
- generic MCP/OAuth Conformance suiteを再実装しない
- Preflight、Runtime Evidence、real-client interopを別の証拠として扱う
- メタデータ互換性を`reach/auth/init/tools` PASSへ昇格させない
- 仕様適合性だけを確認したい場合は公式Conformanceを使う

## localhost fixtureの役割

リポジトリ内のfixtureは、アダプターが本当に実クライアントを観測できているか、隔離やcleanupが壊れていないかをrelease前に検証するためのものです。

これは**アダプターのself-test / release gate**であり、任意のclient/serverを仕様適合と認定するためのsuiteではありません。

## mcp-interopが主張しないこと

Interop PASSは次を証明しません。

- MCP仕様への完全な適合
- server / clientのsecurity
- 各tool実装の正しさ
- 破壊的操作の安全性
- modelが正しいtoolを選ぶこと
- テストしていないclient product / versionとの互換性

内部設計は[アーキテクチャ](architecture.ja.md)を参照してください。
