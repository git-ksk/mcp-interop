# Project direction

[English](project-direction.md) | [日本語](project-direction.ja.md)

この文書は`mcp-interop`のproduct directionを定義するcanonical documentです。MCP Conformance、Inspector、evaluation platform、client capability matrixなど周辺ecosystemが拡大しても、projectの焦点を維持するために使います。

公式MCP Conformanceとの詳細なcategory比較は[Conformance vs. mcp-interop](conformance-vs-interop.ja.md)を参照してください。この文書が決めるのは別の問いです。**このprojectは次に何を作るべきか、何にはならないか、そしてどのevidence品質を絶対に崩さないか。**

## Mission

`mcp-interop`は、次の具体的なinterop tupleに対するrepeatable regression-testing layerを目指します。

```text
Remote MCP deployment
  × real shipping client product
  × exact client version
  × relevant platform/runtime context
```

主に答える問いは次です。

> 実際にdeployしているRemote MCP endpointは、usersが使う実client productとその正確なversionで、real MCP pathを通して今も動くか？

長期的な価値はclient名の数ではありません。特定deployment/client pairingが動いた、失敗した、またはregressionしたことを、信頼でき再現可能なevidenceで示すことです。

## Core product contract

live interoperability resultは、その結果を生成したexecutionにscopeします。観測したendpoint、client product/version、platform/runtime context、auth mode、そのrunで得られたevidenceを超えて一般化してはいけません。

現在ship済みのcore live stageは次です。

```text
reach -> auth -> init -> tools
```

これはexisting output/evidence contractであり、将来のすべてのMCP revisionにwire-level initialization phaseが存在するという仮定ではありません。[roadmap](roadmap.ja.md)では、MCP evolutionに合わせてinteroperabilityのsemantic meaningを維持するprotocol-aware workを定義します。

complete live PASSには、observed/supported pathで必要な全semantic stageがreal-client evidenceによって`pass`であることを要求します。`unknown`、`skip`、metadata compatibility、fixture-only success、sanitized runtime observationをdeployment-specific live PASSへ昇格させてはいけません。

便利なfalse-positiveより、conservativeなfalse-negativeまたは`unknown`を選びます。

## Evidence hierarchy

implementation、output、documentation、将来schemaのすべてで次のlayerを分離します。

1. specification / conformance evidence
2. direct server inspection / debugging
3. product-profile preflight metadata
4. deploymentからのsanitized Runtime Evidence
5. measurement pathを証明するdeterministic adapter / fixture evidence
6. **real shipping clientから得たlive deployment-specific evidence**

対象deploymentが対象client/versionでPASSしたことを証明できるのは6だけです。5が証明するのはadapterが主張どおり測定できることまでで、別のproduction deploymentがPASSしたことではありません。

## Real-client boundaryとして認めるもの

supported public automation surfaceが存在する場合、live adapterはそれを優先します。十分にstableで再現可能かつ制約を文書化できる場合にはobserved product surfaceも利用できますが、client数を増やすためにPASS semanticsを弱めてはいけません。

許容できる例:

- documented CLI MCP management command
- documented local app-server / control protocol
- MCP lifecycle/tool discoveryを直接反映するisolated client-owned state
- より安全なsupported machine interfaceがない場合のactual installed clientへのcontrolled PTY interaction。ただしpathがdeterministicで、evidence解釈がconservativeであること

core PASS oracleとして許容しないもの:

- model promptやLLMのtool選択/tool call
- brittle browser DOM automation
- supported boundaryがないという理由だけで使うprivate/minified product endpointやinternal UI command identifier
- testを通すための通常user authentication credentialのcopy/reuse
- live discovery evidenceなしにconfig presence、enabled state、allowlistをdiscovered toolsとして扱うこと
- server metadataやdirect server inspectionをclient-observed lifecycle evidenceの代わりに使うこと

