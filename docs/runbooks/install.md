# Install runbook

## Production VPS (canonical)

On a clean **Ubuntu LTS** VPS with sudo and outbound HTTPS:

```bash
curl -fsSL https://raw.githubusercontent.com/hsoni92/obelisk/main/install/install.sh | sudo bash
```

The installer downloads a GitHub source archive (default branch `main`), unpacks to `/opt/obelisk`, installs Docker/Go if needed, builds images on the host, and runs `install/bootstrap.sh` with `OBELISK_PROFILE=vps`. First install often takes **15–25 minutes** on an 8 GB VM because of image builds.

Pin a tag or branch when you cut releases:

```bash
curl -fsSL https://raw.githubusercontent.com/hsoni92/obelisk/main/install/install.sh | sudo OBELISK_REF=v0.1.0 bash
```

Safer than piping straight to shell: download, read, then run `sudo bash install.sh`.

After install, open the printed `http://<vps-ip>/` URL. **Platform install** ends here (Kong + admin UI, no data pods). Next: sign in with GitHub and blueprint onboarding ([github-blueprint.md](./github-blueprint.md)), then enable Postgres / Redis / RustFS / Qdrant / SearXNG from the Services tab when needed.

| Variable | Default | Meaning |
|----------|---------|---------|
| `OBELISK_REF` | `main` | GitHub archive ref (branch, tag, or commit) |
| `OBELISK_ROOT` | `/opt/obelisk` | Install tree |
| `OBELISK_ARCHIVE_SHA256` | unset | Optional archive checksum verify |
| `OBELISK_INSTALL_UPGRADE` | `false` | Replace tree in place when `true` |
| `OBELISK_SKIP_GITHUB_WIZARD` | `false` | Skip interactive OAuth wizard after first install |

## Upgrade an existing VPS

```bash
sudo obelisk update
sudo obelisk update --ref v0.2.0    # pin a tag or branch
sudo obelisk update --skip-images   # helm/controller rollout only (no new UI image)
```

This fetches the GitHub archive, rsyncs into `/opt/obelisk` (preserving `/etc/obelisk` and `/var/lib/obelisk`), rebuilds platform images by default, and runs `install/bootstrap.sh`. Each upgrade tags images uniquely so Helm rolls the admin UI and controller pods (rebuilding the same `0.1.0` tag alone does not refresh running pods). Equivalent: `sudo OBELISK_INSTALL_MODE=upgrade bash /opt/obelisk/install/update.sh`.

After each install or upgrade, bootstrap runs **`install/cleanup.sh`** automatically: stale platform Docker/k3s images (keeping the current tag), Helm revision pruning, light dangling-layer prune, and install-log rotation.

### Disk cleanup (manual)

```bash
sudo obelisk cleanup              # stale platform images, Helm history, light Docker prune
sudo obelisk cleanup --dry-run    # preview
sudo obelisk cleanup --deep       # normal + Docker build cache (slower next image build)
```

Set `OBELISK_SKIP_POST_INSTALL_CLEANUP=true` on bootstrap to skip automatic cleanup.

## Uninstall / failed install

If bootstrap failed or you need to reclaim the VPS:

```bash
sudo obelisk uninstall --dry-run
sudo obelisk uninstall --yes
```

Shallow teardown (keep PVCs for debugging): `sudo obelisk uninstall --tier platform --yes`. See [uninstall.md](./uninstall.md).

## Contributor / fallback (git checkout)

1. Clone or copy Obelisk to `/opt/obelisk`.
2. On **VPS**: `OBELISK_PROFILE=vps OBELISK_ROOT=/opt/obelisk OBELISK_BUILD_IMAGES=true sudo -E ./install/bootstrap.sh`
3. On **local** with optional Tailscale: `OBELISK_PROFILE=local OBELISK_TAILSCALE=true sudo ./install/bootstrap.sh`
4. Open admin UI via Kong on **port 9080** (VPS: k3s servicelb maps host :9080 → Kong).

## UI dev on a laptop (no `/etc/obelisk`)

```bash
make dev              # k3d + cluster Helm reconcile, Air + Next.js (one terminal)
make dev-observe      # API + UI only (OBELISK_RECONCILE=off)

# Or separate processes without API hot-reload:
make dev-api          # OBELISK_CONFIG=./dev/obelisk.yaml, API :8080
make dev-ui           # Next.js → proxies /api/v1 to :8080
```

Without `make dev-api`, the UI will error because nothing serves `/api/v1` and config may be missing.

### Local k3s without sudo

Docker + `./install/k3d-dev.sh` (or `make dev-k3d-up`) creates a **k3d** cluster (real k3s in containers). `make dev-api` uses `dev/kubeconfig` and **cluster** reconcile.

### Production VPS (manual bootstrap)

Use the **curl one-liner** above when possible. Manual path: `sudo ./install/bootstrap.sh` installs **k3s on the host** (get.k3s.io) from an existing tree at `OBELISK_ROOT`.

**VM smoke test (Multipass):** [tests/integration/README.md](../../tests/integration/README.md) — `make test-vm` (prod-like curl install; public GitHub required).

### Reconcile off

Set `OBELISK_RECONCILE=off` (or `make dev-observe`) to run the API/UI without Helm. Enabled instances stay **pending** in the UI until you use cluster mode with helm, kubectl, and a kubeconfig.