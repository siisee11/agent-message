# Connecting the agent-messenger CLI to a Remote Server

## Permanently Saving the Server URL

To permanently point the CLI at `https://messaging.example.com`, run the following from the `cli/` directory:

```bash
go run . config set server_url https://messaging.example.com
```

This writes the URL to the config file at `~/.agent-messenger/config` and all subsequent commands will use it automatically — no flags needed.

## Verifying the Setting

You have two ways to confirm the value was saved correctly.

**Check just the server_url key:**

```bash
go run . config get server_url
```

Expected output:
```
https://messaging.example.com
```

**Check the full config as JSON:**

```bash
go run . config get
```

This prints the entire config file, where you should see:
```json
{
  "server_url": "https://messaging.example.com",
  ...
}
```

**Check the config file path (optional):**

```bash
go run . config path
```

This prints the absolute path to the config file (e.g., `/Users/you/.agent-messenger/config`) so you can inspect it directly if needed.

## One-Off Override (Without Saving)

If you ever need to target a different server for a single command without changing your saved config, use the global `--server-url` flag:

```bash
go run . --server-url https://messaging.example.com whoami
```

This overrides the config for that command only and leaves the saved value unchanged.

## Resetting to the Default

If you later want to revert to the default (`http://localhost:8080`):

```bash
go run . config unset server_url
```
