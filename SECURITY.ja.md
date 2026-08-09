# セキュリティポリシー

[English](SECURITY.md) | [日本語](SECURITY.ja.md)

**この日本語版は参考訳です。セキュリティ報告に関する正式なポリシーは英語版`SECURITY.md`を正とします。**

## 脆弱性の報告

セキュリティ脆弱性の疑いがある場合、**public Issueを作成しないでください**。

このrepositoryのGitHub Private Vulnerability Reportingを使用してください。

1. repositoryの**Security**タブを開く。
2. **Report a vulnerability**を選ぶ。
3. 影響するversionまたはcommit、再現手順、想定される影響、分かる場合は緩和策を記載する。

live credential、OAuth authorization code、access/refresh token、cookie、その他secretを報告へ含めないでください。credential形式のデータが必要な再現ではsyntheticまたはrevoked credentialを使用してください。

## 対象範囲

特に次の問題はsecurity reportの対象として重要です。

- OAuth/Bearer credentialの漏えい
- ユーザーが普段使うMCP client設定の意図しない変更
- temporary credential/configのcleanup失敗
- Remote MCP metadataやURLを経由したcommand/argument injection
- client生成authorization URLの危険な取り扱い
- test sessionと通常client profile間のisolation不備
- report/logのredaction bypass

## サポート対象version

security fixは最新の`main` branchを基準に開発します。

現在の`v0.x` seriesでは、報告された脆弱性について、severity、exploitability、公開release利用者への影響を基にpatched tagged releaseが必要か判断します。原則として最新の公開release lineを主要なサポート対象とし、古い`v0.x` releaseへのbackportは保証しません。

公開artifactへ影響する脆弱性を修正版releaseで対応した場合、advisoryまたはrelease notesでfixed versionを明示します。
