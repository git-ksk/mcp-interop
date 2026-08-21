# VS Code Agent Plugin MCP PoC

[English](vscode-agent-plugin-poc.md) | **日本語**

> この文書は英語版`vscode-agent-plugin-poc.md`の日本語訳です。Issue #6向けの実験的調査であり、提供済みアダプターではありません。

## 目的

実VS CodeのMCPクライアントが、localhostのStreamable HTTP MCP fixtureへ到達し、次のlifecycleを**モデルへのプロンプトなし**で実行できるか検証します。

```text
initialize -> notifications/initialized -> tools/list
```

browser / DOM automationやprivate workbench command IDは使いません。

## なぜAgent Plugin経路を調べるのか

VS CodeのAgent Plugin資料では、次の公開機能があります。

- `chat.pluginLocations`でlocal plugin directoryを登録できる
- pluginの`.mcp.json`へMCP serverを定義できる
- plugin有効時にMCP serverが自動起動すると説明されている
- plugin由来のMCP serverはimplicit trustとして扱われる

VS Code自体も隔離起動向けに`--user-data-dir`と`--extensions-dir`を提供しています。

これらを組み合わせれば、通常ユーザーのVS Code設定を触らず、実クライアントがMCP lifecycleを開始するか確認できる可能性があります。

参考:

- https://code.visualstudio.com/docs/agent-customization/agent-plugins
- https://code.visualstudio.com/docs/agent-customization/mcp-servers
- https://code.visualstudio.com/docs/configure/command-line

## PoCの構成

`scripts/poc-vscode-agent-plugin.sh`は次を行います。

1. macOSとinstalled `code` CLIを確認
2. localhost用`internal/e2e/fixture`をbuild
3. 一時`--user-data-dir`、`--extensions-dir`、空workspace、local Agent Pluginを作成
4. Agent Pluginsを有効にし、一時pluginだけを隔離profileへ登録
5. plugin MCP serverをloopback fixtureへ向ける
6. 実VS Codeを起動する
7. server-side wire evidenceを判定の正とする
8. `initialize`、`notifications/initialized`、`tools/list`を確認する
9. `tools/call`が発生したらFAIL
10. 一時`--user-data-dir`を引数に持つVS Code processだけを停止する
11. 通常VS Codeの`settings.json`とuser `mcp.json`のメタデータが変化していないことを確認する

実行:

```bash
bash scripts/poc-vscode-agent-plugin.sh
```

診断情報を残す場合:

```bash
MCP_INTEROP_KEEP_VSCODE_POC_TMP=1 bash scripts/poc-vscode-agent-plugin.sh
```

## 検証結果 — 2026-08-11

maintainerのmacOS環境で確認したbuild:

```text
VS Code 1.132.0
df53daabb18cd157bdb08c7f01c34df936cf12f4
arm64
```

一時local plugin自体は発見されました。

VS Codeはplugin server向けログと`mcpGateway.log`を作成しましたが、入力なし起動を繰り返してもcontrolled fixtureへの**MCP requestは0件**でした。

```text
initialize                  not observed
notifications/initialized  not observed
tools/list                  not observed
tools/call                  not observed
```

`chat.mcp.access=true`、`chat.mcp.autoStart=newAndOutdated`を明示しても結果は同じでした。

外部networkを閉じるproxyを外した診断runでもfixture trafficは0件だったため、network isolationが主因とは考えていません。

## 現在の判定

**direct no-model adapterとしてはBLOCKEDです。**

確認したVS Code buildでは、local Agent Pluginが発見されただけでは、入力なし起動時にplugin MCP serverのwire activityは始まりませんでした。

これはMCP protocol failureを証明するものではありません。

chat/workbench activationなど、追加のproduct lifecycle条件が必要な可能性があります。

ただしPoCでは、その不足をCommand Palette操作やUI automationで埋めません。

Issue #6はopenのまま維持します。

VS Codeが、モデルや壊れやすいUI操作を使わずにMCP startup / status / tool discoveryを行える公式・安定した経路を提供した場合、このハーネスを再実行します。

## 将来PASSした場合に何を証明するか

このPoCのPASSは、検証対象VS Code buildとAgent Plugin feature stateで、公開されたplugin設定だけを使い、**実VS Code MCP clientがモデル参加なしにRemote MCPを初期化し、ツール発見まで実行できた**ことを意味します。

それが確認できれば、同じ隔離session patternを使うreal VS Code adapterを検討できます。

## このPoCが証明しないもの

- OAuth完遂やcredential isolation
- 安定したclient-side machine-readable MCP status API
- organization policyでAgent Pluginsが無効な環境との互換性
- Preview変更後もAgent Plugin interfaceが維持されること
- workspace `.vscode/mcp.json` lifecycleをheadlessに操作できること
- release readiness

OAuthは別のgateです。このPoCからauthenticated interoperabilityを推測しません。

## fixture trafficが無い場合の解釈

MCP requestが来ないことは、自動的にprotocol failureを意味しません。

たとえば次の可能性があります。

- `chat.plugins.enabled`が利用できない、またはpolicyで無効
- 対象VS Code buildが隔離profileでAgent Pluginsをloadしない
- plugin MCP auto-startに追加のproduct/session conditionが必要
- HTTP plugin MCP configuration形式が変わった
- 実行環境からisolated desktop instanceを起動できなかった

この場合はIssue #6をopenのまま維持し、別のsupported surfaceへ進む前にVS Code / fixture diagnosticsを確認します。
