# Stable interoperability contractに向けたRoadmap

[English](roadmap.md) | [日本語](roadmap.ja.md)

このroadmapは、[Project direction](project-direction.ja.md)のproduct principleをmilestone、exit criteria、明示的non-goalへ落とし込む文書です。

これはplanning contractであり、日付のcommitmentではありません。minor version番号はmaturity workの想定順序を表しますが、未完成featureの出荷やevidence boundaryの緩和を強制しません。特に`v1.0.0`を`v0.9.0`や`v0.10.0`の次releaseとして自動予約しません。必要なら`v0.11.0`、`v0.12.0`、それ以降の`v0.x`を継続します。`v1.0.0`へ進むのは、この文書のstable-contract exit criteriaを満たした時だけです。

## 他project documentとの関係

- [Project direction](project-direction.ja.md)はmission、evidence hierarchy、competitive boundary、priority orderを定義します。
- このroadmapはmaturityの想定順序とmilestone完了に必要なevidenceを定義します。
- [Architecture](architecture.ja.md)は現在のimplementation/trust boundaryを説明します。
- [Live result artifact schema v1](live-result-schema-v1.ja.md)は現在ship済みのportable result formatを定義します。このroadmapだけでschemaを黙って変更しません。
- [Conformance vs. mcp-interop](conformance-vs-interop.ja.md)はofficial MCP Conformanceとの境界を定義します。
- [CONTRIBUTING](../CONTRIBUTING.ja.md)はroadmap/contract変更のproposal/review方法を定義します。

文書ごとの目的が異なる場合、roadmap上のaspirationをship済みbehaviorとして扱わないでください。公開versionが実際に何をするかはcurrent code、released schema、release documentationがsource of truthです。

## Roadmap invariants

すべてのmilestoneで次を維持します。

1. **feature数よりevidence correctnessを優先する。** green resultを得やすくする代わりにPASSの意味を弱めない。
2. **live PASSはreal-client-only。** fixture、configuration、metadata、direct-server inspection、diagnostic evidenceをtest対象real shipping clientのlive evidenceへ代用しない。
3. **unknownはunknownのまま。** compatibility matrixを埋めるためにmissing/ambiguous evidenceをsuccessへ推論しない。
4. **safe graduation。** client/capabilityはsupported、またはdeliberately acceptedなdirect automation/evidence boundaryを証明してからgraduationする。version milestoneを理由にcriteriaを弱めない。
5. **backward compatibilityはevidenceで判断する。** existing public contractは可能な範囲で維持し、current modelで具体的requirementを表せないと証明してからnew schema/incompatible contractを導入する。
6. **local-first core。** core valueを得るためにhosted backend、account、dashboard、SaaSを必須にしない。
7. **normal-user credentialを再利用しない。** adapterをpassさせるだけのためにnormal browser/client credential、token、Keychain entry、persistent user stateをcopyしない。

## Core interoperability profile

capability数を増やす前に、core live PASSのminimum claimを明示します。

想定するcore profileは **Remote Tool Interoperability** です。

> real clientがtarget Remote MCP deploymentへ到達し、必要なauthentication boundaryを満たし、観測したprotocol eraで利用可能なMCP protocol pathを成立させ、そのclient pathから実tool-inventory evidenceを取得できる。

core profileではproductionの任意tool実行を必須にしません。tool callにはside effectがあり得るため、discoveryを証明する目的だけでgeneric PASS prerequisiteにしてはいけません。

Resources、Prompts、Tasks、Multi Round-Trip Requests (MRTR)、safe controlled-fixture tool call、将来のMCP extensionは別capability profileとして追加できます。新profile追加でcore profileの意味を黙って広げません。

現在ship済みのpublic result modelは次です。

```text
reach -> auth -> init -> tools
```

これはexisting compatibility contractですが、`initialize`が将来のすべてのMCP revisionで永続するwire-level phaseだという宣言ではありません。protocol-aware normalizationではold outputの意味を守りながら、new protocol eraへfalse assumptionを持ち込まないようにします。

## Protocol-era policy

MCP protocolは進化しています。official `2026-07-28` revisionではlegacyな`initialize` / `notifications/initialized` handshakeとprotocol-level session modelが廃止されています。そのrevisionのrequestはself-describingで、serverは`server/discover`を実装しますが、clientのdiscovery利用自体は必須ではありません。

そのため`mcp-interop`は**observed evidence**と**normalized interoperability meaning**を分離します。

