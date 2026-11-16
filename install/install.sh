#!/usr/bin/env bash
# VPS production entry: fetch GitHub source archive, prepare host, run bootstrap.
set -euo pipefail

OBELISK_ROOT="${OBELISK_ROOT:-/opt/obelisk}"
OBELISK_REPO="${OBELISK_REPO:-hsoni92/obelisk}"
OBELISK_REF="${OBELISK_REF:-main}"
INSTALL_LOG="${OBELISK_INSTALL_LOG:-/var/log/obelisk-install.log}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/fetch-source.sh" ]]; then
  # shellcheck source=install/fetch-source.sh
  source "$SCRIPT_DIR/fetch-source.sh"
fi
if [[ -f "$SCRIPT_DIR/resolve-admin-url.sh" ]]; then
  # shellcheck source=install/resolve-admin-url.sh
  source "$SCRIPT_DIR/resolve-admin-url.sh"
fi

log() {
  printf '[obelisk-install] %s\n' "$*"
  if [[ -w "$(dirname "$INSTALL_LOG")" ]] 2>/dev/null; then
    printf '[obelisk-install] %s\n' "$*" >>"$INSTALL_LOG"
  fi
}
die() { log "ERROR: $*"; exit 1; }

# Inline fallbacks when piped from curl (no sibling files yet).
if ! declare -F obelisk_github_archive_url >/dev/null; then
  obelisk_github_archive_url() {
    echo "https://codeload.github.com/${1:-hsoni92/obelisk}/tar.gz/${2:-main}"
  }
fi
if ! declare -F obelisk_extract_archive >/dev/null; then
  obelisk_extract_archive() {
    local archive="$1" dest="$2" tmp top
    tmp="$(mktemp -d)"
    tar -xzf "$archive" -C "$tmp"
    top="$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -1)"
    [[ -n "$top" ]] || die "archive layout unexpected"
    rm -rf "$dest"
    mkdir -p "$dest"
    rsync -a --exclude node_modules --exclude .next --exclude bin --exclude dev "$top/" "$dest/"
    rm -rf "$tmp"
  }
fi

require_root() {
  [[ "$(id -u)" -eq 0 ]] || die "run as root (e.g. curl … | sudo bash)"
}

preflight_os() {
  require_root
  command -v curl >/dev/null || die "curl required"
  if [[ "$(uname -m)" != "x86_64" && "$(uname -m)" != "amd64" ]]; then
    die "unsupported architecture $(uname -m); v1 supports linux/amd64 only"
  fi
  if [[ -f /etc/os-release ]]; then
    # shellcheck source=/dev/null
  source /etc/os-release
    if [[ "${ID:-}" != "ubuntu" ]]; then
      log "warning: unsupported OS ID=${ID:-unknown}; Ubuntu LTS is the tested target"
    fi
  fi
  log "preflight ok arch=$(uname -m)"
}

ensure_docker() {
  if docker info >/dev/null 2>&1; then
    log "docker ok"
    return
  fi
  log "installing docker.io"
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y docker.io
  systemctl enable --now docker
  log "docker installed"
}

ensure_go() {
  if command -v go >/dev/null 2>&1; then
    log "go $(go version | awk '{print $3}')"
    return
  fi
  if command -v snap >/dev/null 2>&1; then
    log "installing Go via snap"
    snap install go --classic
    return
  fi
  die "go not found and snap unavailable; install Go 1.22+ and re-run"
}

build_cli() {
  [[ -d "$OBELISK_ROOT/cmd/obelisk" ]] || die "missing $OBELISK_ROOT/cmd/obelisk after extract"
  log "building obelisk CLI"
  (cd "$OBELISK_ROOT" && go build -o obelisk ./cmd/obelisk)
}

