# Uninstall runbook (VPS)

Removes **local** Obelisk state only. Does **not** delete GitHub blueprint repos, backup buckets, or OAuth app registrations.

## When to use

| Situation | Command |
|-----------|---------|
| Reclaim VPS / full wipe | `sudo obelisk uninstall` (dry-run first) |
| Failed install, retry with same data | `sudo obelisk uninstall --tier platform --yes` |
| Remove k3s too | Add `--remove-k3s` (extra confirmation) |

## Recommended flow

```bash
sudo obelisk uninstall --dry-run
sudo obelisk uninstall              # type `uninstall` when prompted
# non-interactive:
sudo obelisk uninstall --yes
```

## Flags

| Flag | Effect |
|------|--------|
| `--dry-run` | Inventory only; exit 0, no changes |
| `--yes` | Skip type-to-confirm (k3s still needs extra confirm) |
| `--tier platform` | Helm + namespace; keeps PVCs and `/var/lib/obelisk` |
| `--keep-data` | Skip PVC delete and keep `/var/lib/obelisk` |
| `--remove-k3s` | Runs `k3s-uninstall.sh` after Obelisk cleanup |
| `--resume` | Continue after partial uninstall (reads `/var/lib/obelisk/.uninstall-meta.json`) |

Environment: `OBELISK_UNINSTALL_K3S_CONFIRM=1` skips the k3s type-to-confirm (automation only).

## What is removed (default full tier)

- All Helm releases in namespace `obelisk` (platform + data services)
- PVCs in that namespace
- Namespace `obelisk`, ClusterRole/Binding `obelisk-controller`
- `/etc/obelisk`, `/var/lib/obelisk`, `/opt/obelisk`, `/usr/local/bin/obelisk`
- Obelisk images in k3s containerd (and local Docker tags if present)

## What is not removed

- Remote blueprint repository and S3/R2 `data/`
- `docker.io`, Go snap, or Helm installed by curl installer
- k3s (unless `--remove-k3s`)

## After uninstall

- **Re-install (k3s kept):** `curl -fsSL …/install/install.sh | sudo bash`
- **Stuck namespace `Terminating`:** `kubectl get pvc -A`; delete finalizers per Kubernetes docs, then re-run `sudo obelisk uninstall --yes`

Log file: `/var/log/obelisk-uninstall.log`

Design: [obelisk_uninstaller.md](../designs/obelisk_uninstaller.md)