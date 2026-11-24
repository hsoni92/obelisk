#!/usr/bin/env bash
# Tear down everything `make dev` starts: controller/air, Next.js, port-forwards, and the k3d cluster.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLUSTER="${OBELISK_K3D_CLUSTER:-obelisk}"
KUBECONFIG_FILE="${OBELISK_KUBECONFIG:-$ROOT/dev/kubeconfig}"

log() { printf '[terminate] %s\n' "$*"; }

kill_tree() {
  local pid="$1" child i
  for child in $(pgrep -P "$pid" 2>/dev/null || true); do
    kill_tree "$child"
  done
  kill -0 "$pid" 2>/dev/null || return 0
  kill -TERM "$pid" 2>/dev/null || true
  for i in $(seq 1 20); do
    kill -0 "$pid" 2>/dev/null || return 0
    sleep 0.1
  done
  kill -KILL "$pid" 2>/dev/null || true
}

pids="$(
  {
    pgrep -f "^$ROOT/tmp/obelisk-controller" || true
    pgrep -f "^$ROOT/bin/obelisk-controller" || true
    pgrep -f "scripts/run-controller-dev\.sh" || true
    pgrep -f "scripts/dev\.sh" || true
    pgrep -f "air -c $ROOT/\.air\.toml" || true
    pgrep -f "$ROOT/web/admin/node_modules/\.bin/next" || true
    pgrep -f "scripts/dev-port-forwards\.sh" || true
    pgrep -f "kubectl port-forward -n ${OBELISK_NAMESPACE:-obelisk}" || true
    for pid in $(pgrep -f "next-server" || true); do
      [ "$(readlink -f "/proc/$pid/cwd" 2>/dev/null || true)" = "$ROOT/web/admin" ] && echo "$pid"
    done
  } 2>/dev/null | sort -u
)"
for pid in $pids; do
  [ "$pid" = "$$" ] && continue
  log "stopping pid $pid"
  kill_tree "$pid"
done

if command -v k3d >/dev/null 2>&1 && k3d cluster get "$CLUSTER" >/dev/null 2>&1; then
  log "deleting k3d cluster $CLUSTER"
  k3d cluster delete "$CLUSTER"
else
  log "no k3d cluster $CLUSTER"
fi
rm -f "$KUBECONFIG_FILE"
log "done (config and credentials under dev/ are kept)"
