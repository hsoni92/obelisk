# Integration harness (prod-like VM)

Exercises the same path as a fresh VPS: k3s on the host, container images built locally, `install/bootstrap.sh`, and Kong in front of the admin API/UI.

## Multipass (local)

Requires [Multipass](https://multipass.run/), Docker inside the VM (cloud-init), and enough RAM for image builds (~8G recommended).

```bash
# One-shot: launch VM → install → smoke → print URL
make test-vm

# Or step by step:
./tests/integration/multipass/harness.sh up
./tests/integration/multipass/harness.sh install   # 15–25 min first time
./tests/integration/multipass/harness.sh smoke
./tests/integration/multipass/harness.sh url       # open in browser on the LAN

# Optional: enable postgres after platform smoke
OBELISK_SMOKE_LEVEL=full ./tests/integration/multipass/harness.sh smoke

./tests/integration/multipass/harness.sh destroy
```

Inside the VM, install uses:

- `OBELISK_PROFILE=vps`
- `OBELISK_BUILD_IMAGES=true` (controller + admin UI → `k3s ctr images import`)
- `--set kong.service.type=LoadBalancer` (k3s servicelb exposes HTTP on the VM IP)

## Bare metal / cloud VM

**Rsync from checkout (default harness):**

```bash
export OBELISK_VM_SRC=/path/to/checkout
sudo ./tests/integration/vm/install-in-vm.sh
sudo ./tests/integration/vm/smoke.sh
```

**Archive path (closer to production curl install):**

```bash
export OBELISK_VM_SRC=/path/to/checkout
sudo ./tests/integration/vm/curl-install.sh
sudo ./tests/integration/vm/smoke.sh
```

`curl-install.sh` packs the checkout into a tarball and runs `install/install.sh` (same entry as `curl | sudo bash` on GitHub).

## Environment

| Variable | Default | Meaning |
|----------|---------|---------|
| `OBELISK_VM_NAME` | `obelisk-vm` | Multipass instance name |
| `OBELISK_BUILD_IMAGES` | (install sets `true`) | Build/import images during bootstrap |
| `OBELISK_HELM_EXTRA_ARGS` | LoadBalancer for harness | Extra `helm upgrade` flags |
| `OBELISK_REF` | `main` / `local-test` | Archive ref for `install.sh` |
| `OBELISK_ARCHIVE_URL` | (curl-install builds `file://` tarball) | Override archive download |
| `OBELISK_SMOKE_LEVEL` | `basic` | `full` also enables postgres |

Manual image build on any k3s host:

```bash
sudo OBELISK_BUILD_IMAGES=true ./install/bootstrap.sh
```