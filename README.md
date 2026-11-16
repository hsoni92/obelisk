# obelisk

Single-node middleware appliance: lazy Postgres, Redis, and S3-compatible storage with Kong, backups, and an admin UI.

Design: [docs/designs/obelisk.md](docs/designs/obelisk.md)

Contributing (CI, optional pre-commit hooks): [CONTRIBUTING.md](CONTRIBUTING.md)

## Development

**Prerequisites:** Go **1.22+** on `PATH`, Node.js/npm for `web/admin`. If `make` says `go: No such file or directory`, install Go first (`make check-go` prints distro hints).

The admin UI talks to `obelisk-controller` on port **8080**. Without `/etc/obelisk/obelisk.yaml` on your laptop, point config at a repo-local file:

```bash
go version       # must be 1.22+

# One command: k3d + real Helm reconcile + API (:8080) + UI (:3080)
make install-air
make dev

# UI/API only without Helm (instances show pending)
make dev-observe
```

Split terminals (no hot-reload on the API): `make dev-api` and `make dev-ui`. Cluster-only bootstrap: `make dev-k3d-up`.

Override paths with `OBELISK_CONFIG` and `OBELISK_ROOT`. If `/etc/obelisk/obelisk.yaml` exists (after bootstrap), the controller uses it automatically unless `OBELISK_CONFIG` is set.

```bash
# CLI + tests (requires Go 1.22+)
make build-cli test-go

# Helm charts
make helm-lint
```

## Bootstrap on a host

**VPS (production):**

```bash
curl -fsSL https://raw.githubusercontent.com/hsoni92/obelisk/main/install/install.sh | sudo bash
```

**From a git checkout** (local or contributor):

```bash
export OBELISK_ROOT=/opt/obelisk   # checkout path on the server
export OBELISK_PROFILE=vps        # or local; Tailscale only for local + opt-in
export OBELISK_BUILD_IMAGES=true  # build/import platform images before helm
sudo -E ./install/bootstrap.sh
```

**VM test (Multipass):** `make test-vm` — prod-like VM, curl install from public GitHub (see [tests/integration/README.md](tests/integration/README.md)).

After install: `sudo obelisk configure github` (OAuth wizard). Upgrade: `sudo obelisk update [--ref REF]`. Remove: `sudo obelisk uninstall --dry-run` then `sudo obelisk uninstall` ([uninstall runbook](docs/runbooks/uninstall.md)).

## Layout

| Path | Purpose |
|------|---------|
| `cmd/obelisk` | Operator CLI |
| `cmd/obelisk-controller` | Admin API + reconciler |
| `deploy/helm/` | Platform + data service charts |
| `web/admin` | Next.js admin UI |
| `schemas/obelisk.yaml.schema.json` | Desired-state schema |
| `install/install.sh` | VPS curl entry (archive → bootstrap) |
| `install/bootstrap.sh` | k3s platform installer |
| `install/update.sh` | In-place VPS upgrade (used by `obelisk update`) |

**Profiles:** VPS does not install Tailscale. Local installs may opt into Tailscale when internet reachability is needed.