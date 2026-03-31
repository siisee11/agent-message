# Agent Message — Implementation Plan

## Phase 1: Server Foundation

**Goal**: Go server skeleton with DB, models, and auth.

### Tasks
- [ ] Initialize Go module (`server/go.mod`)
- [ ] Set up project directory structure (`api/`, `ws/`, `store/`, `models/`)
- [ ] Define data models (`User`, `Conversation`, `Message`, `Reaction`)
- [ ] Implement SQLite database layer with schema migrations
- [ ] Auth endpoints: `POST /api/auth/register`, `POST /api/auth/login`, `DELETE /api/auth/logout`
  - bcrypt PIN hashing
  - Opaque session token generation and storage
- [ ] Auth middleware (`Authorization: Bearer <token>` validation)
- [ ] CORS middleware
- [ ] Basic server entry point (`main.go`) with config via env vars

**Deliverable**: Server starts, register/login/logout work, token auth middleware functional.

---

## Phase 2: Core REST API

**Goal**: All REST endpoints for users, conversations, messages, and file upload.

### Tasks
- [ ] `GET /api/users` — search users by username
- [ ] `GET /api/users/me` — current user profile
- [ ] `GET /api/conversations` — list conversations for current user
- [ ] `POST /api/conversations` — start DM (body: `{ username }`)
- [ ] `GET /api/conversations/:id` — conversation details
- [ ] `GET /api/conversations/:id/messages` — list messages (cursor-based: `?before=<id>&limit=20`)
- [ ] `POST /api/conversations/:id/messages` — send message (text or multipart with attachment)
- [ ] `PATCH /api/messages/:id` — edit message (own only)
- [ ] `DELETE /api/messages/:id` — soft-delete message (own only)
- [ ] `POST /api/upload` — upload file/image (max 20 MB), returns `{ url }`
- [ ] Serve static files at `/static/uploads/` from configurable `UPLOAD_DIR`

**Deliverable**: Full CRUD over conversations and messages via HTTP. File upload functional.

---

## Phase 3: WebSocket & Reactions

**Goal**: Real-time event hub and emoji reaction endpoints.

### Tasks
- [ ] WebSocket hub (`ws/hub.go`): manage client connections, broadcast to conversation participants
- [ ] `GET /ws?token=<token>` — authenticate and upgrade connection
- [ ] Emit events on server-side mutations:
  - `message.new` on send
  - `message.edited` on edit
  - `message.deleted` on delete
  - `reaction.added` on react
  - `reaction.removed` on unreact
- [ ] Handle client → server `read` event (`{ type: "read", data: { conversation_id } }`)
- [ ] `POST /api/messages/:id/reactions` — add emoji reaction (toggle / one per emoji per user)
- [ ] `DELETE /api/messages/:id/reactions/:emoji` — remove own reaction

**Deliverable**: WebSocket broadcasts all mutations in real time. Reactions fully functional.

---

## Phase 4: Web Client Foundation

**Goal**: React + TypeScript app scaffold with routing, API client, and auth flow.

### Tasks
- [ ] Scaffold with Vite (`web/`)
- [ ] Install dependencies: React Router, a fetch/WebSocket wrapper (e.g. SWR or React Query), Tailwind CSS or CSS modules
- [ ] Typed API client (`src/api/`) wrapping all REST endpoints
- [ ] Auth state management (context or store)
- [ ] `/login` page — username + PIN form; auto-register on first login
- [ ] Protected route wrapper — redirect to `/login` if unauthenticated
- [ ] WebSocket client hook (`src/hooks/useWebSocket.ts`) with reconnect logic

**Deliverable**: Auth flow works end-to-end in the browser. WebSocket hook connects and receives events.

---

## Phase 5: Web Client Chat UI

**Goal**: Full chat interface — sidebar, message list, input, reactions, attachments.

### Tasks
- [ ] Layout: sidebar (conversation list) + main chat area (`/` route)
- [ ] Conversation list sidebar — shows DM partner name, last message preview
- [ ] Start new DM — user search input → `POST /api/conversations`
- [ ] `/dm/:conversationId` — active chat view
  - Message list with infinite scroll (load older on scroll-up via cursor)
  - Real-time new messages via WebSocket
  - Message bubble: sender name, timestamp, `[수정됨]` badge, `"삭제된 메시지입니다"` for deleted
  - Inline image preview; other attachments as download link
- [ ] Message input bar: text field + emoji picker + file/image attach button
- [ ] Context menu (right-click / long-press on own message): Edit, Delete
- [ ] Edit mode: pre-fill input with current content, submit updates message
- [ ] Reactions bar: grouped emoji + count; click to add/toggle/remove own reaction

**Deliverable**: Fully usable web chat with real-time updates, attachments, and reactions.

---

## Phase 6: CLI Client

**Goal**: `msgr` CLI in Go covering all SPEC commands.

### Tasks
- [ ] Initialize Go module (`cli/go.mod`); use [cobra](https://github.com/spf13/cobra) for commands
- [ ] Config file at `~/.msgr/config` (JSON: `{ "server_url": "...", "token": "..." }`)
- [ ] Auth commands:
  - `register <username> <pin>`
  - `login <username> <pin>`
  - `logout`
- [ ] Conversation commands:
  - `ls` — list conversations
  - `open <username>` — get-or-create DM
- [ ] Message commands:
  - `send <username> <text>`
  - `read <username> [--n N]` — print last N messages with index `[1] <uuid> user: text`
  - `edit <index> <text>` — edit by index from last `read` (index → UUID resolved from local session state)
  - `delete <index>` — soft-delete by index
- [ ] Reaction commands:
  - `react <index> <emoji>`
  - `unreact <index> <emoji>`
- [ ] Watch mode:
  - `watch <username>` — open WebSocket, stream `message.new` events to stdout in real time

**Deliverable**: Full CLI client working against the server.

---

## Phase 7: Integration & Polish

**Goal**: Tie everything together, add PostgreSQL support, and clean up.

### Tasks
- [ ] PostgreSQL store implementation (same interface as SQLite store, switch via `DB_DRIVER` env var)
- [ ] Consistent error responses across all API endpoints (`{ "error": "..." }`)
- [ ] Input validation (username length, PIN 4–6 digits, file size/type checks)
- [ ] End-to-end integration tests (server + SQLite, using `httptest`)
- [ ] Docker Compose setup: server + PostgreSQL for local prod-like testing
- [ ] `README.md` with quickstart for all three components

**Deliverable**: Production-ready stack. All three clients work against a PostgreSQL-backed server.
