#!/usr/bin/env bash
# Obelisk platform bootstrap — idempotent where possible.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/obelisk-paths.sh" ]]; then
  # shellcheck source=install/obelisk-paths.sh
  source "$SCRIPT_DIR/obelisk-paths.sh"
fi
if [[ -f "$SCRIPT_DIR/image-tag.sh" ]]; then
  # shellcheck source=install/image-tag.sh
  source "$SCRIPT_DIR/image-tag.sh"
fi
OBELISK_ROOT="${OBELISK_ROOT:-/opt/obelisk}"
CONFIG_PATH="${OBELISK_CONFIG:-/etc/obelisk/obelisk.yaml}"
NAMESPACE="${OBELISK_NAMESPACE:-obelisk}"
PROFILE="${OBELISK_PROFILE:-vps}"
ENABLE_TAILSCALE="${OBELISK_TAILSCALE:-false}"

log() { printf '[obelisk] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

require_root() {
  [[ "$(id -u)" -eq 0 ]] || die "run as root"
}

preflight() {
  require_root
  command -v curl >/dev/null || die "curl required"
  if [[ "$PROFILE" != "vps" && "$PROFILE" != "local" ]]; then
    die "OBELISK_PROFILE must be vps or local"
  fi
  if [[ "$PROFILE" == "vps" && "$ENABLE_TAILSCALE" == "true" ]]; then
    die "Tailscale is not used on VPS profile"
  fi
  preflight_host_edge
  log "preflight ok profile=$PROFILE tailscale=$ENABLE_TAILSCALE"
}

# preflight_host_edge warns when something already owns the host's web edge.
# Obelisk never binds 80/443 itself; this only surfaces a co-located server so the
# operator knows the appliance is sharing the host rather than replacing it.
preflight_host_edge() {
  command -v ss >/dev/null 2>&1 || return 0
  local port label
  for port in 80 443; do
    if ss -tlnH "sport = :$port" 2>/dev/null | grep -q .; then
      case "$port" in
        80) label=HTTP ;;
        443) label=HTTPS ;;
      esac
      log "note: host :$port ($label) is already in use — Obelisk will not bind 80/443 (uses 9080/9443)"
    fi
  done
}

install_k3s() {
  if command -v k3s >/dev/null; then
    log "k3s already installed"
    return
  fi
  # shellcheck source=/dev/null
  curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable=traefik" sh -
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
  log "k3s installed"
}

ensure_image_tag() {
  if [[ -n "${OBELISK_IMAGE_TAG:-}" ]]; then
    return
  fi
  if declare -F obelisk_resolve_image_tag >/dev/null; then
    export OBELISK_IMAGE_TAG="$(obelisk_resolve_image_tag)"
  else
    export OBELISK_IMAGE_TAG="0.1.0-$(date -u +%Y%m%dT%H%M%SZ)"
  fi
  log "platform image tag: $OBELISK_IMAGE_TAG"
}

build_images() {
  if [[ "${OBELISK_BUILD_IMAGES:-false}" != "true" ]]; then
    log "skipping image build (set OBELISK_BUILD_IMAGES=true to build and import)"
    return
  fi
  if ! docker info >/dev/null 2>&1; then
    die "OBELISK_BUILD_IMAGES=true but Docker is not running; install docker.io or set OBELISK_BUILD_IMAGES=false"
  fi
  if [[ ! -x "$OBELISK_ROOT/install/build-images.sh" ]]; then
    die "OBELISK_BUILD_IMAGES set but $OBELISK_ROOT/install/build-images.sh missing or not executable"
  fi
  ensure_image_tag
  OBELISK_IMAGE_TAG="$OBELISK_IMAGE_TAG" "$OBELISK_ROOT/install/build-images.sh"
}

maybe_tailscale() {
  if [[ "$PROFILE" != "local" || "$ENABLE_TAILSCALE" != "true" ]]; then
    log "skipping tailscale (profile=$PROFILE)"
    return
  fi
  if command -v tailscale >/dev/null; then
    log "tailscale present — ensure 'tailscale up' completed"
  else
    log "install tailscale from https://tailscale.com/download then re-run"
  fi
}

ensure_lib_paths() {
  mkdir -p /var/lib/obelisk/credentials
  chmod 700 /var/lib/obelisk /var/lib/obelisk/credentials 2>/dev/null || true
}

write_config() {
  ensure_lib_paths
  mkdir -p "$(dirname "$CONFIG_PATH")"
  if [[ -f "$CONFIG_PATH" ]]; then
    log "config exists: $CONFIG_PATH"
    return
  fi
  if [[ -x "$OBELISK_ROOT/obelisk" ]]; then
  "$OBELISK_ROOT/obelisk" init-config --profile "$PROFILE" --path "$CONFIG_PATH" \
    $([[ "$ENABLE_TAILSCALE" == "true" ]] && echo --tailscale)
  elif [[ -f "$OBELISK_ROOT/install/obelisk.yaml.example" ]]; then
    sed "s/profile: vps/profile: $PROFILE/" "$OBELISK_ROOT/install/obelisk.yaml.example" >"$CONFIG_PATH"
  else
    die "no obelisk CLI or example config under $OBELISK_ROOT"
  fi
  log "wrote $CONFIG_PATH"
}

