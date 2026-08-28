# Capability profile v1

[English](capability-profile-v1.md) | **日本語**

Capability profile v1は、**optional capability向けの独立したevidence contract**です。既存Remote Tool Interoperability result:

```text
reach -> auth -> init -> tools
```

の意味を変更しません。

Resources、Prompts、Tasks、MRTR、controlled tool-call profile、将来capabilityは、それぞれ正確なevidence contractを定義した後だけcapability profileで表現できます。ここでcapability名を例示すること自体は、現在のshipped adapterが対応済みという意味ではありません。

v0.9.0では、Resources / Prompts / Tasks / MRTRその他optional capabilityについて**capability PASSをまだ1件もemitしません**。まずevidence boundaryを固定します。

## CLI

clientを実行せず、Remote MCPへ接続せずにcapability-profile fileをvalidateします。

```console
mcp-interop capability validate capability-profile.json
mcp-interop capability validate capability-profile.json --json
```

exit `0`はdocumentがstructuralにvalidという意味だけです。core interoperability PASSでも「全capability PASS」でもありません。human outputは各capability stateを表示し、`--json`はvalidated profileだけを再出力します。入力filesystem pathは出しません。

## schema

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/capability-profile",
  "context": {
    "observed_at": "2026-08-28T12:00:00Z",
    "deployment_id": "production-a",
    "deployment_fingerprint": "sha256:...",
    "client": {
      "id": "codex",
      "product": "Codex CLI",
      "version": "codex-cli 0.133.0"
    },
    "platform": {
      "os": "darwin",
      "arch": "arm64"
    },
    "runtime": {
      "mcp_interop_version": "dev",
      "mcp_interop_commit": "...",
      "go_version": "go1.26.6"
    },
    "auth_mode": "default",
    "evidence_provenance": {
      "kind": "real_client_adapter",
      "adapter_id": "codex"
    }
  },
  "capabilities": [
    {
      "capability_id": "resources",
      "state": "pass",
      "evidence_kind": "client_protocol",
      "evidence_id": "resources.list.response"
    }
  ]
}
```

`capability_id` / `evidence_id`は短いproject-defined identifierです。raw protocol payload、client log、UI text、URL、diagnostic messageを入れません。

capability配列は`capability_id`順のdeterministic orderとし、duplicate IDをrejectします。

## exact context

profileは1つのexact contextへ固定します。

- operatorが選ぶ非secret `deployment_id`
- exactなsecret-safe `deployment_fingerprint`
- exact client ID / product / version
- runner/process OS / architecture
- `mcp-interop` runtime identity
- auth mode（`default` / `oauth`）
- real-client adapter provenance
- UTC observation time

endpoint URL/path/query、bearer token、authorization code、cookie、client secret、executable path、raw human/client outputはschemaに存在しません。

将来emitterは、可能な場合validated schema-v2 protected-path live runからcontextを作ります。`ContextFromLiveRunV2`はprotected endpoint pathを受け取らず、非secret deployment ID/fingerprintとexact client/platform/runtime/auth/provenanceだけを引き継ぎます。

## state

- `pass` — このexact contextで、capability固有の成功条件をreal clientから直接観測した
- `fail` — direct real-client attemptで明示的なnegative capability outcomeを観測した
- `unknown` — direct real-client attemptは行ったがevidenceが曖昧/不完全
- `unsupported` — adapterのdocumented policy/boundary上、そのexact client/profileではcapability pathを利用できない。tested failureではない
- `untested` — そのexact contextでcapability attempt/evidenceがない。`unknown` / `unsupported` / `fail`とは別

近いclient version、別platform/auth mode/deploymentへstateを自動継承しません。

## evidence kind

`pass` / `fail` / `unknown`は次の**direct client evidence**のいずれかが必須です。

- `client_protocol` — capability固有operationについてreal clientが直接origin/consumeしたprotocol evidence
- `client_control_surface` — supportedまたはdeliberately acceptedなreal-client management/control surfaceがcapability operation/resultを直接返す
- `client_observed_state` — 実際のcapability execution/discoveryで生成されたboundedなclient-owned stateを、documented observation boundaryとして使う

`client_observed_state`にconfig presence、enabled flag、allowlist、static cached metadata、単なるfeature広告UI textは含みません。

`unsupported`は`adapter_policy`と、documented policy boundaryを示すstable/非secret `evidence_id`が必須です。

`untested`は`none`のみで、`evidence_id`を持てません。

server metadata、client config presence、browser/UI presence、model output、fixture-only server observation、generic feature advertisingを表すvalid evidence kindはschemaに存在しません。したがって、それらをcapability PASSへ変換できません。

controlled fixtureを使う場合も、**real client自身**がaccepted direct evidence surfaceを返す必要があります。fixture server側だけの観測をdeployment-specific capability PASSへ昇格しません。

## PASSはcapability固有

`resources` / `prompts` / `tasks` / `mrtr`のような名前だけではPASSになりません。shipped adapterがcapability PASSをemitする前に、実装/docsで最低限次を定義します。

1. successとなるexact client operation / observed state
2. accepted direct evidence kindとstable `evidence_id`
3. `fail`と`unknown`の境界
4. 明示的な`unsupported`条件
5. isolation / cleanup / secret-safety境界
6. claimed pathのcontrolled real-client fixture coverage

これによりstatic capability matrixやprotocol/server metadataをlive deployment evidenceへ変換することを防ぎます。

## core interoperabilityは不変

capability profileはlive-result artifact v1/v2へ埋め込まず、現在の`test` / `compare` / suite / baseline / compatibilityのPASS/regression判定からも読みません。

```text
capability pass != core live PASS
core live PASS != optional capability pass
```

valid capability profileに`fail` / `unknown` / `unsupported` / `untested`が含まれていても`capability validate`は成功できます。validateはaggregate PASSを作らずevidence contractだけを確認するためです。

## v0.8 compatibility

Capability profile v1はadditiveな別artifactです。次を変更/migrateしません。

- live-result artifact schema v1
- live-result artifact schema v2
- suite manifest/result-set schema
- baseline schema
- compatibility envelope/query semantics
- public `reach/auth/init/tools` result

既存v0.8 artifactは既存readerでそのまま読めます。既存runからcapability profileを作る場合は、raw endpoint pathを新profileへ持ち込まないためvalidated schema-v2 protected-path runを使います。schema-v1 artifactもhistorical evidenceとして引き続きvalidで、capability profileへ黙って変換しません。

## secret / input boundary

profile fileはunknown field rejectのstrict decodeとbounded inputを使います。`endpoint`、`path`、`token`、`authorization_code`、`cookie`、raw output blobのようなad-hoc fieldを受け入れて後からredactする設計ではありません。

writerは既存private JSON replacement pathを使います。evidence identifierは短いlowercaseの非secret project identifierだけにします。
