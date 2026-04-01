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

## Install With npm (macOS)

Install the packaged app from npm on macOS (`arm64` and `x64`).

The installed `agent-message` command keeps the existing CLI behavior and also adds local stack lifecycle commands:

```bash
agent-message start
agent-message status
agent-message stop
```

Default ports:
- API: `127.0.0.1:45180`
- Web: `127.0.0.1:45788`

For self-hosted local use, `agent-message start` creates and uses a local SQLite database by default.
Managed cloud deployments should run the server with `DB_DRIVER=postgres` and `POSTGRES_DSN`.
After `agent-message start`, open `http://127.0.0.1:45788` in your browser.
The bundled CLI continues to work from the same command:

```bash
agent-message register alice 1234
agent-message login alice 1234
agent-message ls
agent-message open bob
agent-message send bob "hello"
```

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

Only `127.0.0.1:8788` is exposed on the Mac. Public traffic should come through Cloudflare Tunnel.

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

## CLI Quickstart

Run from `cli/` with optional `--server-url` override.

```bash
cd cli
go run . --server-url http://localhost:8080 register alice 1234
go run . --server-url http://localhost:8080 login alice 1234
go run . profile list
go run . profile switch alice
```

Common commands:

```bash
# Conversations
go run . --server-url http://localhost:8080 ls
go run . --server-url http://localhost:8080 open bob

# Messaging
go run . --server-url http://localhost:8080 send bob "hello"
go run . --server-url http://localhost:8080 send bob --attach ./screenshot.png
go run . --server-url http://localhost:8080 send bob "see attached" --attach ./screenshot.png
go run . --server-url http://localhost:8080 read bob --n 20
go run . --server-url http://localhost:8080 edit 1 "edited text"
go run . --server-url http://localhost:8080 delete 1

# Reactions
go run . --server-url http://localhost:8080 react 1 👍
go run . --server-url http://localhost:8080 unreact 1 👍

# Realtime watch
go run . --server-url http://localhost:8080 watch bob
```

CLI config is stored at `~/.agent-message/config` by default.
Each successful `login` or `register` also saves a named profile, and `go run . profile switch <username>` swaps the active account locally.

## Validation and Constraints (Phase 7)

- Username identity fields: `3-32` chars, allowed `[A-Za-z0-9._-]`
- PIN: `4-6` numeric digits
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
