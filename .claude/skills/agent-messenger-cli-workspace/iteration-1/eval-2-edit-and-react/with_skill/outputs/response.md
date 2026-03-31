# How to Edit a Message and Add a Reaction

Since you already ran `read alice`, the index from that session is still active — you can use it directly for both operations.

## Fix the typo in message at index 3

Use the `edit` command with the index and the corrected text:

```bash
go run . edit 3 "your corrected text here"
# Output: edited <message-id>
```

Replace `"your corrected text here"` with the full corrected version of the message.

## Add a thumbs-up reaction to message at index 1

Use the `react` command with the index and the emoji:

```bash
go run . react 1 👍
# Output: reaction added <message-id> 👍
```

## Important notes

- Both commands use the 1-based index from the most recent `read` output in your current session. Since you just ran `read alice`, the index is already established and these commands will work immediately.
- If you start a new terminal session or run `read` again before doing the edit/react, the index will be refreshed — just make sure to re-check which index corresponds to which message.
- For `react`: running the same emoji command again (e.g. `react 1 👍` a second time) will toggle the reaction off. Use `unreact 1 👍` explicitly if you want to remove it later.
- Both commands must be run from the `cli/` directory (or using the built binary).
