# Stableな相互運用契約に向けたロードマップ

[English](roadmap.md) | **日本語**

> この文書は英語版`roadmap.md`の日本語訳です。計画の正本は英語版です。

このロードマップは、[プロジェクト方針](project-direction.ja.md)で定めた原則を、milestone・完了条件・非目標へ落とし込む文書です。

日付を約束する計画ではありません。minor versionは想定する作業順序を示しますが、未完成機能を出すために安全境界を緩めるものではありません。

`v0.9.0`や`v0.10.0`の次が自動的に`v1.0.0`になるわけではありません。必要なら`v0.11.0`、`v0.12.0`以降を継続し、stable contractの完了条件を満たした場合だけ`v1.0.0`へ進みます。

## 他の文書との役割分担

- [Project direction](project-direction.ja.md) — ミッション、証拠の優先順位、プロジェクト境界
- このRoadmap — 作業順序と各milestoneの完了条件
- [Architecture](architecture.ja.md) — 現在の実装・信頼境界
- [Live result artifact schema v1](live-result-schema-v1.ja.md) / [schema v2](live-result-schema-v2.ja.md) — 現在提供済みのportable result形式
- [Conformanceとの違い](conformance-vs-interop.ja.md) — 公式MCP Conformanceとの役割分担
- [CONTRIBUTING](../CONTRIBUTING.ja.md) — 変更提案・レビュー方法

ロードマップ上の将来像を、現在提供済みの挙動として扱わないでください。現在の仕様はコード、release documentation、versioned schemaを正とします。

## GitHubでの実行管理

Roadmapは計画と完了条件の正本です。GitHub Milestoneとroadmap tracking Issueは**進捗**の正本として扱います。実装・調査を始める前にmatching Milestone付きのfocused Issueを作り、tracking Issueからリンクします。完了済みIssueは作り直さずmilestone historyとして残します。

