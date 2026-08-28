# プロジェクト方針

[English](project-direction.md) | **日本語**

> この文書は英語版`project-direction.md`の日本語訳です。プロジェクト方針の正本は英語版です。

この文書は、`mcp-interop`が**何を作るプロジェクトなのか、何を優先し、何にはならないのか**を整理します。

MCP Conformance、Inspector、評価基盤、client capability matrixなど周辺ツールが増えても、役割を曖昧にしないための上位方針です。

公式MCP Conformanceとの違いは[Conformanceとmcp-interopの違い](conformance-vs-interop.ja.md)を参照してください。

## ミッション

`mcp-interop`が継続的に検証したい単位は次です。

```text
Remote MCP deployment
  × 実際に配布されているclient product
  × 正確なclient version
  × 関係するplatform / runtime context
```

主な問いはシンプルです。

> 実際にデプロイしているRemote MCPは、ユーザーが今使っているクライアント製品・バージョンから、本物のMCP経路を通して利用できるか？

長期的な価値は「対応クライアント数」ではありません。

特定の組み合わせが動いた、壊れた、退行した、あるいは証拠不足で判断できなかったことを、**再現可能で信頼できる証拠として残せること**を重視します。

## コアとなる公開契約

live resultが証明するのは、その実行で観測した範囲だけです。

- 対象endpoint
- client製品・正確なversion
- platform / runtime
- auth mode
- そのrunで得られた証拠

を超えて一般化してはいけません。

現在の公開live stageは次です。

```text
reach -> auth -> init -> tools
```

完全なPASSには、必要な各段階が実クライアントの証拠で`pass`になっている必要があります。

次をlive PASSの代わりにはしません。

- `unknown`
- `skip`
- 公開メタデータの互換性
- fixtureだけの成功
- sanitized Runtime Evidenceだけの成功

便利なfalse positiveを出すくらいなら、`unknown`や保守的な失敗を選びます。

## 証拠の階層

実装・出力・文書では、次を混ぜません。

1. specification / Conformanceの証拠
2. サーバーを直接調べたデバッグ情報
3. 製品向けPreflightメタデータ
4. デプロイ側から渡された秘密情報を含まないRuntime Evidence
5. アダプターやfixtureの測定経路を検証する証拠
6. **実際に配布されているクライアントから得た、対象デプロイ固有のlive evidence**

対象Remote MCPが対象client/versionでPASSしたと主張できるのは6だけです。

## 実クライアントの観測経路として認めるもの

公式にサポートされた自動操作インターフェースがある場合は、それを最優先します。

十分に安定し、再現でき、制約を文書化できる場合は、製品が実際に生成する観測可能な状態を利用できます。

例:

- 公開されたCLIのMCP管理コマンド
- 公開されたlocal app-server / control protocol
- MCP lifecycleやtool discoveryを直接反映する、隔離されたクライアント所有状態
- より安全なmachine interfaceがない場合の、実クライアントへの限定的なPTY操作

一方、コアPASSの判定には次を使いません。

- model promptやLLMのtool選択
- 壊れやすいbrowser DOM automation
- coverageを増やすためだけのprivate/minified endpointや内部UI command
- 通常ユーザーのcredentialのコピー・再利用
- 実ツール発見の証拠なしに、設定済みtool allowlistをdiscovered toolsとみなすこと
- server metadataや直接調査を、client側のlifecycle evidenceの代わりにすること

この境界を満たせない製品はresearch-only、または`unknown` / `skip`のままにします。

## アダプターの成熟度

### Research-only

安全なdirect boundaryがあるか調査中の状態です。

一部段階を観測できても、必要な証拠がunsafe credential reuse、model prompt、壊れやすいUI、private internalsなどに依存するならsupported adapterへ昇格しません。

### Beta

実クライアント経路が有用で再現可能でも、version / OS coverageが狭い場合はbetaとして提供できます。

betaでも少なくとも次が必要です。

- 設定・credentialの隔離
- exact client versionの記録
- platform/runtime contextの記録
- no-modelなコア実行
- `reach/auth/init/tools`の決定的または保守的な判定
- 証拠不足時の`unknown`
- bounded timeout / cancellation
- owned process / session cleanup
- secret rejection / redaction
- controlled fixture E2E
- 可能な範囲でnormal user stateのbefore/after非変更check

### Stable

1 version / 1 platformだけの偶然ではないことをexact evidenceで示せる場合だけbetaからstableへ昇格します。stableにはbeta gate全部に加えて次が必要です。

- advertised PASS pathごとに少なくとも2つのexact client versionで繰り返しevidenceがあり、連続version rangeへ補間しない
- advertisedする各OS/architecture scopeにretained real-client evidenceがあるか、supported scopeを明示的に狭める
- client measurement surfaceが十分supported、または繰り返し実測されている
- 将来のclient変更に対するexact-point compatibility / regression maintenance pathがある

