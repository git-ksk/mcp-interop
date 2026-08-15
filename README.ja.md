# mcp-interop

[![CI](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml/badge.svg)](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/git-ksk/mcp-interop)](https://github.com/git-ksk/mcp-interop/releases/latest)
[![License](https://img.shields.io/github/license/git-ksk/mcp-interop)](LICENSE)

[English](README.md) | [日本語](README.ja.md)

**Remote MCP serverが、実際のMCP clientで本当に動くかをlive testするinterop runner。**

`mcp-interop`は、Remote Model Context Protocol (MCP) serverを実クライアントでblack-box検証するcross-client test runnerです。

このツールが答えたいのは、spec上の適合性だけでは分からない次の問いです。

> このRemote MCP deploymentは、ユーザーが実際に使っているclientから利用可能なprotocol pathへ到達し、必要な認証を満たし、toolsを発見できるか？

また、安全にheadless automationできる実client surfaceがまだ無い対象向けに、client profileベースの**preflight診断**も提供します。preflightの結果をlive interoperability PASSとして扱うことはありません。

## Status

**現在の公開releaseはv0.5.1です。**

Release: [v0.5.1](https://github.com/git-ksk/mcp-interop/releases/tag/v0.5.1)

v0.5.1で提供するlive adapterは次の通りです。

- **Codex CLI** — live inventory + 明示的opt-in OAuth
- **Cursor CLI (beta)** — dedicated MCP management commandを使うno-auth live inventory + 実Cursor MCP login pathを使う明示的opt-in OAuth。controlled fixtureでauthenticated `mcp list-tools`まで検証済み
- **Antigravity CLI (beta, macOS)** — isolated no-prompt PTY/tool cacheを使うno-auth live inventory + isolated PTY内の実`/mcp` managerを使う明示的opt-in OAuth。authenticationとclient-side tool-cache観測を分離し、generic `init/tools`は必要に応じてconservativeに`unknown`を維持する

v0.5.1はfocused patch releaseです。実Codex/CursorがMCP OAuth registration boundaryへの到達を自ら証明した`DCR_UNSUPPORTED` / `DCR_FAILED`では`reach=pass`を記録し、generic OAuth failureは引き続きconservativeに`unknown`を維持します。CI vulnerability scanとrelease buildもpatched Go 1.26.6へ更新しました。v0.5.0ではportable live-result artifact schema v1、`test --output`、artifact compare、`--fail-on-regression` CI gateを追加し、既存の`test --json` contractとreal-client-only PASS boundaryを維持しました。

v0.5.1以降も新client追加を急がず、品質・最適化を優先します。現在強化している保証は次の通りです。

- live PASSには引き続き4つのreal-client stageすべての`pass`が必要
- diagnostic metadataとRuntime Evidenceはreal-client PASS evidenceと分離
- secret-bearing valueは出力前にrejectまたはredact
- process cleanupはboundedにし、current test sessionが所有するtemporary state/processだけを対象にする
- exact client-version runをsecret-safeなlocal artifactへexportし、既存live verdictを弱めずに比較できる
- CI/release gateでは可能な範囲でformat、vet、unit、race、vulnerability scan、fixture、shell syntax、release archive smokeを検証。cross-platform互換性testはmoduleのGo 1.24 baselineを維持し、security scanとrelease artifact buildはpatched Go 1.26.6へ固定

VS Codeは、別途進めているlifecycle/tool-discovery automation researchがstable live adapterへ昇格するまでresearch-onlyです。

GitHub Copilot CLIはresearch-onlyです。現在の検証では実clientのMCP initializationまでは証明できましたが、projectのno-model evidence contractで`tools/list`までは未証明です ([#48](https://github.com/git-ksk/mcp-interop/issues/48))。Claude Code対応は現時点では優先していません。

ChatGPT real-client対応は、officially supportedなdirect/headless ChatGPT MCP app-management surfaceが利用可能になるまで意図的にBLOCKEDです ([#20](https://github.com/git-ksk/mcp-interop/issues/20))。model prompt、brittleなDOM/UI automation、private endpoint、通常ユーザーのbrowser credentialはreal-client PASSの根拠にしません。

## Install

Go 1.24以降で、現在のstable releaseを固定して入れる場合:

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@v0.5.1
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

[v0.5.1 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.5.1)には、macOS / Linux / Windows向けのamd64 / arm64 archiveと`checksums.txt`があります。

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

JSON outputはarrayです。portable artifact用fieldをこの既存contractへ追加しません。

### Portable regression artifact

同じlive runを、stdoutを変えずにversioned / secret-safeなlocal artifactへexportできます。

```console
mcp-interop test https://example.com/mcp --client codex --output result.json
```

artifactには正確に検出したclient version、OS/architecture、runner/runtime context、invocation auth mode、evidence provenance、既存4 stageのstatus/reasonを保存します。raw endpoint URLはpersistせず、endpoint fingerprintを作る前にquery valueを除外します。human-readableなstage messageやdiagnostic payloadもartifact v1には保存しません。

client version違い・run違いを比較できます。

```console
mcp-interop compare old.json new.json
mcp-interop compare old.json new.json --json
mcp-interop compare old.json new.json --fail-on-regression
```

comparisonは`PASS_TO_FAIL`、`PASS_TO_UNKNOWN`、`PASS_TO_SKIP`、reason-code変更、baseline evidence消失を明示します。client versionが変わっただけならregressionではありません。`--fail-on-regression`はregression/evidence lossを検出したときだけexit `1`、malformed/unsupported artifactなどinput contract違反はexit `2`です。

正確なcompatibility、secret safety、pairing、exit-code contractは[Live interoperability result artifact schema v1](docs/live-result-schema-v1.ja.md) ([English](docs/live-result-schema-v1.md))を参照してください。

OAuth flowは常に明示的opt-inです。

```console
mcp-interop test https://example.com/mcp --client codex --oauth
mcp-interop test https://example.com/mcp --client cursor --oauth
mcp-interop test https://example.com/mcp --client antigravity --oauth
```

Codexはauthorization URLをstderrへ表示し、実Codex OAuth callbackを待ちます。URLには短時間有効なOAuth stateが含まれるため共有しないでください。

Cursorはisolated temporary HOME/workspace内で実Cursor MCP login pathを使い、authenticated `mcp list-tools`でtool discoveryを証明します。callback detailはversion-specificであり、固定portをhard-codeしません。

Antigravityはisolated PTY内で実`/mcp` managerへ入り、OAuth token persistenceをtemporary HOME内に閉じ込めます。authorization codeやtoken内容は`mcp-interop` evidenceへpersistしません。詳細は[Antigravity OAuth live-test boundary](docs/antigravity-oauth.ja.md) ([English](docs/antigravity-oauth.md))を参照してください。

### ChatGPT OAuth/server preflight

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

### Secret-free ChatGPT Runtime Evidence

Authorization Serverのsanitized logからtoken requestの**値ではなくpresence/matchだけ**を観測できる場合は、Runtime Evidenceも相関できます。

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --runtime-evidence runtime-evidence.json
```

`runtime-evidence.json`の最小v3形:

```json
{
  "schema_version": 3,
  "registration": {
    "strategy": "cimd",
    "client_metadata_url": "https://chatgpt.com/oauth/.../client.json"
  },
  "token_request": {
    "resource_matches": true,
    "client_assertion_present": false
  },
  "tool_metadata": {
    "oauth2_security_scheme_present": true
  },
  "tool_challenge": {
    "expected": false
  }
}
```

authorization/token/resource/toolの追加observationは任意です。v3では`tool_metadata`と`tool_challenge`を独立して扱い、v2 `tool_auth`とlegacy v1も互換目的で引き続き受け付けます。未観測なら推測せず`WARN / unknown`になります。未知fieldは拒否するため、token、authorization code、PKCE verifier、raw client assertion、cookieなどを入力しないでください。

Preflight、Runtime Evidence、real-client interoperabilityは別の証拠層です。serverが`PREFLIGHT PASS`でも、Runtime Evidenceが`TOKEN_AUTH_METHOD_MISMATCH`で`FAIL`になることがあります。

### Evidence utility

`diagnose`と同じstrict secret-free decoderを使う補助commandです。

```console
mcp-interop evidence validate runtime-evidence.json
mcp-interop evidence summary runtime-evidence.json
mcp-interop evidence merge authorization.json resource.json tool.json -o runtime-evidence.json
```

`summary`はsection名とsupplied field数だけを表示します。`merge`は競合observationを後勝ちにせず失敗させ、canonical schema v3 JSONを出力します。未知fieldの拒否も共通です。

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

### Current Codex limitations

- OAuthはinteractiveかつexplicitです。`--oauth`なしでserverが`notLoggedIn`を返す場合、testはincompleteのままnon-zero exitになります。
- 現在のCodex app-server versionでは、unreachable serverと正当なzero-tool serverが同じempty-inventory shapeになる場合があります。そのため`mcp-interop`は成功/失敗を推測せず、該当stageを`unknown`として報告します。
- adapterはinstalled Codex app-serverのMCP status/OAuth surfaceに依存します。必要methodを公開しない古い/将来versionでは、adapter更新までinconclusive/errorになる可能性があります。

## Cursor adapter (beta)

Cursor adapterは:

- isolated temporary `HOME` + workspaceを作る
- `<workspace>/.cursor/mcp.json`へtest endpointだけを書く
- 実Cursor CLIのMCP management surfaceを使う
- `mcp enable` → `mcp list` → `mcp list-tools`を使う
- `--oauth`指定時はisolated session内で実Cursor MCP login pathを起動する
- authenticated `mcp list-tools`成功をreal-client evidenceとして扱う
- temporary Cursor stateをcleanupする

Cursor model promptは送信しません。

### 現在の制限

- OAuthは明示的opt-inで、`--oauth`なしにloginを勝手に開始しない
- callback addressはversion-specificで固定portを永久仕様として扱わない
- MCP command outputは専用JSON contractではなくhuman-readable
- 実OAuth validationは検証済みCursor CLIのmacOS pathで完了済み。今後は追加version/OS evidenceを増やす

## Antigravity adapter (beta, macOS)

Antigravity adapterは:

- isolated temporary `HOME` + workspaceを作る
- temporary `~/.gemini/config/mcp_config.json`へ`serverUrl`形式でtargetを書く
- model promptなしで実`agy`をPTY起動する
- no-authではisolated `~/.gemini/antigravity-cli/mcp/<server>/` tool cacheを観測する
- `--oauth`指定時はisolated PTY内で実Antigravity `/mcp` managerへ入る
- OAuth token persistenceはisolated `~/.gemini/antigravity/mcp_oauth_tokens.json`のmetadataだけを観測し、token file内容は開かない
- test PTY wrapper由来と証明できるdescendant processだけを回収する
- temporary HOME/workspaceをcleanupする

### 現在の制限

- live adapterはmacOSのみ
- OAuthは明示的opt-inで、検証済みAntigravityのinteractive `/mcp` surfaceに依存する
- 検証済み`agy 1.1.11`のOAuth pathではauthenticated `initialize` / `tools/list`が完了しても、no-auth時と同じclient-side tool cacheが生成されない場合がある。その場合generic `init/tools`はauthenticationから推測せず`unknown`を維持する
- controlled localhost OAuth E2Eでは別途authenticated `initialize` / `notifications/initialized` / `tools/list`のserver-side evidenceを必須にする。詳細は[Antigravity OAuth live-test boundary](docs/antigravity-oauth.ja.md) ([English](docs/antigravity-oauth.md))
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

Cursor/AntigravityのOAuth専用harnessはcontrolled loopback fixtureに対して実OAuth pathも検証し、authorization code/tokenをpersisted evidenceへ含めません。

通常のGitHub-hosted CIは外部clientをインストールしません。実client E2E用にはself-hosted macOS ARM64向けmanual workflowがあります。

## ドキュメント

- [Architecture](docs/architecture.ja.md) ([English](docs/architecture.md))
- [Project direction](docs/project-direction.ja.md) ([English](docs/project-direction.md))
- [Roadmap to a stable interoperability contract](docs/roadmap.ja.md) ([English](docs/roadmap.md))
- [Conformance vs. interoperability](docs/conformance-vs-interop.ja.md) ([English](docs/conformance-vs-interop.md))
- [Live result artifact schema v1](docs/live-result-schema-v1.ja.md) ([English](docs/live-result-schema-v1.md))
- [Troubleshooting](docs/troubleshooting.ja.md) ([English](docs/troubleshooting.md))
- [Reason code](docs/reason-codes.ja.md) ([English](docs/reason-codes.md))
- [ChatGPT接続診断](docs/chatgpt-diagnostics.ja.md) ([English](docs/chatgpt-diagnostics.md))
- [Antigravity OAuth live-test boundary](docs/antigravity-oauth.ja.md) ([English](docs/antigravity-oauth.md))
- [GitHub Copilot CLI direct MCP inventory PoC](docs/copilot-cli-poc.ja.md) ([English](docs/copilot-cli-poc.md)) — research-only
- [VS Code Agent Plugin MCP PoC](docs/vscode-agent-plugin-poc.ja.md) ([English](docs/vscode-agent-plugin-poc.md)) — experimental research
- [Contributing](CONTRIBUTING.ja.md) ([English](CONTRIBUTING.md))
- [Support](SUPPORT.ja.md) ([English](SUPPORT.md))
- [Security Policy](SECURITY.ja.md) ([English](SECURITY.md))
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [CHANGELOG](CHANGELOG.md) — release historyのcanonical版は英語

## Release process

release archiveは`scripts/build-release.sh`でbuildします。`v*` tagをpushするとrelease workflowが起動し、tag/provenanceを検証したうえでsource quality gateを再実行し、version/commit/build-time metadataを埋め込み、6種類のplatform/architecture archiveと`checksums.txt`を生成します。Linux artifactのembedded versionとpackaged CLI regression smokeも確認してからGitHub Releasesへpublishします。

通常のPull Requestでも、tag作成前にUbuntu上で同じrelease build pathをsmoke testします。cross-platform jobはLinux / macOS / Windowsで通常のGo 1.24-compatible test/build pathを維持し、Ubuntuではrace detector実行後にpinned release/security Go toolchainへ切り替えて`govulncheck`を実行します。tagged release artifactもminimum module versionではなく、このpatched pinned toolchainでbuildします。

## Roadmap

詳細なmilestone、exit criteria、non-goal、`v1.0.0` graduation条件は[Stable interoperability contractに向けたRoadmap](docs/roadmap.ja.md) ([English](docs/roadmap.md))を参照してください。

現在の想定maturity sequenceは次です。version番号はdeadlineやautomatic graduationではありません。必要なら`v0.11.x`以降を継続し、`v1.0.0`はstable-contract exit criteriaを満たした時だけreleaseします。

- **v0.6.x** — protocol-aware core + deployment identity privacy
- **v0.7.x** — repeatable suite / regression workflow + CI trust boundary
- **v0.8.x** — baseline lifecycle + observed compatibility envelope
- **v0.9.x** — coverage / capability profile / safe client graduation
- **v0.10.x** — public contract candidate
- **v0.11.x+** — 必要なだけstabilization
- **v1.0.0** — exit criteria達成時のみstable contract化

roadmap上のfuture capabilityはship済みbehaviorではありません。current release behaviorはcode、release documentation、versioned schemaをsource of truthとして確認してください。

## 現在のnon-goals

- MCP security scanning
- tool quality / LLM-selection benchmark
- runtime sandboxing
- permission/capability governance
- 新しいOAuth/MCP conformance specificationの策定
- 対応clientを実際に動かさず互換性を保証すること

## Contributing / Security

Contributionは[CONTRIBUTING.ja.md](CONTRIBUTING.ja.md)を参照してください。usage/supportの窓口は[SUPPORT.ja.md](SUPPORT.ja.md)、project参加時の基本ルールは[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)を参照してください。

security vulnerabilityの疑いがある場合、public Issueではなく[SECURITY.ja.md](SECURITY.ja.md)に従ってprivate vulnerability reportingを使用してください。

## License

Apache License 2.0です。`LICENSE`を参照してください。
