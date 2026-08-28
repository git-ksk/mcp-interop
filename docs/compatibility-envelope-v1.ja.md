# Compatibility envelope v1

[English](compatibility-envelope-v1.md) | [日本語](compatibility-envelope-v1.ja.md)

Compatibility envelope v1は、相互運用性を**実際に観測したexact client-version pointの集合**として表します。2つのversionでPASSしただけで、その間を連続した対応rangeとして推測しません。

## 証拠モデル

observed pointは次の組で識別します。

- target IDと非secretなdeployment ID
- 検証済みschema-v2 live resultのdeployment fingerprint
- client adapter IDとexact client version文字列
- 検証済みlive resultが持つrunner/process platform OS / architecture
- auth mode

各pointには、元になった全observationを残します。`executed_at`、suite result-set digest、`mcp-interop` runtime identity、real-client adapter provenance、stage/reason evidence、accept済みbaselineに対するregression evidenceを保持します。`executed_at`はwall-clock evidenceであり、独立runner間のportableな全順序を証明する値ではありません。

envelopeは1つのexact suite manifest fingerprintとexecution contextに限定されます。manifest、execution context、logical run identity、deployment fingerprintが異なる証拠は黙って混ぜずfail-closedにします。

ここでの`platform`はlive-result schema v2と同じく、`mcp-interop` runner/process platformを意味し、real client binary architectureの証明ではありません。新しいlocal evidence生成とinstalled-client compatibility queryでは、直接認識できるMach-O / ELF / PE executable metadataを確認し、native executableの実測architecture集合にrunner architectureが含まれない既知mismatchをfail-closedにします。script/wrapper architectureはunknownのままで、hostから推測しません。既存v0.8 artifactは引き続き読めますが、過去のclient-binary architectureを遡って証明するものにはなりません。

## state

各stateは意味を明確に分けます。

- `tested` — exact observed pointに一貫したPASS evidenceがある
- `untested` — query専用state。既知contextで要求されたexact version/platform pointが未観測。observed pointとしてはserializeしない
- `stale` — 本来はtestedなPASS evidenceだが、明示的freshness policyにより古いと判断された
- `known_broken` — exact pointに一貫した実測FAIL evidenceがある
- `regressed` — 比較可能なaccept済みbaselineに対して、current observationの実退行が確認された
- `unknown` — exact pointの証拠自体が不確実または不安定。UNKNOWN/SKIPや保持したretry同士の不一致など

したがって`unknown`はevidence state、`untested`はcoverage stateです。混同しません。

client versionだけが変わったことはregressionではありません。auto-updateされたローカルclientのexact versionをまだ観測していなければ、そのversionは測定されるまで`untested`です。

## stale判定

staleは明示policyで決まり、未観測versionの対応を推測しません。v1は次の2つを扱います。

- `max_age_seconds`: otherwise-tested evidenceが指定期間を超えたらstale。新しくage policyを構築する場合は`trust_executed_at_clock=true`が必須で、CLIでは`--trust-executed-at-clock`を指定します。
- `stale_on_client_version_change`: 同じtarget/deployment/client/auth/platform contextで、collection order上あとに来るobservationが別のexact client-version文字列だった場合、古いotherwise-tested pointをstale

version-changeのchronologyにwall-clock timestampは使いません。accept済みbaselineがある場合はcurrent observationより前、その後は繰り返した`--observation`の指定順をそのままcollection orderとして使います。このpolicyを有効にする場合、callerはoldest -> newestの順でresult setを渡す必要があります。SemVerの大小比較や補間はしません。たとえば`1.2.0`と`1.8.0`を観測しても、`1.5.0`をtested/supportedとは扱いません。

age-based freshnessは別で、性質上`executed_at`へ依存します。明示clock-trust flagは、選択したage thresholdに対してrunner clockが十分同期しているというoperator assertionです。このassertionなしでは、新しい`max_age_seconds` envelopeの構築を拒否し、clock skewから強いfreshness claimを黙って作りません。clockをtrustした場合でも、`evaluated_at`より未来のobservation timestampはfresh扱いせず、`observation_after_evaluation`理由の`stale`にします。

