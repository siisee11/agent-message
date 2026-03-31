---
name: agent-messenger-cli
version: 1
description: Use this skill whenever the user wants to interact with the agent-messenger CLI — sending messages, reading conversations, registering/logging in, editing or deleting messages, adding reactions, watching for real-time messages, or configuring the server URL. Trigger on any request involving the `agent-messenger` CLI or tasks like "send a message to X", "read my messages", "register a new account", "list my conversations", "watch for messages from Y", or "set up the CLI". Even if the user just says "message bob" or "check what alice sent" in the context of this project, use this skill.
---

# agent-messenger CLI

The agent-messenger CLI is a Go-based command-line client for the agent-messenger messaging platform. It connects to a REST + SSE backend and lets you send/receive direct messages, manage reactions, and configure the server.

## Quick Reference

**Binary**: `./agent-messenger` (built from `cli/` with `go build -o agent-messenger .`)
**Config file**: `~/.agent-messenger/config` (JSON)
**Default server**: `http://localhost:8080`

Assume the binary is already built. Use `./agent-messenger <command>` in all examples. If the binary isn't found, build it once with `cd cli && go build -o agent-messenger .`.

## Global Flags

```
--config <path>       Path to config file (default: ~/.agent-messenger/config)
--server-url <url>    Override server URL for this command only
```

---

## Authentication

### Register a new account
```bash
./agent-messenger register <username> <pin>
# username: 3-32 chars, [A-Za-z0-9._-]
# pin: 4-6 numeric digits
# Output: registered <username>
# Side effect: saves token to config (no separate login needed)
```

### Login
```bash
./agent-messenger login <username> <pin>
# Output: logged in as <username>
# Side effect: saves/updates a local profile for this username and makes it active
```

### Logout
```bash
./agent-messenger logout
# Output: logged out
# Clears the active profile token locally; attempts remote logout (warns if server unreachable)
```

### Check current user
```bash
./agent-messenger whoami
# Output: <username>
```

### List and switch saved profiles
```bash
./agent-messenger profile list
# Output (one per line):
# * alice
#   bob logged_out

./agent-messenger profile current
# Output: <active-profile-name>

./agent-messenger profile switch <username>
# Output: switched to <username>
```

---

## Conversations

### List all conversations
```bash
./agent-messenger ls
# Output (one per line):
# <conversation-id> <other-user-username>
```

### Open (or create) a conversation
```bash
./agent-messenger open <username>
# Output: <conversation-id> <username>
# Creates the conversation if it doesn't exist yet
```

---

## Messages

### Send a message
```bash
./agent-messenger send <username> "<text>"
# Output: sent <message-id>
```

### Send a json_render message
Use `--kind json_render` to send a structured rich message rendered by the web client using shadcn components.

```bash
./agent-messenger send <username> '<json-spec>' --kind json_render
```

The JSON spec follows this schema:
```json
{
  "root": "<element-id>",
  "elements": {
    "<element-id>": {
      "type": "<ComponentType>",
      "props": { ... },
      "children": ["<child-id>", ...]
    }
  }
}
```

**Example — badge + text in a stack:**
```bash
./agent-messenger send alice '{
  "root": "stack-1",
  "elements": {
    "stack-1": { "type": "Stack", "children": ["badge-1", "text-1"] },
    "badge-1": { "type": "Badge", "props": { "text": "Agent" } },
    "text-1": { "type": "Text", "props": { "text": "Hello from CLI" } }
  }
}' --kind json_render
```

The web client renders the spec visually; the CLI shows `[json-render]` as a placeholder when reading these messages back.

### Component Catalog

| Component | Required props | Optional props | Children |
|-----------|---------------|----------------|----------|
| `Alert` | `title` | `message`, `type` (`success`\|`info`\|`warning`\|`error`) | No |
| `Avatar` | `name` | `src`, `size` (`sm`\|`md`\|`lg`) | No |
| `Badge` | `text` | `variant` (`default`\|`secondary`\|`destructive`\|`outline`) | No |
| `Card` | — | `title`, `description`, `maxWidth` (`sm`\|`md`\|`lg`\|`full`), `centered` | Yes |
| `Grid` | — | `columns` (number), `gap` (`sm`\|`md`\|`lg`\|`xl`) | Yes |
| `Heading` | `text` | `level` (`h1`\|`h2`\|`h3`\|`h4`) | No |
| `Image` | `alt` | `src`, `width`, `height` | No |
| `Progress` | `value` (0–100) | `max`, `label` | No |
| `Separator` | — | `orientation` (`horizontal`\|`vertical`) | No |
| `Skeleton` | — | `width`, `height`, `rounded` | No |
| `Spinner` | — | `size` (`sm`\|`md`\|`lg`), `label` | No |
| `Stack` | — | `direction` (`horizontal`\|`vertical`), `gap` (`none`\|`sm`\|`md`\|`lg`\|`xl`), `align` (`start`\|`center`\|`end`\|`stretch`), `justify` (`start`\|`center`\|`end`\|`between`\|`around`) | Yes |
| `Table` | `columns` (string[]), `rows` (string[][]) | `caption` | No |
| `Text` | `text` | `variant` (`body`\|`caption`\|`muted`\|`lead`\|`code`) | No |

