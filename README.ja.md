# mcp-interop

[![CI](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml/badge.svg)](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/git-ksk/mcp-interop)](https://github.com/git-ksk/mcp-interop/releases/latest)
[![License](https://img.shields.io/github/license/git-ksk/mcp-interop)](LICENSE)

[English](README.md) | [日本語](README.ja.md)

**Remote MCP serverが、実際のMCP clientで本当に動くかをlive testするinterop runner。**

`mcp-interop`は、Remote Model Context Protocol (MCP) serverを実クライアントでblack-box検証するcross-client test runnerです。

このツールが答えたいのは、spec上の適合性だけでは分からない次の問いです。

> このRemote MCP deploymentは、ユーザーが実際に使っているclientから接続・認証・初期化でき、toolsを発見できるか？

また、安全にheadless automationできる実client surfaceがまだ無い対象向けに、client profileベースの**preflight診断**も提供します。preflightの結果をlive interoperability PASSとして扱うことはありません。

## Status

**v0.1.0を公開済みです。**

Release: [v0.1.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.1.0)

現在のlive adapter:

- **Codex CLI** — live inventory + 明示的opt-in OAuth
- **Cursor CLI (beta)** — dedicated MCP management commandを使うno-auth live inventory。OAuth完遂は未対応
- **Antigravity CLI (beta, macOS)** — isolated no-prompt PTY startupとmachine-readable MCP tool cacheを使うno-auth live inventory。OAuth完遂は意図的に無効

開発branchには、**ChatGPT OAuth/server preflight profile**もあります。ChatGPTの公開された認証仕様に対してMCP/OAuth metadataを診断しますが、実ChatGPT clientを動かしたとは主張しません。

VS Codeは、stableなno-model server-start/tool-discovery surfaceが確認できるまでresearch-onlyです。

GitHub Copilot CLIは今後の候補です。Claude Code対応は現時点では優先していません。

## Install

Go 1.24以降で、現在のstable releaseを固定して入れる場合:

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@v0.1.0
```

最新の公開module versionを追う場合:

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@latest
```

version確認:

```console
mcp-interop version
# または
mcp-interop --version
```

[v0.1.0 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.1.0)には、macOS / Linux / Windows向けのamd64 / arm64 archiveと`checksums.txt`もあります。

## 何を証明するテストか

1 clientの完全なlive testは4 stageです。

1. `reach` — 実clientがRemote MCPへ到達し、live interactionを確認できた
2. `auth` — 必要なclient authenticationが完了した、またはlive tool discoveryにより認証不要を確認できた
3. `init` — MCP session initializationが完了した
4. `tools` — clientがserverのtoolsを発見した

exit code `0`になるのは、**4 stageすべてが`pass`の場合だけ**です。

`fail`、`skip`、`unknown`はすべてnon-zeroです。証拠不足のinterop結果をCIが成功扱いしないための仕様です。

`diagnose` commandは別contractです。公開metadataから`PREFLIGHT PASS` / `PREFLIGHT FAIL`を返しますが、real-clientの`reach/auth/init/tools` PASSの代用にはしません。

`mcp-interop`は次を保証しません。

- serverのsecurity
- 各tool実装の正しさ
- destructive operationの安全性
- AI modelが正しいtoolを選択すること

## CLI

検出可能なclientを確認:

```console
mcp-interop clients
mcp-interop clients --json
```

1 clientをlive test:

```console
mcp-interop test https://example.com/mcp --client codex
mcp-interop test https://example.com/mcp --client cursor
mcp-interop test https://example.com/mcp --client antigravity
```

複数clientを順番にtest:

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity
```

複数client選択時のtext outputはcross-client summaryから始まります。

```text
SUMMARY
CLIENT           REACH  AUTH  INIT  TOOLS  VERSION
Codex CLI        PASS   PASS  PASS  PASS   codex-cli 0.133.0
Cursor CLI       PASS   PASS  PASS  PASS   2026.08.04-aaa8809
Antigravity CLI  PASS   PASS  PASS  PASS   1.1.11
```

JSON outputはarrayです。

## Codex OAuth

OAuthが必要なtargetをCodexで検証する場合は明示的にopt-inします。

```console
mcp-interop test https://example.com/mcp --client codex --oauth
```

`--oauth`を付けてもbrowserを勝手には開きません。authorization URLをstderrへ表示し、実Codex OAuth callbackを待ちます。

URLには短時間有効なOAuth stateが含まれるため、Issueやlog共有時に貼らないでください。

Cursor / AntigravityのOAuth完遂はv0.1.xでは未対応です。

## ChatGPT OAuth/server preflight

Remote MCP serverが公開しているOAuth metadataに、ChatGPTの現在の認証pathと既知のblocking mismatchがないか確認します。

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

このprofileはProtected Resource MetadataとAuthorization Server Metadataを辿り、主に次を確認します。

- HTTPS endpoint
- `authorization_servers`
- `authorization_endpoint` / `token_endpoint`
- Client ID Metadata Documents (CIMD) / Dynamic Client Registration (DCR)
- CIMD時の`none` / `private_key_jwt` token endpoint auth互換性
- PKCE `S256`
- `offline_access`などrefresh token継続性の参考情報
- protected-resource `resource`の整合性

`client_id_metadata_document_supported: true`が広告されているserverは、`registration_endpoint`が無くてもChatGPT registration preflightをPASSできます。ChatGPTはCIMD pathを利用でき、DCRを必須としないためです。

Authorization Serverのsanitized logから、実ChatGPT authorization requestの非secretな`client_id`と`redirect_uri`を確認できる場合は、さらにexactな照合ができます。

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

この拡張診断では、CIMD document、redirect URI、client/server間のtoken endpoint auth method、`private_key_jwt`利用時のJWKS到達性まで確認します。

**このcommandはChatGPT UIを操作せず、OAuthを完遂せず、実ChatGPT client PASSを主張しません。** 詳細は[ChatGPT接続診断](docs/chatgpt-diagnostics.ja.md)を参照してください。

## Codex adapter

Codex adapterは:

1. isolated temporary `CODEX_HOME`を作る
2. OAuth credential storageをtemporary HOME内fileへ強制する
3. test対象Remote MCP endpointだけをisolated configへ書く
4. 実`codex app-server`を起動する
5. app-server control connectionをinitializeする
6. `mcpServerStatus/list`でtool inventoryを確認する
7. `--oauth`指定時のみCodex自身のOAuth flowを使う
8. Codexが実際に観測した結果をreportする
9. OAuth credentialを含むtemporary session全体をcleanupする

model promptは送信しません。live inventory/OAuth testのためにmodel/API usageを必要としません。

### OAuth isolation

temporary configでは次を設定します。

```toml
mcp_oauth_credentials_store = "file"
```

通常のautomatic/keyring storageではなくisolated `CODEX_HOME`内にcredentialを閉じ込めます。

## Cursor adapter (beta)

Cursor adapterは:

- isolated temporary `HOME` + workspaceを作る
- `<workspace>/.cursor/mcp.json`へtest endpointだけを書く
- 実Cursor CLIのMCP management surfaceを使う
- `mcp enable` → `mcp list` → `mcp list-tools`を使う
- successful `mcp list-tools`をreal-client evidenceとして扱う
- temporary Cursor stateをcleanupする

Cursor model promptは送信しません。

### 現在の制限

- OAuth discovery/DCR/PKCE flow start/local callbackまではmaintainer PoCで確認済み
- token exchange + authenticated `tools/list`は未完了
- OAuth-required targetでは自動`mcp login`を実行せずincomplete resultにする
- MCP command outputは専用JSON contractではなくhuman-readable
- 初期live validationはmacOS中心

## Antigravity adapter (beta, macOS)

Antigravity adapterは:

- isolated temporary `HOME` + workspaceを作る
- temporary `~/.gemini/config/mcp_config.json`へ`serverUrl`形式でtargetを書く
- 入力/model promptなしで実`agy`をPTY起動する
- isolated `~/.gemini/antigravity-cli/mcp/<server>/` tool cacheを観測する
- valid tool schemaをreach/init/tools evidenceとして扱う
- test PTY wrapper由来と証明できるdescendant processだけを回収する
- temporary HOME/workspaceをcleanupする

### 現在の制限

- live adapterはmacOSのみ
- OAuth-required discovery/DCRは観測済み
- macOS Keychainと完全に隔離できるsupported mechanismが未確立のためauthorization/token exchangeは無効
- tool cacheが確認できない場合は成功を推測せず`unknown`
- tool cacheはcross-vendor stable APIではなく、Antigravity clientのobserved surface

## Safety / isolation

- **実clientを使う。** emulatorでclient behaviorを再現したことをinterop成功としない
- **model benchmarkではない。** core pathでmodelにtool selection/callを依頼しない
- **user configを変更しない。** safe isolationできなければ`skip`/`unknown`
- **temporary stateをprivateにする。** POSIXではowner-only permissionを使用
- **secret redaction。** Bearer/OAuth materialやcredential-like URL parameterをreportから除去
- **OAuthは明示的。** verified isolated implementationがあるadapterでopt-inされた場合のみ開始
- **preflightはlive evidenceではない。** profile diagnosticでpublic metadata互換性を確認しても、実clientの`reach/auth/init/tools` PASSへ昇格させない
- **hosted backend不要。** core toolはlocal/CIで動作

## Real-client E2E on macOS

repoにはdeterministic localhost MCP fixtureとrelease-gate runnerがあります。

```console
bash scripts/e2e-real-clients.sh
```

defaultではCodex / Cursor / Antigravityを検証します。

subset指定:

```console
MCP_INTEROP_CLIENTS=codex,cursor bash scripts/e2e-real-clients.sh
```

harnessは:

- current checkoutをbuild/test
- `127.0.0.1`だけにbindするGo fixtureを起動
- 各clientを別fixture pathで実行
- `initialize` / `notifications/initialized` / `tools/list`を必須evidenceにする
- `tools/call`が発生したらFAIL
- common model/API key envを子processから除外
- user MCP/config/credential metadataをbefore/after比較
- login Keychain DBをdefaultで比較
- 新規leaked `codex` / `cursor-agent` / `agy` processを検出
- 新規`mcp-interop-*` temporary session漏れを検出

通常のGitHub-hosted CIは外部clientをインストールしません。実client E2E用にはself-hosted macOS ARM64向けmanual workflowがあります。

## ドキュメント

- [Architecture](docs/architecture.ja.md)
- [Troubleshooting](docs/troubleshooting.ja.md)
- [Reason code](docs/reason-codes.ja.md)
- [ChatGPT接続診断](docs/chatgpt-diagnostics.ja.md)
- [Contributing](CONTRIBUTING.ja.md)
- [Security Policy](SECURITY.ja.md)
- [CHANGELOG](CHANGELOG.md) — release historyのcanonical版は英語

## Roadmap

### v0.2 — authentication completeness

- [x] clientが観測したDCR failure向けstructured OAuth reason code
- [x] ChatGPT向けPRM / CIMD / DCR / PKCE / token-auth preflight診断。live-client verdictとは分離
- [ ] profile診断evidenceと追加のreal-client OAuth failureを相関
- [ ] Cursor OAuth token exchange + authenticated tool discovery
- [ ] Antigravity OAuthを安全に完遂できるcredential isolation boundaryの確立
- [ ] 残る`unknown` / incomplete result向けsanitized verbose trace

### v0.3 — client coverage

- [ ] real ChatGPT adapterを追加する前にsupportedなheadless ChatGPT MCP/app-management surfaceを調査。brittleなDOM scrapingはinterop contractに使わない
- [ ] supported direct lifecycle/tool-discovery surfaceが利用可能になったらVS Codeを再検討
- [ ] stable automatable MCP inventory surfaceが確認できたらGitHub Copilot CLIを評価
- [ ] beta adapterのOS/client-version evidenceを拡充

### v0.1.0で提供済み

- [x] `pass` / `fail` / `skip` / `unknown` result model
- [x] isolated test-session lifecycle + secret redaction
- [x] Codex CLI live inventory adapter
- [x] Codex OAuth live flow
- [x] Cursor CLI no-auth live adapter (beta)
- [x] Antigravity CLI no-auth live adapter (beta, macOS)
- [x] cross-client combined text report
- [x] repeatable real-client macOS E2E harness
- [x] versioned release build/release automation

## 現在のnon-goals

- MCP security scanning
- tool quality / LLM-selection benchmark
- runtime sandboxing
- permission/capability governance
- 新しいOAuth/MCP conformance specificationの策定
- 対応clientを実際に動かさず互換性を保証すること

## Contributing / Security

Contributionは[CONTRIBUTING.ja.md](CONTRIBUTING.ja.md)を参照してください。

security vulnerabilityの疑いがある場合、public Issueではなく[SECURITY.ja.md](SECURITY.ja.md)に従ってprivate vulnerability reportingを使用してください。

## License

Apache License 2.0です。`LICENSE`を参照してください。