## evidence gap

suite executionの中にはexact compatibility pointにできない証拠もあります。次のようなものは`evidence_gaps`として別に保持します。

- live-result artifactを作れなかったexecution error
- logical run evidenceの欠落
- real-clientではないprovenance
- exact client versionの欠落

gapにもregression/evidence-loss情報は残せますが、pointへ載せるためにclient versionを捏造しません。

## baselineとの関係

accept済みimmutable baselineを渡した場合、current observationは安全に比較できる範囲だけbaselineと比較します。deployment mismatchはfail-closedです。runner-platform差は無理に同じpointへまとめず、別のobserved pointとして扱います。

`regressed`はPASS-to-FAIL/UNKNOWN/SKIP、reason-code regressionなど、実際のbaseline比較証拠からのみ導出します。client version文字列が違うだけでは`regressed`になりません。

## schema互換性

Compatibility envelope v1は既存contractの上に追加する新しいreporting modelです。次は変更しません。

- live-result artifact schema v2
- suite manifest v1
- suite result-set v1
- suite regression report v1
- real-client live PASSの意味

既存artifactにmigrationは不要です。検証済みの既存evidenceを読み、exact-point compatibility viewを追加します。v0.8が出力した`trust_executed_at_clock`なしのcompatibility-envelope artifactもstructuralには引き続き読めます。変更されるのは、新しいage-based policyを構築するときに明示clock-trust assertionが必要になる点です。

## secret/privacy境界

envelopeには非secret deployment identityと、既存portable evidenceに含まれるdeployment fingerprintを使います。raw Remote MCP endpoint URL、protected endpoint path、OAuth credential、bearer token、executable path、human diagnostic payloadは追加しません。

schema-v2 live resultと同様、credential-safeであることとdeployment-publicであることは別です。originやoperator定義deployment ID自体が運用上privateな場合は、その前提で共有してください。

## インストール済みclientをqueryする

`mcp-interop compatibility query`は、shipped live clientの現在インストール済みexact versionを検出し、明示的に指定したlocal evidenceに対してそのversionを分類します。

```console
mcp-interop compatibility query \
  --client codex \
  --target production-a \
  --deployment-id production-a \
  --baseline baselines/current \
  --observation suite-results-latest \
  --stale-on-client-version-change
```

`--json`ではversionedな`mcp-interop/compatibility-query` machine-readable outputを返します。manifest/execution context、stale policy、任意のaccept済みbaseline fingerprint、検出したexact version、exact query、classification、保持したpoint evidence、関連evidence gapを含みます。一方で、検出したexecutable pathや入力filesystem pathは出力しません。

`--max-age-seconds N`でage-based stale判定を有効にする場合は`--trust-executed-at-clock`も必須です。`--observation`は最大128件まで繰り返せます。`--stale-on-client-version-change`ではその指定順が明示collection orderになるため、oldest -> newestで渡してください。大量入力によって保持observation reportが無制限に大きくなるのを防ぐための上限です。`--baseline`または`--observation`の少なくとも一方が必要です。

既存の`mcp-interop clients` / `clients --json` contractは変更しません。検出metadataだけではcompatibility claimになりません。auto-updateされた新しいinstalled versionが指定evidenceのexact observed pointに存在しなければ`untested`と表示します。

## exact observed matrixを列挙する

`mcp-interop compatibility matrix`は、installed clientを検出・実行せず、明示したbaseline / result setからcompatibility envelope全体を出力します。

```console
mcp-interop compatibility matrix \
  --baseline baselines/current \
  --observation attempt-1 \
  --observation attempt-2 --json
```

各exact pointの全observation、retry、state、evidence gapを保持します。human outputもattempt countと`attemptN:outcome`を表示し、retry後のPASSで先行FAIL/UNKNOWNを隠しません。execution errorやclient unavailableは`known_broken`へ変換せず`evidence_gaps`のままです。入力順・stale policy・128 observation上限は`compatibility query`と同じです。現在repositoryに保持しているexact実測範囲は[Exact observed client coverage](observed-coverage.ja.md)を参照してください。
