# Connect to Agent Message with `--with-tunnel`

Use this guide when you want the local `agent-message start` stack to be reachable through the checked-in Cloudflare named tunnel.

For general self-host production use with your own Cloudflare Tunnel token, prefer `docker-compose.selfhost.yml`. The `--with-tunnel` option is for the local lifecycle command and currently uses the named tunnel configuration in `deploy/agent_tunnel_config.yml`.

## Requirements

- Agent Message is installed on the host Mac.
- `cloudflared` is installed and available on `PATH`.
- The Cloudflare named tunnel exists in the Cloudflare account.
- The tunnel credentials file exists at the path configured in `deploy/agent_tunnel_config.yml`.
- The tunnel hostnames point to the named tunnel in Cloudflare DNS.

## Start Agent Message with the tunnel

Run:

```bash
agent-message start --with-tunnel
```

Then open one of your configured hostnames:

```text
https://agent.example.com
https://agent-message.example.com
```

The command starts:

- the local API server on `127.0.0.1:45180`
- the local web gateway on `127.0.0.1:45788`
- `cloudflared tunnel --config deploy/agent_tunnel_config.yml run <tunnel-name>`

`--with-tunnel` requires the default web listener, so do not override `--web-host` or `--web-port` for this mode.

## Stop the local stack and tunnel

```bash
agent-message stop --with-tunnel
```

`agent-message stop` also stops the tunnel process when the tunnel pidfile exists, but keeping `--with-tunnel` in the command makes the intent explicit.

## Troubleshooting

Check status:

```bash
agent-message status --with-tunnel
```

Check logs:

```bash
tail -f ~/.agent-message/logs/server.log
tail -f ~/.agent-message/logs/gateway.log
tail -f ~/.agent-message/logs/named-tunnel.log
```

If the tunnel does not connect, verify:

- `cloudflared` is installed:

  ```bash
  cloudflared --version
  ```

- the credentials file from `deploy/agent_tunnel_config.yml` exists
- the hostnames are routed to the same named tunnel in Cloudflare
- the local gateway is reachable:

  ```bash
  curl -I http://127.0.0.1:45788
  ```

## Notes

- This mode is useful for testing HTTPS-only browser behavior such as iPhone PWA install and web push.
- The tunnel hostnames and credentials path are currently project-specific. To use your own domain with `agent-message start --with-tunnel`, update `deploy/agent_tunnel_config.yml` and the tunnel name used by the npm wrapper.
- For a long-running self-host deployment, use `docker-compose.selfhost.yml` with `CLOUDFLARE_TUNNEL_TOKEN` instead of `--with-tunnel`.
