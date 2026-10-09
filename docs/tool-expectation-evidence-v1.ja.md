# Expected-tool evidence v1

[English](tool-expectation-evidence-v1.md) | [日本語](tool-expectation-evidence-v1.ja.md)

別スキーマのオプトイン証拠です。**利用者が明示した期待ツール名**が実クライアントで確認できたかを保存し、観測した未指定の名前は公開しません。従来の4段階`reach/auth/init/tools`には追加しません。

## 証拠の生成

`mcp-interop test <url> --client codex --expect-tool ping --output core.json --deployment-id safe-label --tool-evidence expected.json`で、従来のprotected-path v2成果物と**別のtool-evidence v1ファイル**を出力します。期待名が欠落して終了コード`1`になっても、照合結果は保持します。クライアント1件・期待値の指定・未使用の出力先が必須です。`deployment-id`は機密ではない固定ラベルとし、URLパスや認証情報から作成しません。期待名はCLI引数に表示され得るため、秘密文字列は使わないでください。

## データ項目

- `schema_version: 1`
- `artifact_type: mcp-interop/tool-expectation-evidence`
- `run`: 厳格なprotected-path v2準拠の単一実行情報。実クライアント名・バージョン、実行OS・時刻、認証方法、provenance、4段階判定、originと公開deployment fingerprint。生のURL path/queryを保存しません。
- `expectation`: 最大64件の重複なしの期待名（昇順、1件128 UTF-8バイトまで）、期待件数0〜4096、直接観測できた場合の観測件数、欠落した**期待名だけ**、`pass`/`fail`/`unknown`と理由コード。

読込上限は1MiB。未知フィールド、シンボリックリンク入力、整合しない判定・件数・名前を拒否します。`unknown`では観測件数も欠落名も推測しません。直接の実クライアント証拠とcoreの完全PASSだけが既知の名前判定を支えます。保存先がすでにあれば上書きせず、可能な環境で0600権限を使用します。ただし**暗号署名による真実性の証明ではありません**。

## 差分比較

`mcp-interop tools compare baseline.json current.json --json --fail-on-drift`は、別スキーマ`mcp-interop/tool-expectation-diff` v1を出力。deployment/origin、クライアントID、実行OS/arch、認証方法、期待名・期待件数が一致する場合だけ比較します。クライアントのバージョン差だけでは退行と判断しません。

判定は以下のとおりです。

| 判定 | 意味 | ゲート指定時の終了コード |
|---|---|---|
| `clean` | 名前照合・観測件数が変化なし | 0 |
| `regression` | 新たな期待名欠落、または件数減少 | 1 |
| `drift` | 欠落の増加なしで件数増加 | 1 |
| `recovered` | 以前失敗していた期待名・件数が回復 | 0 |
| `non_pass` | 今回も期待値を満たさない | 1 |
| `unknown` | いずれかの試行で正確な証拠なし | 1 |

不正・互換性のない入力は終了コード`2`。`--fail-on-drift`なしでは、有効なレポートを出力した場合の終了コードは`0`です。**指定していないツール名が入れ替わって件数も変化しなければ検出できません**。重要なツール名は`--expect-tool`で明示し、総数の変化は`--expect-tool-count`で確認してください。既存のsuite・baseline・repeatのcore判定は変更しません。
