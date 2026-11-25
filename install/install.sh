#!/usr/bin/env bash
# VPS production entry: fetch GitHub source archive, prepare host, run bootstrap.
set -euo pipefail

OBELISK_ROOT="${OBELISK_ROOT:-/opt/obelisk}"
OBELISK_REPO="${OBELISK_REPO:-hsoni92/obelisk}"
INSTALL_LOG="${OBELISK_INSTALL_LOG:-/var/log/obelisk-install.log}"
RELEASE_BASE_URL="${OBELISK_RELEASE_BASE_URL:-https://github.com}"

# Two modes:
#   release (default): download the CLI binary and use images CI built for tag build-N. No Docker, no Go.
#   dev: build images and the CLI on this machine from a git ref (OBELISK_REF=main, a branch, a SHA).
# OBELISK_REF unset or build-N => release. Any other ref, or OBELISK_SOURCE=dev, => dev.
# OBELISK_REEXEC is set when a release install hands over to the freshly fetched install.sh.
if [[ -z "${OBELISK_SOURCE:-}" ]]; then
  if [[ -n "${OBELISK_REEXEC:-}" ]] || [[ -z "${OBELISK_REF:-}" ]] || [[ "${OBELISK_REF}" =~ ^build-[0-9]+$ ]]; then
    if [[ "${OBELISK_SKIP_SOURCE_FETCH:-false}" == "true" && -z "${OBELISK_REEXEC:-}" ]]; then
      OBELISK_SOURCE=dev
    else
      OBELISK_SOURCE=release
    fi
  else
    OBELISK_SOURCE=dev
  fi
fi
if [[ "$OBELISK_SOURCE" == "dev" ]]; then
  OBELISK_REF="${OBELISK_REF:-main}"
fi
OBELISK_REF="${OBELISK_REF:-}"

# When piped from curl, BASH_SOURCE[0] is unset (set -u would abort); sibling sources are inlined below.
SCRIPT_DIR=""
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
  _install_sh="${BASH_SOURCE[0]}"
  if [[ -f "$_install_sh" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "$_install_sh")" && pwd)"
  fi
fi
if [[ -n "$SCRIPT_DIR" && -f "$SCRIPT_DIR/fetch-source.sh" ]]; then
  # shellcheck source=install/fetch-source.sh
  source "$SCRIPT_DIR/fetch-source.sh"
fi
if [[ -n "$SCRIPT_DIR" && -f "$SCRIPT_DIR/resolve-admin-url.sh" ]]; then
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
  case "$(uname -m)" in
    x86_64|amd64) OBELISK_ARCH=amd64 ;;
    aarch64|arm64) OBELISK_ARCH=arm64 ;;
    *) die "unsupported architecture $(uname -m); supported: amd64, arm64" ;;
  esac
  export OBELISK_ARCH
  if [[ "$OBELISK_SOURCE" == "dev" && "$OBELISK_ARCH" != "amd64" ]]; then
    log "note: source builds on $OBELISK_ARCH rely on Dockerfile.controller TARGETARCH support"
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
  log "installing docker.io and docker-cli"
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y docker.io
  # docker.io ships dockerd only; the `docker` client is a separate package.
  if ! command -v docker >/dev/null 2>&1; then
    apt-get install -y docker-cli
  fi
  command -v docker >/dev/null 2>&1 || die "docker CLI missing after install"
  systemctl enable --now docker
  log "docker installed"
}

ensure_go() {
  # Snap binaries live in /snap/bin, which is missing from PATH on Debian and
  # under sudo's secure_path. Add it so `go` resolves both before and after install.
  case ":$PATH:" in
    *:/snap/bin:*) ;;
    *) export PATH="$PATH:/snap/bin" ;;
  esac
  hash -r

  if command -v go >/dev/null 2>&1; then
    log "go $(go version | awk '{print $3}')"
    return
  fi
  if command -v snap >/dev/null 2>&1; then
    log "installing Go via snap"
    # Freshly installed snapd may still be restarting/seeding; wait before installing.
    snap wait system seed.loaded >/dev/null 2>&1 || true
    snap install go --classic
    hash -r
    command -v go >/dev/null 2>&1 \
      || die "go installed via snap but not found; expected /snap/bin/go"
    log "go $(go version | awk '{print $3}') (from /snap/bin)"
    return
  fi
  die "go not found and snap unavailable; install Go 1.22+ and re-run"
}