```text
real client execution
  -> observed protocol/client evidence, when available
  -> protocol-era-specific interpretation
  -> normalized core interoperability verdict
```

protocol version/eraは推測値ではなくevidenceです。controlled fixtureではnegotiated protocol revisionを直接観測できても、production real-client surfaceでは見えない場合があります。その場合protocol revisionは`unknown`のまま維持します。fixtureで観測したprotocol情報を、production clientが証明したかのように別runへ転記してはいけません。

将来`protocol_ready`のようなinternal semantic stateでlegacy initializationとmodern request readinessをnormalizeする可能性があります。ただしexisting public `init` fieldの置換・再解釈には明示的compatibility designとmigration planが必要です。

### Remote transport boundary

core projectは引き続き**Remote MCP deployment**へ集中します。

- Streamable HTTPをprimary modern remote transportとする。
- real shipping clientが利用している間はlegacy remote HTTP/SSE behaviorも観測対象にできる。
- すべてのhistorical transportをsupportすること自体は目的にしない。
- `stdio` interoperabilityはproject directionを明示的に変更しない限りdeployment-specific Remote MCP core scope外とする。

## v0.6.x — Protocol-aware coreとdeployment privacy

### Goal

existing live-PASS invariantを弱めず、legacy/modern MCP protocol eraをまたいでcore evidence modelを正しくする。

### Required work

- Codex、Cursor、Antigravityのcurrent real-client pathを再観測し、supported/accepted automation surfaceから実際に取得可能なprotocol-era/version evidenceを記録する。
- core Remote Tool Interoperability profile向けprotocol-era-aware interpretationを定義する。
- production real-client pathがnegotiated protocol revisionを証明できない場合は`unknown`を維持する。
- existing public `init` stageを、より正確なinternal semantic modelのcompatibility projectionとして維持できるか判断する。
- controlled fixtureでlegacy/modern protocol behaviorをtestし、可能な範囲でfallback/unsupported-era behaviorも検証する。
- protocol-aware変更中もexisting isolation、timeout、cancellation、process cleanup、state cleanup、secret-redaction gateを再実行する。
- portable artifactを通常のbaseline/CI inputにする前にdeployment-identity privacy boundaryを定義する。
- concreteなprotocol-aware requirementを安全に表せないと証明されるまではportable artifact schema v1を維持する。new schema versionは不足を実証してから導入する。

### Deployment identity privacy

current portable artifactはraw query valueやcredential-bearing URL materialを意図的に除外します。これはartifactをcredential-safeにしますが、secret-safeなhostname/path自体がoperationally privateな場合があります。

```text
credential-safe != deployment-public
```

baseline artifactを通常commit/shareする前に、`production-a`のようなopaque user-defined target identity、またはprivate deployment hostname/pathを公開せずpairing可能な同等mechanismを検討します。candidate deployment identityを推測して照合できる場合があるため、deterministic hashだけを十分なprivacy対策とは扱いません。

### Exit criteria

`v0.6.x`完了条件:

- projectがcoverするlegacy/modern protocol behaviorでlifecycle-model assumptionがfalse live PASSを生成しない。
- 観測できないprotocol情報は明示的にunknownのまま残る。
- existing 3 adapterにactual observable surfaceへ適したprotocol-aware controlled-fixture coverageがある。
- later baseline workflowに利用できるdeployment identityのdocumented secret/privacy modelがある。
- artifact schema変更がある場合、実証済みneedと明示的compatibility/migration rationaleがある。

### Non-goals

- new client graduationは必須ではない。
- hosted history serviceは作らない。
- generic MCP conformance replacementにはしない。
- production tool callを必須にしない。

## v0.7.x — Repeatable regression workflow

### Goal

ship済みartifact/compare primitiveを、repositoryごとのcustom glueなしで複数real clientを実行し、comparable evidenceとdeterministic CI decisionを生成できるoperational workflowへする。

### Required work

- target、client、authentication mode、allowed execution contextを選択するsecret-safe suite/manifest modelを定義する。
- 複数selected clientを実行し、coherentなportable artifact setを生成する。
- hand-written support matrixではなくevidenceからcompatibility/regression reportを生成する。
- stage/reason-code change、client-version change、protocol-evidence change、missing evidenceをmachine-readable outputで保持する。
- retry/flake semanticsを定義する。retryでfirst failureを消してはいけない。mixed attemptは黙ってPASSへ変換せずunstable/ambiguous evidenceとして表現する。
- real-client executionのCI trust boundaryを定義・enforceする。

