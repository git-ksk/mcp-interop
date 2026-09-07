# Antigravity stable acceptance

Antigravity CLIは、明示的にevidenceがある **macOS arm64 / non-OAuth core path** に限って`stable`へ昇格します。

昇格に使うcurrent-main real-client evidence:

- `1.1.22`: `reach/auth/init/tools=PASS`
- `1.1.24`: `reach/auth/init/tools=PASS`
- runner: macOS 26.5 (25F71), arm64
- real client boundary: isolated PTY + bounded observed live MCP tool-cache surface
- protocol evidence: `server/discover`後、`initialize`、`notifications/initialized`、`tools/list`へfallback
- safety gate: user config不変、login Keychain DB不変、新規client processなし、`mcp-interop` session leakなし、`tools/call`なし

Stable gate評価:

- `repeat_path_version_coverage`: **met** — 同じPASS non-OAuth pathを2 exact client versionで確認
- `advertised_platform_coverage`: **met** — stable scopeをmacOS arm64へ明示的に限定
- `measurement_surface_stability`: **met** — bounded observed tool-cache surfaceが複数exact versionで同じPASS boundaryを再現
- その他stable criteriaもすべて **met**

OAuth、non-macOS、modern `server/discover`でのtool discovery成功、semantic-version rangeのstable claimではありません。未観測versionは実測されるまで`untested`です。

## 起動安定化の追加検証（2026-09-08）

`fix/antigravity-stability`で、インストール済み`1.1.25`の非OAuth経路が
macOS 26.5 arm64上で4段階PASSしました。実行コマンドは
`MCP_INTEROP_CLIENTS=antigravity bash scripts/e2e-real-clients.sh`です。
fixtureでは`server/discover`の後にlegacyの`initialize`、
`notifications/initialized`、`tools/list`を観測しました。
通常ユーザーの状態・login Keychain DB不変、process/session cleanup、
`tools/call`なしのgateもPASSしています。公開版ではなく、このブランチの検証結果です。

通常環境で初期設定が完了している場合だけ、3つのbooleanフラグを一時HOMEへ
引き継ぎます。ファイルなし・未完了を完了扱いにはせず、不正な形式・サイズ超過・
シンボリックリンクは拒否します。アカウント・token stateはコピーしません。
PTYは40行×120列で起動します。子プロセスに設定が届くことと、初期設定の
異常入力を回帰テストで確認します。初回利用者のオンボーディングとOAuthは、
今回追加したstable evidenceの対象外です。
