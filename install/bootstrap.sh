#!/usr/bin/env bash
# Obelisk platform bootstrap — idempotent where possible.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/obelisk-paths.sh" ]]; then
  # shellcheck source=install/obelisk-paths.sh
  source "$SCRIPT_DIR/obelisk-paths.sh"
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
  log "preflight ok profile=$PROFILE tailscale=$ENABLE_TAILSCALE"
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
  OBELISK_IMAGE_TAG="${OBELISK_IMAGE_TAG:-0.1.0}" "$OBELISK_ROOT/install/build-images.sh"
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
  local helm_extra="${OBELISK_HELM_EXTRA_ARGS:-}"
  if [[ "$PROFILE" == "vps" && -z "$helm_extra" ]]; then
    helm_extra='--set kong.service.type=LoadBalancer'
    log "VPS default: kong LoadBalancer (k3s servicelb)"
  fi
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
  helm upgrade --install obelisk-platform "$OBELISK_ROOT/deploy/helm/platform" \
    --namespace "$NAMESPACE" --create-namespace --wait --timeout 15m \
    --set obelisk.configHostPath="$CONFIG_PATH" \
    --set obelisk.helmRoot="$OBELISK_ROOT" \
    $([[ -n "$public_host" ]] && echo --set "obelisk.publicHost=$public_host") \
    ${helm_extra}
  log "platform helm release applied"
}

install_cli() {
  if [[ -f "$OBELISK_ROOT/obelisk" ]]; then
    install -m 0755 "$OBELISK_ROOT/obelisk" /usr/local/bin/obelisk
    log "installed /usr/local/bin/obelisk"
  fi
}

main() {
  preflight
  install_k3s
  build_images
  maybe_tailscale
  write_config
  deploy_platform
  install_cli
  log "platform-ready — open admin UI via Kong route on this host"
}

main "$@"