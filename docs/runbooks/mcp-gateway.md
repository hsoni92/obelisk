# MCP gateway

One MCP endpoint on Obelisk for your AI editors and agents. Add it once per editor; nothing is installed locally.

Built-in tools:

| Tool | What it does | Needs |
| --- | --- | --- |
| `web_search` | Top results (title, URL, snippet) from your SearXNG | An enabled SearXNG service (found automatically) |
| `send_mail` | Sends one plain-text mail to one recipient | Mail gateway enabled with a mailbox. Off by default |

Both tools are always listed to the client. A tool that is switched off, or whose backend is missing, returns a message that says what to fix.

## Set up

1. Admin UI → **Exposure** → **MCP gateway**.
2. Turn it on and **Generate** a token. Copy the token and the Cursor config shown once.
3. Add this to Cursor (`~/.cursor/mcp.json`) or any client that supports streamable HTTP MCP with headers:

```json
{
  "mcpServers": {
    "obelisk": {
      "url": "https://<your-host>/mcp",
      "headers": { "Authorization": "Bearer obk_mcp_..." }
    }
  }
}
```

Claude Code: `claude mcp add --transport http obelisk https://<your-host>/mcp --header "Authorization: Bearer obk_mcp_..."`.

## Reaching it

- **VPS profile:** use the same public URL as the other Exposure services.
- **Local machine:** reach it over Tailscale on the same tailnet.
- The token is a bearer token. Over plain HTTP it crosses the network unencrypted. Prefer HTTPS or the tailnet.

## `send_mail` safety

Recipients are not restricted, so a misled agent (for example one that read a hostile web page) could mail any address. Limits that apply:

- One recipient, plain text, body up to 32 KB, single-line subject up to 200 characters.
- At most 10 sends per hour and 30 per day, across all clients. Limits survive restarts.
- Identical sends within 10 minutes count once.
- Every attempt is logged (time, recipient, subject, size, body hash) and shown under **Recent sends**.

Keep `send_mail` off unless you need it. If the token leaks, **Rotate** it at once: the old token stops working immediately, on every machine, with no grace period.

## Files and limits

- Switches: `obelisk.yaml` → `mcp_gateway` (`enabled`, `tools`).
- Token: `mcp.credentials.json`; send log: `mcp.sendlog.json` under `OBELISK_CREDENTIALS_DIR`. The token is included in the encrypted blueprint backup.
- Kong: `/mcp` → controller `:8083`, 120 requests/minute per IP, 1 MB body.
- Protocol: stateless JSON over HTTP POST (`initialize`, `ping`, `tools/list`, `tools/call`). GET returns 405; no sessions or streaming. Every failed auth returns 401, including when the gateway is off.

## Upgrade note

The controller image and the platform chart must be upgraded together (new port `8083` and Kong route). `obelisk update` does both.
