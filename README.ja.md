# mcp-interop

[![CI](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml/badge.svg)](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/git-ksk/mcp-interop)](https://github.com/git-ksk/mcp-interop/releases/latest)
[![License](https://img.shields.io/github/license/git-ksk/mcp-interop)](LICENSE)

[English](README.md) | **日本語**

> この文書は英語版READMEの日本語訳です。内容に差がある場合は英語版を正とします。

**Remote MCPサーバーが、実際のMCPクライアントから本当に使えるかを検証する相互運用性テストランナーです。**

`mcp-interop` は、公開済みのRemote MCPエンドポイントを、実際にインストールされたMCPクライアントでブラックボックス検証します。

確認したいのは、単に「MCP仕様に適合しているか」ではありません。

> このRemote MCPは、ユーザーが実際に使っているクライアントから到達でき、必要な認証を通過し、MCPセッションを成立させ、ツール一覧まで取得できるか？

MCP仕様への適合性は公式のMCP Conformance Test Frameworkが担当します。`mcp-interop` は、その上で**特定のデプロイ × 特定のクライアント製品 × 特定のバージョン**という実運用の組み合わせを検証します。

安全に自動操作できるクライアント向けインターフェースがまだ無い場合は、製品ごとの事前診断（preflight）も提供します。ただし、事前診断の成功を実クライアントでの相互運用PASSとして扱うことはありません。

## 現在の状態

現在の公開リリースは **v0.8.0** です。

Release: [v0.8.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.8.0)

v0.8.0で利用できる実クライアント向けアダプターは次のとおりです。

- **Codex CLI** — 実クライアントのMCP一覧確認と、明示的に指定した場合だけ実行するOAuth認証
- **Cursor CLI（beta）** — MCP管理コマンドを使った認証不要の実ツール確認と、実CursorのMCPログイン経路を使うOAuth認証
- **Antigravity CLI（beta / macOS）** — 隔離したPTYとツールキャッシュを使う認証不要の確認と、実`/mcp`マネージャーを使うOAuth認証

v0.7.0では、単発のlive testだけでなく、同じRemote MCPを複数クライアントで繰り返し検証し、前回結果との退行まで確認できるようになりました。v0.6.0のprotocol-aware coreとprotected-path artifactを土台に、秘密情報をmanifestへ書かないsuite宣言、複数clientの一括実行、baselineとの比較、manual / main-onlyのself-hosted CI境界を追加しています。retry後にPASSしても最初の失敗は消えません。live PASSの意味自体は変えていません。

v0.8.0では、v0.7の繰り返しregression workflowにimmutable baselineのaccept/supersedeとexact observed-point compatibility分類を追加しました。auto-updateされた未観測versionは観測されるまで`untested`のまま、stale evidenceは明示され、version変更だけでregressionにはなりません。公開版の保証は次のとおりです。

- 実クライアントの4段階すべてを確認できた場合だけlive PASSにする
- 診断用メタデータとRuntime Evidenceを、実クライアントのPASS証拠から分離する
- 秘密情報を含む値は、出力前に拒否またはマスクする
- 終了処理では、今回のテストが所有している一時状態・プロセスだけを対象にする
- クライアントの正確なバージョンを含む結果をローカルファイルへ保存し、バージョン間の退行を比較できるようにする
- CI / releaseではformat、vet、unit、race、脆弱性検査、fixture、release archiveを可能な範囲で検証する

VS CodeとGitHub Copilot CLIは調査段階です。ChatGPTは、公式にサポートされた自動操作可能なMCPアプリ管理インターフェースが利用できるまで、実クライアントアダプターを意図的にBLOCKEDとしています。

## インストール

Go 1.24以降が必要です。

現在の安定版を固定してインストールする場合:

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@v0.8.0
```

最新公開版を使う場合:

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@latest
```

バージョン確認:

```console
mcp-interop version
# または
mcp-interop --version
```

[v0.8.0 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.8.0)には、macOS / Linux / Windows向けのamd64 / arm64アーカイブと`checksums.txt`があります。

## 何を検証するのか

1つのクライアントに対するlive testは、次の4段階で構成されます。

1. `reach` — 実クライアントが対象Remote MCPへ到達し、実通信が発生したことを確認できた
2. `auth` — 必要な認証が完了した、またはツール発見によって認証不要と確認できた
3. `init` — 実クライアントが、その後のMCP通信を続けられる準備状態まで到達した。旧protocolの`initialize` handshakeそのものを観測した、という意味に限定しない
4. `tools` — クライアントがサーバーのツールを発見した

**4段階すべてが`pass`の場合だけexit code `0`**です。

`fail`だけでなく、`skip`や`unknown`もnon-zeroになります。これは「証拠が足りないのにCIだけ成功する」状態を防ぐためです。

`diagnose`は別の診断機能です。公開メタデータから`PREFLIGHT PASS` / `PREFLIGHT FAIL`を返しますが、実クライアントの`reach/auth/init/tools`を代替しません。

また、`mcp-interop`のPASSは次を保証しません。

- MCPサーバー自体が安全であること
- 各ツール実装が正しいこと
- 破壊的操作が安全であること
- AIモデルが適切なツールを選ぶこと
- テストしていない別クライアント・別バージョンでも動くこと

## 基本的な使い方

どのコマンドを使えばよいか迷った場合は、まず次の使い分けで考えます。

| やりたいこと | コマンド |
| --- | --- |
| 対応クライアントがインストールされているか確認 | `mcp-interop clients` |
| 今すぐ1つまたは複数の実クライアントをテスト | `mcp-interop test` |
| 1回のlive testを保存して後で比較 | `mcp-interop test --output` |
| suiteファイルを実行せず検証 | `mcp-interop suite validate` |
| 同じ宣言を複数クライアントで繰り返し実行 | `mcp-interop suite run` |
| raw result setと1回以上のattemptを比較 | `mcp-interop suite compare` |
| immutable baselineをaccept | `mcp-interop baseline create` |
| accept済みbaselineとretained attemptを比較 | `mcp-interop baseline compare` |
| インストール済みexact client versionを実測evidenceで分類 | `mcp-interop compatibility query` |
| 実クライアントを動かさない事前診断 | `mcp-interop diagnose` |

検出できるクライアントを確認:

```console
mcp-interop clients
mcp-interop clients --json
```

1クライアントをテスト:

```console
mcp-interop test https://example.com/mcp --client codex
mcp-interop test https://example.com/mcp --client cursor
mcp-interop test https://example.com/mcp --client antigravity
```

複数クライアントを順番にテスト:

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity
```

複数クライアント時のテキスト出力は、たとえば次のようになります。

```text
SUMMARY
CLIENT           REACH  AUTH  INIT  TOOLS  VERSION
Codex CLI        PASS   PASS  PASS  PASS   codex-cli 0.133.0
Cursor CLI       PASS   PASS  PASS  PASS   2026.08.04-aaa8809
Antigravity CLI  PASS   PASS  PASS  PASS   1.1.11
```

`--json`を指定した場合は配列を返します。既存JSON契約へartifact用のフィールドを勝手に追加しません。

## 実行結果を保存・比較する

同じlive runを、既存の結果shapeを変えずに、バージョン付き・秘密情報を含まないローカルartifactへ保存できます。

```console
mcp-interop test https://example.com/mcp --client codex --output result.json
```

このdefaultは従来どおりartifact schema v1です。実際に検出したクライアントバージョン、`mcp-interop` runner/processのOS / architecture、runner/runtime context、認証モード、証拠の出所、4段階の結果とreason codeを保存します。`platform.arch`はreal client binary architectureの証明ではありません。新しいportable evidenceを出力する前に、取得可能なMach-O / ELF / PE executable metadataだけを非侵襲に確認し、runner/client architectureの既知mismatchはfail-closedにします。script/wrapper launcherはhost architectureから推測せず、明示的にunknownのまま扱います。raw endpoint URLは保存せず、query値を除外した識別情報を使います。

endpoint path自体にcredentialが含まれる場合はv1をexportせず、schema v2 protected-path identityを使います。

```console
mcp-interop test 'https://example.com/mcp/<protected-path>' \
  --client codex \
  --output result.json \
  --deployment-id production-a
```

deployment IDはそのまま保存されるため、operatorが決めた安定した非secret labelでなければなりません。protected pathから派生させてはいけません。このmodeではcanonical originとdeployment IDだけをartifactへ保存し、path/query/userinfo/fragmentは保存もhashもしません。通常のtext/JSON出力でもprotected pathを再表示しません。v1↔v2比較はidentity mappingを推測せず明示的にrejectします。

結果を比較:

```console
mcp-interop compare old.json new.json
mcp-interop compare old.json new.json --json
mcp-interop compare old.json new.json --fail-on-regression
```

比較では、たとえば次を区別します。

- `PASS_TO_FAIL`
- `PASS_TO_UNKNOWN`
- `PASS_TO_SKIP`
- reason codeの変化
- baselineにあった証拠の消失

クライアントのバージョンが変わっただけでは退行扱いにしません。

詳しい仕様は[Live interoperability result artifact schema v1](docs/live-result-schema-v1.ja.md)と[artifact schema v2 protected-path identity](docs/live-result-schema-v2.ja.md)を参照してください。

## 複数クライアントを繰り返し検証する（suite）

複数クライアントを同じ条件で繰り返しテストしたい場合は、対象とクライアントを`suite.json`へ宣言します。まず`validate`で内容だけを確認でき、この段階ではクライアントを起動せず、endpointの値も読み込みません。

```console
mcp-interop suite validate suite.json
mcp-interop suite validate suite.json --json
```

`trusted_real_client`形式なら、宣言した対象とクライアントの組み合わせをまとめて実行し、結果をschema v2のartifact setとして保存できます。

```console
export MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A='https://example.com/mcp/<protected-path>'
mcp-interop suite run suite.json --output-dir suite-results
mcp-interop suite run suite.json --output-dir suite-results-json --json
```

suiteは最初のclientを起動する前に全endpointを解決・検証し、各runを`mcp-interop test`と同じlive-test経路で実行します。出力は`index.json`とrunごとのprotected-path schema v2 artifactです。indexへendpoint URLやendpoint環境変数名は保存しません。non-PASSや未インストールclientもsetから落とさず、commandはexit `1`になります。manifest不正、endpoint未解決、既存output directoryはclient起動前にexit `2`です。

Manifest v1にはRemote MCP endpoint URL自体を保存しません。hosted fixture宣言は任意network targetやOAuthを指定できず、v0.8.0でもvalidation-onlyです。repositoryのPR CIは任意suite manifestを実行せず、controlled localhost fixture gateを別経路で使います。trusted real-client suiteはtarget固有の`MCP_INTEROP_SUITE_ENDPOINT_*`変数参照と非secret `deployment_id`を使います。詳細は[Suite manifest v1](docs/suite-manifest-v1.ja.md)と[Suite result set v1](docs/suite-result-set-v1.ja.md)を参照してください。

比較基準として保存した結果（baseline）と、その後の1回以上の実行結果（attempt）を比較できます。

```console
mcp-interop suite compare baseline-results attempt-1 attempt-2
mcp-interop suite compare baseline-results attempt-1 attempt-2 --json
mcp-interop suite compare baseline-results attempt-1 attempt-2 --fail-on-regression
```

reportには全attemptを残します。最初のattemptがFAIL/UNKNOWNでretry後にPASSしてもclean PASSへ上書きせず、`regression_and_unstable`になります。gate指定時はregressionまたはunstable evidenceでexit `1`、cleanで`0`、invalid/unreadable inputで`2`です。詳細は[Suite regression report v1](docs/suite-regression-report-v1.ja.md)を参照してください。

全declared runにcompleteなreal-client evidenceとexact client versionがあるresult setは、immutable baselineとしてacceptできます。

```console
mcp-interop baseline create suite-results --output-dir baselines/current
mcp-interop baseline create suite-results-new --output-dir baselines/next --supersedes baselines/current
mcp-interop baseline compare baselines/current attempt-1 attempt-2 --fail-on-regression
```

compatibilityはexact client-version/platformの実測pointだけで表します。未観測versionは`untested`のままで、version変更だけをregressionにはしません。`tested` / `untested` / `stale` / `known_broken` / `regressed` / `unknown`の意味は[Compatibility envelope v1](docs/compatibility-envelope-v1.ja.md)を参照してください。

保持したresult set / baseline全体からexact version/OS matrixを列挙する場合は`mcp-interop compatibility matrix`を使います。retryとevidence gapも保持されます。現在repositoryに保持している実測範囲は[Exact observed client coverage](docs/observed-coverage.ja.md)を参照してください。
既存`clients --json` schemaを変更せず、現在インストール済みのexact client versionを分類できます。

```console
mcp-interop compatibility query --client codex --target production-a --deployment-id production-a --baseline baselines/current --observation attempt-1 --json
```

version-change staleでは、繰り返した`--observation`をwall-clockで並べ替えず、oldest -> newestの明示collection orderとして扱います。age-based staleには`--max-age-seconds N --trust-executed-at-clock`が必須で、未来timestampは保守的にstale扱いします。

baseline作成は既存directoryを上書きしません。result set自体をbundleへコピーしdeterministic digestで固定し、baselineはpathで明示選択します。そのためretryやclient auto-updateでaccept済み基準が勝手に置換されません。supersede時も旧baselineを変更せずfingerprintを記録します。baseline比較ではmanifest、execution context、deployment fingerprint、platform mismatchをfail-closedにしつつ、出力はsuite regression report v1を維持します。詳細は[Suite baseline v1](docs/suite-baseline-v1.ja.md)を参照してください。

self-hosted real-client GitHub Actionsは別のprivileged pathです。manual macOS workflowはmain-only、main-only Environment policy、exact run provenanceを使い、remote suite endpointやOAuth credentialを受け取りません。詳細は[Self-hosted real-client CI security boundary](docs/self-hosted-ci-security.ja.md)を参照してください。

## OAuth認証

OAuthは**必ず明示的に指定した場合だけ**開始します。

```console
mcp-interop test https://example.com/mcp --client codex --oauth
mcp-interop test https://example.com/mcp --client cursor --oauth
mcp-interop test https://example.com/mcp --client antigravity --oauth
```

### Codex

Codex自身のOAuth経路を使います。authorization URLはstderrへ表示されます。URLには短時間有効なOAuth `state`が含まれるため、共有しないでください。

### Cursor

一時的な`HOME`とworkspaceの中で、実Cursor MCPログイン経路を使います。認証後の`mcp list-tools`成功を、実クライアントがツールを発見した証拠として扱います。

callback addressはクライアントバージョン依存として扱い、固定portを仕様として決め打ちしません。

### Antigravity

隔離したPTY内で実`/mcp`マネージャーを操作します。OAuth tokenは一時`HOME`内に閉じ込めます。

`mcp-interop`はtokenファイルの内容を読みません。ファイルが存在するかなどのメタデータだけを確認します。

詳細は[Antigravity OAuth live-test boundary](docs/antigravity-oauth.ja.md)を参照してください。

## ChatGPT向けの接続診断

Remote MCPが公開しているOAuthメタデータに、ChatGPTとの既知の不整合がないか確認できます。

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

主に次を確認します。

- HTTPS endpoint
- Protected Resource Metadata
- Authorization Server Metadata
- CIMD / DCR
- token endpoint authentication method
- PKCE `S256`
- `offline_access`
- protected resourceの`resource`整合性

ChatGPTがCIMDを利用できるため、`registration_endpoint`が無いだけではFAILにしません。

実ChatGPTの認証要求から、秘密情報ではない`client_id`と`redirect_uri`を取得できる場合は、さらに厳密に照合できます。

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

この診断はChatGPT UIを操作せず、OAuthを完了させず、実ChatGPTクライアントのPASSを主張しません。

詳細は[ChatGPT接続診断](docs/chatgpt-diagnostics.ja.md)を参照してください。

## Runtime Evidence

Authorization ServerやResource Serverで観測した情報を、**値そのものではなく「存在したか」「一致したか」だけ**に変換して診断へ渡せます。

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --runtime-evidence runtime-evidence.json
```

たとえば次のような情報を扱います。

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
  "resource_request": {
    "bearer_present": true,
    "signature_valid": true,
    "audience_matches": true
  }
}
```

未観測の項目は推測せず`WARN / unknown`です。未知フィールドは拒否します。

access token、refresh token、authorization code、PKCE verifier、raw client assertion、cookieなどの秘密情報は入力しないでください。

補助コマンド:

```console
mcp-interop evidence validate runtime-evidence.json
mcp-interop evidence summary runtime-evidence.json
mcp-interop evidence merge authorization.json resource.json tool.json -o runtime-evidence.json
```

`summary`はセクション名と入力されたフィールド数だけを表示します。`merge`は競合した観測値を勝手に上書きせず、エラーにします。

Preflight、Runtime Evidence、実クライアント相互運用テストは**別々の証拠層**です。`PREFLIGHT PASS`でもRuntime EvidenceがFAILになることがありますし、両方PASSでも実ChatGPTのlive PASSにはなりません。

## アダプターの仕組み

### Codex CLI

Codexアダプターは次の流れで動きます。

1. 一時`CODEX_HOME`を作る
2. OAuth credential storageを一時HOME内のファイルへ限定する
3. 対象Remote MCPだけを一時設定へ登録する
4. 実`codex app-server`を起動する
5. app-serverのcontrol connectionを初期化する
6. `mcpServerStatus/list`でMCP状態・ツール一覧を確認する
7. `--oauth`指定時だけCodex自身のOAuthを実行する
8. 実Codexが観測した結果だけを報告する
9. 一時セッションを削除する

モデルへのプロンプトは送りません。

### Cursor CLI（beta）

一時`HOME`とworkspaceを作り、実Cursor CLIの`mcp enable`、`mcp list`、`mcp list-tools`を使います。OAuth時も通常ユーザーの設定や認証情報を使わず、一時環境に閉じ込めます。

### Antigravity CLI（beta / macOS）

一時`HOME`とworkspaceを作り、実`agy`をPTYで起動します。起動前に一時settingsへ`modelProvider: "gemini"`を書き、ambientなGemini credential / endpoint overrideを除去し、固定の非秘密`GEMINI_API_KEY` sentinelを注入します。これによりAntigravityのdocumented no-account modeを使い、通常ユーザーのKeychain account sessionへ依存しません。model promptは送りません。

認証不要の経路ではクライアント自身が生成したツールキャッシュを観測し、OAuthでは実`/mcp`マネージャーを利用します。Remote MCP OAuth tokenは一時HOMEへ閉じ込め、通常ユーザーaccount認証とは分離します。

Keychainのbefore/after比較は非変更gateであり、それ単独では非利用の証明にはしません。通常ユーザーcredential非再利用はdocumented no-account modeと実クライアントE2Eで担保し、core pathは`agy 1.1.22`で再検証しています。

証拠が足りない場合は、認証成功だけを根拠に`init/tools=pass`へ昇格せず`unknown`を維持します。

## 安全性と隔離

- **実クライアントを使う。** クライアントを模倣しただけの結果をinterop成功にしない
- **モデル評価と混ぜない。** 相互運用性の証拠をモデルのツール選択に依存させない
- **通常のユーザー設定を変更しない。** 安全に隔離できなければ`skip` / `unknown`
- **一時状態を保護する。** POSIX環境ではowner-only権限を使う
- **秘密情報を出力しない。** Bearer/OAuth情報やcredential-like URL parameterを拒否・マスクする
- **OAuthは明示的に開始する。** 対応済みアダプターで`--oauth`を指定した場合だけ実行する
- **Preflightをlive PASSにしない。** メタデータ互換性は実クライアントの証拠ではない
- **hosted backendを必須にしない。** コア機能はローカル・CIだけで利用できる

## macOSでの実クライアントE2E

リポジトリには、localhostだけで動くMCP fixtureとrelease-gate用ランナーがあります。

```console
bash scripts/e2e-real-clients.sh
```

デフォルトではCodex / Cursor / Antigravityを検証します。

```console
MCP_INTEROP_CLIENTS=codex,cursor bash scripts/e2e-real-clients.sh
```

ハーネスは主に次を確認します。

- current checkoutをbuild/testする
- `127.0.0.1`だけにbindするfixtureを起動する
- 各クライアントを別のfixture pathで動かす
- `initialize` / `notifications/initialized` / `tools/list`を確認する
- `tools/call`が発生したらFAIL
- 一般的なモデル/APIキー環境変数を子プロセスから除外する
- ユーザー設定・認証情報のメタデータを実行前後で比較する
- 新しく残ったクライアントプロセスや一時ディレクトリを検出する

通常のGitHub-hosted CIには外部MCPクライアントをインストールしません。実クライアントE2Eはself-hosted macOS ARM64向けmanual workflowとして分離しています。

## ドキュメント

- [Architecture / アーキテクチャ](docs/architecture.ja.md)
- [Project direction / プロジェクト方針](docs/project-direction.ja.md)
- [Roadmap / ロードマップ](docs/roadmap.ja.md)
- [MCP Conformanceとの違い](docs/conformance-vs-interop.ja.md)
- [Live result artifact schema v1](docs/live-result-schema-v1.ja.md)
- [Live result artifact schema v2](docs/live-result-schema-v2.ja.md)
- [Suite manifest v1](docs/suite-manifest-v1.ja.md)
- [Suite result set v1](docs/suite-result-set-v1.ja.md)
- [現行real-clientのprotocol-era観測](docs/protocol-era-observations.ja.md)
- [トラブルシューティング](docs/troubleshooting.ja.md)
- [Reason code](docs/reason-codes.ja.md)
- [ChatGPT接続診断](docs/chatgpt-diagnostics.ja.md)
- [Antigravity OAuth](docs/antigravity-oauth.ja.md)
- [GitHub Copilot CLI PoC](docs/copilot-cli-poc.ja.md) — 調査用
- [VS Code Agent Plugin MCP PoC](docs/vscode-agent-plugin-poc.ja.md) — 実験的調査
- [コントリビューションガイド](CONTRIBUTING.ja.md)
- [サポート](SUPPORT.ja.md)
- [セキュリティポリシー](SECURITY.ja.md)
- [行動規範](CODE_OF_CONDUCT.ja.md)
- [CHANGELOG](CHANGELOG.md) — リリース履歴の正本は英語

## リリースとロードマップ

リリース用アーカイブは`scripts/build-release.sh`で生成します。`v*`タグをpushするとrelease workflowが起動し、タグと`main`の関係、source quality/security gate、埋め込みバージョン、アーカイブ、checksums、artifact attestationを確認してからGitHub Releasesへ公開します。

ロードマップの詳細は[Stable interoperability contractに向けたロードマップ](docs/roadmap.ja.md)を参照してください。

現在の想定順序:

- **v0.6.x** — protocol-aware core + deployment identity privacy
- **v0.7.x** — repeatable suite / regression workflow + CI trust boundary
- **v0.8.x** — baseline lifecycle + observed compatibility envelope
- **v0.9.x** — coverage / capability profile / safe client graduation
- **v0.10.x** — public contract candidate
- **v0.11.x+** — 必要なだけstabilization
- **v1.0.0** — exit criteriaを満たした場合だけstable contract化

ロードマップに書かれた将来機能は、現在利用できる機能を意味しません。現在の挙動はコード、リリース文書、バージョン付きschemaを正とします。

## 現在の非目標

- MCPセキュリティスキャナー
- ツール品質やLLMのツール選択ベンチマーク
- ランタイムサンドボックス
- 権限・capability governance
- 新しいOAuth/MCP適合仕様の策定
- 実際に動かしていないクライアントの互換性保証

## コントリビューションとセキュリティ

開発への参加は[CONTRIBUTING.ja.md](CONTRIBUTING.ja.md)を参照してください。利用方法や不具合報告は[SUPPORT.ja.md](SUPPORT.ja.md)、プロジェクト参加時の基本ルールは[CODE_OF_CONDUCT.ja.md](CODE_OF_CONDUCT.ja.md)にまとめています。

セキュリティ上の問題は公開Issueへ書かず、[SECURITY.ja.md](SECURITY.ja.md)に従ってPrivate Vulnerability Reportingを利用してください。

## ライセンス

Apache License 2.0です。`LICENSE`を参照してください。
