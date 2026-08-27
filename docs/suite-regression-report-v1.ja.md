# Suite regression report v1

[English](suite-regression-report-v1.md) | **日本語**

> この文書は英語版`suite-regression-report-v1.md`の日本語訳です。契約の正本は英語版です。

Suite regression report v1は、validation済みbaseline suite result set 1件と、保持したcurrent attempt 1件以上を比較します。per-runのstage / reason-code regression semanticsは既存live-artifact comparatorを再利用し、手書きsupport matrixから判定しません。

## CLI

```console
mcp-interop suite compare baseline-results attempt-1 [attempt-2 ...]
mcp-interop suite compare baseline-results attempt-1 [attempt-2 ...] --json
mcp-interop suite compare baseline-results attempt-1 [attempt-2 ...] --fail-on-regression
```

directoryを渡すと`index.json`へ解決します。index pathを直接指定することもできます。

`--fail-on-regression`指定時のexit contract:

- `0`: decisionが`clean`
- `1`: regressionまたはunstable retry evidenceあり
- `2`: invalid / unreadable / mismatch input、またはreport生成失敗

gate flagなしでは、valid reportならdecisionがcleanでなくてもexit `0`です。既存`compare`のreport-only behaviorと揃えています。

## Retry invariant

retryで過去attemptを置換しません。渡したattemptは順序付きで全件reportへ残します。

例:

```text
baseline: PASS
attempt 1: UNKNOWN
attempt 2: PASS
```

最初のregressionは消えず、suite decisionは`regression_and_unstable`です。後続PASSでevidence historyをcleanへ書き換えられません。

同じmaterial evidenceが繰り返された場合、retryしただけでunstableにはしません。material attempt signatureにはoutcome、endpoint fingerprint、platform、stage status / reason evidenceを含めます。client versionは各attemptへ別途保持し、version-only changeだけではregressionにしません。

## Decision

- `clean`: regressionもunstable / ambiguous attempt evidenceもない
- `regression`: regression evidenceはあるがattempt間のmaterial evidenceは安定
- `unstable`: baseline regressionはないがattemptがmixed / ambiguous
- `regression_and_unstable`: 両方ある

current evidence missingやexecution errorも明示的に保持します。baseline evidenceが存在した場合、それらは黙ってdropせずregression / evidence-loss signalとして扱います。

## Machine-readable evidence

JSON artifact typeは`mcp-interop/suite-regression-report`、schema versionは`1`です。次を保持します。

- baseline / current manifest fingerprint
- attempt countと順序
- target / deployment / client / auth identity
- 利用可能なbaseline outcome / client version / platform / fingerprint / stages
- 利用可能な全current attempt outcome / client version / platform / fingerprint / stages
- direct stage/status/reason-code changeとregression kind
- client-version change marker
- per-run regression / unstable flagとsuite decision

baselineと全current attemptは同一manifest fingerprint必須です。宣言が違うattemptをretryとして扱いません。

## Protocol evidence境界

現在のlive-result artifact schema v2はinternal `ProtocolObservation`を意図的にserializeしません。そのためsuite report v1は`protocol_evidence_status: "not_serialized_in_live_result_v2"`を明示し、fixtureなど間接証拠からprotocol era / revision changeを捏造しません。

将来portable live-result schemaがdirect observed protocol evidenceを保持する場合は、明示的migration contractを持つreport revisionで比較できます。

## Reader / privacy境界

report生成はvalidation済みsuite result setだけを読みます。参照artifactはsymlink解決後もresult-set directory内に留まり、indexの`deployment_id`、client、auth mode、outcomeと一致する必要があります。

reportにはRemote MCP endpoint URL、protected path、endpoint環境変数名/値、credential、OAuth code/token、human diagnostic messageを保存しません。非secret deployment IDとschema-v2 endpoint fingerprintはportable evidenceとして保持します。
