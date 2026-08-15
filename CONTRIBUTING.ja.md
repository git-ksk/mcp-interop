# mcp-interopへのコントリビューション

[English](CONTRIBUTING.md) | [日本語](CONTRIBUTING.ja.md)

real-client MCP interoperability testingの改善に協力いただきありがとうございます。

## Pull Requestを開く前に

1. 関連する既存Issue/PRがないか確認してください。
2. 新しいclient adapterを追加する場合は、そのclientが提供するsupportedまたはsafely observableなMCP surfaceとisolation戦略を記録したIssueを作成または参照してください。
3. new client workは`issue/research -> safe observable-surface proof -> bounded PoC -> implementation`の順で進めてください。先にlive adapterを実装し、後からevidence modelを決めてはいけません。
4. ユーザーが普段使うclient configurationやcredential stateを黙って変更するlive adapterは追加しないでください。
5. unsupported / blocked / experimental / research-only clientは、その状態を明示したままにしてください。partial observationをshipped supportとして表現してはいけません。

## 開発

必要条件:

- `go.mod`で宣言されているGo version

CIと同じ基本checkを実行します。

```console
gofmt -w .
git diff --exit-code
go vet ./...
go test ./...
go build ./cmd/mcp-interop
```

process lifecycle、OAuth、shared state、release gateに関係する変更では、追加で次も実行してください。

```console
go test -race ./...
govulncheck ./...
```

`govulncheck`がローカルに無い場合はPRへ明記し、CIで固定versionのscanを通してください。

Pull Requestでrequiredになっているstatus checkはLinux / macOS / Windowsの`test (...)` jobです。required checkがfailまたはmissingの間はmerge可能な状態とは扱いません。

## Evidence / adapter要件

live client adapterは次を満たすべきです。

- client挙動をemulateせず、実際にインストールされたclientを呼び出す
- client management/control surfaceで結果を証明できる場合、model promptを使わない
- config/credentialをtemporary profile/HOME/config directoryへ隔離する
- client surfaceだけではstageを証明できない場合、成功を推測せず`unknown`を返す
- `reach`、`auth`、`init`、`tools`を別々の観測として扱う
- success/failure両方でtemporary stateとowned client processをcleanupする
- report/errorからBearer/OAuth credential等のsecretをredactする
- 検証したclient versionを記録する
- successおよび主要なfailure/inconclusive pathのtestを追加する

fixtureはmeasurement pathを証明しますが、それだけでdeployment-specific live interoperabilityを証明するものではありません。fixture-only success、configuration presence、metadata compatibility、configured tool allowlistをreal-client live PASSへ昇格させてはいけません。

cleanup failureはtest上重要です。isolated runがtemporary credential/configuration stateやowned processを残した場合、interop結果だけをcleanとして扱わず、test/harnessでfailureとして表面化させてください。

安全なisolationを確立できないclientは、既存ユーザー設定を変更して無理に対応せず、experimental/research-onlyのままにしてください。

## OAuth変更

OAuthは特に慎重に扱います。

- user interactionを発生させる可能性があるauthenticationは明示的opt-inにする
- CLI contractで明示されていないauthorization URLを勝手にopenしない
- test credentialを通常のOS Keychain/client credential storeへ保存しない
- normal-user browser/client credentialをtest profileへcopyしない
- authorization URL、authorization code、callback state、access/refresh token、cookie、client secret、private keyをmachine-readable resultやpublic evidenceへ含めない
- automated testではproduction credentialではなくlocal/synthetic OAuth fixtureを使用する

## Pull Request運用

developmentはPR-firstです。`main`へ入れる変更はfocused branchで作業し、Pull Requestを通してmergeしてください。通常のdevelopment workflowとして`main`へ直接pushしません。

PRは焦点を絞ってください。最低限、次を含めます。

- 該当する場合はrelated issue/research context
- scopeと明示的なnon-goal
- client behaviorに関係する場合は検証したclient/version
- interoperabilityを証明するために使った具体的なobservable surface
- isolation/cleanupの挙動
- local test結果と該当するCI/E2E結果
- secret-safety上の確認
- documentation sync状況
- 意図的に`unknown`として残す制限

通常のPR integrationにはsquash mergeを使用します。required CIがgreenになる前にmergeしません。同じcontract、安全境界、user-facing behaviorを変更する場合はEnglish/Japanese document pairを原則同時に更新してください。`SECURITY.md`が定めるsecurity policyのcanonical版は英語です。

## Release / versioning

release preparationもPR-firstです。通常のrelease sequenceは次です。

1. focused release-prep PRを作成する
2. 必要に応じて`CHANGELOG.md`とEnglish/Japanese READMEのcurrent-release referenceを更新する
3. required CIがgreenになってからmergeする
4. `main`に既に含まれているcommitへrelease tagを作成する
5. `.github/workflows/release.yml`をauthoritative publication gateとして実行する
6. generated archive、`checksums.txt`、embedded version output、packaged CLI regression smokeを確認してからrelease完了とする

release workflowは、`origin/main`に含まれないcommitを指すrelease tagをrejectし、artifact publish前にsource quality/security gateを再実行します。

version numberは`v0.x`期間中もSemVerの意図に沿って扱います。

- **patch**: public contractを意図的に壊さないbug/security fix、documentation、maintenance
- **minor**: backward-compatible featureまたは意味のあるcapability追加
- **major**: project maturity上その区別が有効になった段階で、意図的なbreaking public contract向けに予約

`v0.x`はpre-1.0で進化中のcontractを意味するため、`v1.0`前にcompatibility changeが必要になる可能性はあります。ただし既知のbreaking behaviorはscopeとrelease noteで明示し、routine patch releaseへ紛れ込ませてはいけません。

## Security reporting / support

security vulnerabilityはpublic Issue/PRではなく、[SECURITY.ja.md](SECURITY.ja.md)に従ってprivateに報告してください。security reporting policyのcanonical版は[SECURITY.md](SECURITY.md)です。

bug report、interoperability report、feature request、usage questionは[SUPPORT.ja.md](SUPPORT.ja.md)とrepositoryのIssue templateを参照してください。public reportにはprivate production service identity、sensitive endpoint value、credential materialを含めないでください。
