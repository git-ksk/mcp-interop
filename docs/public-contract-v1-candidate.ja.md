# Public contract candidate

[English](public-contract-v1-candidate.md) | **日本語**

この文書は、将来`v1.x`で維持する公開互換性の候補を記録します。v0.10では**candidate contract**であり、内部実装の全てをpublic APIにするものではありません。

## CLI compatibility

公開documentに記載するtop-level commandとflagをsupported CLI surfaceとします。stable `v1.x`では次を守ります。

- 既存command/flagの意味を黙って別用途へ変更しない
- 削除や意味変更にはdeprecationとmigration、またはmajor version変更が必要
- 新commandやoptional flagの追加はcompatible changeとして可能
- undocumentedな内部実装、temporary file、child-process argument、private adapter internalsはpublic CLI contractではない

## Exit code

CLIは3種類のprocess exit classを予約します。

| Code | Contract |
| ---: | --- |
| `0` | command固有のsuccess contractを満たした。`test`ならcore 4 stageが全てPASS。validatorならdocumentがvalid。`--fail-on-regression`なしのcompareは、regressionをreportしてもreport生成成功として`0`になり得る。 |
| `1` | invocation自体はvalidだが、要求したgateを満たさない、またはread/write/execution等のoperational errorでsuccess resultを得られない。non-PASS live test、preflight failure、`--fail-on-regression`でのregression等。 |
| `2` | usage、option、trust boundary、pre-execution configuration validationの失敗。interoperability PASSを推測してはいけない。 |

将来commandも独自numeric meaningを増やさず、このclassを使います。

## JSON compatibility class

### Unversioned command JSON

`clients --json`、`test --json`、diagnostic command JSON等はportable evidence schemaではなくcommand interfaceです。`v1.x`では既存field名・type・意味を維持します。additive fieldは許可し、consumerは未知fieldを無視する必要があります。field削除・rename・type変更・意味変更はincompatibleです。

local diagnostic JSONには、既存contractに含まれるexecutable path等のlocal-machine factが含まれる場合があります。これは自動的にportable/public-safe evidenceになるわけではありません。

### Versioned report / evidence

portable/versioned artifactは明示的な`schema_version`と、定義される場合は`artifact_type`を持ちます。evolution policyはschema contractでより厳密に定義します。validatorの成功はschema validという意味で、non-PASS observationをPASSへ変えません。

## Reason-code compatibility

`reason_code`は、**既存値をstableに保つopen string enum**です。

- 既存codeを`v1.x`内でrename・削除・別意味へ再利用しない
- 新しいdirect evidence classificationが必要なら新codeを追加できる
- consumerは未知の将来codeを許容し、可能なら保持する
- code欠落時にfree-form textから独自classificationを推測しない
- project-authored messageは改善可能だがmachine identifierではない

現在のcode名は[Reason codes](reason-codes.ja.md)を参照してください。portable artifact readerは、他がvalidなら未知の将来reason-code stringだけを理由にrejectしません。

## Primary live-result JSON

core live resultの意味は次を維持します。

- `client_id` — stable adapter identity
- `client_name` — human-readable product label
- `client_version` — safely observableなexact version
- `endpoint` — command resultのendpoint表示。portable protected-path evidenceは別artifact identity modelを使う
- `stages` — stable core順序`reach`, `auth`, `init`, `tools`
- `diagnostics` — core verdictを単独でupgradeしないsecret-safe supporting evidence

reason codeやdiagnostic変更だけでcore PASSを広げません。

## Compatibility review rule

public command、JSON field、exit-code meaning、既存reason codeを変更する前に、必ず次のいずれかへ分類します。

1. compatible addition
2. machine meaningを変えないbehavior clarification
3. versioned migration/deprecation
4. major-version incompatibility

分類が曖昧ならfail closedとし、公開meaningを黙って変更しません。
