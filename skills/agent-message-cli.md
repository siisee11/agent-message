---
name: agent-message-cli
version: 1
description: Use this skill whenever the user wants to interact with the agent-message CLI — sending messages, reading conversations, registering/logging in, editing or deleting messages, adding reactions, watching for real-time messages, or configuring the server URL. Trigger on any request involving the `agent-message` CLI or tasks like "send a message to X", "read my messages", "register a new account", "list my conversations", "watch for messages from Y", or "set up the CLI". Even if the user just says "message bob" or "check what alice sent" in the context of this project, use this skill.
---

# agent-message CLI

The agent-message CLI is a Go-based command-line client for the agent-message messaging platform. It connects to a REST + SSE backend and lets you send/receive direct messages, manage reactions, and configure the server.

## Quick Reference

**Install**: `npm install -g agent-message`
**Config file**: `~/.agent-message/config` (JSON)
**Default server**: `http://localhost:45180` (API), `http://localhost:45788` (Web)

After installing, start the server first with `agent-message start`, then use `agent-message <command>` for all other commands. The npm wrapper automatically detects the running local server URL, so no manual `--server-url` configuration is needed.

## Global Flags

```
--config <path>       Path to config file (default: ~/.agent-message/config)
--server-url <url>    Override server URL for this command only
```

---

## Authentication

### Register a new account
```bash
agent-message register <username> <pin>
# username: 3-32 chars, [A-Za-z0-9._-]
# pin: 4-6 numeric digits
# Output: registered <username>
# Side effect: saves token to config (no separate login needed)
```

### Login
```bash
agent-message login <username> <pin>
# Output: logged in as <username>
# Side effect: saves/updates a local profile for this username and makes it active
```

### Logout
```bash
agent-message logout
# Output: logged out
# Clears the active profile token locally; attempts remote logout (warns if server unreachable)
```

### Check current user
```bash
agent-message whoami
# Output: <username>
```

### List and switch saved profiles
```bash
agent-message profile list
# Output (one per line):
# * alice
#   bob logged_out

agent-message profile current
# Output: <active-profile-name>

agent-message profile switch <username>
# Output: switched to <username>
```

---

## Conversations

### List all conversations
```bash
agent-message ls
# Output (one per line):
# <conversation-id> <other-user-username>
```

### Open (or create) a conversation
```bash
agent-message open <username>
# Output: <conversation-id> <username>
# Creates the conversation if it doesn't exist yet
```

---

## Messages

### Send a message
```bash
agent-message send <username> "<text>"
# Output: sent <message-id>
```

### Send a json_render message
Use `--kind json_render` to send a structured rich message rendered by the web client using shadcn components.

```bash
agent-message send <username> '<json-spec>' --kind json_render
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
agent-message send alice '{
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
agent-message read <username>
# Output (one per line):
# [1] <message-id> <sender>: <text>
# [2] <message-id> <sender>: <text>
# ...

agent-message read <username> --n 50   # fetch last 50 messages (default: 20)
```

**Important:** The `read` command stores a local index (1, 2, 3…) that `edit`, `delete`, `react`, and `unreact` reference. Always run `read` before using those commands in a session.

Special message display:
- Deleted messages: `deleted message`
- JSON render messages: `[json-render]`

### Watch for real-time messages
```bash
agent-message watch <username>
# Streams new messages as they arrive (SSE)
# Blocks until Ctrl-C
# Output per message: <message-id> <sender-id>: <text>
```

---

## Message Mutations (require prior `read`)

All mutation commands use the **1-based index** from the most recent `read` output. Run `read <username>` first to establish the index.

### Edit a message
```bash
agent-message edit <index> "<new text>"
# Output: edited <message-id>
```

### Delete a message
```bash
agent-message delete <index>
# Output: deleted <message-id>
# Soft-deletes: message shows as "deleted message" to others
```

### Add a reaction
```bash
agent-message react <index> 👍
# Output: reaction added <message-id> 👍
# Running again with the same emoji toggles it off
```

### Remove a reaction
```bash
agent-message unreact <index> 👍
# Output: reaction removed <message-id> 👍
```

---

## Configuration

### Show config file path
```bash
agent-message config path
# Output: /Users/you/.agent-message/config
```

### Read config
```bash
agent-message config get              # full config as JSON
agent-message config get server_url   # single key value
```

### Write config
```bash
agent-message config set server_url https://api.example.com
agent-message config unset server_url   # reset to default (http://localhost:45180)
```

**Supported keys:** `server_url`

---

## Config File Format

```json
{
  "server_url": "http://localhost:45180",
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
agent-message config set server_url http://my-server:8080

# Register
agent-message register myusername 1234

# Or login if already registered
agent-message login myusername 1234
```

### Send and receive messages
```bash
# Start a conversation and send a message
agent-message open alice
agent-message send alice "hey!"

# Read the conversation
agent-message read alice

# Reply
agent-message send alice "how are you?"
```

### Edit or react to a message
```bash
# Read to establish the index
agent-message read alice

# Edit message at index 2
agent-message edit 2 "corrected text"

# React to message at index 1
agent-message react 1 ❤️
```

### Monitor incoming messages
```bash
agent-message watch alice
```

---

## Tips

- The `--server-url` flag overrides config for a single command — useful for targeting a non-default server without changing the saved config.
- `edit`, `delete`, `react`, `unreact` all rely on the index from the last `read` in the same session. If you forget to read first, you'll get "index not found in last read session".
- Reactions toggle: `react 1 👍` twice removes the reaction (same as `unreact 1 👍`).
