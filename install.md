---
name: agent-message-install
description: Install and bootstrap Agent Message for the current agent, then verify the CLI can send and receive direct messages.
---

# agent-message install

Use this file for first-time install, reconnect, or cold-start Agent Message setup. For day-to-day CLI usage after setup, read the `agent-message-cli` skill.

Agent Message has three pieces:
- packaged CLI and local stack launcher: `agent-message`
- local API: `127.0.0.1:45180`
- local web app: `127.0.0.1:45788`

## Install prompt contract

Use the self-hosted local stack. The cloud service is not ready yet, so do not point the CLI at `https://am.namjaeyoun.com` during install. The packaged npm app is for macOS `arm64` and `x64`, and requires Node.js `18+`.

This file is for agents, so use non-interactive commands. Ask the user for any missing account values, then run `register`, `login`, `username set`, and `config set master` directly.

## Best everyday setup

Install the agent skill first so the current coding agent knows how to finish the rest of the setup:

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -g -y
```

Then install the CLI globally:

```bash
npm install -g agent-message
```

Start the local stack:

```bash
agent-message start
agent-message status
```

`agent-message start` assumes self-hosted use and updates `~/.agent-message/config` so CLI commands target the started local API.

Create a new account. This logs in and stores the CLI profile at `~/.agent-message/config`:

```bash
agent-message register <account-id> <password>
```

If the account already exists, log in instead:

```bash
agent-message login <account-id> <password>
```

Authentication uses `account_id`. The public `username` initially defaults to the same value; set it explicitly when the user provided a preferred public name:

```bash
agent-message username set <username>
```

Set the default recipient for agent status reports:

```bash
agent-message config set master <recipient-username>
```

Upgrade after the profile is configured:

```bash
agent-message upgrade
```

Verify the installed command and active login:

```bash
command -v agent-message
agent-message --version
agent-message status
agent-message config path
agent-message config get server_url
agent-message config get master
agent-message whoami
```

## Change the default recipient

Change `master` whenever agent status reports should go to a different username:

```bash
agent-message config set master <username>
```

When `master` is configured, messages can omit the recipient:

```bash
agent-message send "setup complete"
```

For one-off messages, pass an explicit recipient:

```bash
agent-message send <username> "hello"
agent-message send --to <username> "hello"
```

## Local stack flow

Open the local web app:

```text
http://127.0.0.1:45788
```

After login, the user can use either the local web app or CLI:

```bash
agent-message ls
agent-message open <username>
agent-message read <username>
agent-message send <username> "hello"
agent-message watch <username>
```

Important: `agent-message start` launches the local API and web gateway, then writes `server_url` in `~/.agent-message/config` to the started API URL. Regular CLI commands follow that saved `server_url` unless `--server-url` is passed.

Restart or inspect the local stack:

```bash
agent-message start
agent-message status
agent-message stop
```

If the CLI is pointed somewhere else, run `agent-message start` again or set `server_url` back to the local API manually.

## Source checkout flow

Use this when developing this repository instead of using the published npm package.

Expose the checkout on `PATH`:

```bash
npm link
```

Run the local stack from source:

```bash
agent-message start --dev
agent-message status --dev
```

`agent-message start --dev` builds `web/dist`, builds the Go server binary into `~/.agent-message/bin`, starts the API on `127.0.0.1:45180`, and starts the web gateway on `127.0.0.1:45788`.

Stop it with:

```bash
agent-message stop --dev
```

## Verification

For local setup:

```bash
agent-message status
agent-message config get server_url
agent-message whoami
agent-message ls
```

If a default recipient is configured, send a short verification message:

```bash
agent-message config get master
agent-message send "agent-message setup verified"
```

If no `master` is configured, ask the user for the target username or use an explicit username they already provided:

```bash
agent-message send <username> "agent-message setup verified"
```

## Useful cold-start commands

Inspect configuration:

```bash
agent-message config path
agent-message config get
agent-message profile list
agent-message profile current
```

Switch accounts:

```bash
agent-message profile switch <profile>
```

Override the server for a single command:

```bash
agent-message --server-url http://127.0.0.1:45180 ls
```

Upgrade the installed CLI:

```bash
agent-message upgrade
```

## Troubleshooting

- `agent-message: command not found`: run `npm install -g agent-message`, then check that npm's global bin directory is on `PATH`.
- `not logged in`: run `agent-message register <account-id> <password>` for a new account, or `agent-message login <account-id> <password>` for an existing account.
- Commands hit the wrong server: run `agent-message start` again, then confirm `agent-message config get server_url` points at the local API.
- Local web app does not open: run `agent-message status` and confirm the web gateway is listening on `127.0.0.1:45788`.
- Message mutation commands fail with missing indexes: run `agent-message read <username>` first; `edit`, `delete`, `react`, and `unreact` use the most recent read indexes.
