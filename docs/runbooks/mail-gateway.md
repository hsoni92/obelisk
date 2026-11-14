# Mail gateway

1. Admin UI → **Exposure** → **Mail gateway**.
2. Enable gateway, **Generate gateway token**, add mailbox (Gmail/Outlook/Yahoo app password).
3. Call `POST {origin}/mail/api/v1/send` with `Authorization: Bearer <token>`.

## Example

```bash
curl -sS -X POST "${ORIGIN}/mail/api/v1/send" \
  -H "Authorization: Bearer ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"to":"you@example.com","subject":"hello","text":"from obelisk"}'
```

## Credentials

- Metadata: `obelisk.yaml` → `mail_gateway`
- Secrets: `mail.credentials.json` under `OBELISK_CREDENTIALS_DIR` (default beside config, or `/var/lib/obelisk/credentials` on cluster)

## Limits

- Kong: 30 requests/minute per IP, 1 MB JSON body on `/mail`
- v1.0: text/html only; six SMTP provider presets