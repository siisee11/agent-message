<p align="center">
  <img src="agent-message-logo.svg" alt="Agent Message logo" width="88">
</p>

# Agent Message

[English](README.md) | [한국어](README.ko.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md)

<p align="center">
  <img src="docs/readme-screenshot.png" alt="Agent Message screenshot" width="900">
</p>

Agent Message 是讓代理程式與人類在同一個 DM 流程中協作的訊息堆疊。

- HTTP/SSE 伺服器 (`server/`)
- Web 應用程式 (`web/`)
- CLI (`cli/`)

## 為什麼使用 Agent Message

- 訊息工具以 CLI 形式提供，代理程式可以直接傳送、讀取與監看訊息。
- 透過 `json_render`，訊息可以用卡片、表格、徽章、進度區塊等易讀的結構化格式呈現。
- 你可以在手機上開啟 Web 應用程式，持續與代理程式協作。
- `codex-message` 與 `claude-message` 會把 Codex 和 Claude 會話接入同一套 DM 工作流程。

## Wrapper 套件

- `codex-message`：啟動 Codex app-server 會話，並透過 Agent Message DM 轉送對話。
- `claude-message`：啟動 Claude CLI 會話，並轉送提示、失敗訊息與最終回覆。

Landing page 位於 `https://amessage.dev`。託管雲端服務仍在準備中；目前建議使用 self-hosted local stack。

## 支援平台

目前發佈到 npm 的套件只包含 macOS 建置。

| 平台 | 架構 | `agent-message` | `codex-message` | `claude-message` | 備註 |
| --- | --- | --- | --- | --- | --- |
| macOS | Apple Silicon (`arm64`) | 支援 | 支援 | 支援 | 主要打包目標 |
| macOS | Intel (`x64`) | 支援 | 支援 | 支援 | 打包目標 |
| Linux | `x64` / `arm64` | 未打包 | 未打包 | 未打包 | 僅可從原始碼建置 |
| Windows | `x64` / `arm64` | 未打包 | 未打包 | 未打包 | 目前不支援 |

## Setup Prompt

貼到 Claude Code 或 Codex：

```bash
Set up https://github.com/siisee11/agent-message for me.

Read `install.md` and follow the self-host setup flow. Ask me for the account-id before registering, use 0000 only as the temporary initial password, remind me to change it immediately, set the master recipient, and send me a welcome message with agent-message when setup is complete.
```

## 快速設定

雲端服務帳號目前尚不可用。現在請使用 self-hosted local stack。

如果由代理程式替你安裝 Agent Message，請使用上方的 Setup Prompt。

也可以用 npm 手動安裝：

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -g -y
npm install -g agent-message
agent-message start
agent-message status
```

接著建立或登入本機帳號。註冊前先詢問使用者的 `account-id`。`0000` 只能作為暫時初始密碼。

```bash
agent-message register <account-id> 0000
# 如果帳號已存在：
agent-message login <account-id> 0000
```

在瀏覽器開啟 `http://127.0.0.1:45788`，並立即在 Profile 頁面修改密碼。
`agent-message start` 會啟動本機堆疊，並更新 `~/.agent-message/config`，讓 CLI 流量指向已啟動的 API：`http://127.0.0.1:45180`。

設定公開顯示名稱，以及代理程式狀態報告的預設接收者：

```bash
agent-message username set <username>
agent-message config set master <recipient-username>
agent-message whoami
```

## 使用 npm 安裝 (macOS)

macOS (`arm64` 與 `x64`) 可以安裝 npm 打包版本。

```bash
npm install -g agent-message
agent-message start
agent-message status
agent-message stop
agent-message upgrade
```

預設連接埠：
- API：`127.0.0.1:45180`
- Web：`127.0.0.1:45788`

self-hosted local 使用時，`agent-message start` 預設會建立並使用本機 SQLite 資料庫。託管雲端服務仍在準備中。未來的 managed cloud 部署應使用 `DB_DRIVER=postgres` 與 `POSTGRES_DSN` 執行伺服器。

常用 CLI 命令仍透過同一個 `agent-message` 命令使用：

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

必要時可以覆寫執行目錄與連接埠：

```bash
agent-message start --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message status --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message stop --runtime-dir /tmp/agent-message
```

## Web 快速開始

```bash
cd web
npm ci
npm run dev
```

本機開發時，Vite 會把 `/api/...` 與 `/static/uploads/...` 代理到 `http://localhost:8080`。如果 API 在另一個 origin，請設定 `VITE_API_BASE_URL`：

```bash
cd web
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

建置檢查：

```bash
cd web
npm run build
```

## Cloudflare Workers Web 部署

目前 React Web 應用程式可以部署為 Cloudflare Workers static assets。這主要用於先發布公開網站；之後可再接入 API 支援的雲端服務。

```bash
cd web
npm ci
npm run deploy:worker
```

雲端 API 準備好之前，Worker 的 API 請求會回傳 `503`。當 API origin 可用後，在 Worker 上設定 `API_ORIGIN` 並重新部署：

```bash
cd web
npx wrangler secret put API_ORIGIN
npm run deploy:worker
```

例如設定 `API_ORIGIN=https://api.amessage.dev` 後，Web 應用程式仍使用同源 `/api/...` 呼叫，Worker 會把請求代理到 API 服務。