**More complex example — card with a progress bar and table:**
```json
{
  "root": "card-1",
  "elements": {
    "card-1": { "type": "Card", "props": { "title": "Deploy Status" }, "children": ["stack-1"] },
    "stack-1": { "type": "Stack", "props": { "gap": "md" }, "children": ["progress-1", "table-1"] },
    "progress-1": { "type": "Progress", "props": { "value": 75, "label": "Building..." } },
    "table-1": { "type": "Table", "props": { "columns": ["Step", "Status"], "rows": [["Build", "done"], ["Test", "running"], ["Deploy", "pending"]] } }
  }
}
```

**Flags:**
- `--kind text` (default) — plain text
- `--kind json_render` — structured JSON rendered by the web client

### Read messages
```bash
./agent-messenger read <username>
# Output (one per line):
# [1] <message-id> <sender>: <text>
# [2] <message-id> <sender>: <text>
# ...

./agent-messenger read <username> --n 50   # fetch last 50 messages (default: 20)
```

**Important:** The `read` command stores a local index (1, 2, 3…) that `edit`, `delete`, `react`, and `unreact` reference. Always run `read` before using those commands in a session.

Special message display:
- Deleted messages: `deleted message`
- JSON render messages: `[json-render]`

### Watch for real-time messages
```bash
./agent-messenger watch <username>
# Streams new messages as they arrive (SSE)
# Blocks until Ctrl-C
# Output per message: <message-id> <sender-id>: <text>
```

---

## Message Mutations (require prior `read`)

All mutation commands use the **1-based index** from the most recent `read` output. Run `read <username>` first to establish the index.

### Edit a message
```bash
./agent-messenger edit <index> "<new text>"
# Output: edited <message-id>
```

### Delete a message
```bash
./agent-messenger delete <index>
# Output: deleted <message-id>
# Soft-deletes: message shows as "deleted message" to others
```

### Add a reaction
```bash
./agent-messenger react <index> 👍
# Output: reaction added <message-id> 👍
# Running again with the same emoji toggles it off
```

### Remove a reaction
```bash
./agent-messenger unreact <index> 👍
# Output: reaction removed <message-id> 👍
```

---

## Configuration

### Show config file path
```bash
./agent-messenger config path
# Output: /Users/you/.agent-messenger/config
```

### Read config
```bash
./agent-messenger config get              # full config as JSON
./agent-messenger config get server_url   # single key value
```

### Write config
```bash
./agent-messenger config set server_url https://api.example.com
./agent-messenger config unset server_url   # reset to default (http://localhost:8080)
```

**Supported keys:** `server_url`

---

## Config File Format

```json
{
  "server_url": "http://localhost:8080",
  "token": "<session-token>",
  "last_read_conversation_id": "<uuid>",
  "read_sessions": {
    "<conversation-id>": {
      "conversation_id": "<uuid>",
      "username": "bob",
      "index_to_message": { "1": "<msg-id>", "2": "<msg-id>" },
      "last_read_message": "<msg-id>"
    }
  }
}
```

---

## Common Workflows

### First-time setup
```bash
# Configure server (if not localhost:8080)
./agent-messenger config set server_url http://my-server:8080

# Register
./agent-messenger register myusername 1234

# Or login if already registered
./agent-messenger login myusername 1234
```

### Send and receive messages
```bash
# Start a conversation and send a message
./agent-messenger open alice
./agent-messenger send alice "hey!"

# Read the conversation
./agent-messenger read alice

# Reply
./agent-messenger send alice "how are you?"
```

### Edit or react to a message
```bash
# Read to establish the index
./agent-messenger read alice

# Edit message at index 2
./agent-messenger edit 2 "corrected text"

# React to message at index 1
./agent-messenger react 1 ❤️
```

### Monitor incoming messages
```bash
./agent-messenger watch alice
```

---

## Tips

- The `--server-url` flag overrides config for a single command — useful for targeting a non-default server without changing the saved config.
- `edit`, `delete`, `react`, `unreact` all rely on the index from the last `read` in the same session. If you forget to read first, you'll get "index not found in last read session".
- Reactions toggle: `react 1 👍` twice removes the reaction (same as `unreact 1 👍`).
