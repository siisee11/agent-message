# Connect to Agent Message with Tailscale

Use this guide when Agent Message is running on one Mac and you want to open the web UI from another device in the same Tailscale tailnet.

## Requirements

- Agent Message is installed on the host Mac.
- Tailscale is installed and logged in on the host Mac.
- Tailscale is installed and logged in on the other device.
- Both devices are connected to the same tailnet.

## Start Agent Message for Tailscale access

By default, `agent-message start` binds the web UI to `127.0.0.1:45788`, which only works from the same Mac. Bind the web gateway to the Tailscale IP instead:

```bash
TAILSCALE_IP="$(tailscale ip -4)"
agent-message start --web-host "$TAILSCALE_IP"
```

Then open this URL from another Tailscale device:

```text
http://<TAILSCALE_IP>:45788
```

For example:

```text
http://100.80.12.34:45788
```

You can print the host Mac's Tailscale IP at any time:

```bash
tailscale ip -4
```

## Alternative: bind every interface

If you want the web gateway to listen on every network interface:

```bash
agent-message start --web-host 0.0.0.0
```

Then connect from another Tailscale device with:

```text
http://<host-tailscale-ip>:45788
```

Prefer binding to `$(tailscale ip -4)` when you only want Tailscale devices to reach the web UI.

## Stop the local stack

```bash
agent-message stop
```

## Notes

- The API can keep its default `127.0.0.1:45180` bind address. The web gateway runs on the same host and proxies API requests locally.
- macOS may ask whether to allow incoming connections for the Node or Agent Message process. Allow it if Tailscale devices cannot connect.
- This setup uses plain HTTP over the Tailscale private network. Public HTTPS, web push, and installable PWA behavior are better served by the self-host Cloudflare Tunnel setup.
- Change the temporary `0000` setup password immediately. Anyone in the allowed network path who can reach the web UI can access the login page.
