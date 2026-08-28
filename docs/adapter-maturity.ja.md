# Adapter maturity contract

[English](adapter-maturity.md) | **日本語**

Adapter maturityは**evidence reviewに基づくlifecycle decision**です。既存clientの`tier` fieldとは分離します。`tier=v1`はshipped/live-adapter delivery trackに属するという意味で、evidence maturityが`stable`という意味ではありません。

clientを検出・実行せず、現在のmachine/human-readable decisionを確認できます。

```console
mcp-interop maturity
mcp-interop maturity --json
```

JSON reportは`schema_version: 1`、`artifact_type: "mcp-interop/adapter-maturity"`です。installed client version、executable path、endpoint、credential、user-state dataは含みません。

## report contract

各decisionは次を持ちます。

- `client_id` / `display_name` — shipped adapter identity
- `tier` — 既存roadmap/delivery tier。maturityとは意図的に分離
- `maturity` — `research_only` / `beta` / `stable`
- `criteria` — known criterion全件と`met` / `limited` / `missing` status
- `blockers` — beta decisionで`met`ではないstable criterion全件。隠してはいけない
- `evidence_refs` — `docs/...`や`github:pr/<number>`のようなretained repository evidence reference。参照文字列だけで、command自体はfetchしない

unknown criterion、重複evidence reference、未記載beta blocker、non-`met` criterionを持つ`stable` decisionはvalidationでrejectします。このreportはreview済みproject decisionでありlive probeではありません。

## state

- `research_only` — safe direct measurement boundaryが、live adapterとしてshipできる水準まで完成していない
- `beta` — supported pathがminimum evidence/safety gateを満たすが、stable criteriaに`limited` / `missing`が残る
- `stable` — advertised scopeについて、下記beta/stable criteriaをすべて明示的に満たす

project age、人気、release回数、近いclient versionはmaturity evidenceにしません。

## Beta gate

beta adapterは次のcriteriaがすべて`met`でなければなりません。

| Criterion ID | Requirement |
| --- | --- |
| `direct_real_client_boundary` | PASS semanticsがreal clientのdirect supported/observed boundary由来であり、model promptやserver-only substituteに依存しない |
| `isolated_state` | test config/auth stateをnormal user stateから隔離する |
| `owned_cleanup` | testが所有するprocess/session/temp stateをboundedにcleanupする |
| `secret_safety` | secret-bearing valueをreject/redactし、interop evidenceへ保持しない |
| `conservative_failure_semantics` | ambiguous evidenceを`unknown`/`skip`のままにし、coverage目的でPASSを広げない |
| `controlled_fixture_e2e` | repeatable controlled real-client fixture gateでclaimed core pathを証明する |
| `exact_version_platform_evidence` | real-client evidenceにexact client versionとrunner platform contextを保持する |

beta criterionのどれかが未証明になれば、explicit reviewが完了するまで強いmaturity claimを止めます。ただしclient version文字列が変わっただけでは未証明になったと判断しません。

## Stable gate

stableには全beta criterionに加え、次もすべて`met`が必要です。

| Criterion ID | Requirement |
| --- | --- |
| `repeat_path_version_coverage` | PASS claimを構成する各advertised pathについて、少なくとも2つのexact client versionでretained real-client evidenceがある。version rangeへ補間しない |
| `advertised_platform_coverage` | supportedとadvertiseする各OS/architecture scopeにexact real-client evidenceがあるか、advertised scopeを明示的に狭める。unit/hosted build成功だけをreal-client platform evidenceにしない |
| `measurement_surface_stability` | clientのobservation/control surfaceが十分supported、または繰り返し実測され、PASS boundaryが1 buildだけの偶然ではない |
| `regression_maintenance_path` | exact-point compatibility、regression gate、failure semantics、external client変更時のrevalidation pathがある |

`limited`は関連evidenceはあるがstable promotionには不足、`missing`は必要evidenceが無い状態です。どちらも`stable`をblockします。

stableでもsemantic-version support rangeは作りません。stable adapterでもinstalled clientのexact versionが未観測なら`untested`です。

## 現在のshipped adapter decision

v0.9 reviewでは3つのshipped adapterをすべて**beta**に維持します。これは保守的なadapter-level maturity classificationであり、regression resultでもversion changeによる自動降格でもありません。

| Adapter | Decision | betaを支えるevidence | Stable blocker |
| --- | --- | --- | --- |
| Codex CLI | `beta` | isolated app-server real-client boundary、exact Codex CLI `0.133.0`（PR #108）のretained controlled core evidence、cleanup/secret/failure gate | `repeat_path_version_coverage`、`advertised_platform_coverage`: stable gate向けにrepositoryが保持するcurrent core PASS pointは1 exact versionで、real-client coverageもmacOS arm64中心 |
| Cursor CLI | `beta` | isolated supported MCP management/login path、exact `2026.08.04-aaa8809`のOAuth evidence（PR #39）、exact `2026.08.25-3e8eec8`のcontrolled core evidence（PR #108） | `repeat_path_version_coverage`、`advertised_platform_coverage`、`measurement_surface_stability`: 2 versionが別path mode、real-client OS evidenceが狭い、MCP management outputが専用machine contractではなくhuman-readable |
| Antigravity CLI | `beta` | isolated PTY/no-account boundary、exact `agy 1.1.11`のOAuth evidence（PR #40）、exact `agy 1.1.22`のcontrolled core evidence（PR #108） | `repeat_path_version_coverage`、`advertised_platform_coverage`、`measurement_surface_stability`: OAuth/coreのversionが異なる、retained macOS evidenceはarm64のみ、no-auth tool discoveryがboundedなobserved client cache surfaceに依存 |

現在のexact coverage tableは[Exact observed client coverage](observed-coverage.ja.md)で管理します。historical PR evidenceは、そのPRが明示したexact path/version claimだけに使い、近いversionをtestedへ広げません。

## client version変更時

maturityとexact compatibilityは分離してreviewします。

1. installed client version変更だけではadapterをpromote/demote/regressしない
2. `compatibility query` / `compatibility matrix`で新しいexact pointをretained evidenceから分類する
3. 新versionが未観測ならadapterがbeta/stableでも`untested`
4. 新しいretained failure、cleanup/isolation failure、繰り返すevidence gapはexplicit maturity reviewのtriggerになり得る
5. maturity変更はevidence rationale更新を含むreview済みproject decisionとしてcommitし、SemVer順から推測しない

VS Code、GitHub Copilot CLI、ChatGPT、Claudeなどresearch candidateは`mcp-interop maturity`へ出しません。このcommandは**shipped live adapter**だけをreportします。detectやpartial PoCが存在するだけではresearchからbetaへ昇格しません。