このboundaryを満たせないproductはresearch-onlyに維持するか、`unknown`/`skip`を返し、PASSの意味を変更しません。

## Adapter lifecycle / graduation

client supportには明示的なmaturity stateを持たせます。

### Research-only

safe direct boundaryが存在するか調査中の状態です。partial stageを証明できても、不完全なevidenceをsupported live adapterとして扱いません。

required stageがunsafe credential reuse、model prompt、brittle UI scraping、private internals、未証明のlifecycle/tool-discovery pathに依存する間はresearch-onlyです。

### Beta

real-client pathが有用で再現可能でも、version/OS coverageやobserved product surfaceがまだ狭い場合はbetaとしてshipできます。

beta前にsupported pathについて最低限すべて必要です。

- isolated config / credential state
- exact client version capture
- relevant platform/runtime context capture
- no-model core execution
- deterministicなreach/auth/init/tools interpretation
- ambiguous evidenceに対するconservativeな`unknown`
- bounded timeout / cancellation
- owned process/session cleanup
- secret rejection / redaction
- claimed real-client pathを証明するcontrolled fixture E2E
- practicalな範囲でnormal user stateが変わっていないことを示すbefore/after check

### Stable

measurement boundaryがone-version accidentではないことを示せるだけのrealistic client version/platform evidenceを得てからbetaを昇格させます。

Stableはexternal clientが将来変わらないという意味ではありません。documented compatibility envelope、release gate、failure semantics、client変更時のmaintenance pathを持っていることを意味します。

## Priority order

roadmap項目が競合する場合、次の順で優先します。

### P0 — Evidence correctnessを守る

最優先:

- false live PASSを出さない
- stage/result invariant
- secret safety
- process/session cleanup
- deterministic timeout/cancellation
- reproducible fixture gate
- CI/release provenanceとregression coverage

adapter数が増えてもこれらを弱める変更はmergeしません。

### P1 — Regressionをfirst-classにする

次のmajor product layerでは、version間/run間の差分を容易に検出できるようにします。

目標capability:

- versioned interoperability result/evidence schema
- endpoint identity/safe fingerprint、client product、exact version、platform/runtime、auth mode、stage resultを含むstable run identity
- optional secret-free evidence bundle export
- 2 run間のcompare/diff output
- `PASS -> AUTH FAIL`や`TOOLS PASS -> UNKNOWN`などのCI-friendly regression gate
- aggregate PASS/FAILだけでなくmachine-readableなreason change
- core projectがhosted backendを必要としないlocal-file-first history

目標workflow例:

```text
production endpoint + Cursor 2026.08.04 -> PASS
production endpoint + Cursor 2026.08.11 -> AUTH FAIL
                                      ^ regression
```

将来hosted/dashboard layerがこれらartifactをconsumeすることはできますが、core value propositionに必須ではありません。

### P2 — Existing adapterをversion/platform横断で強化

client追加を急ぐ前に、既存shipping adapterのconfidenceを高めます。

- feasibleなら複数のrecent client versionをtest
- client自体がsupportする場合はOS/platform coverageを拡大
- known-good / known-bad compatibility envelopeを記録
- output/control-surface変化をconservativeに検出
- 各supported platformでfixture/cleanup gateを維持

PASS semanticsが異なる複数adapterより、深く信頼できる1 adapterの方が価値があります。

### P3 — New real clientを選択的に追加

credible automation/evidence boundaryを持つclientだけを追加します。人気だけでは十分ではありません。

candidate評価では次を確認します。

1. normal user stateをisolateできるか
2. model promptなしでreal clientにtargetへ到達させられるか
3. authを安全にcomplete/classifyできるか
4. initializationを直接観測、またはdocumented product surfaceからconservativeに判断できるか
5. actual tool discoveryを証明できるか
6. exact version/platform contextを取得できるか
7. sessionをdeterministicにcleanupできるか
8. controlled fixtureでmeasurement pathを証明できるか

