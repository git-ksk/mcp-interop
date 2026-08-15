# GitHub Copilot CLI direct MCP inventory PoC

[English](copilot-cli-poc.md) | [日本語](copilot-cli-poc.ja.md)

このdocumentはissue #48のresearch boundaryを記録するものです。これは**PoCであり、shipped adapter contractではありません**。

## Why this surface is interesting

GitHubの現在のCopilot CLI documentationには、supported non-interactive MCP management surfaceとして次が公開されています。

- `copilot mcp list [--json]`
- `copilot mcp get <name> [--json]`
- `copilot mcp add --transport http <name> <url>`

command referenceでは、`copilot mcp`はinteractive sessionを開始せずcommand lineから利用でき、`mcp get`はserver configurationとtoolsを表示すると説明されています。

CLIはさらに次をsupportします。

- `COPILOT_HOME`: 通常の`~/.copilot` configuration/state directoryを置換する
- `COPILOT_CACHE_HOME`: cache directoryを別途redirectする
- `--additional-mcp-config`: session-only MCP definitionを追加する
- `COPILOT_MCP_TOOL_CACHE=false`: MCP tool snapshot cachingを無効化する

Official references:

- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference
- https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers

## Tested baseline

hosted macOS researchではofficial npm packageを使用しました。

```text
GitHub Copilot CLI 1.0.79
macOS 26 arm64
Node.js 22
```

CLIはisolated temporary `COPILOT_HOME`、`COPILOT_CACHE_HOME`、workspaceで実行しました。一般的なGitHub/model token environment variableは除外し、localhostへの接続を維持したまま、通常のoutbound HTTP(S)は到達不能なloopback proxyへredirectしました。

## Terminal MCP management result

`scripts/poc-copilot-cli-mcp.sh`は次を実行します。

```console
copilot mcp list --json
copilot mcp get mcp-interop-fixture
copilot mcp get mcp-interop-fixture --json
```

controlled configurationでは`deferTools: "never"`を使用し、processには`COPILOT_MCP_TOOL_CACHE=false`を設定します。

Copilot CLI 1.0.79での観測結果:

- 3つのterminal management commandはすべて正常終了する
- textは`Status: Enabled`と`Tools: * (all)`を報告する
- JSONはconfigured `tools: ["*"]`、URL、source、enabled stateを報告する
- localhost fixtureには**MCP requestが一切届かない**
- `initialize`、`notifications/initialized`、`tools/list`は観測されない
- 実fixture toolの`ping`はmanagement outputに返らない

したがって、検証したterminal `mcp list/get` pathは**configuration/inventory managementであり、direct live MCP tool discoveryではありません**。configuration registrationやconfigured `tools` allowlistをinteroperability PASSへ昇格させてはいけません。

## No-input real-client startup result

`scripts/poc-copilot-cli-startup.sh`は、**inputもmodel promptも与えず**実`copilot` TUIをPTY内で起動します。同じisolated stateとloopback-only MCP targetを使用します。

30秒のhosted-macOS observationでは次を確認しました。

```text
initialize                  observed
notifications/initialized  observed
tools/list                  not observed
tools/call                  not observed
```

Copilotのdebug logでも、MCP serviceがclientとしてinitializeされ、fixture serverの`tools` capabilityを認識したことを記録しています。一方、isolated environmentにはmodel backend/account authenticationが存在しませんでした（`Login status unknown`、GitHub auth tokenなし）。

これは有用なpartial boundaryを証明します。**実Copilot CLI startupはmodel promptなしでconfigured Remote MCP serverへ到達しinitializeできます**が、検証したunauthenticated/no-model startupではobservable tool discoveryまで進みません。

`tools/list`が後続のauthenticated/model-session lifecycle stepにgatedされている可能性が高い、というのが現時点の解釈です。これはobserved wire traceとdebug logからの推測であり、documented Copilot contractではありません。

## Current verdict

**complete direct `mcp-interop` adapterとしてはBLOCKEDです。**

projectはtool discoveryを含むcore path全体についてdirect evidenceを要求します。Copilot CLI 1.0.79で現在確認できているのは次です。

- terminal management: safe configuration visibilityはあるがlive server contactはない
- no-input real-client startup: live `initialize` + `notifications/initialized`までは到達するが、authenticated/model backendなしでは`tools/list`がない

`Tools: *`、configuration state、MCP initialization成功だけを根拠に`tools=pass`を報告するCopilot adapterをshipしてはいけません。

## Next safe gate

残るresearch questionは、**isolated authenticated Copilot session**が、ユーザーの通常credential stateをcopy/mutateせず、model promptをinteroperability oracleとして要求せずに`tools/list`まで到達できるかです。

通常ユーザーのtoken、Keychain entry、`~/.copilot` credential stateをPoCへcopyしてはいけません。Copilotがこのtestに適したsupported isolated authentication mechanismを提供しない場合、issue #48はresearch-onlyのまま維持します。

## PASS contract for a future adapter

future PoCがgraduateできるのは、以下をすべて満たす場合だけです。

1. 実際にインストールされたCopilot CLI/versionを記録する
2. config/cache/workspaceとcredential stateを安全にisolateする
3. core evidence mechanismとしてmodel promptを使用しない
4. fixtureが`initialize`を観測する
5. fixtureが`notifications/initialized`を観測する
6. fixtureが`tools/list`を観測する
7. supported surfaceが存在する場合、client-observable inventoryがcontrolled `ping` toolを識別する
8. fixtureが`tools/call`を観測しない
9. 通常ユーザーのconfiguration/credential stateが変化しない
10. owned client processをleakしない

## OAuth

no-auth/tool-discovery boundaryが解決するまではRemote-MCP OAuthをscope外とします。Copilot account authenticationとRemote-MCP OAuthは別のconcernであり、混同してはいけません。

## Running

Terminal management PoC:

```console
bash scripts/poc-copilot-cli-mcp.sh
```

No-input startup PoC:

```console
bash scripts/poc-copilot-cli-startup.sh
```

sanitized diagnosticsを保持する必要がある場合だけ`MCP_INTEROP_KEEP_COPILOT_POC_TMP=1`を設定してください。保持したdirectoryにはclient logが含まれる可能性があるため、共有前に必ずreviewしてください。
