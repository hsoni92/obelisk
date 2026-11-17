#!/usr/bin/env bash
# Start / stop / restart the VPS Obelisk platform (k3s + in-cluster workloads).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=install/obelisk-paths.sh
source "$SCRIPT_DIR/obelisk-paths.sh"

ACTION="${1:-}"
NO_WAIT="${OBELISK_PLATFORM_NO_WAIT:-false}"

log() { printf '[obelisk-platform] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<EOF
Usage: platform-ctl.sh start|stop|restart

Environment:
  OBELISK_NAMESPACE          kubernetes namespace (default $OBELISK_NAMESPACE)
  OBELISK_PLATFORM_NO_WAIT   true to skip rollout / Kong health after start|restart
  OBELISK_PLATFORM_WAIT_SEC  rollout timeout per deployment (default 300)
EOF
}

k3s_unit() {
  if systemctl cat k3s.service >/dev/null 2>&1; then
    echo k3s
    return 0
  fi
  if systemctl cat k3s-agent.service >/dev/null 2>&1; then
    echo k3s-agent
    return 0
  fi
  return 1
}

require_k3s() {
  command -v k3s >/dev/null 2>&1 || die "k3s is not installed — use on a VPS appliance (install runbook)"
  local unit
  unit="$(k3s_unit)" || die "no k3s systemd unit (k3s.service) — is k3s installed?"
  printf '%s' "$unit"
}

wait_k3s_api() {
  export KUBECONFIG="${KUBECONFIG:-$OBELISK_KUBECONFIG_DEFAULT}"
  local sec="${OBELISK_K3S_API_WAIT_SEC:-120}"
  local i=0
  while (( i < sec )); do
    if kubectl cluster-info >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
    ((i += 2)) || true
  done
  die "kubernetes API did not become ready within ${sec}s"
}

wait_platform_rollout() {
  [[ "$NO_WAIT" == "true" ]] && {
    log "skip platform rollout wait (OBELISK_PLATFORM_NO_WAIT)"
    return 0
  }
  export KUBECONFIG="${KUBECONFIG:-$OBELISK_KUBECONFIG_DEFAULT}"
  command -v kubectl >/dev/null 2>&1 || return 0
  kubectl cluster-info >/dev/null 2>&1 || return 0
  local timeout="${OBELISK_PLATFORM_WAIT_SEC:-300}"
  for dep in obelisk-kong obelisk-controller obelisk-admin-ui; do
    if ! kubectl get deployment -n "$OBELISK_NAMESPACE" "$dep" >/dev/null 2>&1; then
      continue
    fi
    log "waiting for $dep rollout (up to ${timeout}s)"
    if ! kubectl rollout status "deployment/$dep" -n "$OBELISK_NAMESPACE" --timeout="${timeout}s"; then
      die "deployment $dep did not become ready — run: sudo obelisk status"
    fi
  done
  if [[ -f "$SCRIPT_DIR/resolve-admin-url.sh" ]]; then
    # shellcheck source=install/resolve-admin-url.sh
    source "$SCRIPT_DIR/resolve-admin-url.sh"
    if ! obelisk_wait_kong_health; then
      die "Kong did not serve /api/v1/health — check obelisk-kong + obelisk-controller pods"
    fi
  fi
}

do_start() {
  local unit
  unit="$(require_k3s)"
  if systemctl is-active --quiet "$unit"; then
    log "$unit already active"
  else
    log "starting $unit"
    systemctl start "$unit"
  fi
  wait_k3s_api
  wait_platform_rollout
  log "platform started — run: sudo obelisk status"
}

do_stop() {
  local unit
  unit="$(require_k3s)"
  if ! systemctl is-active --quiet "$unit"; then
    log "$unit already stopped"
    return 0
  fi
  log "stopping $unit (admin UI, API, and data services on this host will go down)"
  systemctl stop "$unit"
  log "platform stopped"
}

do_restart() {
  local unit
  unit="$(require_k3s)"
  log "restarting $unit"
  systemctl restart "$unit"
  wait_k3s_api
  wait_platform_rollout
  log "platform restarted — run: sudo obelisk status"
}

main() {
  case "$ACTION" in
    start) do_start ;;
    stop) do_stop ;;
    restart) do_restart ;;
    -h | --help | help | "") usage; [[ -n "$ACTION" ]] || exit 1 ;;
    *) die "unknown action $ACTION (use start|stop|restart)" ;;
  esac
}

main "$@"