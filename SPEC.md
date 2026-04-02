# Agent Message — Project Specification

A lightweight Telegram-inspired messenger with a server, web client, and CLI client.

---

## Overview

| Component | Technology |
|-----------|-----------|
| Server    | Go (REST + SSE) |
| Web Client | React + TypeScript |
| CLI Client | Go |
| Database  | SQLite (dev) / PostgreSQL (prod) |

---

## Architecture

```
┌─────────────┐        ┌─────────────┐
│  Web Client │        │  CLI Client │
│ React + TS  │        │     Go      │
└──────┬──────┘        └──────┬──────┘
       │ HTTP / SSE            │ HTTP / SSE
       └──────────┬────────────┘
                  │
         ┌────────▼────────┐
         │     Server      │
         │  Go + REST API  │
         │  + SSE stream   │
         └────────┬────────┘
                  │
         ┌────────▼────────┐
         │    Database     │
         │ SQLite / Postgres│
         └─────────────────┘
```

---

## Authentication

- **Username + PIN login**: Users register and log in with a unique username and a 4–6 digit numeric PIN.
- On login/register, the server issues an **opaque session token** with no expiry (invalidated only on logout).
- Token is passed as `Authorization: Bearer <token>` on all API requests and as a query param for the SSE stream connection.

---

## Data Models

### User
```
id          string (uuid)
username    string (unique)
pin_hash    string        # bcrypt hash of 4–6 digit PIN
created_at  timestamp
```

### Conversation (DM)
```
id            string (uuid)
participant_a string (user_id)
participant_b string (user_id)
created_at    timestamp
```

### Message
```
id              string (uuid)
conversation_id string
sender_id       string (user_id)
content         string (nullable if attachment)
attachment_url  string (nullable)
attachment_type string (image | file | null)
edited          bool
deleted         bool
created_at      timestamp
updated_at      timestamp
```

### Reaction
```
id         string (uuid)
message_id string
user_id    string
emoji      string (e.g. "👍")
created_at timestamp
```

---

## API

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/register` | Register with username + PIN → returns token |
| POST | `/api/auth/login` | Login with username + PIN → returns token |
| DELETE | `/api/auth/logout` | Invalidate session token |

### Users

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/users` | Search users by username |
| GET | `/api/users/me` | Get current user profile |

### Conversations

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/conversations` | List all DM conversations for current user |
| POST | `/api/conversations` | Start a new DM (body: `{ username }`) |
| GET | `/api/conversations/:id` | Get conversation details |

### Messages

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/conversations/:id/messages` | List messages (cursor-based, `?before=<message_id>&limit=20`) |
| POST | `/api/conversations/:id/messages` | Send a message (text or multipart with attachment) |
| PATCH | `/api/messages/:id` | Edit message content |
| DELETE | `/api/messages/:id` | Delete message (soft delete) |

### Reactions

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/messages/:id/reactions` | Add emoji reaction (body: `{ emoji }`) |
| DELETE | `/api/messages/:id/reactions/:emoji` | Remove own reaction |

### File Upload

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/upload` | Upload file/image → returns `{ url }` |

---

## Realtime Stream

- **Endpoint**: `GET /api/events?token=<session_token>`
- One persistent SSE connection per client session.

### Server → Client Events

```jsonc
// New message received
{ "type": "message.new", "data": { /* Message */ } }

// Message edited
{ "type": "message.edited", "data": { /* Message */ } }

// Message deleted
{ "type": "message.deleted", "data": { "id": "..." } }

// Reaction added
{ "type": "reaction.added", "data": { /* Reaction */ } }

// Reaction removed
{ "type": "reaction.removed", "data": { "message_id": "...", "emoji": "...", "user_id": "..." } }
```

## Web Client Features

### Pages / Views

| Route | Description |
|-------|-------------|
| `/login` | Username input → login or auto-register |
| `/` | Sidebar with conversation list + main chat area |
| `/dm/:conversationId` | Active DM conversation |

### Chat UI

- **Message list**: Infinite scroll (load older messages on scroll up).
- **Message input**: Text + emoji picker + file/image attach button.
- **Message bubble**: Shows sender, timestamp, edited badge. Deleted messages show `"삭제된 메시지입니다"` placeholder (soft delete).
- **Reactions bar**: Shows grouped emoji counts; click to add/remove your reaction.
- **Context menu** (right-click / long press): Edit, Delete (own messages only).
- **Attachment preview**: Inline image preview; file shown as download link.

---

## CLI Client Features

```
Usage: agent-message <command> [flags]

Auth:
  login <username> <pin>    Log in (stores token in ~/.agent-message/config)
  register <username> <pin> Register new account
  logout                    Clear stored token

Conversations:
  ls                        List all conversations
  open <username>           Open DM with user (create if not exists)

Messages:
  send <username> <text>    Send a message
  read <username>           Read recent messages (default: last 20)
  read <username> --n 50    Read last N messages
                            Output format:
                              [1] a1b2c3d4-... user: message text
                              [2] e5f6a7b8-... user: message text

  edit <index> <text>       Edit a message by its index from last `read`
  delete <index>            Delete a message by its index

Reactions:
  react <message-id> <emoji>  Add a reaction by message ID
  unreact <message-id> <emoji> Remove a reaction by message ID

Watch mode:
  watch <username>          Stream incoming messages in real-time (SSE)
```

Config is stored at `~/.agent-message/config` (JSON with server URL and session token).

---

## File & Attachment Handling

- Supported upload types: images (jpg, png, gif, webp), files (pdf, txt, zip, etc.)
- Max file size: **20 MB**
- Files stored on the server's local disk under a configurable `UPLOAD_DIR` (default: `./uploads/`).
- Files served from `/static/uploads/`.
- In web client: images render inline; other files show as a download link.
- In CLI: attachments shown as a URL.

---

## Emoji Reactions

- Any valid Unicode emoji is allowed.
- A user can only have **one reaction per emoji per message** (toggle behavior).
- Reactions are grouped by emoji with a count in the UI.

---

## Non-Goals (explicitly out of scope)

- Voice / video calls
- Group chats or channels
- Message forwarding / pinning
- Push notifications (mobile)
- End-to-end encryption
- Phone number / email authentication
- Profile pictures / avatars (text initial shown instead)
- Read receipts
- Online presence / typing indicators

---

## Project Structure

```
agent-message/
├── server/              # Go server
│   ├── main.go
│   ├── api/             # HTTP handlers
│   ├── realtime/        # Realtime event hub
│   ├── store/           # Database layer
│   └── models/
├── web/                 # React + TypeScript client
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── hooks/
│   │   └── api/         # API client
│   └── package.json
├── cli/                 # Go CLI client
│   ├── main.go
│   └── cmd/
├── SPEC.md
└── README.md
```
