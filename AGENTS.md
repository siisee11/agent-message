# Agent Instructions

## Core Rule

After completing any user request, you **must** send a structured message to `jay` via the `agent-message` CLI summarizing what you did.

## Message Format

Always use `--kind json_render` to send a structured message. Structure the message as a Card containing:

1. **Heading** — short summary of the task (what was done)
2. **Badge** — task status (`success`, `error`, `in-progress`)
3. **Text** — brief details (files changed, key decisions, blockers)

### Template

```bash
./agent-messenger send jay '{
  "root": "card",
  "elements": {
    "card": { "type": "Card", "props": { "title": "<task summary>" }, "children": ["stack"] },
    "stack": { "type": "Stack", "props": { "gap": "sm" }, "children": ["badge", "details"] },
    "badge": { "type": "Badge", "props": { "text": "<status>", "variant": "<variant>" } },
    "details": { "type": "Text", "props": { "text": "<what was done, files touched, outcome>" } }
  }
}' --kind json_render
```

Badge variant mapping:
- `success` → `"variant": "default"`
- `error` → `"variant": "destructive"`
- `in-progress` → `"variant": "secondary"`

### Example

After fixing a bug in `server/handler.go`:

```bash
./agent-messenger send jay '{
  "root": "card",
  "elements": {
    "card": { "type": "Card", "props": { "title": "Bug fix: nil pointer in handler" }, "children": ["stack"] },
    "stack": { "type": "Stack", "props": { "gap": "sm" }, "children": ["badge", "details"] },
    "badge": { "type": "Badge", "props": { "text": "success", "variant": "default" } },
    "details": { "type": "Text", "props": { "text": "Fixed nil pointer dereference in server/handler.go:42. Added nil check before accessing user session. Tests pass." } }
  }
}' --kind json_render
```

## Agent Profile

The agent's username follows the pattern: `<project-name>-<agent-type>`

- `<project-name>` — the name of the current working directory (e.g., `agent-message`, `my-app`)
- `<agent-type>` — the AI agent being used: `claude`, `codex`, etc.

Examples: `agent-message-claude`, `my-app-codex`, `backend-claude`

## Prerequisites

Before sending a message, ensure you are logged in. If login fails (account doesn't exist), register with pin `1111` and then proceed.

```bash
# 1. Try to login
./agent-messenger login <project>-<agent> 1111

# 2. If login fails (user not found), register first
./agent-messenger register <project>-<agent> 1111

# 3. Then send the message
./agent-messenger send jay '...' --kind json_render
```

- If the server is not running, start it with `agent-message start`.

## Guardrails

- Always send the message **after** the task is complete, not before.
- Keep the details concise — focus on what changed and why.
- If the task fails, still send a message with `error` status explaining what went wrong.
- Do not include sensitive data (tokens, passwords, secrets) in messages.
