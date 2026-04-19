# codex-message

`codex-message` is a separate Rust wrapper around `codex app-server` that uses
the `agent-message` binary as its transport layer and reuses the default
`agent-message` config/profile store.

Install:

```bash
npm install -g agent-message codex-message
```

`codex-message` expects both `agent-message` and the `codex` CLI to already be
available on your `PATH`.

By default `codex-message` always uses the installed `agent-message` on `PATH`.
If you explicitly want to run the repo checkout's CLI with `go run .`, set
`CODEX_MESSAGE_AGENT_MESSAGE_MODE=source`.

Behavior:

1. Starts a fresh `agent-{chatId}` account with a generated password.
2. Sends the target user a startup message with the generated credentials.
3. Reuses one Codex app-server thread for the DM session.
4. Polls `agent-message read <user>` for new plain-text requests, adds a `👀` reaction to each accepted inbound DM, and relays it into `turn/start`.
5. For approval and input requests, sends readable `json_render` prompts back to that user and waits for a text reply.
6. Tells Codex to send the final user-facing result itself by invoking `agent-message send --from agent-{chatId}` directly, typically as `json_render`.
7. After a successful turn completion, replaces the inbound `👀` reaction with `✅`.

If `--to` is omitted, `codex-message` uses the current `agent-message` `master` value.

Example:

```bash
agent-message config set master jay
codex-message --model gpt-5.4
codex-message --model gpt-5.4 --yolo
codex-message --to alice --model gpt-5.4
codex-message --bg --model gpt-5.4 --cwd /path/to/worktree
codex-message upgrade
```

Useful flags:

- `--to <username>` overrides `agent-message` `master`
- `--cwd /path/to/worktree`
- `--approval-policy on-request`
- `--sandbox workspace-write`
- `--network-access`
- `--yolo` = `--approval-policy never` + `--sandbox danger-full-access`
- `CODEX_MESSAGE_AGENT_MESSAGE_MODE=source` opts into using the repo checkout's `agent-message` CLI instead of the installed binary

Background run:

- `codex-message --bg ...` detaches the wrapper and prints the PID, log path, and metadata path.
- Logs and metadata are written under `~/.agent-message/wrappers/codex-message/`.
- `codex-message list` shows running background sessions. Use `codex-message list --all` to include stale metadata.
- `codex-message kill <session-id|pid|all>` stops one session or every running background session.
