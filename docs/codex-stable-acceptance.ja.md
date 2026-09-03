# Codex stable adapter acceptance

[English](codex-stable-acceptance.md) | [日本語](codex-stable-acceptance.ja.md)

このrecordは、shipped Codex adapterを`beta`から`stable`へ昇格するためのevidence gapを、意図的に狭い次のscopeでcloseします。

- client: Codex CLI
- runner: macOS 26.5 (25F71), arm64
- path: controlled localhost、non-OAuth core interoperability path
- 必須stage: `reach/auth/init/tools=PASS`
- safety gate: normal user config unchanged、login Keychain DB unchanged、new Codex processなし、`mcp-interop` session leakなし、`tools/call`なし

stable claimにはOAuth、macOS amd64、Linux、Windows、推測したCodex version rangeを含めません。

## 複数exact versionのevidence

両方ともcurrent pre-v1 main `8e12549f0563892b09b2ec6127eed2071fa376bd`に対し、同じ`scripts/e2e-real-clients.sh` real-client boundaryを使用しました。

| Exact Codex version | Result | Fixture protocol observation | Safety gates |
| --- | --- | --- | --- |
| `codex-cli 0.133.0` | `reach/auth/init/tools=PASS` | `initialize`で`2025-06-18`を提示 | 全PASS、`tools/call`なし |
| `codex-cli 0.152.1` | `reach/auth/init/tools=PASS` | `initialize`、`notifications/initialized`、`tools/list`で`2025-06-18`を観測 | 全PASS、`tools/call`なし |

`0.133.0` binaryはHomebrew upgrade前に退避し、同じcurrent mainのdetached worktreeに対して一時PATHから実行しました。Homebrewで導入した`0.152.1`も同一harnessをPASSしています。

これはexact observed pointです。`0.133.0`と`0.152.1`の間やそれ以降のversionがtestedであることを意味しません。

## Stable gate decision

advertised stable scopeについて:

- `repeat_path_version_coverage`: **met** — PASS claimを構成する同じnon-OAuth core pathを2つのexact Codex versionで実証
- `advertised_platform_coverage`: **met** — stable advertised scopeをretained exact evidenceのあるmacOS arm64へ明示的に限定
- beta criteria、measurement-surface stability、regression-maintenance criteriaもすべて**met**のまま

OAuthは明示的opt-in機能として利用できますが、このstable maturity scope外です。v1前OAuth revalidationで別管理します。
