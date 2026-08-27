# Self-hosted real-client CI security boundary

[English](self-hosted-ci-security.md) | **日本語**

> この文書は英語版`self-hosted-ci-security.md`の日本語訳です。契約の正本は英語版です。

`mcp-interop`はpublic repositoryです。GitHubはself-hosted runnerがephemeral clean machineではなく、特にpublic repositoryでは高リスクだと明示しています。そのためself-hosted macOS real-client workflowは**privileged trusted path**として扱い、通常のPull Request CIには使いません。

## 現在の境界

通常の`pull_request` CIはGitHub-hosted Ubuntu/macOS/Windows runnerだけで実行し、`self-hosted` labelを使いません。

`.github/workflows/e2e-real-macos.yml`はmanual-onlyで、次の多層gateを持ちます。

1. triggerは`workflow_dispatch`のみ
2. self-hosted jobのGitHub側`if`でcanonical repository、`refs/heads/main`、`main`上のworkflow file、`github.workflow_sha == github.sha`を要求。job-level `if`はrunnerへjobを送る前にGitHub側で評価される
3. jobはGitHub Environment `real-client-e2e`を参照。repository設定のcustom deployment branch policyは`main`だけを許可
4. checkoutはexact `github.sha`へ固定し、`persist-credentials: false`
5. runner上の`scripts/guard-real-client-e2e.sh`でもrepository/ref/workflow/SHAを再検証し、self-hosted macOS ARM64 contextも要求
6. dispatchの`clients`はfree-form textではなく固定`choice`。runner guardもuniqueな`codex` / `cursor` / `antigravity`だけを許可
7. workflowはcontrolled localhost real-client fixtureだけを使う。endpoint URL、suite manifest path、shell command、executable path、environment override、OAuth option、credential inputは受け取らない
8. repository/ref/SHA、workflow ref/SHA、run ID/attempt、actor、選択client、観測client versionを含むprivate provenance JSONを作り、30日workflow artifactとして保存

Environment branch policyはGit fileではなくrepository設定です。maintainerは`real-client-e2e`が`main`だけを許可していることを定期確認してください。

## Remote suite executionとは意図的に接続しない

`mcp-interop suite run`はlocalで`trusted_real_client` manifestを実行できますが、v0.7では任意remote suite executionをself-hosted GitHub Actionsへ**接続しません**。これは意図した設計です。

将来`MCP_INTEROP_SUITE_ENDPOINT_*`を解決したりproduction-equivalent credentialを使うworkflowを追加する場合、privileged network / credential boundaryが拡大するため新しいsecurity reviewが必要です。untrusted Pull Request contentからendpoint値やrunnerをredirectできるmanifestを選択できてはいけません。

## OAuth

manual self-hosted release gateは`--oauth`を渡しません。OAuthはlocal/operatorのexplicit opt-inのままで、既存のisolated credential boundaryを維持します。OAuth credentialやbrowser/session stateをself-hosted CIへ載せることはv0.7 contract外です。

## Runner運用

repository workflow controlだけではpersistent self-hosted machineはsandboxになりません。runnerはdedicated/minimalを推奨し、normal-user browser/client credentialや無関係なsecretを置かず、controlled testに必要なnetwork accessだけへ絞るべきです。可能ならephemeral/disposable runnerを優先します。

organization/enterprise shared runnerを使う場合、GitHub planが対応していればrunner groupを対象repository/workflowだけへ制限してください。

## Threat modelの限界

このcontrolは**untrusted Pull Request / branch content**がordinary CIや誤操作からprivileged runnerへ到達することを防ぐためのものです。trusted `main`、workflow、Environment policy、runner configuration自体を変更できるmalicious/compromised repository admin/write actorまでは防御対象にしません。

## Audit / test gate

`scripts/test-real-client-e2e-guard.sh`はpull-request event、non-main ref、wrong repository、workflow ref/SHA mismatch、hosted runner context、invalid client inputをrejectすることを検証します。workflowがmanual-only/main-gatedであることとprovenance shapeも確認します。このtestはself-hosted runnerを使わずhosted CIとrelease workflowで実行します。
