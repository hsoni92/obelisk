#!/usr/bin/env bash
# Local k3s via k3d (no sudo). Real Helm reconcile for dev; not for production VPS.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLUSTER="${OBELISK_K3D_CLUSTER:-obelisk}"
KUBECONFIG_OUT="${OBELISK_KUBECONFIG:-$ROOT/dev/kubeconfig}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing $1 on PATH"; exit 1; }; }
need docker
need k3d
need kubectl

if k3d cluster get "$CLUSTER" >/dev/null 2>&1; then
  echo "[obelisk] k3d cluster $CLUSTER already exists"
else
  echo "[obelisk] creating k3d cluster $CLUSTER (k3s, traefik disabled)"
  k3d cluster create "$CLUSTER" \
    --k3s-arg "--disable=traefik@server:0" \
    --port "9080:80@loadbalancer"
fi

mkdir -p "$(dirname "$KUBECONFIG_OUT")"
k3d kubeconfig get "$CLUSTER" >"$KUBECONFIG_OUT"
chmod 600 "$KUBECONFIG_OUT"
export KUBECONFIG="$KUBECONFIG_OUT"
kubectl cluster-info
echo "[obelisk] KUBECONFIG=$KUBECONFIG_OUT"
echo "[obelisk] run: make dev-api   # cluster mode when helm+kubectl on PATH"