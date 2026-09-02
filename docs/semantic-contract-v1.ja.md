# Interoperability semantics v1

[English](semantic-contract-v1.md) | **日本語**

この文書はadapter identity、maturity、core PASS、optional capability、protocol-era normalizationのstable v1 meaningを定義します。

## Adapter identity

現在のshipped live-adapter IDは次です。

```text
codex
cursor
antigravity
```

stable `v1.x`では既存adapter IDをrenameせず、別productへ再利用せず、別measurement pathへ黙ってaliasしません。新adapter IDはcommon graduation gate通過後に追加できます。display name改善はidentity変更ではありません。

Detectionはshipped supportより広い範囲を扱います。research clientが`clients`に表示されてもlive adapterにはなりません。

## Tier / maturity / graduationは別概念

`tier`はroadmap/delivery placementで、evidence maturityではありません。

Maturity state:

- `research_only` — shipped evidence claimではない
- `beta` — beta evidence gateは満たすがstable criterionにlimited/missingが残る
- `stable` — advertised scopeについて全stable maturity criterionを満たす

project-levelのv1 contract stabilityとadapter maturityは独立です。mcp-interop本体がstable v1.xをreleaseしても、adapter maturityが`beta`のままshipできます。project v1 stabilityが固定するのはdocumented project contractであり、全adapterがstable gateを満たしたという意味ではありません。各adapterはadvertised scopeのretained evidenceに基づいて個別にpromoteします。

既存maturity-state名と意味をstableにします。criterion IDもstable machine identifierです。新criterionはconservativeに追加できますが、既存criterionを強いclaimが簡単になる意味へ再利用しません。

Research graduationは別gateです。`eligible_for_beta`は全mandatory graduation criterionを満たす意味で、自動shipではありません。shipにはimplementation/review、tier-v1 spec、valid maturity decisionが必要です。

## Core Remote Tool Interoperability PASS

public coreは次の4 stageを維持します。

```text
reach -> auth -> init -> tools
```

complete live PASSは4 stageがこの順序で全て`pass`の場合だけです。`fail` / `skip` / missing / `unknown`はnon-PASSです。

- `reach` — real clientがtarget Remote MCPへlive interactionしたdirect evidence
- `auth` — 必要なclient auth完了、またはtested pathで不要とdirect live evidenceにより証明
- `init` — real-client pathでprotocol readinessをdirectに成立。literal legacy `initialize` request観測を永久に意味しない
- `tools` — real clientがtarget serverのtool inventoryをdiscover

coreで`tools/call`は必須にしません。arbitrary production tool executionをgeneric PASS prerequisiteにしません。

diagnostic、config presence、registration success、metadata、fixture-only observation、model outputはmissing core stageの代替証拠になりません。

## Protocol-era normalization

MCP wire behaviorが変わってもpublic protocol-readiness meaningを維持できます。

`init=pass`は、tool-inventory evidence等、usable protocol readinessをdirectに証明するsupported real-client surfaceからprojectできます。controlled fixtureはadapter implementation/release gateを検証できますが、fixture-only evidenceからdeployment-specific `init=pass`は作れません。

real-client surfaceがnegotiated protocol revisionを出さない場合、deployment-specific revisionはunknownのままです。fixtureだけで観測したrevisionをproduction runへコピーしません。

modern probe、legacy fallback、将来protocol revisionはevidence detailです。internal observation modelは進化できますが、public four-stage PASS meaningを弱めません。

## Optional capability semantics

Capability profileはcore live resultと独立です。

- `pass` / `fail` / `unknown`はそのcapabilityについてdocumented direct real-client evidence surfaceが必要
- `unsupported`はexplicit adapter-policy boundaryが必要
- `untested`はexact contextでevidenceなしを意味し、evidence IDを持たない

capability PASSからcore PASSへupgradeせず、core PASSからResources / Prompts / Tasks / MRTR / controlled tool call等を推測しません。

## Exact-version / platform semantics

Compatibilityはexact observed pointのevidenceで、semantic-version rangeではありません。

- client-version変更だけではregressionにならない
- 未観測exact versionは`untested`
- installed version stringだけの変更でmaturityを自動promote/demoteしない
- portable live artifactのplatform fieldは、より強いclient-executable architecture evidenceが明示されない限りmcp-interop runner/processを表す
- wrapper/script launcherではhost architectureをclient-binary evidenceへ推測しない

## Semantic change policy

次の変更にはexplicit compatibility reviewが必要です。

- core 4 stage名/順序
- aggregate PASS条件
- 既存adapter ID
- maturity/graduation state meaning
- 既存capability state/evidence-kind meaning
- fixture/metadata/configuration evidenceがdeployment-specific PASSを作れるかどうか

evidence requirementを弱める変更はcompatible clarificationではありません。
