# Repository protection policy

`main`はGitHub repository Ruleset **Protect main** で保護します。この保護はGitHub repository settingであり、Git管理ファイルだけでは強制できません。

`refs/heads/main`のactive policy:

- branch削除を禁止
- non-fast-forward / force pushを禁止
- main変更はPull Request経由
- solo maintainerをdeadlockさせないためrequired approvalは0
- review threadの解決を必須化
- merge methodはsquashのみ
- required status checksは`test (ubuntu-latest)`、`test (macos-latest)`、`test (windows-latest)`
- linear history必須
- bypass actorなし

privilegedな`real-client-e2e` Environmentは別途`main`のみに制限します。workflow側でもrepository identity、exact main ref / workflow provenance、trusted self-hosted runner boundaryを検証します。

repository settingの移行・再作成・transferを行った場合はstable release前にRulesetを再監査してください。classic branch protection APIが未設定でも、Rulesetが有効な場合があるため、それだけで`main`が無保護とは判断しません。
