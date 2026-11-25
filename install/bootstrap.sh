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
OBELISK_ROLE="${OBELISK_ROLE:-local}"
ENABLE_TAILSCALE="${OBELISK_TAILSCALE:-false}"

log() { printf '[obelisk] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

require_root() {
  [[ "$(id -u)" -eq 0 ]] || die "run as root"
}

preflight() {
  require_root
  command -v curl >/dev/null || die "curl required"
  # git is needed by helm (plugin installs) and by the platform's source tooling.
  if ! command -v git >/dev/null 2>&1; then
    log "git missing; installing git"
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y git
    command -v git >/dev/null 2>&1 || die "git still missing after install"
  fi
  if [[ "$PROFILE" != "vps" && "$PROFILE" != "local" ]]; then
    die "OBELISK_PROFILE must be vps or local"
  fi
  if [[ "$PROFILE" == "vps" && "$ENABLE_TAILSCALE" == "true" && "$OBELISK_ROLE" != "gateway" ]]; then
    die "Tailscale on VPS requires OBELISK_ROLE=gateway"
  fi
  if [[ "$OBELISK_ROLE" == "gateway" ]]; then
    ENABLE_TAILSCALE="true"
  fi
  preflight_host_edge
  log "preflight ok profile=$PROFILE role=$OBELISK_ROLE tailscale=$ENABLE_TAILSCALE"
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

HELM_VERSION="${OBELISK_HELM_VERSION:-v3.14.4}"

install_helm() {
  # Pinned release with checksum, not the install script from helm's main branch.
  local arch
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die "unsupported architecture $(uname -m) for helm" ;;
  esac
  local tmp url
  tmp="$(mktemp -d)"
  url="https://get.helm.sh/helm-${HELM_VERSION}-linux-${arch}.tar.gz"
  log "installing helm ${HELM_VERSION} (${arch})"
  curl -fsSL "$url" -o "$tmp/helm.tgz"
  echo "$(curl -fsSL "${url}.sha256sum" | cut -d' ' -f1)  $tmp/helm.tgz" | sha256sum -c - >/dev/null \
    || die "helm checksum mismatch"
  tar -xzf "$tmp/helm.tgz" -C "$tmp" "linux-${arch}/helm"
  install -m 0755 "$tmp/linux-${arch}/helm" /usr/local/bin/helm
  rm -rf "$tmp"
}

# platform_image prints the image reference for helm. With OBELISK_IMAGE_REGISTRY set
# (release installs) it is registry/name:tag; otherwise the local name used by source builds.
platform_image() {
  local name="$1"
  if [[ -n "${OBELISK_IMAGE_REGISTRY:-}" ]]; then
    printf '%s/%s:%s' "${OBELISK_IMAGE_REGISTRY%/}" "$name" "$OBELISK_IMAGE_TAG"
  else
    printf '%s:%s' "$name" "$OBELISK_IMAGE_TAG"
  fi
}

prepull_release_images() {
  # Fresh install: k3s was just installed. Pull the release images now so the rollout does not
  # wait on first-pull time. (Upgrades pre-pull earlier, in install.sh, before the tree changes.)
  [[ -n "${OBELISK_IMAGE_REGISTRY:-}" && "${OBELISK_BUILD_IMAGES:-false}" != "true" ]] || return 0
  command -v k3s >/dev/null 2>&1 || return 0
  ensure_image_tag
  local name ref attempt
  for name in obelisk-controller obelisk-admin-ui; do
    ref="$(platform_image "$name")"
    for attempt in 1 2 3 4 5 6; do
      if k3s ctr images pull "$ref" >/dev/null 2>&1; then
        log "pulled $ref"
        continue 2
      fi
      log "waiting for containerd to pull $ref (attempt $attempt)"
      sleep 5
    done
    die "could not pull $ref. Is the package public? (ghcr.io packages start private)"
  done
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

ensure_docker_running() {
  if docker info >/dev/null 2>&1; then
    log "docker running"
    return
  fi
  if ! command -v docker >/dev/null 2>&1; then
    log "docker CLI missing; installing docker.io and docker-cli"
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y docker.io
    # docker.io ships dockerd only; the `docker` client is a separate package.
    if ! command -v docker >/dev/null 2>&1; then
      apt-get install -y docker-cli
    fi
    command -v docker >/dev/null 2>&1 || die "docker CLI still missing after installing docker-cli"
  fi
  log "starting docker daemon"
  systemctl enable --now docker >/dev/null 2>&1 || systemctl restart docker >/dev/null 2>&1 || true
  local i
  for i in $(seq 1 30); do
    if docker info >/dev/null 2>&1; then
      log "docker running"
      return
    fi
    sleep 2
  done
  log "docker daemon status:"
  systemctl --no-pager status docker 2>&1 | tail -n 15 || true
  die "Docker daemon is not running. Check: journalctl -u docker -n 50 — or set OBELISK_BUILD_IMAGES=false to skip image builds"
}

build_images() {
  if [[ "${OBELISK_BUILD_IMAGES:-false}" != "true" ]]; then
    log "skipping image build (set OBELISK_BUILD_IMAGES=true to build and import)"
    return
  fi
  ensure_docker_running
  if [[ ! -x "$OBELISK_ROOT/install/build-images.sh" ]]; then
    die "OBELISK_BUILD_IMAGES set but $OBELISK_ROOT/install/build-images.sh missing or not executable"
  fi
  ensure_image_tag
  OBELISK_IMAGE_TAG="$OBELISK_IMAGE_TAG" "$OBELISK_ROOT/install/build-images.sh"
}

maybe_tailscale() {
  if [[ "$ENABLE_TAILSCALE" != "true" ]]; then
    log "skipping tailscale (profile=$PROFILE role=$OBELISK_ROLE)"
    return
  fi
  if command -v tailscale >/dev/null; then
    if systemctl is-active --quiet tailscaled 2>/dev/null; then
      log "tailscaled running — join the tailnet from Platform → Tailscale or: obelisk gateway enable"
    else
      log "tailscale CLI present — start tailscaled (systemctl enable --now tailscaled) or: sudo obelisk tailscale install"
    fi
  else
    if command -v apt-get >/dev/null 2>&1; then
      curl -fsSL https://pkgs.tailscale.com/stable/debian/"$(. /etc/os-release && echo "$VERSION_CODENAME")".noarmor.gpg | tee /usr/share/keyrings/tailscale-archive-keyring.gpg >/dev/null
      curl -fsSL https://pkgs.tailscale.com/stable/debian/"$(. /etc/os-release && echo "$VERSION_CODENAME")".tailscale-keyring.list | tee /etc/apt/sources.list.d/tailscale.list
      apt-get update -qq && apt-get install -y tailscale
      systemctl enable --now tailscaled || true
      log "installed tailscale; join from admin UI Platform → Tailscale"
    else
      log "install tailscale: sudo obelisk tailscale install   or https://tailscale.com/download"
    fi
  fi
}

ensure_lib_paths() {
  mkdir -p /var/lib/obelisk/credentials
  chmod 700 /var/lib/obelisk /var/lib/obelisk/credentials 2>/dev/null || true
  # appliance-id (mode 0600) is written on first machine.id mint; keep /var/lib/obelisk on reinstall when possible.
}

write_config() {
  ensure_lib_paths
  mkdir -p "$(dirname "$CONFIG_PATH")"
  # The live appliance config is /var/lib/obelisk/obelisk.yaml (writes from the CLI and
  # controller land there); /etc/obelisk/obelisk.yaml is only a seed. Never re-init when
  # either exists, or an upgrade resets profile/role and drops gateway registration.
  local live_config="${OBELISK_LIB_DIR:-/var/lib/obelisk}/obelisk.yaml"
  if [[ -f "$live_config" ]]; then
    log "live config exists: $live_config (keeping profile/role/gateway settings)"
    return
  fi
  if [[ -f "$CONFIG_PATH" ]]; then
    log "config exists: $CONFIG_PATH"
    return
  fi
  if [[ -x "$OBELISK_ROOT/obelisk" ]]; then
  "$OBELISK_ROOT/obelisk" init-config --profile "$PROFILE" --role "$OBELISK_ROLE" --path "$CONFIG_PATH" \
    $([[ "$ENABLE_TAILSCALE" == "true" ]] && echo --tailscale)
  elif [[ -f "$OBELISK_ROOT/install/obelisk.yaml.example" ]]; then
    sed "s/profile: vps/profile: $PROFILE/" "$OBELISK_ROOT/install/obelisk.yaml.example" >"$CONFIG_PATH"
  else
    die "no obelisk CLI or example config under $OBELISK_ROOT"
  fi
  log "wrote $CONFIG_PATH"
}

adopt_unmanaged_helm_resources() {
  # Hot-patched or kubectl-apply'd native edge must be adoptable before helm upgrade.
  export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
  command -v kubectl >/dev/null 2>&1 || return 0
  local rel="obelisk-platform"
  local adopt=(
    "deployment/obelisk-native-edge"
    "service/obelisk-native-edge"
  )
  for obj in "${adopt[@]}"; do
    if ! kubectl get -n "$NAMESPACE" "$obj" >/dev/null 2>&1; then
      continue
    fi
    if kubectl get -n "$NAMESPACE" "$obj" \
      -o jsonpath='{.metadata.labels.app\.kubernetes\.io/managed-by}' 2>/dev/null | grep -qx Helm; then
      continue
    fi
    log "adopting $obj for helm release $rel"
    kubectl annotate -n "$NAMESPACE" "$obj" \
      "meta.helm.sh/release-name=$rel" \
      "meta.helm.sh/release-namespace=$NAMESPACE" \
      --overwrite
    kubectl label -n "$NAMESPACE" "$obj" \
      "app.kubernetes.io/managed-by=Helm" \
      --overwrite
  done
}

deploy_platform() {
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
  command -v helm >/dev/null || install_helm
  ensure_image_tag
  adopt_unmanaged_helm_resources
  local helm_extra="${OBELISK_HELM_EXTRA_ARGS:-}"
  if [[ "$PROFILE" == "vps" && -z "$helm_extra" ]]; then
    helm_extra='--set kong.service.type=LoadBalancer --set nativeEdge.service.type=LoadBalancer'
    log "VPS default: LoadBalancer on 9080 (Kong) + 9443 (native edge); never 80/443"
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
  # Upgrades roll the chart back on failure; --wait makes "applied" mean "pods ready".
  local helm_safety=()
  if [[ "${OBELISK_INSTALL_UPGRADE:-false}" == "true" ]]; then
    helm_safety=(--atomic --wait --timeout "${OBELISK_HELM_TIMEOUT:-10m}")
  fi
  log "applying helm release obelisk-platform (images ${OBELISK_IMAGE_TAG}); waiting for rollouts next"
  helm upgrade --install obelisk-platform "$OBELISK_ROOT/deploy/helm/platform" \
    --namespace "$NAMESPACE" --create-namespace \
    --history-max "$helm_history_max" \
    --set obelisk.configHostPath="$CONFIG_PATH" \
    --set obelisk.helmRoot="$OBELISK_ROOT" \
    --set "obelisk.imageController=$(platform_image obelisk-controller)" \
    --set "obelisk.imageAdminUI=$(platform_image obelisk-admin-ui)" \
    "${helm_safety[@]}" \
    $([[ -n "$public_host" ]] && echo --set "obelisk.publicHost=$public_host") \
    --set "kong.service.httpPort=$edge_http_port" \
    --set "nativeEdge.service.tlsPort=$edge_tls_port" \
    ${helm_extra}
  log "platform helm release applied (images ${OBELISK_IMAGE_TAG})"
}

wait_platform_rollout() {
  export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
  command -v kubectl >/dev/null 2>&1 || return 0
  kubectl cluster-info >/dev/null 2>&1 || return 0
  local timeout="${OBELISK_PLATFORM_WAIT_SEC:-300}"
  for dep in obelisk-kong obelisk-controller obelisk-admin-ui obelisk-native-edge; do
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
    # Copy beside the target then rename: replacing a running binary in place can fail (ETXTBSY).
    install -m 0755 "$OBELISK_ROOT/obelisk" /usr/local/bin/obelisk.new
    mv -f /usr/local/bin/obelisk.new /usr/local/bin/obelisk
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
  prepull_release_images
  maybe_tailscale
  write_config
  deploy_platform
  wait_platform_rollout
  install_cli
  post_install_cleanup
  log "platform-ready — open admin UI via Kong route on this host"
}

main "$@"