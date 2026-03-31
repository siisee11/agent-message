# How to Fix a Typo and Add a Reaction in agent-messenger

## Fix the Typo in Message at Index 3

Use the `edit` command with the message index and the corrected text:

```
agent-messenger edit 3 "Your corrected message text here"
```

The `edit` command takes two arguments: the index number (from the last `read` session) and the new full text for the message. It will replace the entire message content with the text you provide.

**Important:** You must provide the complete corrected message text as the second argument. The CLI does not support partial edits — the entire message body is replaced.

## Add a Thumbs-Up Reaction to Message at Index 1

Use the `react` command with the message index and the emoji:

```
agent-messenger react 1 👍
```

The `react` command takes two arguments: the index number and the emoji string. This will add a thumbs-up reaction to the message at index 1 from your last `read alice` session.

## How It Works

Both commands rely on the read session stored from your last `read alice` run. The CLI maps the displayed index numbers to internal message IDs, so you use the same numeric indices you saw in the `read` output (e.g., `1`, `3`).

- `edit <index> <text>` — edits the message body at the given index
- `react <index> <emoji>` — adds a reaction emoji to the message at the given index
- `unreact <index> <emoji>` — removes a reaction if you need to undo one
- `delete <index>` — deletes a message entirely

If you have navigated away and your read session is stale, re-run `read alice` first to refresh the index mapping before editing or reacting.
