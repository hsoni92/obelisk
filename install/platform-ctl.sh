#!/usr/bin/env bash
# Start / stop / restart the Obelisk platform inside its own namespace.
#
# Default behavior is NAMESPACE-SCOPED: only workloads labelled in
# $OBELISK_NAMESPACE are scaled or restarted. Other namespaces on the same k3s
# cluster (and the k3s service itself) are never touched.
#
# Use --system only on a dedicated appliance where stopping k3s is intended.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=install/obelisk-paths.sh
source "$SCRIPT_DIR/obelisk-paths.sh"

ACTION="${1:-}"
shift || true
SYSTEM=false
NO_WAIT="${OBELISK_PLATFORM_NO_WAIT:-false}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --system) SYSTEM=true ;;
    --no-wait) NO_WAIT=true ;;
    *) break ;;
  esac
  shift
done

log() { printf '[obelisk-platform] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

REPLICAS_ANNOTATION="obelisk.dev/stop-replicas"

usage() {
  cat <<EOF
Usage: platform-ctl.sh start|stop|restart [--system] [--no-wait]

Default: namespace-scoped ($OBELISK_NAMESPACE only). k3s keeps running and
other namespaces are unaffected.

  --system    Whole-node control: systemctl k3s stop|start|restart.
              Only use on a dedicated appliance — this stops every namespace.
  --no-wait   Skip rollout / Kong health waits after start|restart.

Environment:
  OBELISK_NAMESPACE          kubernetes namespace (default $OBELISK_NAMESPACE)
  OBELISK_PLATFORM_NO_WAIT   true to skip rollout / Kong health after start|restart
  OBELISK_PLATFORM_WAIT_SEC  rollout timeout per deployment (default 300)
EOF
}

export_kubeconfig() {
  export KUBECONFIG="${KUBECONFIG:-$OBELISK_KUBECONFIG_DEFAULT}"
}

require_kubectl() {
  export_kubeconfig
  command -v kubectl >/dev/null 2>&1 || die "kubectl is required for namespace-scoped control"
  kubectl get namespace "$OBELISK_NAMESPACE" >/dev/null 2>&1 ||
    die "namespace $OBELISK_NAMESPACE not found — is Obelisk installed?"
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
  export_kubeconfig
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
    log "skip platform rollout wait (--no-wait)"
    return 0
  }
  require_kubectl
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

# --- namespace-scoped primitives -------------------------------------------------

# scale_namespace_down stops Obelisk pods only, remembering replica counts.
scale_namespace_down() {
  require_kubectl
  local ns="$OBELISK_NAMESPACE"
  local kind name replicas found=0

  for kind in deployment statefulset; do
    while read -r name replicas; do
      [[ -z "$name" ]] && continue
      found=1
      kubectl annotate "$kind" "$name" -n "$ns" \
        "${REPLICAS_ANNOTATION}=${replicas}" --overwrite >/dev/null
      log "scaling $kind/$name  ${replicas} -> 0"
      kubectl scale "$kind" "$name" -n "$ns" --replicas=0 >/dev/null
    done < <(kubectl get "$kind" -n "$ns" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.spec.replicas}{"\n"}{end}' 2>/dev/null)
  done

  # Suspend cron jobs so nothing wakes the namespace back up.
  while read -r name suspend; do
    [[ -z "$name" ]] && continue
    [[ "$suspend" == "true" ]] && continue
    found=1
    log "suspending cronjob/$name"
    kubectl patch cronjob "$name" -n "$ns" --type merge \
      -p '{"spec":{"suspend":true}}' >/dev/null
  done < <(kubectl get cronjob -n "$ns" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.spec.suspend}{"\n"}{end}' 2>/dev/null)

  if [[ "$found" -eq 0 ]]; then
    log "no workloads found in namespace $ns"
    return 0
  fi

  if [[ "$NO_WAIT" != "true" ]]; then
    log "waiting for Obelisk pods to terminate (namespace $ns)"
    local deadline=$((SECONDS + ${OBELISK_PLATFORM_WAIT_SEC:-300}))
    while (( SECONDS < deadline )); do
      local running
      running="$(kubectl get pods -n "$ns" --no-headers 2>/dev/null |
        grep -cE 'Running|Pending' || true)"
      [[ "$running" -eq 0 ]] && break
      sleep 3
    done
  fi
  log "obelisk namespace $ns scaled down (k3s and other namespaces untouched)"
}

