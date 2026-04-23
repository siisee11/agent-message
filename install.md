---
name: agent-message-install
description: Install and bootstrap Agent Message for the current agent using the self-hosted local stack.
---

# agent-message install

Use this file only for first-time agent setup. Keep the flow limited to skill install, npm install, account setup, and configuration setup.

## 1. Install the agent skill

Install the `agent-message-cli` skill first so the current coding agent knows the CLI commands:

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -a codex -a claude-code -g -y
```

## 2. Install the npm packages

Always install the packaged CLI and local stack launcher. Before installing, check whether the user's computer has Codex or Claude Code available, and install the matching wrapper package too:

```bash
packages=(agent-message)

if command -v codex >/dev/null 2>&1; then
  packages+=(codex-message)
fi

if command -v claude-code >/dev/null 2>&1 || \
  command -v claude >/dev/null 2>&1 || \
  npm list -g @anthropic-ai/claude-code --depth=0 >/dev/null 2>&1; then
  packages+=(claude-message)
fi

npm install -g "${packages[@]}"
```

Install behavior:
- `agent-message` is always installed.
- `codex-message` is installed only when the `codex` CLI is present.
- `claude-message` is installed when `claude-code`, `claude`, or the global `@anthropic-ai/claude-code` npm package is present.

## 3. Start the local stack

Start the self-hosted local stack:

```bash
agent-message start
agent-message status
```

`agent-message start` updates `~/.agent-message/config` so CLI commands target the started local API.

## 4. Set up the account

Ask the user for the `account-id` before registering. The account ID is used for authentication and must be `3-32` characters using only letters, numbers, `.`, `_`, or `-`.

Use `0000` as the initial default password:

```bash
agent-message register <account-id> 0000
```

If the account already exists, log in with the same default password:

```bash
agent-message login <account-id> 0000
```

Important: `0000` is only a temporary setup password. Tell the user they must change it immediately after setup from the local web app Profile page:

```text
http://127.0.0.1:45788
```

## 5. Set up CLI configuration

If the user wants a public display name different from `account-id`, set it:

```bash
agent-message username set <username>
```

Ask the user who should receive agent status reports, then set that username as `master`:

```bash
agent-message config set master <recipient-username>
```

Verify the setup:

```bash
agent-message config get server_url
agent-message config get master
agent-message whoami
```

## Uninstall

To remove Agent Message later, stop the local stack and uninstall the global npm package:

```bash
agent-message uninstall
```

This keeps `~/.agent-message` by default. To also delete local accounts, SQLite data, uploads, logs, and CLI profiles:

```bash
agent-message uninstall --purge
```
