# Real-client adapter graduation gate

[English](adapter-graduation-gate.md) | **日本語**

Graduation gateは、research clientをshipped live-adapter setへ進めるための**単一のminimum evidence policy**です。人気、automationの難しさ、UI中心/cloud-hosted product、partial PoC済みなどを理由に弱い例外routeを作りません。

clientをdetect/executeせず、現在のreview済みcandidate stateを確認できます。

```console
mcp-interop graduation
mcp-interop graduation --json
```

JSON reportは:

```text
schema_version: 1
artifact_type: mcp-interop/adapter-graduation
```

です。review済みpolicy/evidence referenceだけを含みます。installed version、executable path、Remote MCP endpoint、credential、cookie、user-state dataは含まず、network accessも行いません。

## graduationの意味

このgateは **research -> beta実装/ship候補としてeligible** を判定します。adapterのbeta -> stable maturityとは別です。

```text
research candidate
  -> graduation criterion全件met
  -> eligible_for_beta
  -> 明示adapter implementation/review
  -> shipped beta maturity decision
  -> 将来stable maturity gate
```

`eligible_for_beta`だけで`mcp-interop test`へadapterを自動追加しません。shipには明示実装とvalid shipped maturity decisionが必要です。逆にparser branchやadapter switch caseだけ追加してもpolicyを迂回できません。live `--client` parserの許可IDはvalidated shipped maturity decisionから導出します。

現在のshipped setはCodex / Cursor / Antigravityです。client detection、diagnostics、PoC、このreportにcandidateが存在するだけではrunnableになりません。

## mandatory criterion

全criterionがmandatoryです。`limited` / `missing`はいずれもgraduationをblockします。

| Criterion ID | Requirement |
| --- | --- |
| `direct_real_client_boundary` | supportedまたはdeliberately acceptedなdirect boundaryでreal client自身からcomplete core pathを証明できる。config/registration presence、server metadata、model prompt、browser DOM scraping、private/minified internals、fixture-only observationは代替にならない |
| `isolated_state` | complete claimed pathでclient/config/auth stateをnormal userから隔離し、通常ユーザーcredentialのcopy/reuseへ依存しない |
| `owned_cleanup` | process/session/temp config/test-created stateにbounded ownershipとcleanup semanticsがある |
| `secret_safety` | credential、authorization code、cookie、token、protected endpoint path、raw secret-bearing client outputをinterop evidence/logへ保持しない |
| `conservative_failure_semantics` | missing/ambiguous evidenceをnon-PASSのまま保持し、config successやpartial lifecycle evidenceからcomplete `reach/auth/init/tools` PASSを作らない |
| `controlled_fixture_e2e` | proposed adapterがclaimするcomplete pathをrepeatable controlled real-client E2Eで証明し、core profileが必要とするdirect tool discoveryまで確認する |
| `exact_version_platform_evidence` | tested real-client exact versionとrunner/platform contextを保持する。近いversion/platformへ推測で広げない |
| `supported_platform_scope` | shipped予定OS/architecture scopeを明示し、安全にsupport/revalidateできる範囲へ限定する。1 platformだけの実測でもnarrow beta scopeにはできるが、generalizeせず明記する |

candidate固有のexception criterionはありません。unknown criterion追加はinvalidです。research-only decisionは全`limited` / `missing` criterionをblockerへ出し、隠せません。

## machine semantics

各candidate decisionは次を持ちます。

- `client_id` / `display_name`
- `research_issue`
- `status`: `research_only` / `eligible_for_beta`
- complete mandatory criterionから導出した`eligible`
- mandatory criterion全件と`met` / `limited` / `missing`
- non-`met` criterion全件の`blockers`
- retained repository `evidence_refs`

validation invariantは:

```text
全criteria met
  <=> eligible = true
  <=> status = eligible_for_beta
  <=> blockers empty
```

です。不完全criterionが1件でもあれば`research_only` / `eligible=false`とexact blockerが必須です。duplicate evidence reference、unknown criterion、hidden blocker、不整合eligible stateをrejectします。

## current v0.9 candidate review

現在common graduation gateを通る新real clientは**0件**です。**zero graduated clientsはv0.9の正常な成功結果**です。PASSを弱めず既存evidence品質/policy enforcementが改善すれば目的を満たします。

| Candidate | Issue | Current state | 主なblocker |
| --- | ---: | --- | --- |
| GitHub Copilot CLI | #48 | `research_only` | exact 1.0.80 no-input startupでreach/initializeまでは証明したが`tools/list`未証明。isolated authenticated account/session stateも未証明で、complete fixture/secret-auth boundary/shipped platform scopeが未完成 |
| VS Code | #6 | `research_only` | isolated registrationは証明済みだが、supported external CLIにdirect start/status/tool-discovery pathがなく、no-input isolated startupではfixture MCP requestが0件。complete E2E、cleanup/auth boundary、shipped platform scopeが未完成 |
| ChatGPT | #20 | `research_only` | supported custom-app flowは現在もproduct/UI-drivenで、core live adapterに使えるpublic isolated headless app-management/tool-scan contractがない。exact automatable client version/platform、cleanup、auth isolation、controlled E2Eが未証明 |
| Claude web/Desktop | #68 | `research_only` | Remote custom connectorはinterop targetとして有効だが、supported isolated automatable real-client control/tool-discovery boundaryが未証明。exact version/platform、cleanup、auth isolation、controlled E2Eが未証明 |

これはlinked research evidenceの要約で、compatibility claimではありません。近いversionやproduct docsからsupportを推測しません。

## shipped adapterとの接続

live-test parserは独立したhandwritten allowlistをpolicy sourceにしません。`ShippedLiveAdapterIDs`がvalidated shipped `MaturityDecision`から許可setを導出し、`TierV1` client specとcross-checkします。

そのため将来clientは次だけではshipできません。

- detection `Spec`追加
- research tierを`v1`へ変更
- `switch` branch追加
- partial PoCをdocument

valid shipped maturity decisionが無い`TierV1` specはcatalog policy errorになります。逆にmatching `TierV1` specが無いshipped maturity decisionもerrorです。adapter実装/tests自体は別途必要ですが、policy entry point同士を黙って不一致にできません。

これはfail-closedなpolicy wiringで、cryptographic approvalを主張せずhuman reviewの代替でもありません。

## capability profileとの関係

optional capability graduationは同じevidence philosophyに従いつつ、独立した[Capability profile v1](capability-profile-v1.ja.md)を使います。optional-capability PASSでmissing core real-client boundaryを補えず、core adapter graduationからResources/Prompts/Tasks/MRTR supportも推測しません。

## researchは継続できる

`research_only`はproduct自体のfailure verdictではありません。partial evidenceを保持し、安全に次のexperimentを改善できます。gateの役割はcomplete boundary前にpartial observationがshipped compatibility claimへ変わることを防ぐ点です。
