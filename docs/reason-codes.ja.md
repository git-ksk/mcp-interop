# Reason code

[English](reason-codes.md) | [日本語](reason-codes.ja.md)

`mcp-interop`は、明示的な証拠からfailureを安全に分類できる場合にstableな`reason_code`を出します。

現在は大きく2種類あります。

- **real-client reason code** — Codexなど実MCP clientの観測結果から分類
- **runtime diagnostic reason code** — `diagnose --profile chatgpt --runtime-evidence`へ明示的に渡したsecret-freeなserver observationから分類

Runtime diagnostic codeはreal-client interoperability verdictではありません。Preflight、Runtime Evidence、OpenAI Reference Pattern、real-client実行は別の証拠レイヤーです。

## real-client OAuth code

### `DCR_UNSUPPORTED`

実clientが対象OAuth flowでDynamic Client Registration非対応を明示的に報告した場合です。

推測した`/register`や`/oauth/register`が`404`になっただけでは判定しません。

### `DCR_FAILED`

実clientがDCRを試行し、unsupported以外の理由でregistrationが失敗したと明示的に報告した場合です。

### `OAUTH_CALLBACK_PORT_CONFLICT`

実clientが、そのOAuth flowで選択したloopback callback address/portへlistenerをbindできなかったことを明示的に報告した場合です。

clientから観測できるbind conflictの証拠に基づくcodeです。過去に観測した、または推測したcallback portが使用中というだけでは付与しません。callback addressはclient version固有として扱い、固定portを永久contractとしてhard-codeしません。

## Runtime registration / token request code

### `REGISTRATION_STRATEGY_UNSUPPORTED`

明示的に観測したregistration方式（`cimd`または`dcr`）を、discovery済みAuthorization Server Metadataが広告していない場合です。

`predefined`はpublic metadataだけではpre-registrationを証明できないため、それだけを理由にFAILにはしません。

### `TOKEN_AUTH_METHOD_MISMATCH`

観測したtoken endpoint authenticationが、利用可能なclient/server metadataから選択されるmethodと一致しない場合です。

ChatGPT CIMD profileでは、取得したChatGPT CIMDとAuthorization Server双方が`private_key_jwt`を広告しているのに、token requestにclient assertionが無ければ:

```text
expected: private_key_jwt
observed: none
```

として分類できます。

DCR/predefinedへCIMDの`private_key_jwt`期待値を勝手に適用しません。

### `CLIENT_AUTH_REJECTED`

token endpointがsanitized OAuth error `invalid_client`を返した場合です。

### `TOKEN_REQUEST_REJECTED`

token endpointが、より狭い分類へ落とせないsanitized OAuth errorを返した場合です。

### `RESOURCE_MISMATCH`

OAuth `resource`がcanonical protected MCP resourceと一致しなかったというruntime observationがある場合です。

### `REDIRECT_URI_MISMATCH`

authorization requestのredirect URIが、診断対象のregistered/client metadata値と一致しなかったという観測です。

### `PKCE_S256_MISSING`

ChatGPT profileで期待されるPKCE S256がauthorization requestで観測されなかった場合です。

### `PKCE_VERIFIER_MISSING`

token requestで`code_verifier`が存在しなかった場合です。値そのものは入力しません。

## MCP Resource Server code

OAuth token exchange後のsecret-free observationを分類します。`mcp-interop`自身はBearer token値を取り込みません。

### `ACCESS_TOKEN_NOT_ATTACHED`

後続MCP requestにBearer tokenが付いていないと観測された場合です。

### `TOKEN_SIGNATURE_INVALID`

Resource ServerがBearer tokenのsignature validation失敗を報告した場合です。

### `TOKEN_ISSUER_MISMATCH`

Resource Serverがtoken issuer不一致を報告した場合です。

### `TOKEN_AUDIENCE_MISMATCH`

Resource Serverがtoken audience/resourceとprotected MCP resourceの不一致を報告した場合です。

### `TOKEN_EXPIRED`

Resource Serverがtoken期限切れを報告した場合です。

### `INSUFFICIENT_SCOPE`

MCP operationに必要なscopeがtokenに不足しているとResource Serverが報告した場合です。

これらはOpenAI authentication docsがresource serverに求めるtoken verificationと、`openai/openai-mcpkit`のauthenticated Python MCP scaffoldが示すJWT verification patternに対応します。

## Tool-level OAuth signal code

OAuth challengeが必要だったと明示的に観測できる場合だけconclusive failureにします。未観測なら`WARN / unknown`です。

### `TOOL_OAUTH_METADATA_MISSING`

auth-required toolに期待されるOAuth `securitySchemes` metadataが無い場合です。

### `TOOL_OAUTH_CHALLENGE_MISSING`

tool-level認証/再認証challengeが必要なのに、`_meta["mcp/www_authenticate"]`が無いと観測された場合です。

### `TOOL_OAUTH_CHALLENGE_INVALID`

runtime challenge自体はあるものの、明示的に観測した必須OAuth error / error-description signalが欠けている場合です。

この証拠を取るために`mcp-interop`が勝手にtoolをcallすることはありません。既に得られているsanitized server observationだけを利用します。

## Reason precedence

Runtime Evidenceのtop-level `reason_code`にはdiagnostic evaluation order上で最初のconclusive failureを出します。ただし各checkは個別の`reason_code`を保持します。

たとえば最初に`TOKEN_AUTH_METHOD_MISMATCH`が発生し、token endpointがさらに`CLIENT_AUTH_REJECTED`を返していた場合も、両方のcheckを確認できます。

## Security boundary

Runtime Evidenceへ入力できるのは、限定されたpresence/match boolean、stableな非secret metadata identifier、registration strategy、短いsanitized OAuth error codeだけです。未知JSON fieldは拒否します。

次は絶対に入力・reason detailへ保存しません。

- access / refresh token
- authorization code / OAuth state
- PKCE verifier/challengeの値
- raw client assertion / private key
- client secret
- cookie / credential file

実clientのraw errorにもremote生成文字列が混ざる可能性があります。分類に必要な場合だけmemory内で扱い、通常の出力にはstable codeとproject側で定義したmessageを使います。

## Capability correlation boundary

capability evidenceはMCP Protected Resource MetadataとAuthorization Server discoveryを辿って取得します。registration URLを推測してDCR supportを判定しません。

複数`authorization_servers`がある場合は保守的に扱います。実flowが選択したissuerを証明できない限り、別issuerのcapabilityを混ぜてauth-method期待値を作らず`unknown`にします。
