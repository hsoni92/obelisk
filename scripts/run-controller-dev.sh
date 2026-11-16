#!/usr/bin/env bash
# Wrapper so Air can pass listen addresses from the environment.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/load-dotenv.sh"
load_dotenv "$ROOT"
export KUBECONFIG="${KUBECONFIG:-$ROOT/dev/kubeconfig}"
export OBELISK_RECONCILE="${OBELISK_RECONCILE:-cluster}"
exec "$ROOT/tmp/obelisk-controller" \
  -listen ":8080" \
  -gateway-listen "${GATEWAY_LISTEN:-:18081}" \
  -mail-gateway-listen "${MAIL_GATEWAY_LISTEN:-:18082}"