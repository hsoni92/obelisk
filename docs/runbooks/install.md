# Install runbook

1. Clone or copy Obelisk to `/opt/obelisk`.
2. On **VPS**: `OBELISK_PROFILE=vps OBELISK_ROOT=/opt/obelisk sudo ./install/bootstrap.sh`
3. On **local** with optional Tailscale: `OBELISK_PROFILE=local OBELISK_TAILSCALE=true sudo ./install/bootstrap.sh`
4. Open admin UI via Kong on the host HTTP port. Sign in with GitHub and finish blueprint onboarding ([github-blueprint.md](./github-blueprint.md)).
5. Enable Postgres / Redis / RustFS / Qdrant / SearXNG from the Services tab when needed.

## UI dev on a laptop (no `/etc/obelisk`)

```bash
make dev              # simulate reconcile, Air + Next.js (one terminal)
make dev-cluster      # k3d + cluster reconcile + hot reload

# Or separate processes without API hot-reload:
make dev-api          # OBELISK_CONFIG=./dev/obelisk.yaml, API :8080
make dev-ui           # Next.js → proxies /api/v1 to :8080
```

Without `make dev-api`, the UI will error because nothing serves `/api/v1` and config may be missing.

### Local k3s without sudo

Docker + `./install/k3d-dev.sh` (or `make dev-k3d-up`) creates a **k3d** cluster (real k3s in containers). `make dev-api` uses `dev/kubeconfig` and **cluster** reconcile.

### Production VPS

`sudo ./install/bootstrap.sh` installs **k3s on the host** (get.k3s.io). Requires sudo — not run from the agent without your password.

### Simulate (fallback)

If helm/kubectl are missing and no `KUBECONFIG`, the reconciler simulates desired state only.