fetch_and_extract() {
  if [[ "${OBELISK_SKIP_SOURCE_FETCH:-false}" == "true" ]]; then
    log "skipping fetch (OBELISK_SKIP_SOURCE_FETCH=true)"
    [[ -f "$OBELISK_ROOT/install/bootstrap.sh" ]] || die "bootstrap missing under $OBELISK_ROOT"
    return
  fi

  local url="${OBELISK_ARCHIVE_URL:-}"
  if [[ -z "$url" ]]; then
    url="$(obelisk_github_archive_url "$OBELISK_REPO" "$OBELISK_REF")"
  fi
  log "fetching archive ref=$OBELISK_REF"
  local tmp archive
  tmp="$(mktemp -d)"
  archive="$tmp/obelisk.tar.gz"
  curl -fsSL "$url" -o "$archive"

  if [[ -n "${OBELISK_ARCHIVE_SHA256:-}" ]]; then
    echo "${OBELISK_ARCHIVE_SHA256}  $archive" | sha256sum -c - || die "archive checksum mismatch"
  fi

  log "extracting to $OBELISK_ROOT"
  obelisk_extract_archive "$archive" "$OBELISK_ROOT"
  rm -rf "$tmp"
}

write_install_meta() {
  local meta="$OBELISK_ROOT/.install-meta.json"
  local ts
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  cat >"$meta" <<EOF
{"repo":"$OBELISK_REPO","ref":"$OBELISK_REF","installed_at":"$ts"}
EOF
  log "wrote $meta"
}

print_admin_url() {
  if declare -F obelisk_wait_kong_health >/dev/null; then
    log "waiting for admin API via Kong (up to ${OBELISK_INSTALL_WAIT_SEC:-600}s)"
    if ! obelisk_wait_kong_health; then
      log "warning: /api/v1/health not ready — platform may still be starting"
    fi
  fi
  local base=""
  if declare -F obelisk_resolve_kong_base >/dev/null; then
    base="$(obelisk_resolve_kong_base)"
  else
    base="http://$(hostname -I 2>/dev/null | awk '{print $1}')/"
  fi
  log "platform-ready — admin UI: ${base}"
  log "verify: curl -fsS ${base%/}/api/v1/health"
  log "next: sudo obelisk configure github   # interactive OAuth wizard"
  log "       or: sudo obelisk github oauth set --client-id ... --client-secret ... --public-url ${base%/}"
  log "       then sign in with GitHub (see docs/runbooks/github-blueprint.md)"
  if [[ -t 0 && -t 1 && "${OBELISK_SKIP_GITHUB_WIZARD:-false}" != "true" && "${OBELISK_INSTALL_MODE:-}" != "upgrade" ]]; then
    if command -v obelisk >/dev/null; then
      log "starting GitHub OAuth wizard (OBELISK_SKIP_GITHUB_WIZARD=true to skip)"
      obelisk configure github || log "wizard skipped or failed — configure later with: sudo obelisk configure github"
    fi
  fi
}

invoke_bootstrap() {
  chmod +x "$OBELISK_ROOT"/install/*.sh 2>/dev/null || true
  export OBELISK_ROOT
  export OBELISK_PROFILE="${OBELISK_PROFILE:-vps}"
  export OBELISK_BUILD_IMAGES="${OBELISK_BUILD_IMAGES:-true}"
  if [[ "$OBELISK_PROFILE" != "vps" ]]; then
    die "install.sh is VPS-only; set OBELISK_PROFILE=vps"
  fi
  if [[ -z "${OBELISK_HELM_EXTRA_ARGS:-}" ]]; then
    export OBELISK_HELM_EXTRA_ARGS='--set kong.service.type=LoadBalancer'
  fi
  log "running bootstrap profile=$OBELISK_PROFILE"
  "$OBELISK_ROOT/install/bootstrap.sh"
}

run_install_body() {
  if [[ "${OBELISK_INSTALL_MODE:-}" != "upgrade" ]]; then
    : >"$INSTALL_LOG" 2>/dev/null || true
  else
    log "=== upgrade started $(date -u +%Y-%m-%dT%H:%M:%SZ) ref=${OBELISK_REF} ==="
  fi
  preflight_os
  fetch_and_extract
  if [[ "${OBELISK_BUILD_IMAGES:-true}" == "true" ]]; then
    ensure_docker
  fi
  ensure_go
  build_cli
  invoke_bootstrap
  write_install_meta
  print_admin_url
}

main() {
  if [[ "${OBELISK_INSTALL_MODE:-}" == "upgrade" ]]; then
    export OBELISK_INSTALL_UPGRADE=true
    log "upgrade: rsync source into $OBELISK_ROOT (config and credentials on disk are preserved)"
  fi
  run_install_body
}

main "$@"