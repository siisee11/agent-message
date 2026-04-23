---
name: agent-message-install
description: Install and bootstrap Agent Message for the current agent using the self-hosted local stack.
---

# agent-message install

Use this file only for first-time agent setup. Keep the flow limited to skill install, npm install, account setup, and configuration setup.

## 1. Install the agent skill

Install the `agent-message-cli` skill first so the current coding agent knows the CLI commands:

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -g -y
```

## 2. Install the npm package

Install the packaged CLI and local stack launcher:

```bash
npm install -g agent-message
```

Start the self-hosted local stack:

```bash
agent-message start
agent-message status
```

`agent-message start` updates `~/.agent-message/config` so CLI commands target the started local API.

## 3. Set up the account

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

## 4. Set up CLI configuration

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