required itemが満たせなければresearchに維持します。

### P4 — Live failureを説明するproduct-specific diagnosticsだけを追加

`diagnose --profile <product>`は、documented compatibility patternが実際のdeployment/client mismatchを説明する場合に有用です。

diagnosticsを第二のgeneric OAuth/MCP conformance suiteへ拡張しません。specification questionには公式MCP Conformanceを優先します。

## Competitive boundary

隣接toolとは意図的に補完関係を取り、重複を避けます。

### Official MCP Conformance

specification correctnessとconformance scenarioを担当します。独自certification layerを作るためだけにgeneric protocol requirementを再実装しません。

### Inspector / MCP testing・evaluation platform

interactive server debugging、emulation、playground、model evaluation、広いdeveloper UXを担当します。`mcp-interop`がoverlapするのはdeployment-specific interoperability questionに答えるためdirect real-client executionが必要な部分だけです。

### Static client capability matrix

「client Xは一般にどのfeatureをsupportするか」を担当します。何をtestするか決める入力にはなりますが、static capability dataはdeploymentに対するlive evidenceではありません。

### Security scanner / governance product

vulnerability scanning、policy、permission、sandboxing、security certificationを担当します。connectivity PASSをsecurity resultとして扱いません。

### LLM/tool-selection benchmark

modelがtoolを適切に選択・利用できるかを担当します。model behaviorはdownstreamの別test layerになり得ますが、core interoperability PASSの前提にはしません。

## Ecosystemが変化したときの再評価条件

この方向性が有効なのは、projectがdeployment-specific real-client evidenceというdistinct boundaryを持つ間です。

major adjacent projectが次をfirst-classかつreproducible workflowとしてすべて提供し始めた場合、positioningを再評価します。

- emulate/configureだけでなくactual released client productを実行する
- arbitrary user-supplied Remote MCP deploymentをtargetにできる
- client config/credentialをisolateする
- exact client version/platform contextを記録する
- model promptに依存せずreach/auth/init/tool discoveryを証明する
- regression compareに利用できるmachine-readable per-run evidenceをexportする

その場合、同じlayerを維持する前にmaintenance cost、adoption、evidence quality、extension pointを比較します。重複を守ること自体を目的にせず、より強いupstream projectとのintegration/contributionを選ぶ可能性もあります。

## Decisions under pressure

growthとevidence qualityが衝突した場合、次の順で判断します。

1. PASSの意味を守る
2. user credential/local stateを守る
3. reproducibility/cleanupを守る
4. exact client/version/platform provenanceを守る
5. regression detectionを改善する
6. version/OS coverageを広げる
7. clientを追加する
8. convenience UXを追加する

この順序は意図的です。green resultを得やすくする代わりに信頼性が落ちれば、`mcp-interop`は存在理由を失います。

## Current strategic direction

milestone-level planは[Stable interoperability contractに向けたRoadmap](roadmap.ja.md)で管理します。この文書はより上位のproduct-direction contractであり、roadmapはwork sequenceを定義しても、ここで定義するevidence/safety priorityをoverrideしません。

near-term workは次に集中します。

1. existing real-client-only PASS semanticsを弱めず、core interoperability meaningをprotocol-era-awareにする
2. portable artifactが通常のregression/baseline inputになる前にdeployment privacyを維持できる設計を固める
3. ship済みartifact/compare primitiveをsafeでrepeatableなregression workflowへ発展させる
4. Codex/Cursor/Antigravityをobserved client versionとsupported platform横断で強化する
5. ChatGPT、VS Code、GitHub Copilot CLIなどはdirect automation boundaryがgraduation criteriaを満たすまでresearch-onlyに維持する
6. public contractはreal regression workflowで十分にexerciseしてからstabilizeする

目指すのは「最大のMCP client list」ではありません。**real Remote MCP deploymentが、usersが実際に使うclient versionで今も動くかを最も信頼できる形で答えるproject**です。
