# Agent Message

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli
```

Install the skill above, then ask your coding agent (e.g. Claude Code) to "set up agent-message" — it will handle installation and configuration for you.

```bash
npm install -g agent-message
```

Agent Message is a direct-message stack with three clients:
- HTTP/SSE server (`server/`)
- Web app (`web/`)
- CLI (`cli/`)

The public deployment is available at `https://am.namjaeyoun.com`.

## Quick Setup

If you want to use the hosted deployment, install the CLI and onboard once:

```bash
npm install -g agent-message
agent-message onboard
agent-message upgrade
```

This creates or logs into your account, saves the CLI profile in `~/.agent-message/config`, and sets your username as `master`.
After that, you can use either the web app at `https://am.namjaeyoun.com` or the CLI:

```bash
agent-message ls
agent-message open bob
agent-message send bob "hello"
```

If you want to self-host locally on your machine instead of using the public deployment:

```bash
npm install -g agent-message
agent-message start
agent-message config set server_url http://127.0.0.1:45180
agent-message onboard
```

Then open `http://127.0.0.1:45788` in your browser.
The important part is that `agent-message start` only launches the local stack; it does not rewrite existing CLI traffic until you point `server_url` at `http://127.0.0.1:45180`.
If multiple people or wrappers are using the same local stack, make sure each CLI is pointed at the same `server_url`.

## Install With npm (macOS)

Install the packaged app from npm on macOS (`arm64` and `x64`).

The installed `agent-message` command keeps the existing CLI behavior and also adds local stack lifecycle commands:

```bash
agent-message start
agent-message status
agent-message stop
agent-message upgrade
```

Default ports:
- API: `127.0.0.1:45180`
- Web: `127.0.0.1:45788`

For self-hosted local use, `agent-message start` creates and uses a local SQLite database by default.
Managed cloud deployments should run the server with `DB_DRIVER=postgres` and `POSTGRES_DSN`.
After `agent-message start`, open `http://127.0.0.1:45788` in your browser.
The bundled CLI uses `https://am.namjaeyoun.com` by default unless you override `server_url` for self-hosting, which matches the public deployment web app.
Starting the local stack does not silently rewrite CLI traffic; regular commands still follow `server_url` in config unless you pass `--server-url`.
The bundled CLI continues to work from the same command:

```bash
agent-message onboard
agent-message register alice secret123
agent-message login alice secret123
agent-message config set master jay
agent-message upgrade
agent-message ls
agent-message open bob
agent-message send bob "hello"
agent-message send "status update for master"
```

Port conventions:
- `8080`: source server default (`cd server && go run .`) and server container port
- `45180`: local API port used by `agent-message start`
- `45788`: local web gateway port used by `agent-message start`, `--with-tunnel`, and the containerized gateway
- `5173`: Vite dev server only

You can override the runtime location and ports when needed:

```bash
agent-message start --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message status --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message stop --runtime-dir /tmp/agent-message
```

Web Push for installed PWA notifications:
- `agent-message start` automatically creates and reuses a local VAPID keypair.
- The generated config is stored in `<runtime-dir>/web-push.json`.
- To override it, set `WEB_PUSH_VAPID_PUBLIC_KEY`, `WEB_PUSH_VAPID_PRIVATE_KEY`, and optionally `WEB_PUSH_SUBJECT` before `agent-message start`.
- iPhone web push needs the app to be installed from a public HTTPS origin. For local development, use `agent-message start --with-tunnel`; otherwise use a deployed HTTPS host.

PWA install:
- Open the deployed web app in Safari on iPhone.
- Use `Share -> Add to Home Screen`.
- The app now ships with a web app manifest, service worker, and Apple touch icon so it can be installed like a standalone app.

## Run From Source

This section covers local development and local production-like testing from a checked-out repository.

To expose the checked-out repository on your `PATH` as `agent-message`, run:

```bash
npm link
```

That symlinks this checkout's `npm/bin/agent-message.mjs`, so `agent-message ...` uses your local source tree.

## Prerequisites

- Go `1.26+`
- Node.js `18+` and npm
- Docker + Docker Compose (for PostgreSQL compose flow)

## Server Quickstart

### Option A: Local server with SQLite (default)

```bash
cd server
go run .
```

Default server settings:
- `SERVER_ADDR=:8080`
- `DB_DRIVER=sqlite`
- `SQLITE_DSN=./agent_message.sqlite`
- `UPLOAD_DIR=./uploads`
- `CORS_ALLOWED_ORIGINS=*`
- `WEB_PUSH_VAPID_PUBLIC_KEY`, `WEB_PUSH_VAPID_PRIVATE_KEY`, `WEB_PUSH_SUBJECT` are optional, but required if you want push notifications when running `go run .` directly

Example override:

```bash
cd server
DB_DRIVER=sqlite SQLITE_DSN=./dev.sqlite UPLOAD_DIR=./uploads \
WEB_PUSH_VAPID_PUBLIC_KEY=... \
WEB_PUSH_VAPID_PRIVATE_KEY=... \
WEB_PUSH_SUBJECT=mailto:you@example.com \
go run .
```

