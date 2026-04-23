<p align="center">
  <img src="agent-message-logo.svg" alt="Agent Message logo" width="88">
</p>

# Agent Message

[English](README.md) | [한국어](README.ko.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

<p align="center">
  <img src="docs/readme-screenshot.png" alt="Agent Message screenshot" width="900">
</p>

Agent Message 是一个让智能体和人类在同一个 DM 流程里协作的消息栈。

- HTTP/SSE 服务器 (`server/`)
- Web 应用 (`web/`)
- CLI (`cli/`)

## 为什么使用 Agent Message

- 消息工具以 CLI 形式暴露，智能体可以直接发送、读取和监听消息。
- 通过 `json_render`，消息可以以卡片、表格、徽章、进度块等易读的结构化格式呈现。
- 你可以在手机上打开 Web 应用，继续和智能体协作。
- `codex-message` 和 `claude-message` 会把 Codex 与 Claude 会话接入同一套 DM 工作流。

## Wrapper 包

- `codex-message`：启动 Codex app-server 会话，并通过 Agent Message DM 转发对话。
- `claude-message`：启动 Claude CLI 会话，并转发提示、失败信息和最终回复。

Landing page 位于 `https://amessage.dev`。托管云服务仍在准备中；目前推荐使用 self-hosted local stack。

## 支持的平台

当前发布到 npm 的包只包含 macOS 构建。

| 平台 | 架构 | `agent-message` | `codex-message` | `claude-message` | 备注 |
| --- | --- | --- | --- | --- | --- |
| macOS | Apple Silicon (`arm64`) | 支持 | 支持 | 支持 | 主要打包目标 |
| macOS | Intel (`x64`) | 支持 | 支持 | 支持 | 打包目标 |
| Linux | `x64` / `arm64` | 未打包 | 未打包 | 未打包 | 仅可从源码构建 |
| Windows | `x64` / `arm64` | 未打包 | 未打包 | 未打包 | 当前不支持 |

## Setup Prompt

粘贴到 Claude Code 或 Codex：

```bash
Set up https://github.com/siisee11/agent-message for me.

Read `install.md` and follow the self-host setup flow. Ask me for the account-id before registering, use 0000 only as the temporary initial password, remind me to change it immediately, set the master recipient, and send me a welcome message with agent-message when setup is complete.
```

## 快速设置

云服务账号目前还不可用。现在请使用 self-hosted local stack。

如果由智能体帮你安装 Agent Message，请使用上面的 Setup Prompt。

也可以用 npm 手动安装：

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -g -y
npm install -g agent-message
agent-message start
agent-message status
```

然后创建或登录本地账号。注册前先询问用户的 `account-id`。`0000` 只能作为临时初始密码使用。

```bash
agent-message register <account-id> 0000
# 如果账号已存在：
agent-message login <account-id> 0000
```

在浏览器打开 `http://127.0.0.1:45788`，并立即在 Profile 页面修改密码。
`agent-message start` 会启动本地栈，并更新 `~/.agent-message/config`，让 CLI 流量指向已启动的 API：`http://127.0.0.1:45180`。

设置公开显示名和用于智能体状态报告的默认接收人：

```bash
agent-message username set <username>
agent-message config set master <recipient-username>
agent-message whoami
```

## 使用 npm 安装 (macOS)

macOS (`arm64` 和 `x64`) 可以安装 npm 打包版本。

```bash
npm install -g agent-message
agent-message start
agent-message status
agent-message stop
agent-message upgrade
```

默认端口：
- API：`127.0.0.1:45180`
- Web：`127.0.0.1:45788`

self-hosted local 使用时，`agent-message start` 默认会创建并使用本地 SQLite 数据库。托管云服务仍在准备中。未来的 managed cloud 部署应使用 `DB_DRIVER=postgres` 和 `POSTGRES_DSN` 运行服务器。

常用 CLI 命令仍然通过同一个 `agent-message` 命令使用：

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

需要时可以覆盖运行目录和端口：

```bash
agent-message start --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message status --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message stop --runtime-dir /tmp/agent-message
```

## Web 快速开始

```bash
cd web
npm ci
npm run dev
```

本地开发时，Vite 会把 `/api/...` 和 `/static/uploads/...` 代理到 `http://localhost:8080`。如果 API 在另一个 origin，请设置 `VITE_API_BASE_URL`：

```bash
cd web
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

构建检查：

```bash
cd web
npm run build
```

## Cloudflare Workers Web 部署

当前 React Web 应用可以作为 Cloudflare Workers static assets 部署。这主要用于先发布公开网站；后续可以再接入 API 支持的云服务。

```bash
cd web
npm ci
npm run deploy:worker
```

云 API 准备好之前，Worker 的 API 请求会返回 `503`。当 API origin 可用后，在 Worker 上设置 `API_ORIGIN` 并重新部署：

```bash
cd web
npx wrangler secret put API_ORIGIN
npm run deploy:worker
```

例如设置 `API_ORIGIN=https://api.amessage.dev` 后，Web 应用仍然使用同源 `/api/...` 调用，Worker 会把请求代理到 API 服务。

## 从源码运行

要让当前 checkout 在 `PATH` 上以 `agent-message` 使用，请在仓库根目录运行：

```bash
npm link
```

从本地源码树启动时加上 `--dev`：

```bash
agent-message start --dev
agent-message stop --dev
```

前置条件：
- Go `1.26+`
- Node.js `18+` 和 npm
- Docker + Docker Compose（用于 PostgreSQL compose 流程）

## 服务器快速开始

SQLite 本地服务器：

```bash
cd server
go run .
```

## Self-host Container Deploy

在家用 Mac 服务器上，可以完全用容器运行 self-hosted stack。

```bash
cp .env.selfhost.example .env.selfhost
make publish
docker compose --env-file .env.selfhost -f docker-compose.selfhost.yml ps
docker compose --env-file .env.selfhost -f docker-compose.selfhost.yml logs -f
make unpublish
```

必填值：
- `APP_HOSTNAME`
- `POSTGRES_PASSWORD`
- `CLOUDFLARE_TUNNEL_TOKEN`

该 stack 包含：
- `postgres`
- `server`
- `gateway`
- `cloudflared`

Mac 上不需要暴露 host port。公网流量应通过 Cloudflare Tunnel 进入。

## Claude Code Skill

安装 Agent Message CLI skill 后，Claude Code 可以了解本项目的 CLI 命令、参数和 `json_render` 组件目录。

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli
```

## codex-message

`codex-message` 是 Codex 示例应用。它包装 `codex app-server`，并使用 `agent-message` 作为 DM transport。

```bash
npm install -g agent-message codex-message
```

前置条件：
- 已安装并登录 `agent-message`
- 目标用户已有 `agent-message` 账号
- 已安装并认证 `codex` CLI

典型设置：

```bash
agent-message config set master jay
codex-message --model gpt-5.4 --cwd /path/to/worktree
codex-message --model gpt-5.4 --cwd /path/to/worktree --yolo
codex-message --to alice --model gpt-5.4 --cwd /path/to/worktree
codex-message --bg --model gpt-5.4 --cwd /path/to/worktree
```

运行后：
- 每个会话都会创建一个新的 `agent-{chatId}` 账号。
- 向目标用户发送包含生成凭据的启动 DM。
- 将一个 Codex app-server thread 绑定到该 DM 对话。
- 普通文本 DM 会被转发给 Codex。
- 审批、输入、失败和状态提示由 wrapper 以 `json_render` 发回。
- 最终面向用户的结果由 Codex 使用 `agent-message send --from agent-{chatId}` 直接发送。

常用参数：
- `--to <username>`
- `--cwd /path/to/worktree`
- `--model gpt-5.4`
- `--approval-policy on-request`
- `--sandbox workspace-write`
- `--network-access`
- `--yolo`
- `--bg`

## claude-message

`claude-message` 是 Claude 示例应用。它运行 `claude -p --output-format json`，并通过 `agent-message` 中继会话。

```bash
npm install -g agent-message claude-message
```

前置条件：
- 已安装并登录 `agent-message`
- 目标用户已有 `agent-message` 账号
- 已安装并认证 `claude` CLI

示例：

```bash
claude-message --to jay --model sonnet --permission-mode accept-edits
claude-message --bg --to jay --model sonnet --permission-mode accept-edits
```

该设置与 `codex-message` 类似：wrapper 会创建临时 `agent-{chatId}` 账号，并在同一个 DM 对话中等待普通文本消息。成功的 turn 会由智能体直接发送最终用户结果；wrapper 负责启动消息、reaction 和失败通知。

常用参数：
- `--to jay`
- `--cwd /path/to/worktree`
- `--model sonnet`
- `--permission-mode accept-edits`
- `--allowed-tools Read,Edit`
- `--bare`
- `--bg`

## CLI 快速开始

self-hosting 时，先用 `agent-message start` 启动本地栈，或者用 `--server-url` / `config set server_url` 让 source CLI 指向本地 API。

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

CLI 配置默认存放在 `~/.agent-message/config`。`onboard` 是面向人的交互式命令。智能体安装应遵循 `install.md`：安装 skill、安装 npm 包、启动本地栈、询问 `account-id`，然后使用 `register` 或 `login`。

## 校验与限制

- Account ID 和 username：`3-32` 个字符，允许 `[A-Za-z0-9._-]`
- Password：`4-72` 个字符
- 上传最大文件大小：`20 MB`
- 不支持的文件类型会被拒绝

## 开发检查

```bash
cd server
go test ./...

cd cli
go test ./...

cd web
npm run build
```
