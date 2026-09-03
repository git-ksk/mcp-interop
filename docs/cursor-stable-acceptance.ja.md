# Cursor stable acceptance

Cursor CLIは、明示的にevidenceがある **macOS arm64 / non-OAuth core path** に限って`stable`へ昇格します。

昇格に使うcurrent-main real-client evidence:

- `2026.08.04-aaa8809`: `reach/auth/init/tools=PASS`
- `2026.09.02-c22c1a3`: `reach/auth/init/tools=PASS`
- runner: macOS 26.5 (25F71), arm64
- real client boundary: isolated Cursor CLIの`mcp enable`、`mcp list`、`mcp list-tools`
- safety gate: user config不変、login Keychain DB不変、新規client processなし、`mcp-interop` session leakなし、`tools/call`なし

exact `2026.08.25-3e8eec8`のhistorical retained core evidenceもPR #108にあります。

Stable gate評価:

- `repeat_path_version_coverage`: **met** — 同じPASS core pathをcurrent-main上の2 exact client versionで確認
- `advertised_platform_coverage`: **met** — stable scopeをmacOS arm64へ明示的に限定
- `measurement_surface_stability`: **met** — supported Cursor MCP management surfaceを複数exact versionで繰り返し実証
- その他stable criteriaもすべて **met**

OAuth、Linux/Windows/macOS amd64、semantic-version rangeのstable claimではありません。未観測versionは実測されるまで`untested`です。