Stableは「外部clientが将来変わらない」という意味ではありません。新しくinstalledされたexact versionが未観測なら引き続き`untested`で、version文字列が変わっただけでadapterを自動promote/demote/regressしません。maturity変更にはexplicit evidence reviewが必要です。

canonicalなmachine/human-readable criteriaと現在のshipped-adapter decisionは[Adapter maturity contract](adapter-maturity.ja.md)で定義します。`mcp-interop maturity`はclientを検出・実行せず、そのdecisionをreportします。

## 優先順位

### P0 — PASSの正しさを守る

最優先です。

- false live PASSを出さない
- stage / result invariantを守る
- secret safety
- process / session cleanup
- timeout / cancellation
- fixtureによる再現性
- CI / release provenance

クライアント数が増えても、これらを弱める変更は優先しません。

### P1 — 退行検出を第一級機能にする

目標:

- version付き結果・証拠schema
- endpoint、client、version、platform、auth modeを含むrun identity
- 秘密情報を含まないartifact export
- run間compare / diff
- `PASS -> AUTH FAIL`などのCI gate
- reason code変化のmachine-readable化
- hosted backend不要のlocal-file-first history

### P2 — 既存アダプターを深く検証する

新クライアント追加より先に、Codex / Cursor / Antigravityのconfidenceを高めます。

- 複数の最近のclient versionを確認
- 安全に可能ならOS/platformを拡大
- known-good / known-badな観測点を記録
- クライアント側の出力変更を保守的に検出
- 各platformでfixture / cleanup gateを維持

浅いアダプターを多数持つより、深く信頼できるアダプターを優先します。

### P3 — 新しいクライアントは選択的に追加する

人気があるだけでは追加理由になりません。

少なくとも次を確認します。

1. 通常ユーザー状態を隔離できるか
2. model promptなしにRemote MCPへ到達できるか
3. authを安全に完了・分類できるか
4. initialization / protocol readinessを観測できるか
5. 実tool discoveryを証明できるか
6. exact version / platformを取得できるか
7. sessionを確実に片付けられるか
8. controlled fixtureで測定経路を証明できるか

満たせなければresearch-onlyです。

### P4 — 必要な製品固有診断だけ追加する

`diagnose --profile <product>`は、実際のdeployment/client mismatchを説明できる場合に追加します。

第二のgeneric OAuth/MCP Conformance suiteにはしません。

## 周辺ツールとの役割分担

### 公式MCP Conformance

仕様適合性を担当します。`mcp-interop`は独自certification layerを作りません。

### Inspector / testing・evaluation platform

server debugging、playground、model evaluationなど広い開発体験を担当します。

### Static client capability matrix

「製品Xが一般に何をsupportするか」を整理する用途です。テスト対象選定には使えますが、特定デプロイのlive evidenceではありません。

### Security scanner / governance product

脆弱性検査、policy、permission、sandboxingなどを担当します。接続PASSをsecurity resultにしません。

### LLM / tool-selection benchmark

モデルがツールを適切に選ぶかを評価します。コアinterop PASSとは分離します。

## 周辺ecosystemが変わった場合

主要な別プロジェクトが次をすべて第一級機能として提供するようになった場合は、このプロジェクトの位置付けを再評価します。

- 実際に配布されているクライアント製品を起動する
- 任意のRemote MCP deploymentを対象にできる
- client config / credentialを隔離する
- exact version / platformを記録する
- model promptなしにreach/auth/init/tool discoveryを証明する
- 比較可能なmachine-readable evidenceをexportする

その場合、重複を守ること自体を目的にせず、より強いupstreamとの統合や貢献を選ぶ可能性があります。

## 判断に迷ったときの順序

成長と証拠品質が衝突した場合は、次の順で優先します。

1. PASSの意味
2. ユーザーのcredential / local state
3. 再現性とcleanup
4. exact client/version/platform provenance
5. regression detection
6. version / OS coverage
7. client追加
8. convenience UX

## 現在の方向性

具体的なmilestoneは[Roadmap](roadmap.ja.md)で管理します。

直近では次に集中します。

1. real-client-only PASSを弱めず、MCP protocolの世代差を扱えるcoreへする
2. portable artifactを日常的に共有する前にdeployment identity privacyを整理する
3. artifact / compareを安全で繰り返し可能なregression workflowへ発展させる
4. Codex / Cursor / Antigravityをversion/platform横断で強化する
5. ChatGPT / VS Code / GitHub Copilot CLIは、必要なdirect automation boundaryを満たすまでresearch-onlyにする
6. public contractは実運用で十分に検証してからstable化する

目指すのは「最も多くのMCPクライアントに対応するツール」ではありません。

**実際のRemote MCPが、ユーザーの実クライアント・実バージョンで今も動くかを、最も信頼できる形で答えること**が目標です。
