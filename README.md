# Agent Messenger

Agent Messenger is a direct-message stack with three clients:
- HTTP/SSE server (`server/`)
- Web app (`web/`)
- CLI (`cli/`)

This README covers a Phase 7 quickstart for local development and local production-like testing.

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
- `SQLITE_DSN=./agent_messenger.sqlite`
- `UPLOAD_DIR=./uploads`
- `CORS_ALLOWED_ORIGINS=*`

Example override:

```bash
cd server
DB_DRIVER=sqlite SQLITE_DSN=./dev.sqlite UPLOAD_DIR=./uploads go run .
```

### Option B: Local production-like stack (Server + PostgreSQL)

```bash
docker compose up --build
```

This starts:
- `postgres` on `localhost:5432`
- `server` on `localhost:8080` with:
  - `DB_DRIVER=postgres`
  - `POSTGRES_DSN=postgres://agent:agent@postgres:5432/agent_messenger?sslmode=disable`

To stop and remove containers:

```bash
docker compose down
```

To also remove persisted DB/uploads volumes:

```bash
docker compose down -v
```

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

## Local Bundle Commands

From the project root, you can start the SQLite-backed API server and the production-like local web gateway together:

```bash
./dev-up
```

This will:
- build `web/dist`
- build the Go server binary into `~/.agent-messenger/bin`
- start the API on `127.0.0.1:18080`
- start the local web gateway on `127.0.0.1:8788`

To stop both processes:

```bash
./dev-stop
```

If you also want to start or stop the named tunnel that serves `https://agent.namjaeyoun.com`, use:

```bash
./dev-up --with-tunnel
./dev-stop --with-tunnel
```

PWA install:

- Open the deployed web app in Safari on iPhone.
- Use `Share -> Add to Home Screen`.
- The app now ships with a web app manifest, service worker, and Apple touch icon so it can be installed like a standalone app.

## CLI Quickstart

Run from `cli/` with optional `--server-url` override.

```bash
cd cli
go run . --server-url http://localhost:8080 register alice 1234
go run . --server-url http://localhost:8080 login alice 1234
```

Common commands:

```bash
# Conversations
go run . --server-url http://localhost:8080 ls
go run . --server-url http://localhost:8080 open bob

# Messaging
go run . --server-url http://localhost:8080 send bob "hello"
go run . --server-url http://localhost:8080 read bob --n 20
go run . --server-url http://localhost:8080 edit 1 "edited text"
go run . --server-url http://localhost:8080 delete 1

# Reactions
go run . --server-url http://localhost:8080 react 1 👍
go run . --server-url http://localhost:8080 unreact 1 👍

# Realtime watch
go run . --server-url http://localhost:8080 watch bob
```

CLI config is stored at `~/.msgr/config` by default.

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
