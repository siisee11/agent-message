# claude-message

`claude-message` wraps `claude -p --output-format json` and uses `agent-message`
as its transport layer.

Behavior:

1. Starts a fresh `agent-{chatId}` account with a generated password.
2. Sends the `--to` user a startup message with the generated credentials.
3. Reuses the Claude `session_id` for the DM session and resumes later turns.
4. Watches `agent-message` DMs for plain-text requests, adds a `👀` reaction to
   each accepted inbound DM, and posts Claude's JSON result back as
   `json_render`.
5. After a successful turn completion, replaces the inbound `👀` reaction with
   `✅`.

Example:

```bash
cd claude-message
cargo run -- --to jay --model sonnet --permission-mode accept-edits
```

Build from the repo root:

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

- `claude-message` always passes `--dangerously-skip-permissions` to Claude.
- The wrapper therefore does not pause for Claude permission approvals.
