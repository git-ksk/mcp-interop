# Schema evolution v1 candidate

[English](schema-evolution-v1-candidate.md) | **日本語**

この文書はportable evidence、suite state、baseline、derived machine reportのv1-candidate evolution policyを定義します。[Public contract candidate](public-contract-v1-candidate.ja.md)を補完します。

## Schema family

### Strict portable/input schema

mcp-interopがpersist後にfail-closed validationで再読込するdocumentです。

| Family | Current schema | Artifact type / identity |
| --- | ---: | --- |
| Live result artifact | v1 / v2 | `mcp-interop/live-results` |
| Suite manifest | v1 | manifest contract |
| Suite result set index | v1 | `mcp-interop/suite-results` |
| Suite baseline descriptor | v1 | `mcp-interop/suite-baseline` |
| Capability profile | v1 | `mcp-interop/capability-profile` |
| Runtime Evidence | v1 / v2 / v3 | diagnostic evidence contract |

これらは、古いstrict readerがrejectする構造変更なら新しいschema versionが必要です。同じschema versionへrequired/unknown fieldを黙って追加してcompatibleとは扱いません。

新schema versionでは次を明示します。

- 新versionが必要になったsemantic problem
- 引き続きread可能な旧version
- deterministic migrationの有無
- 推測できず再観測/再生成が必要な情報
- comparison identity変更の有無

### Versioned derived report

regression、compatibility、maturity、graduation、baseline-verification等は主にoutputでもschema versionを持ちます。同一schema内では既存field名・type・意味を維持します。既存meaningを変えないoptional additive fieldは許可できますが、削除・rename・type変更・semantic repurposeにはreport schema bumpが必要です。

## 現在のlive-result v1/v2境界

live-result v1/v2は`artifact_type=mcp-interop/live-results`を共有しますがendpoint identity semanticsが異なります。

- v1はcanonical endpoint identity pathを含む
- v2は明示的な非secret `deployment_id`とorigin bindingによるprotected-path identityを使う

両方read可能ですが、**暗黙compareはしません**。`compare`はschema-specific identityを混ぜないためv1-v2 pairingをrejectします。

genericな自動v1 -> v2 migrationはありません。protected deployment IDをv1 artifactから安全に推測できないためです。v2 identityが必要なら、明示的な非secret deployment IDで再生成/再観測します。

## Cross-schema comparison policy

比較identityのsemantic equivalenceを明示的なcomparison/migration ruleで証明できる場合だけcross-schema compareを許可します。それ以外はfail closedです。

次は禁止します。

- client IDやendpoint stringが似ているだけでrunをpairする
- endpoint pathからprotected deployment IDを推測する
- schema固有identity fieldをbest effortでmergeする
- migration失敗を`untested` / `pass` / `regressed` evidenceへ変換する

将来migration toolを追加する場合も別の明示操作とし、source artifactを保持します。lossy/non-derivable fieldは捏造せずreportします。

## Baseline contract

Baseline v1が提供するのはlocal workflow immutabilityとcontent-consistency bindingです。

- output directoryはno-clobber
- result-set snapshotをcopyしてdigest
- descriptor/result-set mismatchをreject
- `supersedes`は明示指定時にexact predecessor baseline fingerprintへbind

baseline fingerprintが証明するのはdocumented algorithm上のdescriptor content identityです。signature、authenticated reviewer identity、timestamp authority、誰が実行/acceptしたかの証明ではありません。

team/CI authenticityが必要ならexact fingerprintを外部review/signature/attestationへbindします。将来native authenticated provenanceを追加する場合もbaseline v1の意味を変更せず、明示的なversioned envelopeを使います。

## Compatibility-envelope contract

compatibility reportはretained exact evidenceだけから導出し、連続client-version rangeを作りません。

- `tested`はselected policy下でmatching exact evidenceがある
- `untested`はexact query pointが未観測
- stale / known-broken / regressedはdocumented evidence conditionが必要
- version変更だけではregressionにならない
- retry/evidence gapをclean PASSへ潰さず保持する

これらの意味変更には明示的なreport/schema contract revisionが必要で、docsだけで意味を読み替えません。

## Deprecation / removal policy

stable `v1.x`では次を守ります。

- v1.0時点でsupportedと記載したportable schemaは、具体的なsecurity issueで危険にならない限りread継続
- default writerを新schemaへ移しても旧supported evidenceをunreadableにしない
- public schema/CLI deprecationはremoval前にdocument化
- removal/incompatible reinterpretationは原則次major versionまで待つ
- security exceptionはrelease noteへ明示し、旧evidenceを黙って読み替えずfail closed

historical evidenceをdefaultでin-place rewriteしません。

## Evidence authenticity boundary

portable artifactはdefaultではevidence recordでありcryptographic attestationではありません。digest/fingerprintが示すのはdocumented content identity/consistencyです。別のauthenticated provenance mechanismが明示しない限り、actor identity、machine identity、trusted time、execution authenticityは証明しません。
