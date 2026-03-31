# Configuring the Remote Server URL

## How to Permanently Save the Server URL

Use the `config set` subcommand to persist the server URL:

```
agent-messenger config set server_url https://messaging.example.com
```

This writes the value to the config file at `~/.agent-messenger/config` (a JSON file). The command prints the normalized URL that was saved, so you can confirm it was accepted immediately.

The only supported key for `config set` is `server_url`. The value must begin with `http://` or `https://` and must include a host — the CLI validates this before writing.

## How to Verify It Is Set Correctly

**Option 1 — read just the server_url key:**

```
agent-messenger config get server_url
```

This prints only the stored server URL.

**Option 2 — read the full config as JSON:**

```
agent-messenger config get
```

This prints the entire config file contents, including `server_url`, `token`, and any read-session state.

**Option 3 — check the config file path and inspect it directly:**

```
agent-messenger config path
```

This prints the path to the config file (default: `~/.agent-messenger/config`). You can then open or `cat` that file to see the raw JSON.

## One-time Override (Without Saving)

If you want to use the remote server for a single command without changing the persisted config, use the `--server-url` flag on any command:

```
agent-messenger --server-url https://messaging.example.com <subcommand>
```

This overrides the config for that invocation only; it does not write anything to disk.

## How It Works Internally

- The config file is stored at `~/.agent-messenger/config` by default (JSON format).
- The relevant field is `"server_url"` in the `Config` struct (`cli/internal/config/config.go`).
- On every command, the CLI reads the config file, then applies any `--server-url` flag override on top.
- `config set server_url` writes the new value to the config file and immediately reloads it, so the change takes effect for all subsequent commands.
- If you ever want to revert to the default (`http://localhost:8080`), run `agent-messenger config unset server_url`.
