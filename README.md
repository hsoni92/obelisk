# obelisk

Single-node middleware appliance: lazy Postgres, Redis, and S3-compatible storage with Kong, backups, and an admin UI.

Design: [docs/designs/obelisk.md](docs/designs/obelisk.md)

## Development

The admin UI talks to `obelisk-controller` on port **8080**. Without `/etc/obelisk/obelisk.yaml` on your laptop, point config at a repo-local file:

```bash
make build-controller

# One-time: local k3s via Docker (no sudo)
make dev-k3d-up

# Terminal 1 — API (cluster reconcile via dev/kubeconfig)
make dev-api

# Terminal 2 — UI (proxies /api/v1 → :8080)
make dev-ui
```

Override paths with `OBELISK_CONFIG` and `OBELISK_ROOT`. If `/etc/obelisk/obelisk.yaml` exists (after bootstrap), the controller uses it automatically unless `OBELISK_CONFIG` is set.

```bash
# CLI + tests (requires Go 1.22+)
make build-cli test-go

# Helm charts
make helm-lint
```

## Bootstrap on a host

```bash
export OBELISK_ROOT=/opt/obelisk   # checkout path on the server
export OBELISK_PROFILE=vps        # or local; Tailscale only for local + opt-in
sudo ./install/bootstrap.sh
```

## Layout

| Path | Purpose |
|------|---------|
| `cmd/obelisk` | Operator CLI |
| `cmd/obelisk-controller` | Admin API + reconciler |
| `deploy/helm/` | Platform + data service charts |
| `web/admin` | Next.js admin UI |
| `schemas/obelisk.yaml.schema.json` | Desired-state schema |
| `install/bootstrap.sh` | k3s platform installer |

**Profiles:** VPS does not install Tailscale. Local installs may opt into Tailscale when internet reachability is needed.