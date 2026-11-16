#!/usr/bin/env bash
# Post-install smoke: platform health via Kong (prod-like k3s on host).
set -euo pipefail

NAMESPACE="${OBELISK_NAMESPACE:-obelisk}"
KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
OBELISK_ROOT="${OBELISK_ROOT:-/opt/obelisk}"
CONFIG_PATH="${OBELISK_CONFIG:-/etc/obelisk/obelisk.yaml}"
LEVEL="${OBELISK_SMOKE_LEVEL:-basic}"
TIMEOUT="${OBELISK_SMOKE_TIMEOUT:-600}"

export KUBECONFIG

log() { printf '[obelisk-smoke] %s\n' "$*"; }
die() { log "FAIL: $*"; exit 1; }

curl_kong() {
  local path="$1"
  curl -fsS --max-time 15 "$KONG_BASE$path"
}

wait_platform_pods() {
  log "waiting for platform pods (timeout ${TIMEOUT}s)"
  local end=$((SECONDS + TIMEOUT))
  while (( SECONDS < end )); do
    local not_ready
    not_ready="$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | grep -vE 'Running|Completed' | grep -v '^$' || true)"
    local total
    total="$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
    if [[ "$total" -ge 3 && -z "$not_ready" ]]; then
      kubectl get pods -n "$NAMESPACE"
      return 0
    fi
    sleep 5
  done
  kubectl get pods -n "$NAMESPACE" || true
  die "platform pods not ready within ${TIMEOUT}s"
}

resolve_kong_base() {
  local helper="${OBELISK_ROOT}/install/resolve-admin-url.sh"
  if [[ -f "$helper" ]]; then
    # shellcheck source=install/resolve-admin-url.sh
    source "$helper"
    KONG_BASE="$(obelisk_resolve_kong_base)"
    log "Kong base: $KONG_BASE"
    return
  fi
  if [[ -n "${OBELISK_KONG_URL:-}" ]]; then
    KONG_BASE="${OBELISK_KONG_URL%/}/"
    log "using OBELISK_KONG_URL=$KONG_BASE"
    return
  fi
  local ip
  ip="$(hostname -I | awk '{print $1}')"
  local lb_port
  lb_port="$(kubectl get svc -n "$NAMESPACE" obelisk-kong -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || echo 80)"
  KONG_BASE="http://${ip}:${lb_port}/"
  log "Kong via node IP (k3s servicelb): $KONG_BASE"
}

main() {
  command -v kubectl >/dev/null || die "kubectl missing (is k3s installed?)"
  wait_platform_pods
  resolve_kong_base

  log "GET /api/v1/health"
  curl_kong "/api/v1/health" | grep -q '"status":"ok"' || die "/api/v1/health unexpected body"

  log "GET /api/v1/status"
  curl_kong "/api/v1/status" | grep -q 'profile' || die "/api/v1/status unexpected body"

  log "GET / (admin UI)"
  curl_kong "/" | grep -qi html || die "admin UI root did not return HTML"

  if [[ -x "$OBELISK_ROOT/obelisk" ]]; then
    log "local CLI status"
    "$OBELISK_ROOT/obelisk" status --config "$CONFIG_PATH" >/dev/null
  fi

  if [[ "$LEVEL" == "full" ]]; then
    log "enabling postgres (full smoke)"
    "$OBELISK_ROOT/obelisk" service enable postgres --config "$CONFIG_PATH" --helm-root "$OBELISK_ROOT"
    local end=$((SECONDS + TIMEOUT))
    while (( SECONDS < end )); do
      if kubectl get statefulset -n "$NAMESPACE" obelisk-postgres -o jsonpath='{.status.readyReplicas}' 2>/dev/null | grep -q 1; then
        log "postgres ready"
        break
      fi
      sleep 10
    done
    kubectl get statefulset -n "$NAMESPACE" 2>/dev/null || true
  fi

  log "smoke passed (level=$LEVEL)"
}

main "$@"