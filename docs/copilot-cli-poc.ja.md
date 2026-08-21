# GitHub Copilot CLI direct MCP inventory PoC

[English](copilot-cli-poc.md) | **日本語**

> この文書は英語版`copilot-cli-poc.md`の日本語訳です。Issue #48の調査記録であり、提供済みアダプターの仕様ではありません。

## 何を調べているか

GitHub Copilot CLIには、MCP設定を非対話で扱う公式コマンドがあります。

- `copilot mcp list [--json]`
- `copilot mcp get <name> [--json]`
- `copilot mcp add --transport http <name> <url>`

さらに次の隔離手段も公開されています。

- `COPILOT_HOME`
- `COPILOT_CACHE_HOME`
- `--additional-mcp-config`
- `COPILOT_MCP_TOOL_CACHE=false`

これらを使えば、通常ユーザーの設定・認証情報を触らず、実Copilot CLIがRemote MCPへどこまで到達するか確認できる可能性があります。

公式資料:

- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference
- https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers

## 検証環境

macOS上で公式npm packageを使って確認しました。

```text
GitHub Copilot CLI 1.0.79
macOS 26 arm64
Node.js 22
```

一時`COPILOT_HOME`、`COPILOT_CACHE_HOME`、workspaceを使い、通常のGitHub/model token環境変数は除外しました。

localhostへの接続だけを維持し、一般的なoutbound HTTP(S)は到達不能なloopback proxyへ向けました。

## `copilot mcp list/get`の結果

`scripts/poc-copilot-cli-mcp.sh`は次を実行します。

```console
copilot mcp list --json
copilot mcp get mcp-interop-fixture
copilot mcp get mcp-interop-fixture --json
```

Copilot CLI 1.0.79で確認できたこと:

- 3コマンドは正常終了する
- textには`Status: Enabled`と`Tools: * (all)`が表示される
- JSONには設定済み`tools: ["*"]`、URL、source、enabled stateが出る
- しかしlocalhost fixtureには**MCP requestが1件も来ない**
- `initialize`、`notifications/initialized`、`tools/list`は観測されない
- fixtureの実tool `ping`も表示されない

つまり、このmanagement commandが見せているのは**設定情報**であり、実Remote MCPから取得したツール一覧ではありません。

`Tools: *`や設定済みallowlistを`tools=pass`の証拠にしてはいけません。

## 入力なしで実Copilot CLIを起動した結果

`scripts/poc-copilot-cli-startup.sh`は、モデルへのプロンプトやユーザー入力を与えず、実`copilot` TUIをPTYで起動します。

30秒の観測では次を確認しました。

```text
initialize                  observed
notifications/initialized  observed
tools/list                  not observed
tools/call                  not observed
```

debug logでもMCP clientとしてinitializeし、fixture serverの`tools` capabilityを認識したことを確認しています。

ただし、隔離環境にはmodel backend / account authenticationがありませんでした。

現時点で言えるのは次までです。

> 実Copilot CLIは、モデルへのプロンプトなしで設定済みRemote MCPへ到達し、initializeまでは実行できる。ただし、検証した未認証・no-model起動では`tools/list`まで確認できない。

`tools/list`が後続のauthenticated/model-session lifecycleに依存する可能性はありますが、これは観測結果からの推測であり、公式契約ではありません。

## 現在の判定

**完全な`mcp-interop`アダプターとしてはBLOCKEDです。**

コアPASSにはtool discoveryまでの直接証拠が必要です。

現在確認できているのは:

- terminal management — 設定は見えるが、Remote MCPへ接続しない
- no-input startup — `initialize`までは到達するが、`tools/list`は未観測

したがって、設定状態やMCP initialization成功だけを理由に`tools=pass`を返すアダプターは追加しません。

## 次に確認すべき安全な条件

残る問いは、**隔離した認証済みCopilot sessionで、通常ユーザーのcredentialをコピー・変更せず、モデルをinterop判定器として使わずに`tools/list`まで到達できるか**です。

通常ユーザーのtoken、Keychain、`~/.copilot` credential stateをPoCへコピーしません。

適切なsupported isolation mechanismが無ければ、Issue #48はresearch-onlyのままにします。

## 将来アダプターへ昇格する条件

最低限、次をすべて満たす必要があります。

1. 実Copilot CLIの正確なversionを記録できる
2. config / cache / workspace / credentialを隔離できる
3. model promptをコア証拠として使わない
4. fixtureで`initialize`を観測できる
5. `notifications/initialized`を観測できる
6. `tools/list`を観測できる
7. 可能ならclient-side inventoryでもfixtureの`ping`を識別できる
8. `tools/call`が発生しない
9. 通常ユーザーの設定・credentialが変化しない
10. owned client processを残さない

## OAuth

認証不要のtool-discovery境界が解決するまでは、Remote MCP OAuthはscope外です。

Copilotアカウントへの認証と、Remote MCPのOAuthは別の問題なので混同しません。

## 実行方法

MCP management PoC:

```console
bash scripts/poc-copilot-cli-mcp.sh
```

入力なしstartup PoC:

```console
bash scripts/poc-copilot-cli-startup.sh
```

一時diagnosticを残す場合のみ:

```console
MCP_INTEROP_KEEP_COPILOT_POC_TMP=1 bash scripts/poc-copilot-cli-startup.sh
```

保存したdirectoryにはclient logが含まれる可能性があるため、共有前に必ず内容を確認してください。
