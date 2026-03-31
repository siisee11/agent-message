# Register a New Account and Send a Message

Here are the exact commands to register a new account with username `testuser` and pin `5678`, then send a message to alice.

## Step 1: Navigate to the CLI directory

```bash
cd /path/to/agent-messenger/cli
```

## Step 2: Register the new account

```bash
go run . register testuser 5678
```

Expected output:
```
registered testuser
```

This also saves the session token to `~/.agent-messenger/config` automatically, so you are logged in immediately after registering.

## Step 3: Send a message to alice

```bash
go run . send alice "hello from testuser"
```

Expected output:
```
sent <message-id>
```

## Full sequence

```bash
cd /path/to/agent-messenger/cli
go run . register testuser 5678
go run . send alice "hello from testuser"
```

## Notes

- Replace `/path/to/agent-messenger/cli` with the actual path to the `cli/` directory in your local clone of the repo (e.g., `/Users/dev/git/agent-messenger/cli`).
- The `register` command requires a username of 3–32 characters using `[A-Za-z0-9._-]` and a pin of 4–6 numeric digits. Both `testuser` and `5678` satisfy these requirements.
- After registering, you do not need to run `login` — the token is saved immediately.
- The `send` command will create a conversation with alice if one does not already exist.
