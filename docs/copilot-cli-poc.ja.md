# GitHub Copilot CLI direct MCP inventory PoC

[English](copilot-cli-poc.md) | **日本語**

> この文書は英語版`copilot-cli-poc.md`の日本語訳です。Issue #48の調査記録であり、提供済みアダプターの仕様ではありません。

## 何を調べているか

GitHub Copilot CLIには、MCP設定を非対話で扱う公式コマンドがあります。

- `copilot mcp list [--json]`
- `copilot mcp get <name> [--json]`
- `copilot mcp add --transport http <name> <url>`

さらに、設定・cacheの隔離とheadless認証に使える公式の入力があります。

- `COPILOT_HOME`
- `COPILOT_CACHE_HOME`
- `--additional-mcp-config`
- `COPILOT_MCP_TOOL_CACHE=false`
- 非対話認証用の`COPILOT_GITHUB_TOKEN`、`GH_TOKEN`、`GITHUB_TOKEN`

GitHubはCI/CD、containerなどの非対話環境では環境変数による認証を推奨しています。`COPILOT_GITHUB_TOKEN`は保存済みcredentialより優先されます。利用できるuser-scoped tokenには、**Copilot Requests** account permissionを持つfine-grained PATや対応OAuth/user tokenが含まれ、classic `ghp_` PATは非対応です。

公式資料:

- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
- https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/authenticate-copilot-cli
- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-programmatic-reference
- https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers

## 検証環境

最初のhosted macOS検証:

```text
GitHub Copilot CLI 1.0.79
macOS 26 arm64
Node.js 22
```

2026-08-23にPR #69で再検証した環境:

```text
GitHub Copilot CLI 1.0.80
GitHub-hosted macOS 15.7.7 arm64
```

どちらも一時`COPILOT_HOME`、`COPILOT_CACHE_HOME`、workspaceを使用し、通常のGitHub/model token環境変数を除外しました。localhostへの接続だけを維持し、一般的なoutbound HTTP(S)は到達不能なloopback proxyへ向けています。MCP tool cacheも無効化し、model promptは使っていません。

## `copilot mcp list/get`の結果

`scripts/poc-copilot-cli-mcp.sh`は次を実行します。

```console
copilot mcp list --json
copilot mcp get mcp-interop-fixture
copilot mcp get mcp-interop-fixture --json
```

1.0.79と1.0.80の両方で確認できたこと:

- 3コマンドは正常終了する
- textには`Status: Enabled`と`Tools: * (all)`が表示される
- JSONには設定済み`tools: ["*"]`、URL、source、enabled stateが出る
- しかしlocalhost fixtureには**MCP requestが1件も来ない**
- `initialize`、`notifications/initialized`、`tools/list`は観測されない
- fixtureの実tool `ping`も表示されない

つまり、このmanagement commandが見せているのは**設定情報**であり、実Remote MCPから取得したツール一覧ではありません。

`Tools: *`や設定済みallowlistを`tools=pass`の証拠にしてはいけません。

## 入力なしで実Copilot CLIを起動した結果

`scripts/poc-copilot-cli-startup.sh`は、model promptやユーザー入力を与えず、実`copilot` TUIをPTYで起動します。隔離状態とcontrolled localhost MCP targetを使います。

検証した未認証の両バージョンで、決定的なwire evidenceは同じでした。

```text
initialize                  observed
notifications/initialized  observed
tools/list                  not observed
tools/call                  not observed
```

1.0.80のdebug logでもMCP clientとしてinitializeし、fixture serverの`tools` capabilityを認識したことを確認しています。一方で、隔離環境では`Login status unknown`、利用可能なmodel backendなし、GitHub authentication tokenなしでした。

現時点で言えるのは次までです。

> 実Copilot CLIは、model promptなしで設定済みRemote MCPへ到達し、initializeまでは実行できる。ただし、検証した未認証・no-model起動では`tools/list`まで確認できない。

`tools/list`が後続のauthenticated/model-session lifecycleに依存する可能性はありますが、これは観測結果からの推測であり、公式契約ではありません。

## Copilot CLI 1.0.80再検証 — 2026-08-23

PR #69でstable `@github/copilot` 1.0.80をGitHub-hosted macOSへ固定インストールし、既存2本のcontrolled PoCを再実行しました。

結果は境界を変えるものではなく、1.0.79の結果を再現しました。

- terminal `mcp list/get`は引き続き設定情報のみで、fixtureへMCP lifecycle trafficを送らない
- no-input startupでは`initialize`と`notifications/initialized`まで到達する
- no-input startupでも`tools/list`は送信されない
- `tools/call`は発生しない
- bounded PTY終了後にCopilot processは残らない

`copilot plugins list --kind mcp --json`は今回の判定証拠には使っていません。設定/resource inventoryとfixtureで観測したlive discoveryは引き続き分離します。

