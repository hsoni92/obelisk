#!/usr/bin/env bash
# One command: controller (Air hot-reload) + Next.js admin UI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/load-dotenv.sh"
load_dotenv "$ROOT"

CLUSTER=0
for arg in "$@"; do
  case "$arg" in
    --cluster) CLUSTER=1 ;;
    -h|--help)
      cat <<'EOF'
Usage: scripts/dev.sh [--cluster]

  (default)  simulate reconcile — no k3d required; good for API + UI work.
  --cluster  OBELISK_RECONCILE=cluster; starts k3d if dev/kubeconfig is missing.

Env: repo-root .env (optional), OBELISK_CONFIG, OBELISK_ROOT, KUBECONFIG, GATEWAY_LISTEN, MAIL_GATEWAY_LISTEN
EOF
      exit 0
      ;;
  esac
done

if [ "${OBELISK_DEV_CLUSTER:-}" = "1" ]; then
  CLUSTER=1
fi

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[dev] missing $1 on PATH"
    exit 1
  }
}
need go
need npm

if ! command -v air >/dev/null 2>&1; then
  echo "[dev] air not found; install with: go install github.com/air-verse/air@latest"
  echo "[dev] or: make install-air"
  exit 1
fi

export OBELISK_ROOT="$ROOT"
export OBELISK_CONFIG="${OBELISK_CONFIG:-$ROOT/dev/obelisk.yaml}"
export OBELISK_CREDENTIALS_DIR="${OBELISK_CREDENTIALS_DIR:-$ROOT/dev/credentials}"
mkdir -p "$OBELISK_CREDENTIALS_DIR" "$ROOT/tmp"

if [ ! -f "$OBELISK_CONFIG" ]; then
  if [ -f "$ROOT/install/obelisk.yaml.example" ]; then
    cp "$ROOT/install/obelisk.yaml.example" "$OBELISK_CONFIG"
    echo "[dev] created $OBELISK_CONFIG from install/obelisk.yaml.example"
  else
    echo "[dev] missing config: $OBELISK_CONFIG"
    exit 1
  fi
fi

if [ "$CLUSTER" = "1" ]; then
  export OBELISK_RECONCILE=cluster
  unset OBELISK_DEV
  if [ ! -f "${KUBECONFIG:-$ROOT/dev/kubeconfig}" ]; then
    echo "[dev] cluster mode: bringing up k3d..."
    make dev-k3d-up
  fi
  export KUBECONFIG="${KUBECONFIG:-$ROOT/dev/kubeconfig}"
  echo "[dev] reconcile: cluster (KUBECONFIG=$KUBECONFIG)"
else
  export OBELISK_RECONCILE="${OBELISK_RECONCILE:-simulate}"
  export OBELISK_DEV=1
  echo "[dev] reconcile: simulate (use --cluster or make dev-cluster for k3d)"
fi

export GATEWAY_LISTEN="${GATEWAY_LISTEN:-:18081}"
export MAIL_GATEWAY_LISTEN="${MAIL_GATEWAY_LISTEN:-:18082}"

if [ ! -d "$ROOT/web/admin/node_modules" ]; then
  echo "[dev] installing web/admin dependencies..."
  (cd "$ROOT/web/admin" && npm install)
fi

PIDS=()
cleanup() {
  local pid
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "[dev] API http://127.0.0.1:8080  (restarts on .go changes via air)"
echo "[dev] UI  http://localhost:3080  (Next.js hot reload; API is :8080)"

air -c "$ROOT/.air.toml" &
PIDS+=($!)

( cd "$ROOT/web/admin" && npm run dev ) &
PIDS+=($!)

wait -n
EXIT=$?
cleanup
exit "$EXIT"