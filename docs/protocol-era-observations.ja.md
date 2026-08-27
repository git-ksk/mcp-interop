# 現行real-clientのprotocol-era観測

[English](protocol-era-observations.md) | **日本語**

> この文書は英語版`protocol-era-observations.md`の日本語訳です。証拠記録の正本は英語版です。

この文書は、v0.6 Issue #99として現在提供しているCodex、Cursor、Antigravity adapterを再観測した結果です。#100のprotocol-aware設計と#101のcontrolled-fixture matrixへの入力であり、この文書だけで現在の公開`reach -> auth -> init -> tools`契約を変更するものではありません。

## 証拠の境界

次の2層を混ぜません。

1. **deployment-specific adapter evidence** — 対象deploymentに対して、adapterが利用するsupported / deliberately acceptedな実client surfaceから`mcp-interop test`自身が証明できる情報
2. **controlled-fixture wire evidence** — 同じadapterで実clientをlocalhost fixtureへ接続したとき、fixtureがwire上で直接観測できた情報

protocol revisionをfixtureだけが観測しても、本番runへ転記しません。実client adapter surfaceがnegotiated protocol revisionを返さない場合、deployment-specificなprotocol revisionは`unknown`のままです。

公式MCP `2026-07-28` releaseでは、この作業に関係するprotocol eraを大きく2つに分けられます。legacy revisionは`initialize` / `notifications/initialized`を使い、modern `2026-07-28` revisionではそのhandshakeが削除され、必要に応じて`server/discover`を利用します。詳細は[MCP 2026-07-28 specification release](https://blog.modelcontextprotocol.io/posts/2026-07-28/)を参照してください。

## 再観測環境

2026-08-27に次の環境で観測しました。

```text
macOS 26.5 (25F71)
arm64
mcp-interop main d848de52d74e6c7857adaf75802108fa4d05b5c2
```

実client:

```text
Codex CLI        codex-cli 0.133.0
Cursor CLI       2026.08.25-3e8eec8
Antigravity CLI  1.1.22
```

Cursorは現在の公式CLI installerで導入しました。supported MCP management surfaceには`cursor-agent mcp list`と`cursor-agent mcp list-tools <identifier>`があります。[Cursor CLI parameter reference](https://docs.cursor.com/en/cli/reference/parameters)を参照してください。

Antigravityのdocumented interactive MCP management surfaceは引き続き`/mcp`です。[Antigravity CLI reference](https://antigravity.google/docs/cli/reference/)を参照してください。

## 結果

| Client | deployment-specific evidenceに使うadapter surface | そのsurfaceからprotocol revisionが見えるか | controlled fixtureでのwire観測 | fixture上のera結論 |
| --- | --- | --- | --- | --- |
| Codex CLI 0.133.0 | isolated `codex app-server`とapp-server MCP inventory/status surface | 見えない。live inventory/auth/tool stateは取得できるがMCP wire revisionは返らない。 | `initialize`で`2025-06-18`を提示し、`notifications/initialized`、`tools/list`へ進行。 | legacy pathを観測。 |
| Cursor CLI 2026.08.25-3e8eec8 | isolated workspace/HOMEで`cursor-agent mcp enable`、`mcp list`、`mcp list-tools` | 見えない。supported command outputはlive tool discoveryを証明するがnegotiated MCP revisionは返さない。 | management commandごとにlegacy sessionを開始し、`initialize`は`2025-11-25`。`list-tools` pathで`tools/list`まで到達。 | legacy pathを観測。 |
| Antigravity CLI 1.1.22 | isolated PTY + bounded live MCP tool-cache観測。`/mcp`はexplicit OAuth management用 | 見えない。cacheはlive tool materializationを証明するがnegotiated MCP revisionは返さない。 | 最初に`server/discover`を`2026-07-28`で送信。その後`initialize` `2025-11-25`へfallbackし、`notifications/initialized`、`tools/list`へ進行。 | modern probe + legacy fallback成功を観測。modern tool discovery成功は未証明。 |

3clientとも既存real-client release gateを通過しました。

- adapter resultは`reach` / `auth` / `init` / `tools`すべてPASS
- fixtureが`tools/list`を観測
- `tools/call`なし
- normal user configuration変更なし
- login Keychain database変更なし
- 新しいclient process残留なし
- `mcp-interop` temporary session漏れなし

interoperability oracleとしてmodel promptは使っていません。

## この観測が証明するもの

現在のadapterがcontrolled localhost deploymentに対して既存real-client evidence pathを引き続き成立させられること、およびfixtureでは実際に使われたprotocol eraを区別できることを証明します。

ただし、本番deploymentが同じrevisionを利用したことは証明しません。現在の3adapterはいずれもdeployment-specific surfaceからnegotiated MCP protocol revisionを安全に取得できません。そのため:

```text
fixture protocol revision != production-run protocol revision
```

future real-client surfaceが直接revisionを返すまではproduction-run revisionは`unknown`です。

## #100への入力

今回の観測からprotocol-aware core設計には次の制約があります。

1. public `init=pass`を永久に「literalな`initialize` requestを観測した」と定義できない。modern MCPにはそのwire phaseがない
2. protocol detailを捏造せず既存public `init`へprojectできるprotocol-readinessの内部semanticが必要
3. protocol revision/eraはoptional evidenceとし、client surfaceから見えないdeployment runでは`unknown`をdefaultにする
4. fixture-only protocol evidenceにはfixture evidenceであることを明示し、本番runをupgradeしてはいけない
5. Antigravityで観測した`2026-07-28` probe -> legacy fallbackを、modern-era semantics完成前に#101 fixture matrixで明示的にcoverする

## controlled protocol-era fixture matrix

Issue #101では、fixtureをdeployment-specific evidenceへ昇格させずにprotocol-aware behaviorを検証するため、3つのmodeを明示します。

| Mode | controlled behavior | 目的 |
| --- | --- | --- |
| `legacy` | `server/discover`をrejectし、handshake revisionは`initialize`、`notifications/initialized`、`tools/list`を利用できる | legacy readiness projectionを維持できることを証明する |
| `modern` | legacy `initialize`をrejectし、`server/discover`と`tools/list`には明示的な`2026-07-28` request version evidenceを要求する。discovery/list responseには保守的なcache hintを付ける | handshakeを捏造せずstateless modern readinessを証明する |
| `fallback` | `server/discover`へ意図的にnon-definitiveなresponseを返し、その後legacy initializeを利用できる | 現在のAntigravityで観測したmodern probe -> legacy fallbackのような安全なfallbackを再現する |

core interoperability proofで`tools/call`は不要です。modern protocol versionが無い、またはunsupportedな場合はmodern successへ暗黙変換せずrejectします。

real-client release gateもprotocol-era-awareにします。controlled fixture readinessがPASSになるのは次のどちらかだけです。

- completeなlegacy `initialize -> notifications/initialized -> tools/list`を観測した場合
- `tools/list`自体に明示的な`2026-07-28` protocol evidenceがある場合

modern `server/discover` probeだけでは不足です。adapter自身はdeployment-specificなreal-client surfaceから独立してPASSする必要があり、fixture evidenceはrelease gate / self-test evidenceのままです。

現在の製品に関係しないhistorical remote HTTP/SSE variantをcoverage数のためだけに追加しません。matrixは現在のRemote MCP scopeと、shipping clientで観測済みまたは想定されるprotocol eraへ集中します。

## 主張しないこと

この再観測では次を主張しません。

- 新adapter追加
- shipping adapterのnative modern `2026-07-28` tool discovery対応
- production tool callの必須化
- OAuth挙動変更
- existing live-PASS invariantの緩和
