# Suite repeat report v1

[English](suite-repeat-report-v1.md) | [日本語](suite-repeat-report-v1.ja.md)

`suite repeat`は、**同じ検証済みtrusted manifest**を2〜5回実行し、各試行を独立したsuite result setとして保持する機能です。最初の試行を自動で正式baselineに昇格せず、MCPツール実行やモデルへのプロンプト送信も行いません。

## コマンド

```console
export MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A='https://example.com/mcp/<protected-path>'
mcp-interop suite repeat suite.json \
  --output-dir repeat-results \
  --attempts 3 \
  --timeout 45s \
  --json
```

`--attempts`は**2〜5回**、`--timeout`はクライアント1件ごとに**1秒〜10分**で、どちらも必須です。宣言した実行時間上限の合計が**45分**を超える場合は実行前に拒否します。出力先は未作成のディレクトリが必要です。PRのhosted CIから任意のRemote MCP URLを実行可能にするものではありません。

出力先は可能なOSで0700に制限し、次を保持します。

```text
repeat-results/
  manifest.json             # 検証済みの固定manifest。endpointの値は含まない
  attempt-01/index.json     # suite result set v1
  attempt-01/artifacts/...  # protected-path live-result v2
  attempt-02/index.json
  attempt-02/artifacts/...
  attempt-03/index.json
  attempt-03/artifacts/...
  repeat-report.json        # 別schemaの派生集計結果
```

全endpointは開始前に1回だけ解決し、後の各試行でメモリ内の同一値と固定manifestを使います。各試行のディレクトリは既存`suite run`で原子的に確定します。後続の失敗・キャンセルで**完了済みの試行を削除・上書きしません**。中断した試行の結果が未確定な場合は`incomplete`にし、保存済みの試行だけを根拠に集計します。

## 派生レポートの仕様

`repeat-report.json`は**別個のschema v1の派生レポート**です。strictなsuite index・live-result artifact・署名済み証明ではありません。

- `schema_version`: `1`
- `artifact_type`: `mcp-interop/suite-repeat-report`
- `manifest_fingerprint`: manifest宣言のfingerprint。生のendpoint値はhashしない
- `requested_attempts` / `completed_attempts`: 要求回数と完了回数
- `complete`: 途中キャンセルなどで未完了なら`false`
- `decision`: `clean`、`non_pass`、`unstable`、`non_pass_and_unstable`、`incomplete`
- `has_non_pass` / `has_unstable`: 完了済みの全試行から計算した真偽値
- `attempt_indexes`: `attempt-01/index.json`等の順序付き相対参照
- `runs[]`: target・deployment・client・auth単位の`all_pass`、`unstable`、全試行
- `runs[].attempts[]`: 試行番号と、結果・exit code・成果物参照・実クライアントのexact version・実行platform・endpoint fingerprint・4段階のstatus/reason等。実行エラーの場合、根拠のないステージ情報は作らない

`clean`は、予定した全試行が完了してすべてPASSし、観測された状態が一致した場合のみです。client version、ステージ状態・reason、実行platform、endpoint fingerprint、結果の変化も区別します。全回失敗なら`non_pass`、FAIL→PASSやPASS→FAILなら`non_pass_and_unstable`、中断なら完了した試行がPASSでも`incomplete`です。

終了コードは全試行が一貫してPASSなら**0**、失敗・不安定・中断・実行エラーなら**1**、manifest・設定値・endpoint・実行予算・既存出力先などの実行前の不正には**2**です。

集計結果は生のendpoint path/query、認証情報、モデルプロンプト、実行ログ、未指定のツール名を保存しません。成果物の真実性はreportだけでは保証できません。試行ごとのindexとschema v2成果物を証拠として保持してください。

## 正式baselineとの比較

`repeat-report`は**同じ試行が安定してPASSしたか**を判定します。**受け入れ済みのbaselineから退行したか**は、`suite compare`で全試行を指定して検証します。

```console
mcp-interop suite compare baseline-results \
  repeat-results/attempt-01 \
  repeat-results/attempt-02 \
  repeat-results/attempt-03 \
  --json --fail-on-regression
```

attempt-01を暗黙に正式baselineへ昇格しません。既存の`suite compare` / `baseline compare`のschemaと意味も維持します。また、optionalな期待ツール名照合はlive-result v2に含まれないため、suite repeatのPASSだけで期待名の一致を保証するものではありません。
