# VS Code Agent Plugin MCP PoC

[English](vscode-agent-plugin-poc.md) | [日本語](vscode-agent-plugin-poc.ja.md)

Status: #6向けのexperimental researchです。まだrelease adapterではありません。

## Goal

実VS Code MCP clientがlocalhost Streamable HTTP MCP fixtureへ到達し、次のdirect lifecycleを実行できるか検証します。

```text
initialize -> notifications/initialized -> tools/list
```

model prompt、browser/DOM automation、private workbench command IDは使用しません。

## Why this path is worth testing

現在のVS Code Agent Plugin documentationにはsupported local-plugin surfaceとして次が公開されています。

- `chat.pluginLocations`でlocal plugin directoryを登録できる
- pluginは`.mcp.json`内にMCP server definitionを含められる
- plugin MCP serverはplugin有効時にautomatic startするとされている
- plugin MCP serverはimplicitly trustedで、workspace-MCPの別trust promptを使用しない

VS Codeはisolated instance向けに`--user-data-dir`と`--extensions-dir`もdocumentしています。これらを組み合わせることで、#6のno-auth部分について不足しているdirect no-model lifecycle pathを提供できる可能性があります。

References:

- https://code.visualstudio.com/docs/agent-customization/agent-plugins
- https://code.visualstudio.com/docs/agent-customization/mcp-servers
- https://code.visualstudio.com/docs/configure/command-line

## PoC design

`scripts/poc-vscode-agent-plugin.sh`は次を行います。

1. macOSとinstalled `code` CLIを必須とする
2. 既存のlocalhost `internal/e2e/fixture`をbuildする
3. temporary VS Code `--user-data-dir`、temporary `--extensions-dir`、empty workspace、local Agent Pluginを作成する
4. Agent Pluginsを有効化し、temporary pluginだけをisolated profile内へ登録する
5. plugin MCP serverをloopback fixtureへ向ける
6. real VS Code executableを起動し、loopback accessを維持しつつ一般的なexternal-network proxy variableをclosed loopback portへ向ける
7. server-side wire evidenceをauthoritativeとして扱う
8. dedicated fixture pathで`initialize`、`notifications/initialized`、`tools/list`を必須にする
9. `tools/call`が発生したらfailする
10. command lineにunique temporary `--user-data-dir` pathを含むVS Code processだけを停止する
11. 通常VS Codeの`settings.json`とuser `mcp.json` metadataが変化していないことを確認する

Run locally:

```bash
bash scripts/poc-vscode-agent-plugin.sh
```

Keep diagnostics when needed:

```bash
MCP_INTEROP_KEEP_VSCODE_POC_TMP=1 bash scripts/poc-vscode-agent-plugin.sh
```

初期research時に使用したtemporary self-hosted PoC workflowは結果取得後に削除済みです。harness自体はexplicit local rerun用として残しています。

## Tested result — 2026-08-11

maintainer macOS machineで次を検証しました。

```text
VS Code 1.132.0
df53daabb18cd157bdb08c7f01c34df936cf12f4
arm64
```

isolated local pluginはdiscoveryされました。VS Codeはplugin-provided server向けに専用の`mcpServer.plugin...mcp-interop-vscode-poc.log`を作成し、`mcpGateway.log`もinitializeしました。しかし、no-input launchを繰り返してもcontrolled fixtureへの**MCP requestは0件**でした。`initialize`、`notifications/initialized`、`tools/list`、`tools/call`はいずれも観測されませんでした。

`chat.mcp.access`を`true`、`chat.mcp.autoStart`を`newAndOutdated`へ明示設定しても結果は同じでした。closed outbound proxyを外したdiagnostic runでもfixture trafficは0件だったため、network-isolation proxyがblockerではありません。

### Current verdict

**direct no-model adapter contractとしてはBLOCKEDです。**

このstable buildでは、local Agent Plugin discoveryだけではno-input VS Code launchにplugin MCP serverをstartさせるには不十分です。public documentationはplugin有効時にplugin MCP serverがautomatic startするとしていますが、検証したlaunch pathではdiscovered serverにwire activityがありませんでした。これはprotocol failureではなく、追加のproduct lifecycle condition（例: chat/workbench activation）が必要である可能性があります。PoCは不足するdirect lifecycleをCommand Palette/UI automationで置き換えません。

issue #6はopenのまま維持します。VS Codeがmodel promptやbrittle UI controlを使わずdeterministicに動くsupported startup/status/tool-discovery pathを公開または修正した場合、このharnessを再実行します。

## PASS meaning

PASSは、検証したVS Code buildとAgent Plugin feature stateにおいて、supported public plugin configuration pathが実VS Code MCP clientに、model participationなしでdirect no-auth initializationとtool discoveryを実行させられることを証明します。

これは#6のno-auth部分を以前のCLI-only BLOCKED stateから進め、同じisolated-session patternに基づくreal VS Code adapter実装を正当化するのに十分です。

## What this does not prove

- OAuth completionまたはcredential isolation
- stable client-side machine-readable MCP status API
- organization policyでAgent Pluginsがdisabledの場合のcompatibility
- Preview変更後もAgent Plugin surfaceがstableであること
- workspace `.vscode/mcp.json` lifecycleをheadlessにdriveできること
- release readiness

OAuthは別gateのままです。このPoCからauthenticated interoperabilityを推測してはいけません。

## Failure interpretation

fixture trafficがないことは、自動的にprotocol failureを意味しません。次のいずれかである可能性があります。

- `chat.plugins.enabled`が利用できない、またはpolicyでdisabled
- installed VS Code buildがisolated profileでAgent Pluginsをloadしない
- plugin MCP auto-startにrunner上では満たされないproduct/session conditionが必要
- HTTP plugin MCP configuration shapeが変更された
- execution contextからVS Codeがisolated desktop instanceをlaunchできなかった

この場合は#6をopenのまま維持し、別のsupported surfaceが存在するか判断する前にVS Code/fixture diagnosticsを取得します。