### CI trust boundary

untrusted pull-request contentからself-hosted real-client runnerをarbitrary network/credential execution surfaceとして利用できないようにします。

Default policy:

- ordinary/untrusted PR CIはhosted runner + controlled localhost fixtureだけを使う。
- self-hosted real-client executionはtrusted branch、explicit manual dispatch、または同等のdeliberately approved execution pathに限定する。
- untrusted PRのsuite/manifestからprivileged runnerをprivate hostやproduction-equivalent credential stateへredirectできないようにする。
- OAuthは引き続きexplicit opt-inで、existing isolation/secret-safety guaranteeを維持する。

### Exit criteria

`v0.7.x`はdeclared test suiteからreal-client artifact、comparison、evidence-derived compatibility reporting、deterministic CI exit decisionまで一周でき、同時にCI trust boundaryを維持できた時に完了します。

## v0.8.x — Baselineとcompatibility envelope

### Goal

client auto-updateや複数tested client version/platformをまたぐ継続regression testを運用可能にする。

### Required work

- baseline lifecycle: create/select、compare、intentional accept、retire/supersedeを定義する。
- accidental baseline replacementでregressionが隠れないようにする。
- confidenceへ影響するstale/missing baseline evidenceを検出する。
- 推測したcontinuous version rangeではなく**observed tested point**でadapter compatibility envelopeを定義する。
- decisionに有用な範囲で`tested`、`untested`、`stale`、`known-broken`を区別する。
- `mcp-interop clients`または同等machine-readable commandからadapter/client compatibility情報を表示し、untested versionをcompatibleと主張しない。
- exact client version、platform/architecture、auth mode、test time/context、relevant evidence provenanceをcompatibility reportへ保持する。

Example principle:

```text
Tested:
  Cursor X on macOS arm64 -> PASS
  Cursor Y on macOS arm64 -> PASS

Does not imply:
  XからYまでの全versionがsupported
```

### Exit criteria

`v0.8.x`はbaseline changeがintentional/auditableであり、client auto-updateをsupport rangeを捏造せずtested、untested、stale、regressedとして分類できる時に完了します。

## v0.9.x — Coverage、capability profile、safe graduation

### Goal

existing adapterのconfidenceを深め、十分強いevidence boundaryを持つproduct/capabilityだけをgraduationする。

### Priority order

1. Codex、Cursor、Antigravityをrealisticなcurrent client version横断で強化する。
2. real client自体がsupportしsafe execution可能な範囲でOS/platform coverageを広げる。
3. project ageやproduct popularityではなくobserved evidenceからbeta-to-stable promotionを判断する。
4. Resources、Prompts、Tasks、MRTRなどはPASS claimをpreciseに定義できる場合だけcapability profileとして追加する。
5. established adapter criteriaを満たしたnew real clientだけgraduationする。

GitHub Copilot CLI、VS Code、ChatGPTなどcurrent research candidateは、supported/accepted direct boundaryで必要なreal-client evidenceを安全に証明できるまでresearch-onlyを維持します。

**new clientが0件でも**evidence quality、tested coverage、adapter maturityがmaterialに改善すれば`v0.9.x`は成功です。

### Exit criteria

existing adapterにdocumented tested envelopeとmaturity statusがある。new client/capabilityを追加した場合も、弱いspecial caseではなく同じevidence/isolation/cleanup standardを満たす。

## v0.10.x — Public contract candidate

### Goal

将来`v1.x`でstable維持を約束する可能性があるpublic contractを整理する。

### Required work

必要に応じて次をstabilizeします。

- CLI command/flag semantics
- primary JSON output compatibility
- portable artifact schema evolutionとcross-schema comparison/migration policy
- reason-code naming/compatibility policy
- exit-code contract
- adapter identity/maturity-state semantics
- core/capability-profile meanings
- protocol-era compatibility policy
- baseline/compatibility-report semantics
- deprecation/removal policy
- release/security/privacy/cleanup guarantees

portable artifactはevidence recordであり、defaultではcryptographic attestationではありません。将来provenance signing、attestation、tamper evidenceを追加する場合はtrust modelを明示します。v1 contractでもunsigned local artifactが「誰が実行したか」を証明すると暗黙に主張してはいけません。

### Exit criteria

public surfaceをreviewし、maintainerが**「これらの意味をv1.xで維持する、または必要時にdeliberately version/migrateする準備がある」**と合理的に言えること。

