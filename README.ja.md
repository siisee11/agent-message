<p align="center">
  <img src="agent-message-logo.svg" alt="Agent Message logo" width="88">
</p>

# Agent Message

[English](README.md) | [한국어](README.ko.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md)

<p align="center">
  <img src="docs/readme-screenshot.png" alt="Agent Message screenshot" width="900">
</p>

Agent Message は、エージェントと人間が同じ DM フローで作業するためのメッセージングスタックです。

- HTTP/SSE サーバー (`server/`)
- Web アプリ (`web/`)
- CLI (`cli/`)

## Agent Message を使う理由

- メッセンジャーが CLI として提供されているため、エージェントが直接メッセージを送信、読み取り、監視できます。
- `json_render` により、カード、テーブル、バッジ、進捗ブロックなどの読みやすい構造化メッセージを受け取れます。
- Web アプリをスマートフォンで開き、エージェントとの作業を継続できます。
- `codex-message` と `claude-message` は Codex と Claude のセッションを同じ DM ワークフローに接続します。

## Wrapper パッケージ

- `codex-message`: Codex app-server セッションを開始し、Agent Message DM で会話を中継します。
- `claude-message`: Claude CLI セッションを開始し、プロンプト、失敗、最終返信を Agent Message で中継します。

ランディングページは `https://amessage.dev` で公開されています。ホスト型クラウドサービスはまだ準備中です。現時点では self-hosted local stack の利用を推奨します。

## 対応プラットフォーム

現在 npm で公開されているパッケージは macOS ビルドのみです。

| プラットフォーム | アーキテクチャ | `agent-message` | `codex-message` | `claude-message` | 備考 |
| --- | --- | --- | --- | --- | --- |
| macOS | Apple Silicon (`arm64`) | 対応 | 対応 | 対応 | 主なパッケージ対象 |
| macOS | Intel (`x64`) | 対応 | 対応 | 対応 | パッケージ対象 |
| Linux | `x64` / `arm64` | 未パッケージ | 未パッケージ | 未パッケージ | ソースビルドのみ |
| Windows | `x64` / `arm64` | 未パッケージ | 未パッケージ | 未パッケージ | 現在未対応 |

## Setup Prompt

Claude Code または Codex に貼り付けてください。

```bash
Set up https://github.com/siisee11/agent-message for me.

Read `install.md` and follow the self-host setup flow. Ask me for the account-id before registering, use 0000 only as the temporary initial password, remind me to change it immediately, set the master recipient, and send me a welcome message with agent-message when setup is complete.
```

## クイックセットアップ

クラウドサービスのアカウントはまだ利用できません。現時点では self-hosted local stack を使用してください。

エージェントに Agent Message のインストールを任せる場合は、上の Setup Prompt を使ってください。

手動でインストールする場合は npm を使います。

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -g -y
npm install -g agent-message
agent-message start
agent-message status
```

次に、ローカルアカウントを作成またはログインします。登録前にユーザーへ `account-id` を確認してください。`0000` は一時的な初期パスワードとしてのみ使います。

```bash
agent-message register <account-id> 0000
# すでにアカウントがある場合:
agent-message login <account-id> 0000
```

ブラウザで `http://127.0.0.1:45788` を開き、Profile ページからすぐにパスワードを変更してください。
`agent-message start` はローカルスタックを起動し、CLI の通信先が起動済み API (`http://127.0.0.1:45180`) になるように `~/.agent-message/config` を更新します。

公開表示名と、エージェントのステータス報告を受け取る既定の宛先を設定します。

```bash
agent-message username set <username>
agent-message config set master <recipient-username>
agent-message whoami
```

## npm でインストール (macOS)

macOS (`arm64` と `x64`) では npm のパッケージ版をインストールできます。

```bash
npm install -g agent-message
agent-message start
agent-message status
agent-message stop
agent-message upgrade
```

既定ポート:
- API: `127.0.0.1:45180`
- Web: `127.0.0.1:45788`

self-hosted local で使用する場合、`agent-message start` は既定でローカル SQLite データベースを作成して使用します。ホスト型クラウドサービスはまだ準備中です。将来の managed cloud デプロイでは `DB_DRIVER=postgres` と `POSTGRES_DSN` でサーバーを実行します。

通常の CLI コマンドも同じ `agent-message` コマンドから使えます。

```bash
agent-message register alice 0000
agent-message login alice 0000
agent-message username set jay-ui-bot
agent-message config set master jay
agent-message ls
agent-message open bob
agent-message send bob "hello"
agent-message send "status update for master"
```

必要に応じてランタイムディレクトリとポートを変更できます。

```bash
agent-message start --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message status --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message stop --runtime-dir /tmp/agent-message
```

## Web クイックスタート

```bash
cd web
npm ci
npm run dev
```

ローカル開発では、Vite が `/api/...` と `/static/uploads/...` を `http://localhost:8080` にプロキシします。API が別 origin にある場合は `VITE_API_BASE_URL` を設定してください。

```bash
cd web
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

ビルド確認:

```bash
cd web
npm run build
```

## Cloudflare Workers Web デプロイ

現在の React Web アプリは Cloudflare Workers の static assets としてデプロイできます。まず公開 Web 面を出す用途で、API ベースのクラウドサービスは後から接続できます。

```bash
cd web
npm ci
npm run deploy:worker
```

クラウド API の準備ができるまでは、Worker の API リクエストは `503` を返します。API origin が用意できたら Worker に `API_ORIGIN` を設定して再デプロイしてください。

```bash
cd web
npx wrangler secret put API_ORIGIN
npm run deploy:worker
```

たとえば `API_ORIGIN=https://api.amessage.dev` を設定すると、Web アプリは同一 origin の `/api/...` 呼び出しを維持し、Worker が API サービスへプロキシします。

