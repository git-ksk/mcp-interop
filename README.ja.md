# mcp-interop

[![CI](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml/badge.svg)](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/git-ksk/mcp-interop)](https://github.com/git-ksk/mcp-interop/releases/latest)
[![License](https://img.shields.io/github/license/git-ksk/mcp-interop)](LICENSE)

[English](README.md) | [日本語](README.ja.md)

**あなたのRemote MCPサーバーは、ユーザーの使うクライアントで動く？**

同じエンドポイントを、実際のCodex・Cursor・Antigravity CLIでテスト。どの段階で接続が止まるかを確認し、結果を保存して、サーバーのデプロイやクライアント更新後の不具合を見つけられます。ホスト型サービスへの登録は不要で、ローカルで動きます。

[まず試す](#まず試す) · [対応クライアント](#対応クライアント) · [利用ガイド](docs/usage.ja.md) · [開発に参加する](CONTRIBUTING.ja.md)

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity
```

出力形式のイメージです。バージョン欄は仮置きで、対応範囲を示すものではありません。

```text
SUMMARY
CLIENT           REACH  AUTH  INIT  TOOLS  VERSION
Codex CLI        PASS   PASS  PASS  PASS   <exact version>
Cursor CLI       PASS   PASS  PASS  PASS   <exact version>
Antigravity CLI  PASS   PASS  PASS  PASS   <exact version>
```

## こんなときに

- **Remote MCPサーバーを公開する前に：** 実クライアントが接続し、必要な認証を済ませ、ツールを発見できるか確認する。
- **クライアントが更新されたら：** 保存した結果と比較して、挙動の変化を見つける。バージョン番号が変わっただけでは不具合扱いにしません。
- **「つながらない」と報告されたら：** 到達・認証・プロトコル準備・ツール発見のどこで止まるかを絞り込む。
- **リリース前の確認を定型化したいときに：** 同じテストを複数クライアントで繰り返し、失敗した試行も含めて記録する。

プロトコルへの適合と、実クライアントでの動作は、それぞれ確認する対象が異なります。既存の適合性テストに加えて、ユーザー側の接続結果を確認できます。[適合性テストとの使い分け →](docs/conformance-vs-interop.ja.md)

## まず試す

最初は、3アダプターのstable範囲として実測済みの **Apple Silicon搭載Mac** で試してください。**Go 1.24以降**と、対応クライアントの実行ファイルが少なくとも1つ必要です（`codex`、`cursor-agent`、`agy`のいずれかを`PATH`へ追加）。

### 1. インストール

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@latest
```

バイナリで使う場合は、[GitHub Releases](https://github.com/git-ksk/mcp-interop/releases/latest)からOS・CPUに合うアーカイブをダウンロードし、`checksums.txt`と照合して、実行ファイルを`PATH`へ配置してください。Goが必要なのはソースからインストールする場合です。

Goでインストールしたコマンドが見つからない場合は、`$(go env GOPATH)/bin`（`GOBIN`を設定している場合はその場所）を`PATH`へ追加してください。

**公開版：[v0.10.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.10.0)。** `main`にはv1.0.0の準備とその後の修正が含まれていますが、v1.0.0は未公開です。`@latest`で入るのは公開済みモジュールであり、現在のチェックアウトではありません。変更履歴は[CHANGELOG](CHANGELOG.md)へ。

### 2. クライアントを確認

```console
mcp-interop version
mcp-interop clients
```

### 3. エンドポイントをテスト

URLを自分のRemote MCPエンドポイントへ置き換え、インストール済みのクライアントを指定します。

```console
mcp-interop test https://example.com/mcp --client codex
```

**4段階すべてがPASSなら成功**し、終了コードは`0`です。

| 段階 | 実クライアントから確認すること |
| --- | --- |
| `reach` | エンドポイントと実際に通信できた |
| `auth` | 必要な認証が完了した、または認証なしでツール発見できた |
| `init` | MCPのやり取りを続けられるプロトコル状態になった |
| `tools` | サーバーのツールを発見できた |

`FAIL`・`SKIP`・`UNKNOWN`があれば終了コードは非ゼロです。`UNKNOWN`は、結果を確定する証拠が足りない状態を示し、原因を調べる手がかりになります。[トラブルシューティング →](docs/troubleshooting.ja.md)

OAuthが必要なサーバーでは、対話的な認証を明示的に開始します。

```console
mcp-interop test https://example.com/mcp --client cursor --oauth
```

OAuthは指定時だけ開始し、非OAuth経路のstable判定には含まれません。[OAuthの詳細 →](docs/usage.ja.md#oauth認証)

## 対応クライアント

現在の`main`では、3アダプターを共通の基準で判定しています。

| クライアント | 実クライアントの観測方法 | stableの範囲 |
| --- | --- | --- |
| Codex CLI | `codex app-server`のMCP状態・ツール一覧 | macOS arm64、非OAuthの基本経路 |
| Cursor CLI | `mcp list-tools`などのMCP管理コマンド | macOS arm64、非OAuthの基本経路 |
| Antigravity CLI | 隔離したPTYとクライアント生成のツールキャッシュ | macOS arm64、非OAuthの基本経路 |

stableは、確認済みのアダプターの範囲を示します。すべてのバージョン・接続先を保証するものではありません。[実測バージョン一覧](docs/observed-coverage.ja.md)と[判定基準](docs/adapter-maturity.ja.md)を参照してください。あるOS用のバイナリが存在することと、そのOSで実クライアントを検証済みであることも別です。

VS Code・GitHub Copilot CLI・ChatGPT・Claude web/Desktopは調査段階です。ChatGPT向けにはメタデータを使う`diagnose --profile chatgpt`がありますが、その診断成功は実ChatGPTでの接続成功を意味しません。[対応に向けた調査状況 →](docs/adapter-graduation-gate.ja.md)

## 一度の確認から、継続的な比較へ

変更前の結果を保存し、変更後にもう一度実行して比較します。

```console
mcp-interop test https://example.com/mcp --client codex --output before.json
# サーバーを変更するか、クライアントを更新してから再実行。
mcp-interop test https://example.com/mcp --client codex --output after.json
mcp-interop compare before.json after.json --fail-on-regression
```

PASSから`FAIL`・`UNKNOWN`・`SKIP`への変化など、成功の証拠が失われたことを検出できます。URLのパスに認証情報が含まれる場合は、[`--deployment-id`による保護された結果保存](docs/usage.ja.md#saved-results)を使ってください。

複数の接続先やクライアントを継続的に確認するなら、[suiteと固定した比較基準](docs/usage.ja.md#suites)を利用できます。再試行で成功しても、その前の失敗は記録に残ります。

## 結果を信頼するために

- **実クライアントから確認する。** 基本テストではモデルにプロンプトを送らず、サーバーのツールも呼び出さずに、インストール済みクライアントを観測します。
- **テスト環境を隔離する。** 設定・認証状態は一時領域で扱い、テストが所有するプロセスとファイルを時間制限付きで片付けます。
- **不明な結果をそのまま伝える。** メタデータ、サーバー側の観測、部分的な証拠から接続成功を推測しません。
- **比較できる結果を残す。** バージョン付きの形式で保存し、秘密情報を含む値は[セキュリティ仕様](docs/security-contract-v1.ja.md)に従って拒否・マスクします。

確認するのは接続とツール発見です。ツールの処理内容、モデルのツール選択、サーバー全体の安全性は、それぞれ別のテストで確認してください。

## 次に読む

| やりたいこと | ドキュメント |
| --- | --- |
| コマンド・OAuth・接続診断を使いこなす | [利用ガイド](docs/usage.ja.md) |
| 予想と違う結果の原因を調べる | [トラブルシューティング](docs/troubleshooting.ja.md) · [Reason code](docs/reason-codes.ja.md) |
| 対応バージョンと証拠を確認する | [実測範囲](docs/observed-coverage.ja.md) · [アダプターの成熟度](docs/adapter-maturity.ja.md) |
| CIへ継続的な確認を組み込む | [Suite manifest](docs/suite-manifest-v1.ja.md) · [Self-hosted CI](docs/self-hosted-ci-security.ja.md) |
| 実装を理解・拡張する | [アーキテクチャ](docs/architecture.ja.md) · [プロジェクト方針](docs/project-direction.ja.md) |
| 仕様・スキーマ・調査資料を探す | [ドキュメント一覧](docs/README.ja.md) |

## 一緒に、クライアント間の接続を確かに

再現できる接続レポート、わかりやすい使用例、日英ドキュメントの修正も、このプロジェクトを支える貢献です。新しいアダプターを実装する以外にも参加方法があります。

[接続の問題を報告する](https://github.com/git-ksk/mcp-interop/issues/new/choose)際は、クライアントのバージョン・OS・コマンド・秘密情報を除いた段階別の結果を添えてください。コード変更は[コントリビューションガイド](CONTRIBUTING.ja.md)、質問は[サポート](SUPPORT.ja.md)、今後の計画は[ロードマップ](docs/roadmap.ja.md)へ。

脆弱性は[セキュリティポリシー](SECURITY.ja.md)に従って非公開で報告してください。参加時のルールは[行動規範](CODE_OF_CONDUCT.ja.md)にまとめています。

## ライセンス

[Apache License 2.0](LICENSE)。
