# OpenRouter gateway

## Enable

1. Admin UI → **Exposure** → **OpenRouter gateway**.
2. **Generate gateway token**, add one or more OpenRouter API keys (labels). Each key is validated via OpenRouter `GET /api/v1/key`; **duplicate keys are rejected** (stored fingerprint + upstream label / key tail in `obelisk.yaml`).
3. Turn **Enable gateway** on.
4. Point clients at `{origin}/openrouter/api/v1` with the gateway bearer token.

Direct controller port (cluster): `http://<host>:8081/api/v1`.

Local `make dev-api` uses **:18081** by default (`OBELISK_GATEWAY_LISTEN` / `GATEWAY_LISTEN` in Makefile) so it does not clash with other services on host :8081.

## SDK example

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://<vps>/openrouter/api/v1",
    api_key="<gateway_token>",
)
```

## Operations

- **Rotate token:** generates a new gateway token; update all clients.
- **Test:** admin **Test GET /models** hits `127.0.0.1:8081` on the controller host.
- Credentials file: `openrouter.credentials.json` beside `obelisk.yaml` (mode 0600).

## Kong

Route ` /openrouter` → controller `:8081` with `strip_path: true` in `kong-dbless.yaml`.