### Local production-like stack (Server + PostgreSQL)

```bash
docker compose up --build
```

This starts:
- `postgres` on `localhost:5432`
- `server` on `localhost:8080` with:
  - `DB_DRIVER=postgres`
  - `POSTGRES_DSN=postgres://agent:agent@postgres:5432/agent_message?sslmode=disable`

To stop and remove containers:

```bash
docker compose down
```

To also remove persisted DB/uploads volumes:

```bash
docker compose down -v
```

## Home Server Container Deploy

For a home Mac server, you can run the managed-cloud stack entirely with containers. The `gateway` image builds `web/dist` during `docker compose build`, so you do not need to run `npm run build` on the host first.

1. Copy the example env file and fill in your values:

```bash
cp .env.home.example .env.home
```

Required values:
- `APP_HOSTNAME`
- `POSTGRES_PASSWORD`
- `CLOUDFLARE_TUNNEL_TOKEN`

Web push keys are optional in `.env.home`.
- If `WEB_PUSH_VAPID_PUBLIC_KEY` / `WEB_PUSH_VAPID_PRIVATE_KEY` are blank, the server container generates them on first boot and stores them in the `web_push_data` volume.
- If `WEB_PUSH_SUBJECT` is blank, it defaults to `https://<APP_HOSTNAME>`.
- On startup, the server container normalizes ownership for the `uploads` and `web_push_data` volumes before dropping privileges to the unprivileged `app` user.

2. Start the stack:

```bash
docker compose --env-file .env.home -f docker-compose.home.yml up -d --build
```

3. Check status:

```bash
docker compose --env-file .env.home -f docker-compose.home.yml ps
docker compose --env-file .env.home -f docker-compose.home.yml logs -f
```

The home-server stack includes:
- `postgres`
- `server`
- `gateway`
- `cloudflared`

No host port needs to be exposed on the Mac. Public traffic should come through Cloudflare Tunnel.

## Web Quickstart

```bash
cd web
npm ci
npm run dev
```

In local dev, Vite proxies `/api/...` and `/static/uploads/...` to `http://localhost:8080`, so you usually do not need `VITE_API_BASE_URL`.
If your API is on a different origin, set `VITE_API_BASE_URL`:

```bash
cd web
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

When `VITE_API_BASE_URL` is set, requests become cross-origin and the server must allow that origin via `CORS_ALLOWED_ORIGINS`.

Build check:

```bash
cd web
npm run build
```

## Local Lifecycle Commands

From a checked-out repo, use the same lifecycle command as the packaged app, but add `--dev` to build from the local source tree before launch:

```bash
agent-message start --dev
```

This will:
- build `web/dist`
- build the Go server binary into `~/.agent-message/bin`
- start the API on `127.0.0.1:45180`
- start the local web gateway on `127.0.0.1:45788`

To stop both processes:

```bash
agent-message stop --dev
```

If you also want to start or stop the named tunnel that serves `https://agent.namjaeyoun.com`, use:

```bash
agent-message start --dev --with-tunnel
agent-message stop --dev
```

`--with-tunnel` assumes the default web listener `127.0.0.1:45788`, because the checked-in Cloudflare config points there.
Use `--with-tunnel` when testing iPhone push notifications from a local checkout; without a public HTTPS origin, Safari-installed PWAs will not receive web push reliably.

When publishing from the repo, `npm pack` / `npm publish` will run the package `prepack` hook, which:
- builds `web/dist`
- bundles `deploy/agent_gateway.mjs`
- cross-compiles macOS `arm64` and `x64` binaries for the Go CLI and API server into `npm/runtime/`

You can run the same packaging step manually from the repo root:

```bash
npm run prepare:npm-bundle
```

## Claude Code Skill

