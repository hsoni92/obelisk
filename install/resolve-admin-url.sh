#!/usr/bin/env bash
# Resolve Kong admin URL on a k3s VPS (sourced by install.sh).
set -euo pipefail

obelisk_resolve_kong_base() {
  local ns="${OBELISK_NAMESPACE:-obelisk}"
  local port="80"

  if [[ -n "${OBELISK_KONG_URL:-}" ]]; then
    printf '%s' "${OBELISK_KONG_URL%/}/"
    return
  fi

  if command -v kubectl >/dev/null && [[ -r /etc/rancher/k3s/k3s.yaml ]]; then
    export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
    port="$(kubectl get svc -n "$ns" obelisk-kong -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || echo 80)"
    local lb_ip=""
    lb_ip="$(kubectl get svc -n "$ns" obelisk-kong -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)"
    if [[ -n "$lb_ip" ]]; then
      printf 'http://%s:%s/' "$lb_ip" "$port"
      return
    fi
    local ext=""
    ext="$(kubectl get svc -n "$ns" obelisk-kong -o jsonpath='{.spec.externalIPs[0]}' 2>/dev/null || true)"
    if [[ -n "$ext" ]]; then
      printf 'http://%s:%s/' "$ext" "$port"
      return
    fi
  fi

  local ip
  for ip in $(hostname -I 2>/dev/null); do
    case "$ip" in
      10.42.*|10.43.*|127.*|169.254.*) continue ;;
      *) printf 'http://%s:%s/' "$ip" "$port"; return ;;
    esac
  done
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  printf 'http://%s:%s/' "${ip:-127.0.0.1}" "$port"
}

obelisk_wait_kong_health() {
  local base timeout="${OBELISK_INSTALL_WAIT_SEC:-600}"
  local end=$((SECONDS + timeout))
  base="$(obelisk_resolve_kong_base)"
  base="${base%/}"

  while (( SECONDS < end )); do
    if curl -fsS --max-time 5 "${base}/api/v1/health" 2>/dev/null | grep -q '"status":"ok"'; then
      return 0
    fi
    sleep 5
  done
  return 1
}