# scale_namespace_up restores replica counts recorded at stop time.
scale_namespace_up() {
  require_kubectl
  local ns="$OBELISK_NAMESPACE"
  local kind name desired original found=0

  for kind in deployment statefulset; do
    while read -r name desired; do
      [[ -z "$name" ]] && continue
      original="$(kubectl get "$kind" "$name" -n "$ns" \
        -o jsonpath="{.metadata.annotations.obelisk\\.dev/stop-replicas}" 2>/dev/null || true)"
      # Only restore workloads WE scaled down. A service the operator disabled
      # (replicas 0, no annotation) must stay disabled.
      if [[ -z "$original" ]]; then
        continue
      fi
      if [[ "$desired" == "$original" ]]; then
        kubectl annotate "$kind" "$name" -n "$ns" "${REPLICAS_ANNOTATION}-" >/dev/null 2>&1 || true
        continue
      fi
      found=1
      log "scaling $kind/$name  ${desired} -> ${original}"
      kubectl scale "$kind" "$name" -n "$ns" --replicas="$original" >/dev/null
      kubectl annotate "$kind" "$name" -n "$ns" "${REPLICAS_ANNOTATION}-" >/dev/null 2>&1 || true
    done < <(kubectl get "$kind" -n "$ns" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.spec.replicas}{"\n"}{end}' 2>/dev/null)
  done

  while read -r name suspend; do
    [[ -z "$name" ]] && continue
    [[ "$suspend" != "true" ]] && continue
    found=1
    log "resuming cronjob/$name"
    kubectl patch cronjob "$name" -n "$ns" --type merge \
      -p '{"spec":{"suspend":false}}' >/dev/null
  done < <(kubectl get cronjob -n "$ns" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.spec.suspend}{"\n"}{end}' 2>/dev/null)

  if [[ "$found" -eq 0 ]]; then
    log "no workloads needed scaling in namespace $ns"
  fi
  wait_platform_rollout
}

# --- actions ---------------------------------------------------------------------

do_start() {
  if [[ "$SYSTEM" == "true" ]]; then
    local unit
    unit="$(require_k3s)"
    if systemctl is-active --quiet "$unit"; then
      log "$unit already active"
    else
      log "starting $unit (ALL namespaces on this host)"
      systemctl start "$unit"
    fi
    wait_k3s_api
    wait_platform_rollout
    log "platform started — run: sudo obelisk status"
    return 0
  fi
  log "starting Obelisk workloads in namespace $OBELISK_NAMESPACE"
  scale_namespace_up
  log "platform started — run: sudo obelisk status"
}

do_stop() {
  if [[ "$SYSTEM" == "true" ]]; then
    local unit
    unit="$(require_k3s)"
    if ! systemctl is-active --quiet "$unit"; then
      log "$unit already stopped"
      return 0
    fi
    log "stopping $unit — ALL namespaces on this host will go down"
    systemctl stop "$unit"
    log "platform stopped (whole node)"
    return 0
  fi
  log "stopping Obelisk workloads in namespace $OBELISK_NAMESPACE (other namespaces untouched)"
  scale_namespace_down
}

do_restart() {
  if [[ "$SYSTEM" == "true" ]]; then
    local unit
    unit="$(require_k3s)"
    log "restarting $unit (ALL namespaces on this host)"
    systemctl restart "$unit"
    wait_k3s_api
    wait_platform_rollout
    log "platform restarted — run: sudo obelisk status"
    return 0
  fi
  require_kubectl
  local ns="$OBELISK_NAMESPACE"
  log "rolling restart of Obelisk workloads in namespace $ns (other namespaces untouched)"
  kubectl rollout restart deployment -n "$ns" >/dev/null 2>&1 || true
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
