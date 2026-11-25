# Install runbook

## Production VPS (canonical)

On a clean **Ubuntu LTS** VPS with sudo and outbound HTTPS:

```bash
curl -fsSL https://github.com/hsoni92/obelisk/releases/latest/download/install.sh | sudo bash
```

The installer finds the latest release (`build-N`), downloads the `obelisk` CLI for this machine's architecture (amd64 or arm64) and checks its SHA-256, unpacks that release's source tree to `/opt/obelisk`, and runs `install/bootstrap.sh`. The platform images were built by CI and k3s pulls them from GHCR. **No Docker, Go, or image build runs on the machine.** See [release.md](./release.md) for how releases are made.

Pin a build:

```bash
curl -fsSL https://github.com/hsoni92/obelisk/releases/latest/download/install.sh | sudo OBELISK_REF=build-142 bash
```

Build from source on the machine instead (development; needs Docker and Go, takes 15–25 minutes):

```bash
curl -fsSL https://raw.githubusercontent.com/hsoni92/obelisk/main/install/install.sh | sudo OBELISK_REF=main bash
```

Safer than piping straight to shell: download, read, then run `sudo bash install.sh`.

After install, open the printed `http://<vps-ip>/` URL. **Platform install** ends here (Kong + admin UI, no data pods). Next: sign in with GitHub and blueprint onboarding ([github-blueprint.md](./github-blueprint.md)), then enable Postgres / Redis / RustFS / Qdrant / SearXNG from the Services tab when needed.

| Variable | Default | Meaning |
|----------|---------|---------|
| `OBELISK_REF` | latest release | `build-N` pins a release. Any other ref (branch, tag, commit) is a **dev** install that builds on the machine |
| `OBELISK_IMAGE_REGISTRY` | `ghcr.io/<repo owner>` | Registry holding `obelisk-controller` and `obelisk-admin-ui` for release installs |
| `OBELISK_ROOT` | `/opt/obelisk` | Install tree |
| `OBELISK_ARCHIVE_SHA256` | unset | Optional archive checksum verify |
| `OBELISK_INSTALL_UPGRADE` | `false` | Replace tree in place when `true` |
| `OBELISK_SKIP_GITHUB_WIZARD` | `false` | Skip interactive OAuth wizard after first install |

## Upgrade an existing VPS

```bash
sudo obelisk update --check         # exit 0 up to date, 10 update available, 11 cannot tell (no root needed)
sudo obelisk update                 # upgrade only if the latest build is higher than the installed one
sudo obelisk update --to build-140  # exact build; going lower also needs --force
sudo obelisk update --ref main      # developer: build from a git ref on this machine
```

`update` compares the installed build number (in `/opt/obelisk/.install-meta.json`, `obelisk version`) with the latest GitHub Release. If the latest is higher it downloads and verifies the new CLI, checks the images can be pulled, pulls them into k3s, replaces the source tree (preserving `/etc/obelisk` and `/var/lib/obelisk`), and runs `helm upgrade --atomic --wait`. If anything fails after the tree is replaced, the previous tree is restored and Helm rolls back. The recorded build changes only after success. It never builds images and never compiles Go. Install scripts take a lock, so two updates cannot run at once.

A machine installed before this change has no build number (counted as build 0). Its first `sudo obelisk update` still builds on the machine once, then later updates are download-only. Equivalent: `sudo OBELISK_INSTALL_MODE=upgrade bash /opt/obelisk/install/update.sh`.

After each install or upgrade, bootstrap runs **`install/cleanup.sh`** automatically: stale platform Docker/k3s images (keeping the current tag), Helm revision pruning, light dangling-layer prune, and install-log rotation.

### Disk cleanup (manual)

```bash
sudo obelisk cleanup              # stale platform images, Helm history, light Docker prune
sudo obelisk cleanup --dry-run    # preview
sudo obelisk cleanup --deep       # normal + Docker build cache (slower next image build)
```

Set `OBELISK_SKIP_POST_INSTALL_CLEANUP=true` on bootstrap to skip automatic cleanup.

### Platform start / stop / restart

By default these commands act **only on the `obelisk` namespace**. k3s keeps running
and every other namespace on the host (your own apps, ingress, data services) is
untouched.

```bash
sudo obelisk stop      # scale obelisk workloads to 0; other namespaces unaffected
sudo obelisk start     # restore the recorded replica counts; wait for health
sudo obelisk restart   # rolling restart of obelisk deployments only
sudo obelisk start --no-wait   # return without waiting for rollout / Kong health
```

Replica counts are recorded in the `obelisk.dev/stop-replicas` annotation at stop
time and restored on start, so a scaled-down service (e.g. one you paused in the UI)
does not come back unexpectedly. CronJobs in the namespace are suspended and resumed.

#### Whole-node control (dedicated appliances only)

On a box that runs **only** Obelisk, `--system` stops the k3s service itself —
this takes down **every namespace**:

```bash
sudo obelisk stop --system     # systemctl stop k3s — everything on the host goes down
sudo obelisk start --system    # start k3s; wait for platform rollouts + Kong health
sudo obelisk restart --system  # systemctl restart k3s
```

Never use `--system` on a shared VPS that also hosts other workloads.

Use **`sudo obelisk status`** after start or restart to verify URLs and component health.

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

### Host port policy (never 80/443)

An Obelisk appliance **never binds host ports 80 or 443**. Those are reserved for a
co-located web edge (an app's own Ingress, nginx, or Kong). Obelisk exposes:

| Purpose | Port |
|---------|------|
| Admin UI, `/api/v1`, HTTP data routes (S3/Qdrant/SearXNG) | **9080** |
| Native data services (Redis/Postgres/MySQL/Mongo, TLS + SNI) | **9443** |

Overrides: `OBELISK_HTTP_PORT` and `OBELISK_TLS_PORT` (or Helm
`kong.service.httpPort` / `kong.service.tlsPort`). Setting either to 80/443 is
**rejected** — the config validator errors and bootstrap fails fast. Set
`nativeEdge.enabled=false` to run admin-only with no native TLS edge (port 9443).

Sharing a host with another app: leave `kong.service.type=ClusterIP` and route to
Obelisk from that app's ingress, or use `LoadBalancer` on 9080/9443 only.

## UI dev on a laptop (no `/etc/obelisk`)

```bash
make dev              # k3d + cluster Helm reconcile, Air + Next.js, auto port-forwards (Kafka :9092, …)
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