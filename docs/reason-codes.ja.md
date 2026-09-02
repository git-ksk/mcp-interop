# Reason code

[English](reason-codes.md) | **日本語**

> この文書は英語版`reason-codes.md`の日本語訳です。定義に差がある場合は英語版を正とします。

`mcp-interop`は、失敗原因を**明示的な証拠から安全に分類できる場合だけ**、安定した`reason_code`を返します。

大きく2種類あります。

- **実クライアント由来** — Codex / Cursorなど、実際に起動したMCPクライアントの観測結果から分類
- **Runtime Evidence由来** — `diagnose --profile chatgpt --runtime-evidence`へ渡された、秘密情報を含まないserver-side observationから分類

Runtime Evidenceのreason codeは、実クライアントのinterop verdictではありません。

Preflight、Runtime Evidence、OpenAI Reference Pattern、real-client runは別々の証拠です。

## 実クライアントOAuth

### `DCR_UNSUPPORTED`

実クライアントが、対象OAuth flowでDynamic Client Registrationをサポートしていないと明示した場合です。

推測した`/register`や`/oauth/register`が404だっただけでは、このcodeを付けません。

### `DCR_FAILED`

実クライアントがDCRを実行し、「unsupported」以外の理由でregistration失敗を明示した場合です。

Codex / Cursorでは、`DCR_UNSUPPORTED`と`DCR_FAILED`は実クライアントがMCP OAuth registration境界まで到達した証拠にもなります。

そのため、この経路では次のように判定できます。

```text
reach=pass
auth=fail
後続stage=skip
```

一般的なOAuth startup failureだけで`reach=pass`にはしません。

### `OAUTH_CALLBACK_PORT_CONFLICT`

実クライアントが、そのOAuth flowで選んだloopback callback address / portへlistenerをbindできなかったと明示した場合です。

「以前このportだった」「たぶんこのportが使用中」といった推測では付けません。

callback addressはclient version固有として扱います。

## Registration / token request

### `REGISTRATION_STRATEGY_UNSUPPORTED`

実際に観測したregistration方式（`cimd`または`dcr`）を、discoveryしたAuthorization Server Metadataが広告していない場合です。

`predefined`は公開メタデータだけでは事前登録の有無を証明できないため、それだけでFAILにはしません。

### `TOKEN_AUTH_METHOD_MISMATCH`

観測したtoken endpoint authenticationと、利用可能なclient/server metadataから期待される方式が一致しない場合です。

ChatGPT CIMD profileで、ChatGPT CIMDとAuthorization Serverの両方が`private_key_jwt`を広告しているのに、token requestにclient assertionが無ければ:

```text
expected: private_key_jwt
observed: none
```

として分類できます。

DCR / predefinedへCIMDの期待値を勝手に適用しません。

### `CLIENT_AUTH_REJECTED`

token endpointが、秘密情報を除去したOAuth error `invalid_client`を返した場合です。

### `TOKEN_REQUEST_REJECTED`

より具体的なreason codeへ分類できないOAuth errorでtoken requestが拒否された場合です。

### `RESOURCE_MISMATCH`

OAuth `resource`がcanonical protected MCP resourceと一致しなかった、という明示的観測がある場合です。

### `REDIRECT_URI_MISMATCH`

authorization requestのredirect URIが、登録済みまたはclient metadataの値と一致しなかった場合です。

### `PKCE_S256_MISSING`

ChatGPT profileで期待されるPKCE S256をauthorization requestで確認できなかった場合です。

### `PKCE_VERIFIER_MISSING`

token requestに`code_verifier`が存在しなかった場合です。値そのものは入力しません。

## MCP Resource Server

OAuth token exchange後の観測を分類します。`mcp-interop`自身はBearer token値を受け取りません。

### `ACCESS_TOKEN_NOT_ATTACHED`

後続MCP requestにBearer tokenが付いていないと観測された場合です。

### `TOKEN_SIGNATURE_INVALID`

Resource Serverがtoken signature validation失敗を報告した場合です。

### `TOKEN_ISSUER_MISMATCH`

token issuerが一致しない場合です。

### `TOKEN_AUDIENCE_MISMATCH`

token audience / resourceとprotected MCP resourceが一致しない場合です。

### `TOKEN_EXPIRED`

tokenが期限切れの場合です。

### `INSUFFICIENT_SCOPE`

MCP operationに必要なscopeが不足している場合です。

## Tool-level OAuth signal

未観測のものをFAILと推測しません。必要なsignalを明示的に観測できた場合だけconclusive failureにします。

### `TOOL_OAUTH_METADATA_MISSING`

認証対象toolに期待されるOAuth `securitySchemes` metadataが無い場合です。

### `TOOL_OAUTH_CHALLENGE_MISSING`

tool-levelの認証・再認証challengeが必要なのに、`_meta["mcp/www_authenticate"]`が無い場合です。

### `TOOL_OAUTH_CHALLENGE_INVALID`

runtime challenge自体はあるものの、必須のOAuth error / error-description signalが欠けている場合です。

この証拠を得るために`mcp-interop`が勝手にtoolを実行することはありません。すでに得られている秘密情報を含まないserver observationだけを使います。

## Reason codeの優先順位

Runtime Evidenceのtop-level `reason_code`には、評価順で最初に見つかったconclusive failureを表示します。

ただし各checkは個別の`reason_code`を保持します。

たとえば最初に`TOKEN_AUTH_METHOD_MISMATCH`があり、その後token endpointが`CLIENT_AUTH_REJECTED`も返した場合、各checkで両方を確認できます。

## 秘密情報の境界

Runtime Evidenceへ入力できるのは、限定されたboolean、stableな非secret metadata identifier、registration strategy、短いsanitized OAuth error codeだけです。

未知JSON fieldは拒否します。

次は入力・reason detailへ保存しません。

- access / refresh token
- authorization code / OAuth state
- PKCE verifier / challengeの値
- raw client assertion / private key
- client secret
- cookie / credential file

実クライアントのraw errorにRemote側の文字列が含まれる場合も、分類に必要な範囲だけmemory内で扱い、通常出力にはstable codeとプロジェクト側で定義したmessageを使います。

## Capability情報を相関するときの境界

capability evidenceはProtected Resource MetadataとAuthorization Server discoveryから取得します。

registration URLを推測してDCR対応を判定しません。

複数の`authorization_servers`がある場合も、実際のflowがどのissuerを選んだか証明できなければ、別issuerのcapabilityを混ぜて期待値を作らず`unknown`にします。

## Compatibility policy

Reason codeは既存値をstableに保つopen string enumです。stable major line内では既存codeをrename・削除・別意味へ再利用せず、新しいdirect evidenceには新codeを追加できます。consumerは未知のnon-empty codeを許容し、free-form messageから代替codeを推測してはいけません。詳細は[Public contract v1](public-contract-v1.ja.md)を参照してください。
