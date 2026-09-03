# Security, privacy, cleanup, and release contract v1

[English](security-contract-v1.md) | **日本語**

この文書はstable v1.x contractで維持するminimum security / operational guaranteeを定義します。将来releaseがunsafe inputをより厳しくrejectしたりisolationを強化することはcompatibility regressionとは扱いません。

## Secret-safe evidence

client/endpointがexecution中にcredential materialを出してもportable evidenceへpersistしません。

- access/refresh token、authorization code、OAuth state、PKCE secret、client secret、private key、cookie、credential-file contentはportable evidence fieldにしない
- unknown secret-bearing Runtime Evidence fieldはfail closed
- free-form client/remote errorはordinary outputへ出る前にredact
- portable artifact fileはprivate modeでwriteし、symlinkをfollowせず置換
- schema-v2 protected-path identityはprotected endpoint pathをpersistもhashもしない

schema v1はread継続しますがendpoint path自体がidentityなのでnon-secretである必要があります。credential-bearing pathではnormal authenticationとschema v2 protected-path identityを使います。

**credential-safe != deployment-public**です。schema v2にもcanonical originは残るため、private/sensitive hostnameをpath保護だけでpublicにしてよいとは限りません。

## Credential isolation

real-client PASSのためにnormal-user token、browser cookie、Keychain entry、persistent credential fileをtest profileへcopyしません。

shipped adapter evidence pathはisolated temporary stateまたはdeliberately accepted bounded client surfaceを使います。controlled real-client release gateでは関連normal-user config/credential metadataをbefore/after比較します。必要なauthenticated boundaryを安全にisolateできないresearch candidateはresearch-onlyのままです。

dedicated test credentialは明示的にisolatedなresearch/test flowだけで利用でき、ordinary user credentialコピーの根拠にはなりません。

## OAuth material

interactive OAuthはopt-inです。authorization navigation情報はexplicit interactive flow内だけで表示できます。token、callback code、refresh token、PKCE verifier、client secretをportable evidence/log artifactへpersistしません。

Tool OAuth release fixtureはsynthetic credentialを使い、secret materialがretained evidenceへ漏れないことをassertします。

## Owned cleanup

cleanupはtest ownershipでboundedです。

- runが作成したtemporary directory/configはsuccess/failure両方でremove
- process cleanupはtest harness所有のdescendant/sessionだけを対象にし、名前一致だけでarbitrary user processをkillしない
- isolationをclaimするadapterではnormal user stateを変更しない
- core 4 stageがPASSでもcleanup/isolation failureはtest/release-gate failure

## Privileged real-client CI boundary

self-hosted real-client workflowはordinary pull-request CIと分離します。

次を維持します。

- manual `workflow_dispatch` only
- repository/main/exact-workflow-SHA guard
- `real-client-e2e` GitHub Environment保護
- labeled self-hosted macOS arm64 runner限定
- exact trusted SHA checkout + `persist-credentials: false`
- execution前に`guard-real-client-e2e.sh`
- fixed shipped-client choiceとcontrolled fixture execution限定

untrusted PR contentからprivileged runnerをarbitrary endpoint/command/credential stateへ向けられないようにします。

## Ordinary CI boundary

Pull Request CIはGitHub-hosted runnerとread-only repository contents permissionを使います。unit/fixture/release-smokeは実行できますが、privileged real-client workflowやproduction targetは実行しません。

format、vet、unit、race、vulnerability、fixture、trust-guard、release archive smoke gateを維持します。

## Tagged release guarantee

Tagged releaseでは次を維持します。

- tag syntax validationとtagged commitが`origin/main`に含まれることの確認
- GitHub Actionsのfull SHA pin
- format / vet / unit / race / `govulncheck`
- real-client trust guardとcontrolled OAuth fixture gate
- deterministic 6-target archive build + checksums
- native runner上でのembedded-version / packaged CLI smoke（Linux amd64、macOS arm64、macOS amd64、Windows amd64。arm64 Linux/Windowsはcross-build + checksumのまま明示的に未native-smoke）
- OIDC permissionを使うGitHub artifact attestation
- verified-tag GitHub Release作成

`release.yml` / `e2e-real-macos.yml` / `ci.yml`は`scripts/test-security-contract.sh`で検証し、これらの境界が誤って消えるとCI failureになります。

## Provenance boundary

GitHub release attestationはGitHub attestation modelに従ってrelease workflowのpublished build artifactをauthenticateします。ordinary local live-result artifactやbaseline fingerprintまでauthenticated execution attestationになるわけではありません。

local baseline/result fingerprintはschema contractに記載したcontent identity/consistencyの狭い意味を維持します。

## Security-driven compatibility exception

既存accepted input/behaviorにcredential exposure、untrusted privileged execution、その他具体的security defectが見つかった場合、stable releaseでもmajor versionを待たずtighten/rejectできます。その変更はrelease noteへ明示しfail closedにします。unsafe old evidenceを黙ってvalid PASSへ読み替えません。
Repository-level `main` protection is an external GitHub setting; the maintained policy is documented in [Repository protection policy](repository-protection.ja.md).

