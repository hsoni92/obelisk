# Integration harness

## Prod-like manual test (Multipass)

`make test-vm` launches a **bare Ubuntu 24.04 VPS-shaped VM**: no repo mount, no Docker/Go preinstall. You SSH in and run the **same curl command** as production. The harness **fails** if `https://raw.githubusercontent.com/hsoni92/obelisk/main/install/install.sh` is not HTTP 200 (repo must be **public** with `install/install.sh` on `main`).

```bash
make test-vm          # launch VM + print SSH + install command
# on VM:
curl -fsSL https://raw.githubusercontent.com/hsoni92/obelisk/main/install/install.sh | sudo bash
# from host after install:
make vm-smoke
./tests/integration/multipass/harness.sh url
make vm-destroy
```

Specs match design floor: 4 CPU, 8G RAM, 40G disk (`OBELISK_VM_CPUS`, `OBELISK_VM_MEM`, `OBELISK_VM_DISK`).

SSH: `ubuntu@<ip>` / password `obelisk-vm` (Multipass manual cloud-init only).

## Dev automation (not prod-like)

`make vm-test-all` mounts this checkout, preinstalls Docker/Go, and runs `install-in-vm.sh` + smoke — useful for fast iteration, **not** the operator install path.

```bash
make vm-test-all
# or:
OBELISK_VM_MODE=full ./tests/integration/multipass/harness.sh all
```

Inside that path, install uses `OBELISK_PROFILE=vps`, `OBELISK_BUILD_IMAGES=true`, Kong LoadBalancer for k3s servicelb.

## Scripts on a real host (checkout present)

For machines where you already have the git tree (not the curl prod path):

```bash
export OBELISK_VM_SRC=/path/to/checkout
sudo ./tests/integration/vm/install-in-vm.sh
sudo ./tests/integration/vm/smoke.sh
```

`curl-install.sh` builds a local tarball — **dev harness only**, not used by `make test-vm`.

## Environment

| Variable | Default | Meaning |
|----------|---------|---------|
| `OBELISK_VM_NAME` | `obelisk-vm` | Multipass instance name |
| `OBELISK_VM_MODE` | `manual` via `test-vm` | `manual` = prod-like; `full` = dev mount |
| `OBELISK_VM_MOUNT` | `false` in manual | Never enable for prod-like test |
| `OBELISK_SMOKE_LEVEL` | `basic` | `full` also enables postgres |
| `OBELISK_BUILD_IMAGES` | `true` on install | Build/import images during bootstrap |
| `OBELISK_HELM_EXTRA_ARGS` | LoadBalancer (dev harness) | Extra `helm upgrade` flags |