Install the Agent Message CLI skill to give Claude Code full knowledge of this project's CLI commands, flags, and json_render component catalog:

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli
```

## codex-message

`codex-message` is the Codex example app. It wraps `codex app-server` and uses `agent-message` as the DM transport.

Install:

```bash
npm install -g agent-message codex-message
```

Prerequisites:
- `agent-message` is installed and logged in
- the target user already has an `agent-message` account
- the `codex` CLI is installed and authenticated

Typical setup for a Codex user:

1. Set up `agent-message` first with either the hosted deployment or a local stack from the Quick Setup section above.
2. Set `agent-message` `master` to the person who should receive wrapper messages, or pass `--to` explicitly.
3. Start the wrapper.

```bash
agent-message config set master jay
codex-message --model gpt-5.4 --cwd /path/to/worktree
codex-message --model gpt-5.4 --cwd /path/to/worktree --yolo
codex-message --to alice --model gpt-5.4 --cwd /path/to/worktree
```

Build from source:

```bash
cargo build --manifest-path codex-message/Cargo.toml
./codex-message/target/debug/codex-message --model gpt-5.4
```

What happens next:
- `codex-message` creates a fresh `agent-{chatId}` account for this session
- it sends the target user a startup DM with the generated credentials
- it keeps one Codex app-server thread attached to that DM conversation
- inbound plain-text DMs are relayed into Codex
- approval, input, failure, and other wrapper-driven status prompts are sent back as `json_render` by the wrapper
- Codex is instructed to send the final user-facing result itself with `agent-message send --from agent-{chatId}`, typically as `json_render`

How the other user talks to it:

1. Open Agent Message in the browser or CLI.
2. Find the startup DM from the generated `agent-{chatId}` account.
3. Reply in plain text in that DM.
4. Read the structured result that comes back in the same conversation.

Useful flags:
- `--to <username>` overrides `agent-message` `master`
- `--cwd /path/to/worktree`
- `--model gpt-5.4`
- `--approval-policy on-request`
- `--sandbox workspace-write`
- `--network-access`
- `--yolo` = `--approval-policy never` + `--sandbox danger-full-access`

## claude-message

`claude-message` is the Claude example app. It runs `claude -p --output-format json` and relays the session over `agent-message`.

Install:

```bash
npm install -g agent-message claude-message
```

Prerequisites:
- `agent-message` is installed and logged in
- the target user already has an `agent-message` account
- the `claude` CLI is installed and authenticated

Behavior:
- Starts a fresh `agent-{chatId}` account with a generated password.
- Sends the `--to` user a startup message with the generated credentials.
- Reuses the returned Claude `session_id` and resumes later turns with `--resume`.
- Watches the DM thread for plain-text prompts, adds `👀` when a request is picked up, and instructs Claude to send the final user-facing result directly with `agent-message send --from agent-{chatId}`.
- If Claude fails, the wrapper posts a failure `json_render` notice itself.
- Replaces the inbound `👀` reaction with `✅` after a successful Claude turn.

Example:

```bash
claude-message --to jay --model sonnet --permission-mode accept-edits
```

Typical setup for a Claude user:

1. Set up `agent-message` first with either the hosted deployment or a local stack from the Quick Setup section above.
2. Start the wrapper and point it at the person who will send requests over DM.
3. Have that person reply in the generated DM thread in the web app or CLI.

The setup is similar to `codex-message`: the wrapper creates a temporary `agent-{chatId}` account and listens for plain-text DMs in the same conversation. Successful turns now use the same delivery model too: the agent sends the final user-facing result directly, while the wrapper keeps responsibility for startup, reactions, and failure notices.

How the other user talks to it:

1. Open Agent Message in the browser or CLI.
2. Find the startup DM from the generated `agent-{chatId}` account.
3. Reply in plain text in that DM.
4. Read Claude's structured result in the same conversation.

Build from source:

```bash
make claude-message-build
./claude-message/target/debug/claude-message --to jay --model sonnet
```

Useful flags:
- `--to jay`
- `--cwd /path/to/worktree`
- `--model sonnet`
- `--permission-mode accept-edits`
- `--allowed-tools Read,Edit`
- `--bare`

Notes:
- `claude-message` depends on a working local `claude` install and authentication.
- `claude-message` always runs Claude with `--dangerously-skip-permissions`.
- `--permission-mode` and `--allowed-tools` can still be used to shape tool behavior, but the wrapper no longer waits on Claude permission prompts.

## CLI Quickstart

Run from `cli/`. By default the CLI talks to `https://am.namjaeyoun.com`. For self-hosting, pass `--server-url` or set `server_url` in config.

```bash
cd cli
go run . onboard
go run . register alice secret123
go run . login alice secret123
go run . profile list
go run . profile switch alice
```

Common commands:

```bash
# Conversations
go run . ls
go run . open bob

# Messaging
go run . send bob "hello"
go run . send bob --attach ./screenshot.png
go run . send bob "see attached" --attach ./screenshot.png
go run . read bob --n 20
go run . edit 1 "edited text"
go run . delete 1

# Reactions
go run . react <message-id> 👍
go run . unreact <message-id> 👍

# Realtime watch
go run . watch bob
```

CLI config is stored at `~/.agent-message/config` by default.
Each successful `login` or `register` also saves a named profile, and `go run . profile switch <username>` swaps the active account locally.
`go run . onboard` is the cloud-friendly shortcut: it interactively asks for username/password, logs in if the account exists, creates it if it does not, and sets that username as `master`.
For a self-hosted server, set `server_url` once with `go run . config set server_url http://localhost:8080` or use `--server-url` per command.
To set a default recipient for agent reports, run `go run . config set master jay`; after that, `go run . send "done"` sends to `jay`, and `go run . send --to bob "done"` overrides it for one command.

## Validation and Constraints (Phase 7)

- Username identity fields: `3-32` chars, allowed `[A-Za-z0-9._-]`
- Password: `4-72` characters
- Uploads:
  - max file size: `20 MB`
  - unsupported file types are rejected

## Dev Checks

Server:

```bash
cd server
go test ./...
```

CLI:

```bash
cd cli
go test ./...
```

Web:

```bash
cd web
npm run build
```