## 從原始碼執行

若要讓目前 checkout 在 `PATH` 上以 `agent-message` 使用，請在儲存庫根目錄執行：

```bash
npm link
```

從本機原始碼樹啟動時加上 `--dev`：

```bash
agent-message start --dev
agent-message stop --dev
```

前置條件：
- Go `1.26+`
- Node.js `18+` 和 npm
- Docker + Docker Compose（用於 PostgreSQL compose 流程）

## 伺服器快速開始

SQLite 本機伺服器：

```bash
cd server
go run .
```

包含 PostgreSQL 的本機 production-like stack：

```bash
make stack-up
docker compose down
```

## Home Server Container Deploy

在家用 Mac 伺服器上，可以完全用容器執行 self-hosted stack。

```bash
cp .env.home.example .env.home
make publish
docker compose --env-file .env.home -f docker-compose.home.yml ps
docker compose --env-file .env.home -f docker-compose.home.yml logs -f
```

必填值：
- `APP_HOSTNAME`
- `POSTGRES_PASSWORD`
- `CLOUDFLARE_TUNNEL_TOKEN`

該 stack 包含：
- `postgres`
- `server`
- `gateway`
- `cloudflared`

Mac 上不需要暴露 host port。公開流量應透過 Cloudflare Tunnel 進入。

## Claude Code Skill

安裝 Agent Message CLI skill 後，Claude Code 可以了解本專案的 CLI 命令、參數與 `json_render` 元件目錄。

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli
```

## codex-message

`codex-message` 是 Codex 範例應用程式。它包裝 `codex app-server`，並使用 `agent-message` 作為 DM transport。

```bash
npm install -g agent-message codex-message
```

前置條件：
- 已安裝並登入 `agent-message`
- 目標使用者已有 `agent-message` 帳號
- 已安裝並認證 `codex` CLI

典型設定：

```bash
agent-message config set master jay
codex-message --model gpt-5.4 --cwd /path/to/worktree
codex-message --model gpt-5.4 --cwd /path/to/worktree --yolo
codex-message --to alice --model gpt-5.4 --cwd /path/to/worktree
codex-message --bg --model gpt-5.4 --cwd /path/to/worktree
```

執行後：
- 每個會話都會建立新的 `agent-{chatId}` 帳號。
- 向目標使用者傳送包含產生憑證的啟動 DM。
- 將一個 Codex app-server thread 綁定到該 DM 對話。
- 普通文字 DM 會被轉送給 Codex。
- 核准、輸入、失敗與狀態提示由 wrapper 以 `json_render` 傳回。
- 最終面向使用者的結果由 Codex 使用 `agent-message send --from agent-{chatId}` 直接傳送。

常用參數：
- `--to <username>`
- `--cwd /path/to/worktree`
- `--model gpt-5.4`
- `--approval-policy on-request`
- `--sandbox workspace-write`
- `--network-access`
- `--yolo`
- `--bg`

## claude-message

`claude-message` 是 Claude 範例應用程式。它執行 `claude -p --output-format json`，並透過 `agent-message` 中繼會話。

```bash
npm install -g agent-message claude-message
```

前置條件：
- 已安裝並登入 `agent-message`
- 目標使用者已有 `agent-message` 帳號
- 已安裝並認證 `claude` CLI

範例：

```bash
claude-message --to jay --model sonnet --permission-mode accept-edits
claude-message --bg --to jay --model sonnet --permission-mode accept-edits
```

該設定與 `codex-message` 類似：wrapper 會建立暫時 `agent-{chatId}` 帳號，並在同一個 DM 對話中等待普通文字訊息。成功的 turn 會由代理程式直接傳送最終使用者結果；wrapper 負責啟動訊息、reaction 和失敗通知。

常用參數：
- `--to jay`
- `--cwd /path/to/worktree`
- `--model sonnet`
- `--permission-mode accept-edits`
- `--allowed-tools Read,Edit`
- `--bare`
- `--bg`

## CLI 快速開始

self-hosting 時，先用 `agent-message start` 啟動本機堆疊，或用 `--server-url` / `config set server_url` 讓 source CLI 指向本機 API。

```bash
cd cli
go run . --server-url http://localhost:8080 register alice 0000
go run . --server-url http://localhost:8080 login alice 0000
go run . username set jay-ui-bot
go run . profile list
go run . profile switch alice
```

常用命令：

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

CLI 設定預設存放在 `~/.agent-message/config`。`onboard` 是面向人的互動式命令。代理程式安裝應遵循 `install.md`：安裝 skill、安裝 npm 套件、啟動本機堆疊、詢問 `account-id`，然後使用 `register` 或 `login`。

## 驗證與限制

- Account ID 和 username：`3-32` 個字元，允許 `[A-Za-z0-9._-]`
- Password：`4-72` 個字元
- 上傳最大檔案大小：`20 MB`
- 不支援的檔案類型會被拒絕

## 開發檢查

```bash
cd server
go test ./...

cd cli
go test ./...

cd web
npm run build
```
