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
  # Exists but may be stopped (docker restart, reboot): start is a no-op when running.
  echo "[obelisk] k3d cluster $CLUSTER exists; ensuring it is running"
  k3d cluster start "$CLUSTER" >/dev/null 2>&1 || true
else
  echo "[obelisk] creating k3d cluster $CLUSTER (k3s, traefik disabled)"
  k3d cluster create "$CLUSTER" \
    --k3s-arg "--disable=traefik@server:0" \
    --port "9080:9080@loadbalancer"
fi

mkdir -p "$(dirname "$KUBECONFIG_OUT")"
k3d kubeconfig get "$CLUSTER" >"$KUBECONFIG_OUT"
chmod 600 "$KUBECONFIG_OUT"
export KUBECONFIG="$KUBECONFIG_OUT"

echo "[obelisk] waiting for API server..."
for _ in $(seq 1 60); do
  kubectl get --raw /readyz >/dev/null 2>&1 && break
  sleep 2
done
kubectl get --raw /readyz >/dev/null 2>&1 || { echo "[obelisk] cluster API not reachable"; exit 1; }
# metrics-server is required for kubectl top (overview vitals sampler).
if ! kubectl get deployment -n kube-system metrics-server >/dev/null 2>&1; then
  echo "[obelisk] installing metrics-server (k3d dev)"
  kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
  # k3d/k3s often needs insecure TLS on the metrics-server API.
  kubectl patch deployment metrics-server -n kube-system --type=json \
    -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]' \
    2>/dev/null || true
fi

kubectl cluster-info
echo "[obelisk] KUBECONFIG=$KUBECONFIG_OUT"
echo "[obelisk] run: make dev-api   # cluster mode when helm+kubectl on PATH"