fundamentalなCLI、schema、evidence-model redesignがまだ起きそうなら`v0.x`を継続します。

## v0.11.x以降 — Stabilization buffer

`v0.11.0`以降のpre-1.0 minor releaseへ特定feature setを予約しません。

real useで見つかった次のような問題を追加`v0.x` milestoneで解消します。

- 想定外のmodern/legacy client difference
- artifact schema migration gap
- cross-platform lifecycle/cleanup problem
- baseline/manifest usability problem
- new OAuth behavior
- real-client automation surfaceの変更/消失
- evidence-model adaptationが必要なprotocol revision/extension

prematureに`v1.0.0`へ進むより、`v0.12.0`、`v0.13.0`以降を出すことを問題としません。

## v1.0.0 — Stable-contract exit criteria

`v1.0.0`は次のcategoryをすべて満たした時だけreleaseします。

### Evidence correctness

- core live PASSの意味がprecise/documentedである。
- PASSにはtest対象real clientのevidenceが必要で、fixture/configuration/metadata/diagnostic evidenceをdeployment-specific real-client evidenceの代替にしない。
- protocol-era differenceをnormalizeしつつ、unobserved protocol detailをknown扱いしない。
- client surfaceから安全に観測できる範囲でexact client versionとrelevant platform/runtime/auth contextを保持する。
- ambiguous evidenceはfail-closedまたは`unknown`を維持する。

### Stable real-client adapters

stable宣言する各adapterについて:

- normal client configuration/credential stateからのisolationがdocumented/tested。
- exact version captureがbounded/deterministic。
- timeout/cancellation/owned-process cleanupがbounded。
- controlled fixtureがclaimed measurement pathを証明する。
- documented envelopeを正当化できるrealistic version/platform横断のcompatibility evidenceがある。
- client surface change時にfalse PASSではなくconservativeなfailure/unknownとなる。

supported client数そのものは**v1 exit criterionにしません**。

### Regression operation

次のcoherent local-first pathを提供する。

- portable versioned artifact
- suite/multi-client execution
- run/baseline comparison
- intentional baseline lifecycle
- evidence-derived compatibility report/matrix
- CI regression gate
- explicit retry/flake semantics
- self-hosted real-client run向けtrusted execution policy

### Public stability

次をpreserve、またはdeliberately version/migrateする準備がある。

- CLI behavior
- primary JSON contract
- artifact schema
- exit code
- stable reason code
- adapter ID/maturity semantics
- core/capability-profile meanings

### Security and privacy

- secret-bearing valueをoutput/portable evidenceからreject/redactする。
- normal-user credentialをtest profileへcopyしてpassさせない。
- OAuth authorization materialをexplicit safe flow外へpersist/exposeしない。
- deployment identity privacyにdocumented sharing/baseline modelがある。
- self-hosted CIをuntrusted changeからarbitrary privileged executionへ誘導できない。
- cleanupはtest sessionが所有するtemporary state/processだけを対象にする。
- release provenance/security gateを維持する。

### Scope boundary

v1でも次をclaimしません。

- official MCP Conformanceのreplacement
- security certification/scanner
- LLM tool-selection benchmark
- brittle GUI/DOM automation framework
- hosted SaaS requirement
- every MCP feature/every MCP clientが動くという証明
- arbitrary production toolを安全に実行できるという証明

## Minor release前のdecision gate

planned capabilityを次minor releaseへ割り当てる前に確認します。

1. deployment-specific real-client evidenceのtrustworthinessまたはoperational usefulnessを改善するか。
2. real-client-only PASS boundaryを維持できるか。
3. normal-user credential copyやunsafe automation surfaceなしで実装できるか。
4. official MCP Conformance、Inspector、security tool、model benchmarkではなくこのprojectに属するか。
5. evidence surfaceを実際に観測し、実装できる程度にstableか。それともまだresearchか。

答えが弱いfeatureはroadmap milestoneに空きがあってもdeferします。

## Roadmapへ影響するprotocol reference

protocol-aware milestoneはofficial MCP `2026-07-28` release/schemaを参照しています。

- [`2026-07-28` specification release](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/blog/content/posts/2026-07-28-spec-ga/index.md)
- [`2026-07-28` schema](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/schema/2026-07-28/schema.ts)

legacy initialization handshakeをpermanent universal interoperability stageとしてroadmapへ固定しない理由は、このprotocol変更にあります。
