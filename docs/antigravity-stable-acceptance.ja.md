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
