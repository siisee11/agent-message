# Register and Send Message with agent-messenger CLI

## Overview

The agent-messenger CLI is a Go-based messaging tool located at `/Users/dev/git/agent-messenger/cli/`. The binary can be built and run from that directory. The default server URL is `http://localhost:8080` and configuration is stored at `~/.agent-messenger/config`.

## Step 1: Build the CLI

First, build the binary from the CLI directory:

```sh
cd /Users/dev/git/agent-messenger/cli && go build -o agent-messenger .
```

## Step 2: Register a new account

Use the `register` command with `<username>` and `<pin>` as positional arguments:

```sh
./agent-messenger register testuser 5678
```

Expected output:
```
registered testuser
```

This command registers the account and automatically saves an auth token to `~/.agent-messenger/config`, so you are logged in immediately after registration.

## Step 3: Send a message to alice

Use the `send` command with `<username>` and `<text>` as positional arguments:

```sh
./agent-messenger send alice 'hello from testuser'
```

Expected output:
```
sent <message-id>
```

The `send` command will open (or reuse) a direct-message conversation with `alice` and send the text message.

## Complete command sequence

```sh
cd /Users/dev/git/agent-messenger/cli
go build -o agent-messenger .
./agent-messenger register testuser 5678
./agent-messenger send alice 'hello from testuser'
```

## Notes

- If running the binary from outside the CLI directory, use the full path: `/Users/dev/git/agent-messenger/cli/agent-messenger`
- The default server must be running at `http://localhost:8080`. To use a different server, add the `--server-url` flag:
  ```sh
  ./agent-messenger --server-url http://your-server:port register testuser 5678
  ./agent-messenger --server-url http://your-server:port send alice 'hello from testuser'
  ```
- After `register`, the token is persisted in `~/.agent-messenger/config`. Subsequent commands use it automatically without needing to log in again.
- The `send` command accepts an optional `--kind` flag (default: `text`). No flag is needed for plain text messages.