build_cli() {
  [[ -d "$OBELISK_ROOT/cmd/obelisk" ]] || die "missing $OBELISK_ROOT/cmd/obelisk after extract"
  log "building obelisk CLI"
  (cd "$OBELISK_ROOT" && go build -o obelisk ./cmd/obelisk)
  verify_cli_capabilities
}

verify_cli_capabilities() {
  local help
  help="$("$OBELISK_ROOT/obelisk" 2>&1 || true)"
  if [[ "$help" != *"configure deployment"* ]]; then
    log "warning: CLI built from ref=$OBELISK_REF lacks 'configure deployment'"
    log "         run: sudo obelisk update"
  fi
}

# ---- release mode helpers -------------------------------------------------

resolve_release_tag() {
  # Pin to OBELISK_REF=build-N, or follow github.com/<repo>/releases/latest (a web redirect:
  # no API rate limit). Sets OBELISK_REF to the tag.
  if [[ "$OBELISK_REF" =~ ^build-[0-9]+$ ]]; then
    return
  fi
  local loc
  loc="$(curl -sS -o /dev/null -w '%{redirect_url}' "$RELEASE_BASE_URL/$OBELISK_REPO/releases/latest")" \
    || die "cannot reach $RELEASE_BASE_URL to find the latest release"
  OBELISK_REF="${loc##*/}"
  [[ "$OBELISK_REF" =~ ^build-[0-9]+$ ]] \
    || die "no release has been published yet for $OBELISK_REPO. To build from source: OBELISK_REF=main"
  log "latest release: $OBELISK_REF"
}

