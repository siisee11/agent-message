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

Behavior:

1. Starts a fresh `agent-{chatId}` account with a generated password.
2. Sends the target user a startup message with the generated credentials.
3. Reuses one Codex app-server thread for the DM session.
4. Polls `agent-message read <user>` for new plain-text requests, adds a `👀` reaction to each accepted inbound DM, and relays it into `turn/start`.
5. For approval and input requests, sends readable `json_render` prompts back to that user and waits for a text reply.
6. Sends final Codex results back as `json_render` reports and, after a successful turn completion, replaces the inbound `👀` reaction with `✅`.

If `--to` is omitted, `codex-message` uses the current `agent-message` `master` value.

Example:

```bash
agent-message config set master jay
codex-message --model gpt-5.4
codex-message --model gpt-5.4 --yolo
codex-message --to alice --model gpt-5.4
```

Useful flags:

- `--to <username>` overrides `agent-message` `master`
- `--cwd /path/to/worktree`
- `--approval-policy on-request`
- `--sandbox workspace-write`
- `--network-access`
- `--yolo` = `--approval-policy never` + `--sandbox danger-full-access`