## ソースから実行

チェックアウトしたリポジトリを `agent-message` として `PATH` に公開するには、リポジトリルートで次を実行します。

```bash
npm link
```

ローカルソースツリーから起動する場合は `--dev` を追加します。

```bash
agent-message start --dev
agent-message stop --dev
```

前提条件:
- Go `1.26+`
- Node.js `18+` と npm
- Docker + Docker Compose (PostgreSQL compose フロー用)

## サーバークイックスタート

SQLite ローカルサーバー:

```bash
cd server
go run .
```

PostgreSQL を含むローカル production-like stack:

```bash
make stack-up
docker compose down
```

## Home Server Container Deploy

家庭用 Mac サーバーでは、self-hosted stack 全体をコンテナで実行できます。

```bash
cp .env.home.example .env.home
make publish
docker compose --env-file .env.home -f docker-compose.home.yml ps
docker compose --env-file .env.home -f docker-compose.home.yml logs -f
```

必須値:
- `APP_HOSTNAME`
- `POSTGRES_PASSWORD`
- `CLOUDFLARE_TUNNEL_TOKEN`

stack に含まれるもの:
- `postgres`
- `server`
- `gateway`
- `cloudflared`

Mac で host port を公開する必要はありません。公開トラフィックは Cloudflare Tunnel 経由にします。

## Claude Code Skill

Agent Message CLI skill をインストールすると、Claude Code がこのプロジェクトの CLI コマンド、フラグ、`json_render` コンポーネントカタログを理解できます。

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli
```

## codex-message

`codex-message` は Codex のサンプルアプリです。`codex app-server` をラップし、`agent-message` を DM transport として使用します。

```bash
npm install -g agent-message codex-message
```

前提条件:
- `agent-message` がインストール済みでログイン済み
- 対象ユーザーがすでに `agent-message` アカウントを持っている
- `codex` CLI がインストール済みで認証済み

典型的な設定:

```bash
agent-message config set master jay
codex-message --model gpt-5.4 --cwd /path/to/worktree
codex-message --model gpt-5.4 --cwd /path/to/worktree --yolo
codex-message --to alice --model gpt-5.4 --cwd /path/to/worktree
codex-message --bg --model gpt-5.4 --cwd /path/to/worktree
```

実行後:
- セッションごとに新しい `agent-{chatId}` アカウントを作成します。
- 生成された認証情報を含む開始 DM を対象ユーザーに送信します。
- 1 つの Codex app-server thread をその DM 会話に接続します。
- 通常テキストの DM を Codex に中継します。
- 承認、入力、失敗、状態プロンプトは wrapper が `json_render` として返します。
- 最終的なユーザー向け結果は Codex が `agent-message send --from agent-{chatId}` で直接送るよう指示されます。

便利なフラグ:
- `--to <username>`
- `--cwd /path/to/worktree`
- `--model gpt-5.4`
- `--approval-policy on-request`
- `--sandbox workspace-write`
- `--network-access`
- `--yolo`
- `--bg`

## claude-message

`claude-message` は Claude のサンプルアプリです。`claude -p --output-format json` を実行し、セッションを `agent-message` で中継します。

```bash
npm install -g agent-message claude-message
```

前提条件:
- `agent-message` がインストール済みでログイン済み
- 対象ユーザーがすでに `agent-message` アカウントを持っている
- `claude` CLI がインストール済みで認証済み

例:

```bash
claude-message --to jay --model sonnet --permission-mode accept-edits
claude-message --bg --to jay --model sonnet --permission-mode accept-edits
```

設定は `codex-message` と似ています。wrapper は一時的な `agent-{chatId}` アカウントを作成し、同じ DM 会話で通常テキストの DM を待ちます。成功した turn ではエージェントが最終的なユーザー向け結果を直接送信し、wrapper は開始メッセージ、reaction、失敗通知を担当します。

便利なフラグ:
- `--to jay`
- `--cwd /path/to/worktree`
- `--model sonnet`
- `--permission-mode accept-edits`
- `--allowed-tools Read,Edit`
- `--bare`
- `--bg`

## CLI クイックスタート

self-hosting では、まず `agent-message start` でローカルスタックを起動するか、`--server-url` または `config set server_url` で source CLI をローカル API に向けます。

```bash
cd cli
go run . --server-url http://localhost:8080 register alice 0000
go run . --server-url http://localhost:8080 login alice 0000
go run . username set jay-ui-bot
go run . profile list
go run . profile switch alice
```

一般的なコマンド:

```bash
go run . ls
go run . open bob
go run . send bob "hello"
go run . send bob --attach ./screenshot.png
go run . read bob --n 20
go run . edit 1 "edited text"
go run . delete 1
go run . react <message-id> 👍
go run . unreact <message-id> 👍
go run . watch bob
```

CLI config は既定で `~/.agent-message/config` に保存されます。`onboard` は人間向けの対話型コマンドです。エージェントによるセットアップは `install.md` に従い、skill のインストール、npm パッケージのインストール、ローカルスタックの起動、`account-id` の確認、`register` または `login` の順に進めます。

## 検証と制約

- Account ID と username: `3-32` 文字、許可文字 `[A-Za-z0-9._-]`
- Password: `4-72` 文字
- アップロード最大ファイルサイズ: `20 MB`
- サポート外のファイル形式は拒否されます

## 開発チェック

```bash
cd server
go test ./...

cd cli
go test ./...

cd web
npm run build
```
