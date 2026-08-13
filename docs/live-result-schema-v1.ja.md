# Live interoperability result artifact schema v1

[English](live-result-schema-v1.md) | [日本語](live-result-schema-v1.ja.md)

この文書は、deployment-specificなreal-client interoperability runを保存・比較するための最初のportable result artifactを定義します。既存の`mcp-interop test --json`出力contractとは意図的に分離します。

schema v1の目的は、`reach -> auth -> init -> tools`の意味を一切弱めずに、正確なclient version間のregressionをlocal fileだけで継続比較できるようにすることです。

## Compatibility boundary

既存commandはbackward-compatibleなままです。

```console
mcp-interop test https://example.com/mcp --client codex --json
```

stdoutは従来どおり、live `Result`のJSON arrayです。schema v1の導入によって、この既存payloadへ`schema_version`、timestamp、platform、artifact metadataなどを追加しません。

portable artifactは明示的に指定したときだけ別fileへ出力します。

```console
mcp-interop test https://example.com/mcp --client codex --output result.json
```

`--json`と`--output`は併用できます。`--output`がstdoutをredirectしたり、既存text/JSON出力を置き換えたりすることはありません。

## Top-level shape

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/live-results",
  "runs": [
    {
      "executed_at": "2026-08-12T15:00:00Z",
      "endpoint": {
        "identity": "https://example.com/mcp",
        "fingerprint": "sha256:..."
      },
      "client": {
        "id": "codex",
        "product": "Codex CLI",
        "version": "codex-cli 0.147.0"
      },
      "platform": {
        "os": "darwin",
        "arch": "arm64"
      },
      "runtime": {
        "mcp_interop_version": "v0.5.0",
        "mcp_interop_commit": "...",
        "go_version": "go1.24.x"
      },
      "auth_mode": "default",
      "evidence_provenance": {
        "kind": "real_client_adapter",
        "adapter_id": "codex"
      },
      "stages": [
        {"stage": "reach", "status": "pass"},
        {"stage": "auth", "status": "pass"},
        {"stage": "init", "status": "pass"},
        {"stage": "tools", "status": "pass"}
      ]
    }
  ]
}
```

各stageには、既存のstableな`reason_code`を必要に応じて含めます。一方、human-readableなstage messageやdiagnostic payloadはv1 artifactへ保存しません。regression比較には不要であり、secret-bearingな値をportable fileへ持ち出すsurfaceを減らすためです。

## Endpoint identityとsecret safety

portable artifactにはraw target URLを保存しません。

`endpoint.identity`へ残すのは次だけです。

```text
http(s)://lowercase-host[:explicit-port]/path
```

userinfo、query parameter、query value、fragmentは除外します。legacy redactionがcredential-likeと認識できないparameter名であっても、query valueは例外なく除外します。

`endpoint.fingerprint`は、このsecret-safeなidentityに対するSHA-256で、`sha256:` prefixを付けます。raw URL自体をhashしないため、secretから派生したfingerprintもportable artifactへ残しません。

その代わり、v1ではquery parameterだけが異なる2つのdeployment targetを区別できません。これは意図したtrade-offで、secret safetyを優先します。

## Run context

各runには次を保存します。

- UTCの`executed_at`
- secret-safeなendpoint identity / fingerprint
- real-client adapter runの場合、client ID / product name / 正確に検出したclient version
- OS / architecture
- `mcp-interop` version / commitとGo runtime version
- invocation時のauth mode（現時点では`default`または明示的`oauth`）
- evidence provenance
- 順序固定の`reach` / `auth` / `init` / `tools` stage result

`auth_mode`はrunnerをどう起動したかを示します。server metadataから認証要否を推測した値ではありません。

## Evidence provenanceとPASS

`evidence_provenance.kind`は次のどちらかです。

- `real_client_adapter` — installed real clientを実際に実行したrun。`adapter_id`と正確なclient versionが必須です。
- `runner_observation` — client未installなど、real-client adapterを実行する前にrunner自身が観測した状態です。

`runner_observation`に`pass` stageを含めることは禁止します。artifact layerは新しいPASS evidenceを作らず、adapter結果を再解釈もしません。complete live PASSの条件は従来どおり、4 stageすべてがreal-client evidenceにより`pass`であることです。

## Strict validation

schema v1をcomparison inputとして読むときはstrictにvalidateします。

- `schema_version`は`1`
- `artifact_type`は`mcp-interop/live-results`
- unknown JSON fieldはreject
- 少なくとも1 run必須
- artifact内のcomparison identityは重複不可
- `executed_at`はUTC必須
- endpoint fingerprintはcanonicalなsecret-safe identityと一致必須
- stageは`reach`, `auth`, `init`, `tools`の4つをこの順序でちょうど1回ずつ
- statusは従来どおり`pass`, `fail`, `skip`, `unknown`のみ

不足・不明なevidenceを都合よくsuccessへ正規化することはありません。invalidならrejectし、validな不確定状態はnon-PASSのまま保持します。

## Comparison identity

`mcp-interop compare`は次の組み合わせでbaseline/new runをpairingします。

```text
endpoint fingerprint
+ client.id
+ auth_mode
+ platform.os
+ platform.arch
```

正確なclient version、execution time、runner/runtime versionはcontextとして保存しますが、pairing keyには含めません。これによりclient upgrade後のrunを「別物」として切り離さず、直前versionとのregression比較に使えます。

client versionだけが変わり、stage/reason evidenceが同じならregressionではありません。

## Regression semantics

```console
mcp-interop compare old.json new.json
mcp-interop compare old.json new.json --json
mcp-interop compare old.json new.json --fail-on-regression
```

comparisonは次を明示的に分類します。

- `PASS_TO_FAIL`
- `PASS_TO_UNKNOWN`
- `PASS_TO_SKIP`
- `REASON_CODE_CHANGED` — reason codeの追加・削除・変更
- `NEW_EVIDENCE_MISSING` — baselineにあったrunとpairingできるrunがnew artifactに存在しない

new-only runはreportには出しますが、それ自体はregressionではありません。stage/reasonが変わらないclient version変更もregression gateを失敗させません。

## Compare exit codes

`mcp-interop compare`のexit contractは次です。

- `0` — validなcomparisonが完了。`--fail-on-regression`なしの場合、report内にregressionがあっても0です。
- `1` — `--fail-on-regression`指定時に、1件以上のregressionまたはbaseline evidence lossを検出。
- `2` — usage error、input fileを読めない、unsupported/invalid artifact schema、comparison output failure。

通常の`mcp-interop test`のexit contractは変更しません。portable artifactの生成・書き込み失敗はexecution failureとして`1`ですが、`test`自体がsuccessになる条件は今までどおり、必要なreal-client stageがすべてpassした場合だけです。

## Scope

schema v1はlocal-file-firstです。hosted service、database、SQLite history、dashboard、新client adapter、generic MCP conformance suite、security scanner、LLM evaluation layerは導入しません。

将来artifact contractを変更する場合は、v1の意味をsilentに変更せず、新しい`schema_version`を使います。