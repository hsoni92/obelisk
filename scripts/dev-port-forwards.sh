#!/usr/bin/env bash
# Background port-forwards for make dev (host clients use instance edge hostnames on 127.0.0.1).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export KUBECONFIG="${KUBECONFIG:-$ROOT/dev/kubeconfig}"
NS="${OBELISK_NAMESPACE:-obelisk}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "[dev-pf] missing $1" >&2; exit 1; }; }
need kubectl

log() { printf '[dev-pf] %s\n' "$*" >&2; }

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

declare -A PF_PID=()
declare -A PF_SVC=()

forward_alive() {
  local lport="$1"
  local pid="${PF_PID[$lport]:-}"
  [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

ensure_forward() {
  local svc="$1" lport="$2" rport="$3"
  if forward_alive "$lport"; then
    return 0
  fi
  if port_listening "$lport"; then
    log "port :$lport in use (not our forward for $svc); leaving as-is"
    return 0
  fi
  log "forward svc/$svc ${lport}:${rport}"
  kubectl port-forward -n "$NS" "svc/$svc" "${lport}:${rport}" >/dev/null 2>&1 &
  PF_PID[$lport]=$!
  PF_SVC[$lport]="$svc"
}

forward_release() {
  local svc="$1"
  case "$svc" in
    obelisk-unimeter-*)
      ensure_forward "$svc" 7001 7001
      ensure_forward "$svc" 9090 9090
      ;;
    obelisk-redpanda-*)
      # Host bootstrap is *:9092; broker external listener is pod :19092 (see redpanda chart devAdvertiseHost).
      ensure_forward "$svc" 9092 19092
      ensure_forward "$svc" 8081 8081
      ;;
    obelisk-redis-*)
      ensure_forward "$svc" 6379 6379
      ;;
    obelisk-postgres-*)
      ensure_forward "$svc" 5432 5432
      ;;
    obelisk-mysql-*)
      ensure_forward "$svc" 3306 3306
      ;;
    obelisk-mongodb-*)
      ensure_forward "$svc" 27017 27017
      ;;
    obelisk-qdrant-*)
      ensure_forward "$svc" 6333 6333
      ;;
    obelisk-rustfs-*)
      ensure_forward "$svc" 9000 9000
      ;;
    obelisk-searxng-*)
      ensure_forward "$svc" 18080 8080
      ;;
  esac
}

reconcile_forwards() {
  local svc
  while read -r svc; do
    [[ -z "$svc" ]] && continue
    forward_release "$svc"
  done < <(kubectl get svc -n "$NS" -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)
}

if ! kubectl cluster-info >/dev/null 2>&1; then
  log "cluster not reachable; exiting"
  exit 0
fi

log "watching $NS services (Kafka :9092, Schema Registry :8081, Redis :6379, …)"
while true; do
  reconcile_forwards || true
  sleep 10
done