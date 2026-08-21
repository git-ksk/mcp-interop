# サポート

[English](SUPPORT.md) | **日本語**

> この文書は英語版`SUPPORT.md`の日本語訳です。内容に差がある場合は英語版を正とします。

`mcp-interop`はコミュニティで管理しているオープンソースプロジェクトです。商用SLAや個別のプライベートサポートは提供していません。

## 問い合わせ先

- **不具合 / 相互運用性の問題** — GitHub Issueで、bug reportまたはreal-client interoperability用のtemplateを使ってください。
- **機能追加・改善の提案** — feature request templateを使ってください。
- **使い方に関する質問** — question templateを使ってください。
- **セキュリティ上の問題** — 公開Issueを作らず、[SECURITY.ja.md](SECURITY.ja.md)に従ってGitHub Private Vulnerability Reportingを利用してください。正式なポリシーは[SECURITY.md](SECURITY.md)です。

このリポジトリではGitHub Discussionsを有効にしていないため、サポート窓口として案内していません。

## 診断情報を共有する前に

次の情報は削除してください。

- access / refresh token
- OAuth authorization code / `state`
- 生のauthorization URL
- cookie
- client secret
- private key
- credential fileの内容
- 機密性のあるendpoint query値

不具合の再現に必要でも、秘密情報そのものを公開Issueへ貼らないでください。
