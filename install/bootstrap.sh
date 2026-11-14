#!/usr/bin/env bash
# Obelisk platform bootstrap — idempotent where possible.
set -euo pipefail

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

write_config() {
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
  helm upgrade --install obelisk-platform "$OBELISK_ROOT/deploy/helm/platform" \
    --namespace "$NAMESPACE" --create-namespace --wait --timeout 15m \
    --set obelisk.configHostPath="$CONFIG_PATH"
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
  maybe_tailscale
  write_config
  deploy_platform
  install_cli
  log "platform-ready — open admin UI via Kong route on this host"
}

main "$@"