deploy_platform() {
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
  command -v helm >/dev/null || {
    curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
  }
  ensure_image_tag
  local helm_extra="${OBELISK_HELM_EXTRA_ARGS:-}"
  if [[ "$PROFILE" == "vps" && -z "$helm_extra" ]]; then
    helm_extra='--set kong.service.type=LoadBalancer'
    log "VPS default: kong LoadBalancer (k3s servicelb) on 9080 + 9443 (never 80/443)"
  fi
  # Obelisk must never claim the host's HTTP/HTTPS edge (80/443): a co-located app
  # (e.g. an existing k3s Ingress/Kong) already owns those. Non-invasive by default.
  local edge_tls_port="${OBELISK_TLS_PORT:-9443}"
  local edge_http_port="${OBELISK_HTTP_PORT:-9080}"
  case "$edge_tls_port" in
    80|443) die "OBELISK_TLS_PORT=$edge_tls_port collides with the host web edge; use 9443" ;;
  esac
  case "$edge_http_port" in
    80|443) die "OBELISK_HTTP_PORT=$edge_http_port collides with the host web edge; use 9080" ;;
  esac
  local public_host="${OBELISK_PUBLIC_HOST:-}"
  if [[ -z "$public_host" && "$PROFILE" == "vps" && -f "$SCRIPT_DIR/resolve-admin-url.sh" ]]; then
    # shellcheck source=install/resolve-admin-url.sh
    source "$SCRIPT_DIR/resolve-admin-url.sh"
    local kong_base
    kong_base="$(obelisk_resolve_kong_base)"
    public_host="${kong_base#*://}"
    public_host="${public_host%%/*}"
    public_host="${public_host%%:*}"
    if [[ -n "$public_host" ]]; then
      log "VPS public host for connection hints: $public_host"
    fi
  fi
  # shellcheck disable=SC2086
  local helm_history_max="${OBELISK_HELM_HISTORY_MAX:-10}"
  helm upgrade --install obelisk-platform "$OBELISK_ROOT/deploy/helm/platform" \
    --namespace "$NAMESPACE" --create-namespace --wait --timeout 15m \
    --history-max "$helm_history_max" \
    --set obelisk.configHostPath="$CONFIG_PATH" \
    --set obelisk.helmRoot="$OBELISK_ROOT" \
    --set "obelisk.imageController=obelisk-controller:${OBELISK_IMAGE_TAG}" \
    --set "obelisk.imageAdminUI=obelisk-admin-ui:${OBELISK_IMAGE_TAG}" \
    $([[ -n "$public_host" ]] && echo --set "obelisk.publicHost=$public_host") \
    --set "kong.service.httpPort=$edge_http_port" \
    --set "kong.service.tlsPort=$edge_tls_port" \
    ${helm_extra}
  log "platform helm release applied (images ${OBELISK_IMAGE_TAG})"
}

wait_platform_rollout() {
  export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
  command -v kubectl >/dev/null 2>&1 || return 0
  kubectl cluster-info >/dev/null 2>&1 || return 0
  local timeout="${OBELISK_PLATFORM_WAIT_SEC:-300}"
  for dep in obelisk-kong obelisk-controller obelisk-admin-ui; do
    if ! kubectl get deployment -n "$NAMESPACE" "$dep" >/dev/null 2>&1; then
      continue
    fi
    log "waiting for $dep rollout (up to ${timeout}s)"
    if ! kubectl rollout status "deployment/$dep" -n "$NAMESPACE" --timeout="${timeout}s"; then
      die "deployment $dep did not become ready"
    fi
  done
  if [[ -f "$SCRIPT_DIR/resolve-admin-url.sh" ]]; then
    # shellcheck source=install/resolve-admin-url.sh
    source "$SCRIPT_DIR/resolve-admin-url.sh"
    if ! obelisk_wait_kong_health; then
      die "Kong did not serve /api/v1/health — use port 9080 (not :3000) and check obelisk-kong + obelisk-controller pods"
    fi
  fi
}

install_cli() {
  if [[ -f "$OBELISK_ROOT/obelisk" ]]; then
    install -m 0755 "$OBELISK_ROOT/obelisk" /usr/local/bin/obelisk
    log "installed /usr/local/bin/obelisk"
  fi
  local envdrop="/etc/profile.d/obelisk.sh"
  if [[ -w "$(dirname "$envdrop")" ]]; then
    cat >"$envdrop" <<'EOF'
# Obelisk CLI: live config is under /var/lib (not /etc seed on k3s).
export OBELISK_CONFIG="${OBELISK_CONFIG:-/var/lib/obelisk/obelisk.yaml}"
EOF
    chmod 644 "$envdrop"
    log "wrote $envdrop (OBELISK_CONFIG)"
  fi
}

post_install_cleanup() {
  local script="$OBELISK_ROOT/install/cleanup.sh"
  if [[ ! -f "$script" ]]; then
    return 0
  fi
  chmod +x "$script" 2>/dev/null || true
  export OBELISK_IMAGE_TAG="${OBELISK_IMAGE_TAG:-}"
  OBELISK_NAMESPACE="$NAMESPACE" OBELISK_ROOT="$OBELISK_ROOT" bash "$script"
}

main() {
  preflight
  install_k3s
  build_images
  maybe_tailscale
  write_config
  deploy_platform
  wait_platform_rollout
  install_cli
  post_install_cleanup
  log "platform-ready — open admin UI via Kong route on this host"
}

main "$@"