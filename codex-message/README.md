# codex-message

`codex-message` is a separate Rust wrapper around `codex app-server` that uses
the `agent-message` binary as its transport layer and reuses the default
`agent-message` config/profile store.

Behavior:

1. Starts a fresh `agent-{chatId}` account with a random numeric PIN.
2. Sends the `--to` user a startup message with the generated credentials.
3. Reuses one Codex app-server thread for the DM session.
4. Polls `agent-message read <user>` for new plain-text requests, adds a `👀` reaction to each accepted inbound DM, and relays it into `turn/start`.
5. For approval and input requests, sends readable `json_render` prompts back to Jay and waits for a text reply.
6. Sends final Codex results back as `json_render` reports and, after a successful turn completion, replaces the inbound `👀` reaction with `✅`.

Example:

```bash
cd codex-message
cargo run -- --to jay --model gpt-5.4
```

Useful flags:

- `--to jay`
- `--cwd /path/to/worktree`
- `--approval-policy on-request`
- `--sandbox workspace-write`
- `--network-access`
