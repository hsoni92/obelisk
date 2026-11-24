#!/usr/bin/env bash
# One command: controller (Air hot-reload) + Next.js admin UI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/load-dotenv.sh"
load_dotenv "$ROOT"

CLUSTER=1
OBSERVE_ONLY=0
for arg in "$@"; do
  case "$arg" in
    --cluster) CLUSTER=1 ;;
    --observe-only) CLUSTER=0; OBSERVE_ONLY=1 ;;
    -h|--help)
      cat <<'EOF'
Usage: scripts/dev.sh [--observe-only]

  (default)  k3d + cluster reconcile (real Helm). Requires docker, k3d, helm, kubectl.
  --observe-only  OBELISK_RECONCILE=off — API + UI without applying Helm (instances show pending).

Env: repo-root .env (optional), OBELISK_CONFIG, OBELISK_ROOT, KUBECONFIG, GATEWAY_LISTEN, MAIL_GATEWAY_LISTEN
EOF
      exit 0
      ;;
  esac
done

if [ "${OBELISK_DEV_CLUSTER:-}" = "1" ]; then
  CLUSTER=1
fi
if [ "${OBELISK_RECONCILE:-}" = "off" ]; then
  CLUSTER=0
  OBSERVE_ONLY=1
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

if [ "$OBSERVE_ONLY" = "1" ]; then
  export OBELISK_RECONCILE=off
  unset OBELISK_DEV
  unset OBELISK_PUBLIC_HOST
  echo "[dev] reconcile: off (instances stay pending; use default dev for k3d)"
else
  export OBELISK_RECONCILE=cluster
  export OBELISK_DEV=1
  export OBELISK_PUBLIC_HOST="${OBELISK_PUBLIC_HOST:-localhost}"
  need docker
  need k3d
  need helm
  need kubectl
  export KUBECONFIG="${KUBECONFIG:-$ROOT/dev/kubeconfig}"
  # kubeconfig can outlive a stopped/deleted cluster, so probe the API rather than the file.
  if [ ! -f "$KUBECONFIG" ] || ! kubectl get --raw /readyz >/dev/null 2>&1; then
    echo "[dev] cluster not reachable; bringing up k3d..."
    OBELISK_KUBECONFIG="$KUBECONFIG" make dev-k3d-up
  fi
  echo "[dev] reconcile: cluster (KUBECONFIG=$KUBECONFIG)"
fi

export GATEWAY_LISTEN="${GATEWAY_LISTEN:-:18081}"
export MAIL_GATEWAY_LISTEN="${MAIL_GATEWAY_LISTEN:-:18082}"

if [ ! -d "$ROOT/web/admin/node_modules" ]; then
  echo "[dev] installing web/admin dependencies..."
  (cd "$ROOT/web/admin" && npm install)
fi

is_ancestor() {
  local target="$1" cur="$$"
  while [ -n "$cur" ] && [ "$cur" -gt 1 ] 2>/dev/null; do
    [ "$cur" = "$target" ] && return 0
    cur="$(ps -o ppid= -p "$cur" 2>/dev/null | tr -d ' ')" || break
  done
  return 1
}

kill_pid_graceful() {
  local pid="$1" i
  kill -0 "$pid" 2>/dev/null || return 0
  kill -TERM "$pid" 2>/dev/null || true
  for i in $(seq 1 20); do
    kill -0 "$pid" 2>/dev/null || return 0
    sleep 0.1
  done
  kill -KILL "$pid" 2>/dev/null || true
}

kill_tree() {
  local pid="$1" child
  for child in $(pgrep -P "$pid" 2>/dev/null || true); do
    kill_tree "$child"
  done
  kill_pid_graceful "$pid"
}

find_stale_dev_pids() {
  local pid cwd
  {
    pgrep -f "^$ROOT/tmp/obelisk-controller" 2>/dev/null || true
    pgrep -f "^$ROOT/bin/obelisk-controller" 2>/dev/null || true
    pgrep -f "scripts/run-controller-dev\.sh" 2>/dev/null || true
    pgrep -f "air -c $ROOT/\.air\.toml" 2>/dev/null || true
    pgrep -f "$ROOT/web/admin/node_modules/\.bin/next" 2>/dev/null || true
    pgrep -f "scripts/dev-port-forwards\.sh" 2>/dev/null || true
    pgrep -f "kubectl port-forward -n ${OBELISK_NAMESPACE:-obelisk}" 2>/dev/null || true
    for pid in $(pgrep -f "next-server" 2>/dev/null || true); do
      cwd="$(readlink -f "/proc/$pid/cwd" 2>/dev/null || true)"
      [ "$cwd" = "$ROOT/web/admin" ] && echo "$pid"
    done
  } | sort -u
}

cleanup_stale_dev() {
  local pid stale=()
  for pid in $(find_stale_dev_pids); do
    is_ancestor "$pid" && continue
    stale+=("$pid")
  done
  if [ "${#stale[@]}" -gt 0 ]; then
    echo "[dev] cleaning up stale dev processes from previous run: ${stale[*]}"
    for pid in "${stale[@]}"; do
      kill_tree "$pid"
    done
  fi
}

port_listening() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltnH "sport = :$port" 2>/dev/null | grep -q .
  elif command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
  else
    return 1
  fi
}

check_ports_free() {
  local port busy=0
  # Data-plane ports (:9092, :6379, …) are bound by scripts/dev-port-forwards.sh after startup.
  local ports=("8080" "3080" "${GATEWAY_LISTEN##*:}" "${MAIL_GATEWAY_LISTEN##*:}")
  for port in "${ports[@]}"; do
    [ -z "$port" ] && continue
    if port_listening "$port"; then
      echo "[dev] error: port :$port is already in use:"
      if command -v ss >/dev/null 2>&1; then
        ss -ltnp "sport = :$port" 2>/dev/null || true
      elif command -v lsof >/dev/null 2>&1; then
        lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true
      fi
      busy=1
    fi
  done
  if [ "$busy" = "1" ]; then
    echo "[dev] stop the conflicting process or override GATEWAY_LISTEN / MAIL_GATEWAY_LISTEN and retry"
    exit 1
  fi
}

cleanup_stale_dev
check_ports_free

PIDS=()
CLEANED=0
cleanup() {
  [ "$CLEANED" = "1" ] && return
  CLEANED=1
  local pid
  for pid in "${PIDS[@]:-}"; do
    kill_tree "$pid"
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "[dev] API http://127.0.0.1:8080  (restarts on .go changes via air)"
echo "[dev] UI  http://localhost:3080  (Next.js hot reload; API is :8080)"

if [ "$OBSERVE_ONLY" != "1" ]; then
  echo "[dev] data services: kubectl port-forwards on :9092 (Kafka), :8081 (Schema Registry), …"
  "$ROOT/scripts/dev-port-forwards.sh" &
  PIDS+=($!)
fi

air -c "$ROOT/.air.toml" &
PIDS+=($!)

( cd "$ROOT/web/admin" && npm run dev ) &
PIDS+=($!)

wait -n
EXIT=$?
cleanup
exit "$EXIT"