# セキュリティポリシー

[English](SECURITY.md) | **日本語**

> この日本語版は参考訳です。セキュリティ報告に関する正式なポリシーは英語版`SECURITY.md`を正とします。

## 脆弱性を報告する場合

セキュリティ上の問題が疑われる場合は、**公開Issueを作成しないでください**。

このリポジトリのGitHub Private Vulnerability Reportingを利用してください。

1. リポジトリの **Security** タブを開く
2. **Report a vulnerability** を選ぶ
3. 影響するバージョンまたはcommit、再現手順、想定される影響、分かる場合は緩和策を記載する

実際のcredential、OAuth authorization code、access/refresh token、cookieなどの秘密情報は報告へ含めないでください。credential形式のデータが必要な再現では、syntheticまたは失効済みの値を使ってください。

## 特に報告してほしい問題

次のような問題は、セキュリティ報告の対象として特に重要です。

- OAuth / Bearer credentialが出力・ログ・artifactへ漏れる
- ユーザーが普段使っているMCPクライアント設定を意図せず変更する
- 一時credentialや一時設定の削除に失敗する
- Remote MCPのメタデータやURLを経由してコマンド・引数を注入できる
- クライアントが生成したauthorization URLを危険な形で扱う
- テスト用セッションと通常のクライアントprofileが十分に隔離されていない
- reportやlogの秘密情報除去を回避できる

## サポート対象バージョン

セキュリティ修正は最新の`main`を基準に開発します。

現在の`v0.x`系列では、報告された問題の深刻度、悪用可能性、公開リリース利用者への影響を見て、修正版のtagged releaseが必要か判断します。

原則として最新の公開release lineを主なサポート対象とし、古い`v0.x`へのbackportは保証しません。

公開artifactへ影響する問題を修正版releaseで対応した場合は、advisoryまたはrelease noteで修正済みバージョンを明示します。