## 対応済みの認証隔離経路

現行のGitHub公式資料により、以前の「Copilot CLIにsupportedなheadless認証入力があるか」という問いは解決しました。**あります。** 自動化環境ではsupportedなuser-scoped tokenを環境変数で渡せ、`COPILOT_GITHUB_TOKEN`はKeychainや`gh` fallback credentialより優先されます。

startup PoCには、ambient credentialと明確に分離したresearch-onlyの認証モードを追加しています。

```console
MCP_INTEROP_COPILOT_TEST_TOKEN='github_pat_...' \
MCP_INTEROP_COPILOT_ALLOW_NETWORK=1 \
bash scripts/poc-copilot-cli-startup.sh
```

このモードの安全ルール:

- ambientな`COPILOT_GITHUB_TOKEN`、`GH_TOKEN`、`GITHUB_TOKEN`は使わず、`MCP_INTEROP_COPILOT_TEST_TOKEN`を明示した場合だけ認証モードに入る
- 専用tokenはchild processへ`COPILOT_GITHUB_TOKEN`として渡し、値を出力しない
- Copilot account/APIへ到達する必要があるため、別途`MCP_INTEROP_COPILOT_ALLOW_NETWORK=1`を明示しないと実行しない
- 認証モードではstartup outputとCopilot debug log本文を表示しない
- 認証モードでは`MCP_INTEROP_KEEP_COPILOT_POC_TMP=1`による一時状態保持を拒否する
- 実行前後で通常`~/.copilot`状態とlogin Keychain fileのmetadata/hashを比較する
- `tools/call`は禁止のままにし、fixtureで直接観測したtool discoveryだけを昇格判定に使う

日常利用中のtokenを流用せず、必要最小限のCopilot権限を持つ専用・revoke可能なtest credentialを使います。通常のrepository/Actions installation tokenを、公式に記載されたuser-scoped Copilot認証tokenの代用品とはみなしません。

このharnessを追加したこと自体は、認証済み`tools/list`成功の証拠ではありません。実際のcontrolled authenticated runでevidenceとstate-isolation gateを満たすまではBLOCKEDです。

## 現在の判定

**完全な`mcp-interop`アダプターとしてはBLOCKEDです。**

コアPASSにはtool discoveryまでの直接証拠が必要です。1.0.79 / 1.0.80の検証済み未認証境界は次のとおりです。

- terminal management — 設定は見えるが、Remote MCPへ接続しない
- no-input startup — `initialize`までは到達するが、認証済みmodel backendなしでは`tools/list`は未観測

したがって、設定状態やMCP initialization成功だけを理由に`tools=pass`を返すアダプターは追加しません。

## 次に確認すべき安全な条件

残るゲートは「隔離方法の発見」ではなく、**実際のcontrolled authenticated run**です。

最低限、次を要求します。

1. 専用のsupported user-scoped Copilot test tokenを明示opt-in経路から渡す
2. 一時config / cache / workspaceとno-model core pathを維持する
3. fixtureで`tools/list`を直接観測する
4. `tools/call`が発生しないことを維持する
5. 通常ユーザーのconfig / credential stateが変化しないことを証明する
6. owned processをすべてcleanupする

この条件でもmodel promptなしで`tools/list`を証明できない場合や、credential隔離を弱める必要がある場合はIssue #48をresearch-onlyのままにします。

## 将来アダプターへ昇格する条件

最低限、次をすべて満たす必要があります。

1. 実Copilot CLIの正確なversionを記録できる
2. config / cache / workspace / credentialを隔離できる
3. model promptをコア証拠として使わない
4. fixtureで`initialize`または対象protocol世代に相当するreadiness evidenceを観測できる
5. 必要なinitialized/readiness progressionを観測できる
6. `tools/list`または同等の実クライアント由来tool-discovery evidenceを観測できる
7. supported surfaceがある場合はclient-side inventoryでもfixtureの`ping`を識別できる
8. `tools/call`が発生しない
9. 通常ユーザーの設定・credentialが変化しない
10. owned client processを残さない

## OAuth

Copilot account authenticationとRemote MCP OAuthは別の問題なので混同しません。Remote MCP向けに文書化されている`client_credentials`はCopilotからMCP serverへの認証方法であり、Copilot account/model-backendの認証境界そのものを解決するものではありません。

## 実行方法

未認証startup PoC:

```console
bash scripts/poc-copilot-cli-startup.sh
```

明示的な認証済みresearch PoC:

```console
MCP_INTEROP_COPILOT_TEST_TOKEN='github_pat_...' \
MCP_INTEROP_COPILOT_ALLOW_NETWORK=1 \
bash scripts/poc-copilot-cli-startup.sh
```

`MCP_INTEROP_KEEP_COPILOT_POC_TMP=1`は未認証のsanitized diagnosticにだけ使えます。認証モードでは一時状態保持を拒否します。
