#!/usr/bin/env bash
# Wrapper so Air can pass listen addresses from the environment.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/load-dotenv.sh"
load_dotenv "$ROOT"
export KUBECONFIG="${KUBECONFIG:-$ROOT/dev/kubeconfig}"
export OBELISK_RECONCILE="${OBELISK_RECONCILE:-cluster}"
export OBELISK_PUBLIC_HOST="${OBELISK_PUBLIC_HOST:-localhost}"
# Cluster dev (make dev) must reconcile Redpanda with a host-reachable Kafka advertise address.
if [ "$OBELISK_RECONCILE" = "cluster" ]; then
  export OBELISK_DEV="${OBELISK_DEV:-1}"
fi
exec "$ROOT/tmp/obelisk-controller" \
  -listen ":8080" \
  -gateway-listen "${GATEWAY_LISTEN:-:18081}" \
  -mail-gateway-listen "${MAIL_GATEWAY_LISTEN:-:18082}"