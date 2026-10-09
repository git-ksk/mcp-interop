# オフラインHTMLとCIサマリー（v0.11.0）

[English](offline-suite-report.md) | [日本語](offline-suite-report.ja.md)

オプトインの`report suite`で、既存の**検証済みsuite回帰結果**から単独で閲覧できる静的HTMLと簡潔なCI用Markdownを生成します。実クライアントの起動、Remote MCPへのアクセス、baselineの自動昇格、既存のstrictな成果物スキーマ変更は行いません。

## 使用例

```console
mcp-interop report suite baseline-results \
  repeat-results/attempt-01 repeat-results/attempt-02 \
  --html review.html --ci-summary ci-summary.md --fail-on-regression
```

baselineと全試行は同じmanifest fingerprint・trusted execution contextのsuite result setである必要があります。既存のsuite readerで個別成果物のsymlink/path・protected deployment識別の検証を行います。最大20試行・192件の論理runまで。それ以上は巨大HTMLを生成せず拒否します。

`--html`と`--ci-summary`はいずれか1つ以上を必須とし、出力先は別々の未作成ファイルです。既存ファイル・symlinkを上書きせず、可能なOSでは0600の専用権限で新規作成します。入力不正は終了コード`2`、書込エラーは`1`。`--fail-on-regression`を指定すると、退行または不安定な結果があれば**出力後**に終了コード`1`です。ゲート指定なしでは有効なレポートを出力した場合は`0`です。

## 機密保護・セキュリティ

HTMLとMarkdownの表示内容は、検証済み回帰モデルから**必要な情報だけに限定**します。suiteのdecision、退行・不安定フラグ、target/client/auth、baseline結果、**全試行の結果と証拠状態**、ステージ遷移を表示。deployment label、クライアントの生version文字列（変更有無のみ表示）、endpoint fingerprint、URL/path/query、ローカルファイルパス、ログ、OAuthトークン、任意のreason code文字列、未指定ツール名は**出力しません**。欠落・未観測・skip・errorをPASS扱いにしません。

HTMLはGo標準の`html/template`で値をエスケープし、JavaScript・イベント処理・外部画像・フォント・CSS・リンク・通信を一切使用しません。Content Security Policyは`default-src 'none'`で、スタイルは固定の埋込CSSのみです。CI用Markdownも検証済みの列挙値と制限されたIDだけから生成し、クライアント入力によるHTMLやリンク注入を防ぎます。

**限界:** HTML自体には署名・認証された実行証明はありません。実データの真実性は入力成果物の信頼境界に依存します。オプションの`test --tool-evidence`で保存した期待ツール差分は従来のcore suiteとは別であり、HTMLへ暗黙に混ぜません。詳細なクライアントversionや診断が必要な場合は、信頼できる環境で元の成果物を確認してください。CI用Markdownは自分のtrusted workflowで内容を確認してから`$GITHUB_STEP_SUMMARY`へ追加できます。公開PR CIから任意のRemote MCP接続先を実行する仕組みは追加しません。
