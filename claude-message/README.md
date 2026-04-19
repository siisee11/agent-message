# claude-message

`claude-message` wraps `claude -p --output-format json` and uses `agent-message`
as its transport layer.

Install:

```bash
npm install -g agent-message claude-message
```

`claude-message` expects both `agent-message` and the `claude` CLI to already
be available on your `PATH`.

If `--to` is omitted, `claude-message` uses the current `agent-message` `master` value.

Behavior:

1. Starts a fresh `agent-{chatId}` account with a generated password.
2. Sends the `--to` user a startup message with the generated credentials.
3. Reuses the Claude `session_id` for the DM session and resumes later turns.
4. Watches `agent-message` DMs for plain-text requests, adds a `👀` reaction to
   each accepted inbound DM, and passes the request to Claude with explicit
   instructions to send the final user-facing result directly with
   `agent-message send --from agent-{chatId}`.
5. If Claude fails, the wrapper sends a failure `json_render` notice itself.
6. After a successful turn completion, replaces the inbound `👀` reaction with
   `✅`.

`claude-message` now follows the same delivery model as `codex-message` for
successful turns: the agent is expected to send the final result itself. The
wrapper still handles startup, reactions, and failure notices.

Example:

```bash
agent-message config set master jay
claude-message --model sonnet --permission-mode accept-edits
claude-message --to alice --model sonnet --permission-mode accept-edits
claude-message --bg --to alice --model sonnet --permission-mode accept-edits
claude-message upgrade
```

Build from the repo root:

```bash
make claude-message-build
./claude-message/target/debug/claude-message --to jay --model sonnet
```

Useful flags:

- `--to <username>` overrides `agent-message` `master`
- `--cwd /path/to/worktree`
- `--model sonnet`
- `--permission-mode accept-edits`
- `--allowed-tools Read,Edit`
- `--bare`

Background run:

- `claude-message --bg ...` detaches the wrapper and prints the PID, log path, and metadata path.
- Logs and metadata are written under `~/.agent-message/wrappers/claude-message/`.
- `claude-message list` shows running background sessions. Use `claude-message list --all` to include stale metadata.
- `claude-message kill <session-id|pid|all>` stops one session or every running background session.

Notes:

- `claude-message` always passes `--dangerously-skip-permissions` to Claude.
- The wrapper therefore does not pause for Claude permission approvals.
