#!/usr/bin/env bash
# Exercise uninstall on a VM with an existing platform install.
set -euo pipefail

OBELISK_ROOT="${OBELISK_ROOT:-/opt/obelisk}"
NAMESPACE="${OBELISK_NAMESPACE:-obelisk}"
KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"

export KUBECONFIG

log() { printf '[obelisk-vm-uninstall] %s\n' "$*"; }
die() { log "FAIL: $*"; exit 1; }

[[ "$(id -u)" -eq 0 ]] || die "run with sudo"

CLI="${OBELISK_ROOT}/obelisk"
[[ -x "$CLI" ]] || CLI="/usr/local/bin/obelisk"
command -v obelisk >/dev/null || [[ -x "$CLI" ]] || die "obelisk CLI missing"

run_cli() {
  if [[ -x "$CLI" ]]; then
    "$CLI" "$@"
  else
    obelisk "$@"
  fi
}

log "dry-run"
run_cli uninstall --dry-run --root "$OBELISK_ROOT" --yes | grep -q 'inventory' || die "dry-run missing inventory"

if [[ -f "${OBELISK_ROOT}/install/resolve-admin-url.sh" ]]; then
  # shellcheck source=install/resolve-admin-url.sh
  source "${OBELISK_ROOT}/install/resolve-admin-url.sh"
  base="$(obelisk_resolve_kong_base)"
  base="${base%/}"
  if curl -fsS --max-time 5 "${base}/api/v1/health" 2>/dev/null | grep -q '"status":"ok"'; then
    log "Kong healthy before uninstall (expected)"
  fi
fi

log "full uninstall"
OBELISK_UNINSTALL_YES=true run_cli uninstall --yes --root "$OBELISK_ROOT"

[[ ! -f /usr/local/bin/obelisk ]] || die "CLI still present"
[[ ! -d /etc/obelisk ]] || die "/etc/obelisk still present"

pods="$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l | tr -d ' ' || echo 0)"
[[ "${pods:-0}" -eq 0 ]] || die "pods remain in $NAMESPACE"

log "uninstall smoke ok"