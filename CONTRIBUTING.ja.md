# mcp-interopへのコントリビューション

[English](CONTRIBUTING.md) | **日本語**

> この文書は英語版`CONTRIBUTING.md`の日本語訳です。運用上の差異がある場合は英語版を正とします。

`mcp-interop`の改善に協力いただき、ありがとうございます。

このプロジェクトで最も重要なのは、対応クライアントの数ではなく、**相互運用性PASSの意味を信頼できる状態に保つこと**です。

## Pull Requestを作る前に

1. 関連するIssueやPull Requestがないか確認してください。
2. 新しいクライアント向けアダプターを追加する場合は、そのクライアントで安全に観測できるMCPインターフェースと、設定・認証情報を隔離する方法をIssueへ記録してください。
3. 新しいクライアント対応は、原則として `Issue / 調査 → 安全に観測できる経路の確認 → 範囲を限定したPoC → 実装` の順で進めます。先にアダプターを作り、あとから「何を証拠とするか」を決めないでください。
4. ユーザーが普段使っているクライアント設定や認証情報を、黙って変更する実装は追加しないでください。
5. unsupported / blocked / experimental / research-onlyの状態は正確に表示してください。一部しか観測できないものを「対応済み」と表現しないでください。

## 開発環境

必要なGoバージョンは`go.mod`を確認してください。

通常はCIと同じ基本チェックを実行します。

```console
gofmt -w .
git diff --exit-code
go vet ./...
go test ./...
go build ./cmd/mcp-interop
```

プロセス管理、OAuth、共有状態、release gateに関係する変更では、追加で次も確認してください。

```console
go test -race ./...
govulncheck ./...
```

`govulncheck`をローカルで実行できなかった場合はPRへ明記し、CIの固定バージョンによる検査結果を確認してください。

必須status checkが失敗中、または未実行の状態ではmerge可能とは扱いません。

## 実クライアント向けアダプターの要件

アダプターは次の原則を守ってください。

- クライアントの挙動を模倣せず、実際にインストールされたクライアントを起動する
- クライアント自身の管理・制御インターフェースで確認できる場合は、モデルへのプロンプトを使わない
- 設定や認証情報を一時profile / `HOME` / config directoryへ隔離する
- クライアント側から結果を証明できない場合は、成功を推測せず`unknown`を返す
- `reach`、`auth`、`init`、`tools`を別々の観測結果として扱う
- 成功・失敗にかかわらず、一時状態と今回のテストが所有するプロセスを片付ける
- reportやerrorからBearer/OAuth情報などの秘密情報を除去する
- 実際に検証したクライアントの正確なバージョンを記録する
- 成功経路だけでなく、主要な失敗・判定不能経路にもテストを追加する

fixtureの成功は、アダプターの測定経路が正しく動くことを示すためのものです。別の本番Remote MCPが実クライアントでPASSしたことまでは証明しません。

設定ファイルに登録できた、メタデータ上は互換だった、許可リストにツール名があった、といった事実だけをlive PASSへ昇格させないでください。

安全な隔離ができないクライアントは、無理にユーザー設定を変更して対応せず、research-onlyや`unknown` / `skip`のままにしてください。

## OAuthを変更するとき

OAuthは特に慎重に扱います。

- 人間の操作を必要とする可能性がある認証は、必ず明示的なopt-inにする
- クライアント契約で明示されていないauthorization URLを自動で開かない
- テスト用credentialを通常のOS Keychainやクライアントの永続credential storeへ保存しない
- 通常ユーザーのブラウザ・クライアント認証情報を一時profileへコピーしない
- authorization URL、authorization code、callback state、access/refresh token、cookie、client secret、private keyをmachine-readable resultや公開証拠へ含めない
- 自動テストでは本番credentialではなく、ローカルまたはsyntheticなOAuth fixtureを使う

## Pull Requestの運用

開発はPR-firstです。`main`へ入れる変更は専用branchで作業し、Pull Request経由でmergeします。

PRには最低限、次を記載してください。

- 関連Issueや調査背景
- 変更範囲と明確な非目標
- クライアント挙動に関係する場合は、確認した製品名とバージョン
- 相互運用性を証明するために使った具体的な観測手段
- 設定・認証情報の隔離方法と終了処理
- ローカルテスト、CI、必要に応じてE2Eの結果
- 秘密情報が出力されないことの確認
- 関連ドキュメントの更新状況
- 意図的に`unknown`として残す制限

通常のPRはsquash mergeを使用します。必須CIがgreenになる前にmergeしません。

同じ公開契約、安全境界、ユーザー向け挙動を変更する場合は、英語版と日本語版の文書を原則として同じPRで更新してください。

## Project directionとRoadmap

`docs/project-direction*.md`と`docs/roadmap*.md`は役割が異なります。

- Project direction — プロジェクトが何を目指し、何を優先するか
- Roadmap — その方針をどの順序・完了条件で進めるか

ロードマップ上の将来機能を、現在提供済みの機能としてREADMEへ書かないでください。

## リリースとバージョニング

リリース準備もPR-firstで行います。

通常の流れ:

1. release-prep PRを作成する
2. 必要に応じて`CHANGELOG.md`とREADMEの日英ペアを更新する
3. 必須CIがgreenになってからmergeする
4. `main`に含まれるcommitへrelease tagを作る
5. `.github/workflows/release.yml`を実行する
6. アーカイブ、`checksums.txt`、埋め込みバージョン、CLI smoke、artifact attestationを確認する

release workflowは、`origin/main`に含まれないcommitを指すtagを拒否します。また、artifact公開前にsource quality/security gateを再実行します。

`v0.x`でもSemVerの意図に沿って扱います。

- **patch** — 公開契約を意図的に壊さないbug/security fix、documentation、maintenance
- **minor** — backward-compatibleな機能追加、意味のあるcapability追加
- **major** — プロジェクトの成熟後、意図的なbreaking public contract向け

`v0.9.0`の次が`v1.0.0`とは限りません。必要なら`v0.10.0`、`v0.11.0`以降を継続し、[roadmap](docs/roadmap.ja.md)のstable-contract完了条件を満たした場合だけ`v1.0.0`へ進みます。

## セキュリティ報告とサポート

セキュリティ上の問題は公開Issueへ書かず、[SECURITY.ja.md](SECURITY.ja.md)に従ってPrivate Vulnerability Reportingを利用してください。正式なセキュリティポリシーは[SECURITY.md](SECURITY.md)です。

通常の不具合、相互運用性レポート、機能要望、使い方の質問は[SUPPORT.ja.md](SUPPORT.ja.md)とIssue templateを参照してください。

公開レポートには、本番サービスの機密識別情報、秘密のendpoint値、credentialを含めないでください。