| Roadmap milestone | Tracking Issue | 現在のfocused Issue |
| --- | --- | --- |
| v0.6.x | [#102](https://github.com/git-ksk/mcp-interop/issues/102) | **完了:** [#87](https://github.com/git-ksk/mcp-interop/issues/87), [#99](https://github.com/git-ksk/mcp-interop/issues/99), [#100](https://github.com/git-ksk/mcp-interop/issues/100), [#101](https://github.com/git-ksk/mcp-interop/issues/101) |
| v0.7.x | [#103](https://github.com/git-ksk/mcp-interop/issues/103) | **完了:** [#112](https://github.com/git-ksk/mcp-interop/issues/112), [#113](https://github.com/git-ksk/mcp-interop/issues/113), [#114](https://github.com/git-ksk/mcp-interop/issues/114), [#115](https://github.com/git-ksk/mcp-interop/issues/115) |
| v0.8.x | [#104](https://github.com/git-ksk/mcp-interop/issues/104) | **完了:** [#125](https://github.com/git-ksk/mcp-interop/issues/125), [#126](https://github.com/git-ksk/mcp-interop/issues/126), [#127](https://github.com/git-ksk/mcp-interop/issues/127) |
| v0.9.x | [#105](https://github.com/git-ksk/mcp-interop/issues/105) | 調査候補: [#6](https://github.com/git-ksk/mcp-interop/issues/6), [#20](https://github.com/git-ksk/mcp-interop/issues/20), [#48](https://github.com/git-ksk/mcp-interop/issues/48), [#68](https://github.com/git-ksk/mcp-interop/issues/68) |
| v0.10.x | [#106](https://github.com/git-ksk/mcp-interop/issues/106) | contract review開始時にfocused audit/fix Issueへ分割 |

この対応は双方向です。roadmap作業は実装開始前にGitHub Issueを持ち、Issueがroadmapのscopeや完了条件を変える場合は同じPRで英語・日本語Roadmapも更新します。

## 全milestoneで守る原則

1. **機能数より証拠の正しさを優先する。** PASSを出しやすくするために意味を弱めない。
2. **live PASSは実クライアントの証拠だけ。** fixture、config、metadata、direct server inspection、diagnostic evidenceで代用しない。
3. **分からないものは`unknown`のまま。** 一覧を埋めるために成功を推測しない。
4. **対応済みへの昇格は安全性を証明してから。** version milestoneを理由に基準を下げない。
5. **互換性変更は実証してから。** 現行契約で表現できない具体的問題を確認してからnew schemaやbreaking contractを導入する。
6. **コアはlocal-first。** hosted backendやアカウントを必須にしない。
7. **通常ユーザーのcredentialを使い回さない。** テストを通すためにbrowser/client tokenやKeychain stateをコピーしない。

## コア相互運用プロファイル

コアとなる最小の主張は **Remote Tool Interoperability** です。

> 実クライアントが対象Remote MCPへ到達し、必要な認証境界を通過し、その時代のMCPで利用可能なprotocol pathを成立させ、そのクライアント自身から実ツール一覧の証拠を取得できる。

任意の本番tool callはコアPASSの必須条件にしません。ツール実行には副作用があり得るためです。

Resources、Prompts、Tasks、MRTR、controlled fixtureでの安全なtool callなどは、将来別capability profileとして追加できます。

現在の公開result modelは次です。

```text
reach -> auth -> init -> tools
```

## MCP protocol世代への対応

MCP protocolは変化します。

英語正本が参照している公式`2026-07-28` revisionでは、従来の`initialize` / `notifications/initialized` handshakeとprotocol-level session modelが廃止され、requestはself-describingになっています。serverは`server/discover`を実装しますが、clientがdiscoveryを使うこと自体は必須ではありません。

そのため`mcp-interop`は、**実際に観測したprotocol情報**と、**相互運用性として正規化した意味**を分けます。

```text
実クライアントを実行
  -> 観測できたprotocol / client evidence
  -> protocol世代ごとの解釈
  -> コア相互運用性の判定
```

protocol versionを推測しません。productionのクライアント側から見えなければ`unknown`のままです。

fixtureで分かったprotocol versionを、別の本番runへ転記して「実クライアントが証明した」ことにはしません。

### Remote transportの範囲

コアは引き続きRemote MCP deploymentへ集中します。

- Streamable HTTPをprimary modern remote transportとする
- 実クライアントが利用している間はlegacy HTTP/SSEも観測対象にできる
- 過去のすべてのtransportを永続サポートすることは目的にしない
- `stdio`は方針変更がない限りcore scope外

## v0.6.x — protocol-aware coreとdeployment privacy

**GitHub tracking:** [#102](https://github.com/git-ksk/mcp-interop/issues/102)。focused workは[#99](https://github.com/git-ksk/mcp-interop/issues/99)、[#100](https://github.com/git-ksk/mcp-interop/issues/100)、[#101](https://github.com/git-ksk/mcp-interop/issues/101)。protected-path deployment identityの[#87](https://github.com/git-ksk/mcp-interop/issues/87)は完了済みです。

**Status:** 完了。#87 / #99 / #100 / #101でv0.6.xの完了条件を満たしています。v0.7.xも完了済みで、現在の次milestoneはv0.8.x / #104です。

### 目的

現在のlive PASSの意味を弱めず、古いMCPと新しいMCPのprotocol世代差を正しく扱えるようにします。

### 主な作業

- Codex / Cursor / Antigravityで実際に観測できるprotocol情報を再確認する
- Remote Tool Interoperability向けのprotocol-awareな解釈を定義する
- production clientからprotocol revisionを証明できない場合は`unknown`を維持する
- 公開`init` stageはprotocol readinessの互換projectionとして維持し、stable semanticをliteral legacy `initialize` handshakeへ固定しない
- fixtureでlegacy / modern protocol behaviorを確認する
- 隔離、timeout、cleanup、secret redactionを維持する
- portable artifactを日常的にbaseline共有する前にdeployment identity privacy境界を維持する。schema v2は明示的な非secret deployment IDでcredential-bearing pathを除去するが、private originの共有は別問題として残す
- schema v1の既存semanticsを維持する。schema v2は#87でprotected-path limitationを実証したため導入済みであり、今後のschema revisionも具体的必要性と明示的なcompatibility/migration rationaleを必須にする

### deployment identityのプライバシー

credentialを除去したURLでも、hostname/path自体が運用上の機密情報である場合があります。

```text
credential-safe != deployment-public
```

単純なdeterministic hashだけでは、候補URLを推測して照合できる可能性があります。

schema v2では`production-a`のようなユーザー定義opaque identityをprotected-path endpointに使い、canonical public originとその非secret identityの組でpairします。protected path自体はhashもしません。ただしcanonical originは保存するため、credential-bearing pathの問題は解消してもprivate hostnameまで公開可能になるわけではありません。

### 完了条件

- protocol世代差によりfalse PASSを出さない
- 観測できないprotocol情報は`unknown`
- 既存3アダプターに適切なfixture coverageがある
- deployment identityのsecret/privacy modelを文書化できる
- schema変更がある場合は、必要性とmigration rationaleが明確

### 非目標

- 新クライアント追加は必須ではない
- hosted history serviceは作らない
- generic MCP Conformanceの代替にはしない
- production tool callを必須にしない

## v0.7.x — 繰り返し可能な退行テスト

**GitHub tracking:** [#103](https://github.com/git-ksk/mcp-interop/issues/103)。focused workの[#112](https://github.com/git-ksk/mcp-interop/issues/112)、[#113](https://github.com/git-ksk/mcp-interop/issues/113)、[#114](https://github.com/git-ksk/mcp-interop/issues/114)、[#115](https://github.com/git-ksk/mcp-interop/issues/115)は完了済みです。

**Status:** 完了。#112 / #113 / #114 / #115でv0.7.xの完了条件を満たし、次に進むRoadmap milestoneはv0.8.x / #104です。

### 目的

現在のartifact / compare機能を、各リポジトリで大量の独自glueを書かなくても使える運用workflowへ発展させます。

### 主な作業

- target、client、auth mode、許可された実行環境を記述するsecret-safeなsuite / manifest
- 複数クライアントの実行とartifact生成
- evidenceからcompatibility / regression reportを生成
- stage / reason code / client version / protocol evidenceの変化をmachine-readableに保持
- retry / flake semanticsの定義
- real-client CIのtrust boundaryを明確化

### CIの信頼境界

self-hosted real-client runnerを、信頼できないPull Requestから任意ネットワーク・credential実行環境として悪用できないようにします。

原則:

- 通常のuntrusted PRはhosted runner + localhost fixtureだけ
- self-hosted real-client executionはtrusted branch、manual dispatchなど明示的に承認された経路だけ
- PR内のmanifestからprivate hostやproduction credentialへ勝手に向けられないようにする
- OAuthは引き続きexplicit opt-in

### 完了条件

宣言したsuiteから、real-client artifact生成、比較、compatibility report、deterministic CI decisionまで一連で実行でき、同時にCI trust boundaryを守れること。

## v0.8.x — baselineと互換性の観測範囲

**GitHub tracking:** [#104](https://github.com/git-ksk/mcp-interop/issues/104)。focused workは[#125](https://github.com/git-ksk/mcp-interop/issues/125)、[#126](https://github.com/git-ksk/mcp-interop/issues/126)、[#127](https://github.com/git-ksk/mcp-interop/issues/127)です。

**Status:** 完了。#125 / #126 / #127でv0.8.xの完了条件を満たし、次のRoadmap milestoneはv0.9.x / #105です。

### 目的

client auto-updateや複数version/platformをまたぐ継続的な退行検出を運用できるようにします。

### 主な作業

- baselineの作成・選択・更新・廃止手順
- accidental baseline replacementで退行が隠れない仕組み
- stale / missing baseline evidenceの検出
- 連続したversion rangeを推測せず、**実際に検証した点**でcompatibility envelopeを表す
- `tested` / `untested` / `stale` / `known-broken`を必要に応じて区別
- client version、runner platform/architecture、auth mode、test時刻・context、証拠の出所を保持

例:

```text
Tested:
  Cursor X on runner macOS arm64 -> PASS
  Cursor Y on runner macOS arm64 -> PASS

これは次を意味しない:
  XからYまでの全versionがsupported
```

## v0.9.x — coverage、capability profile、安全な昇格

**GitHub tracking:** [#105](https://github.com/git-ksk/mcp-interop/issues/105)。既存research候補[#6](https://github.com/git-ksk/mcp-interop/issues/6)、[#20](https://github.com/git-ksk/mcp-interop/issues/20)、[#48](https://github.com/git-ksk/mcp-interop/issues/48)、[#68](https://github.com/git-ksk/mcp-interop/issues/68)は将来のgraduation判断先としてここへ紐付けます。調査自体はそれ以前でも継続できます。

### 目的

既存アダプターの信頼性を深め、十分な証拠境界を持つ製品・capabilityだけを対応済みにします。

### Focused implementation Issue

1. [#133](https://github.com/git-ksk/mcp-interop/issues/133) — cross-runner clock skew chronology hardening
2. [#134](https://github.com/git-ksk/mcp-interop/issues/134) — runner platformとreal client executable architectureの区別
3. [#136](https://github.com/git-ksk/mcp-interop/issues/136) — shipped clientのexact observed version / OS coverage matrix
4. [#135](https://github.com/git-ksk/mcp-interop/issues/135) — baseline authenticity / acceptance provenance境界
5. [#137](https://github.com/git-ksk/mcp-interop/issues/137) — evidence-based adapter maturityとbeta -> stable基準
6. [#138](https://github.com/git-ksk/mcp-interop/issues/138) — capability profile evidence contractと正確なPASS semantics
7. [#139](https://github.com/git-ksk/mcp-interop/issues/139) — 新real client向けequal-evidence graduation gate

依存順: #133 -> #134 -> #136 -> #135 -> #137 -> #138 -> #139。

Baseline acceptanceはv0.9でもbaseline schema v1のlocal-firstを維持します。digest/fingerprintが証明するのは内部content consistencyであり、authenticated acceptanceではありません。team/CIでauthenticityが必要な場合はexact baseline fingerprintを外部のreview/signature/attestation recordへbindします。将来native signed provenanceを追加する場合も、baseline-v1 metadataの意味を変えず別versioned envelopeを使います。

明示的なv0.9 maturity reviewでは、現在Codex / Cursor / Antigravityをすべて`beta`と分類します。`tier=v1`はdelivery/roadmap designationのままで、stable evidence claimではありません。stable promotionは[Adapter maturity contract](adapter-maturity.ja.md)に記載するexact criterion gapでblockし、client version変更だけではmaturityを自動変更しません。

優先順:

1. Codex / Cursor / Antigravityを現実的な最新version横断で検証
2. 安全に可能な範囲でOS/platform coverageを拡大
3. beta -> stableはプロジェクト年齢ではなく証拠で判断
4. Resources / Prompts / Tasks / MRTRなどは、PASSの意味を正確に定義できる場合だけ追加
5. 新クライアントは既存と同じ基準を満たす場合だけ昇格

新クライアントが0件でも、証拠品質や既存アダプターの成熟度が大きく改善すれば成功です。

## v0.10.x — 公開契約候補

**GitHub tracking:** [#106](https://github.com/git-ksk/mcp-interop/issues/106)。contract review開始時にfocused audit/fix Issueへ分割します。

将来`v1.x`で維持する可能性がある公開契約を整理します。

対象例:

- CLI command / flag semantics
- primary JSON output
- artifact schema evolution
- reason code
- exit code
- adapter identity / maturity state
- core / capability profileの意味
- protocol-era policy
- baseline / compatibility report
- deprecation / removal policy
- security / privacy / cleanup保証

根本的なCLI・schema・evidence model変更がまだ起こりそうなら、`v0.x`を継続します。

## v0.11.x以降 — 安定化のための余白

`v0.11.0`以降へ特定featureを予約しません。

実運用で見つかったprotocol差、schema migration、cross-platform cleanup、baseline UX、OAuth変更などを必要なだけ追加`v0.x`で解消します。

`v1.0.0`を急ぐより、`v0.12.0`、`v0.13.0`以降を出すことを問題としません。

## v1.0.0 — Stable contractの完了条件

### 証拠の正しさ

- core live PASSの意味が明確
- PASSには対象実クライアントの証拠が必須
- protocol世代差を扱っても、観測していない情報を既知扱いしない
- exact client versionと関係するplatform/runtime/auth contextを保持
- 曖昧な証拠はfail-closedまたは`unknown`

### Stable real-client adapter

stableとする各アダプターで:

- 通常ユーザー設定・credentialからの隔離が文書化・テスト済み
- version取得がbounded / deterministic
- timeout / cancellation / process cleanupがbounded
- fixtureが測定経路を証明
- 現実的なversion / platform範囲の証拠がある
- client変更時にfalse PASSではなく保守的なfailure / unknownになる

対応クライアント数そのものはv1条件にしません。

### 退行テスト運用

次を一連で利用できること。

- portable versioned artifact
- suite / multi-client execution
- baseline比較
- intentional baseline lifecycle
- evidence-derived compatibility report
- CI regression gate
- retry / flake semantics
- self-hosted real-client向けtrusted execution policy

### 公開契約の安定性

次を維持する、または意図的にversion / migrateできる状態にする。

- CLI behavior
- primary JSON contract
- artifact schema
- exit code
- stable reason code
- adapter ID / maturity semantics
- core / capability profile

### Security / privacy

- 秘密情報をoutput / artifactから拒否・マスクする
- 通常ユーザーcredentialをコピーしてテストを通さない
- OAuth authorization materialを安全な経路外へ保存・露出しない
- deployment identity privacyの共有モデルがある
- untrusted PRからself-hosted runnerを任意実行面として悪用できない
- cleanup対象は今回のtest sessionが所有するものだけ
- release provenance / security gateを維持

### v1でも非目標

- 公式MCP Conformanceの代替
- security certification / scanner
- LLM tool-selection benchmark
- 壊れやすいGUI/DOM automation framework
- hosted SaaS必須化
- すべてのMCP機能・すべてのクライアントが動くという保証
- 任意のproduction toolを安全に実行できるという保証

## Minor releaseへ入れる前の判断

新しいcapabilityを次のminor releaseへ入れる前に確認します。

1. deployment固有のreal-client evidenceの信頼性または運用価値を上げるか
2. real-client-only PASSを維持できるか
3. 通常ユーザーcredentialのコピーやunsafe automationなしに実装できるか
4. 公式Conformance、Inspector、security tool、model benchmarkではなく、このprojectに属するか
5. 観測経路は実装できる程度に安定しているか、それともまだresearchか

答えが弱いものは、ロードマップに空きがあっても延期します。

## 参照しているprotocol変更

protocol-aware milestoneは公式MCP `2026-07-28` release/schemaを参照しています。

- [`2026-07-28` specification release](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/blog/content/posts/2026-07-28-spec-ga/index.md)
- [`2026-07-28` schema](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/schema/2026-07-28/schema.ts)

従来のinitialization handshakeを永続的な相互運用stageとして固定しない理由は、このprotocol変更にあります。
