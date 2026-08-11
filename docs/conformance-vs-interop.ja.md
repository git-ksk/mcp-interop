# MCP Conformance と mcp-interop の違い

[English](conformance-vs-interop.md) | [日本語](conformance-vs-interop.ja.md)

`mcp-interop` は、公式の [MCP Conformance Test Framework](https://github.com/modelcontextprotocol/conformance) を置き換えるものではありません。両者は別の問いに答えるツールであり、**補完関係にあるテストレイヤー**として使うのが適切です。

## 中心となる違い

| | MCP Conformance | `mcp-interop` |
| --- | --- | --- |
| 主な問い | このMCP implementationは仕様どおりに動くか？ | このRemote MCP deploymentは、このclient product/versionで実際に動くか？ |
| 比較軸 | implementation × specification | deployment × client product × client version |
| 正解の基準 | MCP specification、scenario、expected check | 実際にインストールされたclientが観測した事実 |
| Client testの構成 | conformance test server → client under test | user指定Remote MCP ↔ 実client product |
| Server testの構成 | conformance client → server under test | 実client product群 → 同じRemote MCP deployment |
| 主な結果 | scenario/checkのconformance PASS/FAIL | `reach` / `auth` / `init` / `tools` のinterop evidence |
| 製品固有挙動 | conformance scenario違反時に問題になる | 結果そのものの重要な一部 |

違いは「synthetic testかreal softwareか」ではありません。公式Conformanceは**実client commandを起動でき、実server URLも直接テストできます**。違うのは、テストの構成と何を正解として判定するかです。

## 公式Conformanceが検証するもの

Client testでは、公式frameworkがscenario-controlledなMCP test serverを起動し、client-under-testのcommandを実行し、protocol interactionを収集してMCP仕様上のexpected behaviorと比較します。

Server testでは、指定されたserver URLへconformance clientとして接続し、scenarioを実行して仕様への適合性を判定します。

したがって、次のような問いは公式Conformanceの担当です。

- client/serverがMCP wire/lifecycle要件を満たしているか
- 特定のMCP specification revisionに準拠しているか
- OAuth/MCP authorization scenarioを仕様どおり実装しているか
- JSON-RPC messageやprotocol behaviorが仕様に適合しているか

`mcp-interop` は、これらのgeneric conformance checkを重複実装しません。

## mcp-interopが検証するもの

`mcp-interop test` は、ユーザーが実際に公開・利用しようとしている**Remote MCP deployment**を入力として始まります。

そのdeploymentを、各製品自身のMCP surfaceを使って実clientに登録し、同じserverが複数clientでどう見えるかを確認します。現在のadapterにはCodex CLI、Cursor CLI、Antigravity CLIがあります。

```text
same Remote MCP deployment
        |
        +--> Codex CLI version X
        +--> Cursor CLI version Y
        +--> Antigravity CLI version Z
```

各clientについて、次のevidenceを観測します。

```text
reach -> auth -> init -> tools
```

serverとclientが個別には仕様適合しているように見えても、製品固有の設定、OAuth discovery順序、credential storage、callback handling、registration strategy、released version regressionなどにより、特定の組み合わせだけ失敗することがあります。

そのため`mcp-interop`では、**client productとversion自体がinterop evidenceの一部**です。

## Static client compatibility matrixではない

client capability matrixが答えるのは「product/version Xが一般に何をsupportするか」です。これは有用ですが、`mcp-interop` は別のuniversal feature tableを手作業で競うべきではありません。

このprojectが最も強く証明できるのは、deployment-specificで再現可能なevidenceです。

```text
endpoint A + client X version 1 -> PASS
endpoint A + client X version 2 -> AUTH FAIL
endpoint A + client Y version 7 -> PASS
```

つまりversion間のregression detectionをfirst-class use caseとして扱います。resultは必ず、実際にtestしたendpoint、client product、client version、必要に応じてOS/runtime context、そしてそのrunで観測したevidenceにscopeします。実行していないproduct/versionへの互換性へ一般化しません。

static compatibility documentationは「どのtestを走らせるか」を決める材料にはなりますが、対象deploymentに対するlive resultの代わりにはなりません。

## Evidence hierarchy

次のevidence layerを分離します。

1. specification / conformance evidence
2. direct server inspection / debugging
3. product-profile preflight metadata
4. deploymentから提供されたsanitized Runtime Evidence
5. **live deployment-specific real-client evidence**

対象deploymentについて`mcp-interop`のreal-client `reach/auth/init/tools` PASSを出せるのは5だけです。

fixtureやlocalhost adapter self-testが証明するのはmeasurement pathの正しさです。別のproduction deploymentがPASSしたことまでは証明しません。

## Adapter graduation criteria

client名を増やすことより、evidenceの信頼性を守ることを優先します。live adapterをresearch/betaから昇格させる前に、measurement boundaryを明確に文書化・再現できる必要があります。

少なくとも次を評価します。

- **isolation** — 通常userのconfig / credentialを再利用・変更しない
- **supportedまたは意図的に観測するclient surface** — coverage数を増やすためだけにprivate/minified UI internalsへ依存しない
- **no-model core path** — core interop evidenceをLLMのtool選択正しさへ依存させない
- **machine-readableまたは保守的に解釈可能なevidence** — evidence不足は`unknown`であり、PASSを推測しない
- **cleanup** — temporary credential / config / process / stateを除去、または独立確認する
- **version context** — shipping client versionと関連platform contextを記録する
- **deterministic fixture proof** — controlled E2Eでadapterが主張どおりreal client pathを観測していることを示す

この境界を満たせないclientは、project全体のPASS意味を弱めるよりresearch-onlyに維持します。

## 両方を使う理由

release pipelineでは、次のように二段階で使えます。

```text
1. protocol/specification correctness
   -> modelcontextprotocol/conformance

2. 実Remote MCP endpointをdeploy

3. product-level interoperability
   -> usersが実際に使うclientでmcp-interop
```

Conformance PASSだけでは、特定deploymentがすべてのreleased client productで動くことまでは証明しません。逆に`mcp-interop` PASSだけでも、MCP仕様への完全な適合性は証明しません。

## OAuth / diagnose の境界

OAuthは最も重複しやすい領域なので、明確な境界を維持します。

**GenericなMCP/OAuth protocol conformanceは公式Conformanceの担当です。**

`mcp-interop diagnose --profile <product>` は、特定client product向けの**product compatibility profile**です。たとえばChatGPT profileでは、公開metadataやsanitized runtime observationを、その製品の公開された認証挙動と照合します。これをgeneric MCP conformanceとして扱いません。

Diagnostic profileのルール:

- product-specific expectationを明示する
- generic MCP/OAuth conformance scenarioを第二のconformance suiteとして再実装しない
- `PREFLIGHT`、sanitized Runtime Evidence、real-client interoperabilityを別のevidence layerとして扱う
- metadata互換性をreal-client `reach/auth/init/tools` PASSへ昇格させない
- 「仕様適合しているか」だけが問いなら公式Conformanceを優先する

## localhost fixtureの役割

repo内のdeterministic localhost MCP fixtureは、adapterが本当に実clientを観測できているか、isolation/cleanupが壊れていないかをrelease前に検証するためのものです。

これは**adapterのself-test / release gate**であり、任意のclient/serverをMCP conformantと認定するためのconformance suiteではありません。

## `mcp-interop` が主張しないこと

Interop PASSは次を証明しません。

- MCP仕様への完全なconformance
- server/clientのsecurity
- 各tool実装の正しさ
- destructive operationの安全性
- modelが正しいtoolを選ぶこと
- 実際に走らせていないclient product/versionとの互換性

runtime設計の詳細は[Architecture](architecture.ja.md)を参照してください。