release_image_registry() {
  if [[ -n "${OBELISK_IMAGE_REGISTRY:-}" ]]; then
    printf '%s' "${OBELISK_IMAGE_REGISTRY%/}"
    return
  fi
  local owner
  owner="$(printf '%s' "${OBELISK_REPO%%/*}" | tr '[:upper:]' '[:lower:]')"
  printf 'ghcr.io/%s' "$owner"
}

check_images_pullable() {
  # Fail before touching the host if the release's images cannot be pulled anonymously
  # (for example a GHCR package that is still private).
  local registry
  registry="$(release_image_registry)"
  case "$registry" in
    ghcr.io/*) ;;
    *) log "skipping image pull check for registry $registry"; return 0 ;;
  esac
  local path="${registry#ghcr.io/}" image token code
  for image in obelisk-controller obelisk-admin-ui; do
    token="$(curl -fsS "https://ghcr.io/token?scope=repository:${path}/${image}:pull" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')" \
      || die "cannot get a pull token for ghcr.io/${path}/${image}"
    [[ -n "$token" ]] || die "no pull token for ghcr.io/${path}/${image}"
    code="$(curl -s -o /dev/null -w '%{http_code}' \
      -H "Authorization: Bearer $token" \
      -H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json' \
      "https://ghcr.io/v2/${path}/${image}/manifests/${OBELISK_REF}")"
    [[ "$code" == "200" ]] \
      || die "ghcr.io/${path}/${image}:${OBELISK_REF} is not pullable anonymously (HTTP $code). The package must be Public."
  done
  log "images for $OBELISK_REF are pullable from $registry"
}

stage_cli() {
  # Download and verify the CLI for this architecture into a temp dir. Nothing on the host changes yet.
  local base="$RELEASE_BASE_URL/$OBELISK_REPO/releases/download/$OBELISK_REF"
  local name="obelisk-linux-$OBELISK_ARCH" expected
  OBELISK_CLI_STAGE="$(mktemp -d)"
  export OBELISK_CLI_STAGE
  log "downloading $name ($OBELISK_REF)"
  curl -fsSL "$base/$name" -o "$OBELISK_CLI_STAGE/obelisk" || die "cannot download $base/$name"
  curl -fsSL "$base/SHA256SUMS" -o "$OBELISK_CLI_STAGE/SHA256SUMS" || die "cannot download $base/SHA256SUMS"
  expected="$(awk -v f="$name" '$2 == f {print $1}' "$OBELISK_CLI_STAGE/SHA256SUMS")"
  [[ -n "$expected" ]] || die "SHA256SUMS has no entry for $name"
  echo "$expected  $OBELISK_CLI_STAGE/obelisk" | sha256sum -c - >/dev/null || die "checksum mismatch for $name"
  chmod 0755 "$OBELISK_CLI_STAGE/obelisk"
  log "cli checksum ok"
}

place_cli() {
  [[ -f "${OBELISK_CLI_STAGE:-}/obelisk" ]] || die "staged CLI missing; re-run the update"
  install -m 0755 "$OBELISK_CLI_STAGE/obelisk" "$OBELISK_ROOT/obelisk"
  verify_cli_capabilities
}

prepull_images() {
  # Upgrade only: pull the new images into k3s before the tree is replaced, so a slow or failed
  # pull leaves the machine untouched. On a fresh install k3s is not there yet; bootstrap handles it.
  command -v k3s >/dev/null 2>&1 || return 0
  local registry image
  registry="$(release_image_registry)"
  for image in obelisk-controller obelisk-admin-ui; do
    log "pulling $registry/$image:$OBELISK_REF"
    k3s ctr images pull "$registry/$image:$OBELISK_REF" >/dev/null || die "pull failed for $registry/$image:$OBELISK_REF"
  done
}

acquire_lock() {
  mkdir -p /var/lock 2>/dev/null || true
  exec 9>/var/lock/obelisk-install.lock
  flock -n 9 || die "another obelisk install or update is already running"
}

backup_tree() {
  [[ -d "$OBELISK_ROOT/install" ]] || return 0
  rm -rf "$OBELISK_ROOT.prev"
  cp -a "$OBELISK_ROOT" "$OBELISK_ROOT.prev"
}

restore_tree_on_failure() {
  local rc=$?
  trap - EXIT
  if [[ $rc -ne 0 && -d "$OBELISK_ROOT.prev" ]]; then
    log "update failed (exit $rc); restoring previous source tree from $OBELISK_ROOT.prev"
    rsync -a --delete "$OBELISK_ROOT.prev/" "$OBELISK_ROOT/" || log "WARNING: could not restore $OBELISK_ROOT.prev"
  fi
  exit "$rc"
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

meta_value() {
  # Read a string field from the previous .install-meta.json (empty if absent).
  sed -n "s/.*\"$1\": *\"\([^\"]*\)\".*/\1/p" "$OBELISK_ROOT/.install-meta.json" 2>/dev/null | head -n1 || true
}

write_install_meta() {
  local meta="$OBELISK_ROOT/.install-meta.json"
  local ts tag build release_tag
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  tag="${OBELISK_IMAGE_TAG:-}"
  build=0
  release_tag=""
  if [[ "$OBELISK_SOURCE" == "release" ]]; then
    build="${OBELISK_REF#build-}"
    release_tag="$OBELISK_REF"
  fi
  # Write to a temp file and rename so a crash never leaves a half-written meta file.
  cat >"$meta.tmp" <<EOF
{"schema":2,"repo":"$OBELISK_REPO","ref":"$OBELISK_REF","tag":"$release_tag","build":$build,"source":"$OBELISK_SOURCE","profile":"${OBELISK_PROFILE:-vps}","arch":"${OBELISK_ARCH:-}","installed_at":"$ts","image_tag":"$tag"}
EOF
  mv -f "$meta.tmp" "$meta"
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
  local pub=""
  pub="$(echo "${base%/}" | sed -n 's#.*://\([^/:]*\).*#\1#p')"
  log "next: sudo obelisk configure deployment --profile vps${pub:+ --public-host $pub}"
  log "       sudo obelisk configure github   # interactive OAuth wizard"
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
  local existing_profile
  existing_profile="$(meta_value profile)"
  export OBELISK_PROFILE="${OBELISK_PROFILE:-${existing_profile:-vps}}"
  if [[ "$OBELISK_SOURCE" == "release" ]]; then
    export OBELISK_BUILD_IMAGES=false
    export OBELISK_IMAGE_REGISTRY="$(release_image_registry)"
    export OBELISK_IMAGE_TAG="$OBELISK_REF"
  else
    export OBELISK_BUILD_IMAGES="${OBELISK_BUILD_IMAGES:-true}"
  fi
  if [[ "$OBELISK_PROFILE" != "vps" && "${OBELISK_INSTALL_MODE:-}" != "upgrade" ]]; then
    die "install.sh first install is VPS-only; set OBELISK_PROFILE=vps"
  fi
  if [[ -z "${OBELISK_HELM_EXTRA_ARGS:-}" && "$OBELISK_PROFILE" == "vps" ]]; then
    export OBELISK_HELM_EXTRA_ARGS='--set kong.service.type=LoadBalancer'
  fi
  log "running bootstrap profile=$OBELISK_PROFILE source=$OBELISK_SOURCE"
  "$OBELISK_ROOT/install/bootstrap.sh"
}

run_dev_install() {
  preflight_os
  fetch_and_extract
  if [[ "${OBELISK_BUILD_IMAGES:-true}" == "true" ]]; then
    ensure_docker
  fi
  ensure_go
  build_cli
  if [[ -f "$OBELISK_ROOT/install/image-tag.sh" ]]; then
    # shellcheck source=install/image-tag.sh
    source "$OBELISK_ROOT/install/image-tag.sh"
    export OBELISK_IMAGE_TAG="$(obelisk_resolve_image_tag)"
    log "platform image tag: $OBELISK_IMAGE_TAG"
  fi
  invoke_bootstrap
  write_install_meta
  print_admin_url
}

run_release_install() {
  if [[ -z "${OBELISK_REEXEC:-}" ]]; then
    # Phase A: runs the script from the CURRENTLY installed build (or the curl|bash copy).
    # Keep it small and stable: everything that can fail before the host changes comes first.
    preflight_os
    acquire_lock
    resolve_release_tag
    check_images_pullable
    stage_cli
    prepull_images
    if [[ "${OBELISK_INSTALL_MODE:-}" == "upgrade" ]]; then
      # Read the profile now: fetch_and_extract replaces the tree, including .install-meta.json.
      if [[ -z "${OBELISK_PROFILE:-}" ]]; then
        OBELISK_PROFILE="$(meta_value profile)"
      fi
      export OBELISK_PROFILE
      backup_tree
    fi
    fetch_and_extract
    if [[ -x "$OBELISK_ROOT/install/install.sh" ]]; then
      # Hand over to the install.sh that ships with the release being installed, so the
      # steps below always match the release (not the build we are upgrading from).
      export OBELISK_REEXEC=1 OBELISK_REF OBELISK_CLI_STAGE
      log "continuing with install.sh from $OBELISK_REF"
      exec bash "$OBELISK_ROOT/install/install.sh"
    fi
    die "release $OBELISK_REF has no install/install.sh"
  fi
  # Phase B: running the install.sh that shipped with the release.
  preflight_os
  if [[ "${OBELISK_INSTALL_MODE:-}" == "upgrade" ]]; then
    trap restore_tree_on_failure EXIT
  fi
  place_cli
  invoke_bootstrap
  write_install_meta
  trap - EXIT
  rm -rf "${OBELISK_CLI_STAGE:-}" "$OBELISK_ROOT.prev" 2>/dev/null || true
  print_admin_url
}

run_install_body() {
  if [[ -n "${OBELISK_REEXEC:-}" ]]; then
    : # phase B of a release install: keep the log written by phase A
  elif [[ "${OBELISK_INSTALL_MODE:-}" != "upgrade" ]]; then
    : >"$INSTALL_LOG" 2>/dev/null || true
  else
    log "=== upgrade started $(date -u +%Y-%m-%dT%H:%M:%SZ) source=${OBELISK_SOURCE} ref=${OBELISK_REF:-latest} ==="
  fi
  if [[ "$OBELISK_SOURCE" == "release" ]]; then
    run_release_install
  else
    run_dev_install
  fi
}

main() {
  if [[ "${OBELISK_INSTALL_MODE:-}" == "upgrade" ]]; then
    export OBELISK_INSTALL_UPGRADE=true
    log "upgrade: rsync source into $OBELISK_ROOT (config and credentials on disk are preserved)"
  fi
  run_install_body
}

main "$@"