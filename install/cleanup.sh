#!/usr/bin/env bash
# Reclaim disk after install/upgrade: old platform images, Helm history, dangling Docker layers.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OBELISK_ROOT="${OBELISK_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"
OBELISK_NAMESPACE="${OBELISK_NAMESPACE:-obelisk}"
KEEP_HELM_REVISIONS="${OBELISK_HELM_HISTORY_MAX:-10}"
DRY_RUN="${OBELISK_CLEANUP_DRY_RUN:-false}"

log() { printf '[obelisk-cleanup] %s\n' "$*"; }

docker_cmd() {
  if docker info >/dev/null 2>&1; then
    docker "$@"
  elif sudo docker info >/dev/null 2>&1; then
    sudo docker "$@"
  else
    return 1
  fi
}

read_keep_platform_tag() {
  KEEP_PLATFORM_TAG="${OBELISK_IMAGE_TAG:-}"
  if [[ -n "$KEEP_PLATFORM_TAG" ]]; then
    return
  fi
  local meta="$OBELISK_ROOT/.install-meta.json"
  if [[ -f "$meta" ]] && command -v python3 >/dev/null; then
    KEEP_PLATFORM_TAG="$(python3 -c "import json; print(json.load(open('$meta')).get('image_tag','') or '')" 2>/dev/null || true)"
  fi
  if [[ -n "$KEEP_PLATFORM_TAG" ]]; then
    return
  fi
  export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
  if command -v kubectl >/dev/null 2>&1 && kubectl cluster-info >/dev/null 2>&1; then
    local img
    img="$(kubectl get deployment -n "$OBELISK_NAMESPACE" obelisk-admin-ui \
      -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true)"
    if [[ "$img" == *":"* ]]; then
      KEEP_PLATFORM_TAG="${img##*:}"
    fi
  fi
}

run_or_dry() {
  local desc="$1"
  shift
  if [[ "$DRY_RUN" == "true" ]]; then
    log "dry-run: $desc"
    log "  would run: $*"
    return 0
  fi
  log "$desc"
  "$@"
}

image_matches_keep_tag() {
  local ref="$1"
  local tag="${ref#*:}"
  [[ -n "${KEEP_PLATFORM_TAG:-}" && "$tag" == "$KEEP_PLATFORM_TAG" ]]
}

prune_docker_platform_images() {
  docker_cmd info >/dev/null 2>&1 || {
    log "docker not available — skip image prune"
    return 0
  }
  if [[ "${OBELISK_CLEANUP_ALL_PLATFORM_IMAGES:-false}" == "true" ]]; then
    while IFS= read -r ref; do
      [[ -z "$ref" ]] && continue
      run_or_dry "remove docker image $ref" docker_cmd image rm -f "$ref" || true
    done < <(docker_cmd images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | grep -E '(^|/)obelisk-(controller|admin-ui):' || true)
    return
  fi
  read_keep_platform_tag
  while IFS= read -r ref; do
    [[ -z "$ref" ]] && continue
    if image_matches_keep_tag "$ref"; then
      continue
    fi
    if [[ -z "${KEEP_PLATFORM_TAG:-}" ]]; then
      continue
    fi
    run_or_dry "remove stale docker platform image $ref" docker_cmd image rm -f "$ref" || true
  done < <(docker_cmd images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | grep -E '(^|/)obelisk-(controller|admin-ui):' || true)
}

prune_k3s_platform_images() {
  command -v k3s >/dev/null 2>&1 || {
    log "k3s not available — skip containerd image prune"
    return 0
  }
  read_keep_platform_tag
  while IFS= read -r ref; do
    [[ -z "$ref" ]] && continue
    [[ "$ref" =~ obelisk-(controller|admin-ui) ]] || continue
    if [[ "${OBELISK_CLEANUP_ALL_PLATFORM_IMAGES:-false}" == "true" ]]; then
      run_or_dry "remove k3s image $ref" k3s ctr images rm "$ref" || true
      continue
    fi
    if [[ -n "${KEEP_PLATFORM_TAG:-}" && "$ref" == *":${KEEP_PLATFORM_TAG}" ]]; then
      continue
    fi
    if [[ -z "${KEEP_PLATFORM_TAG:-}" ]]; then
      continue
    fi
    run_or_dry "remove stale k3s platform image $ref" k3s ctr images rm "$ref" || true
  done < <(k3s ctr images ls 2>/dev/null | awk '{print $1}' | grep -E 'obelisk-(controller|admin-ui)' || true)
}

prune_docker_dangling() {
  [[ "${OBELISK_CLEANUP_BUILD_CACHE:-false}" == "true" ]] || return 0
  docker_cmd info >/dev/null 2>&1 || return 0
  run_or_dry "prune dangling docker images and build cache" docker_cmd image prune -af || true
  run_or_dry "prune docker builder cache" docker_cmd builder prune -af || true
}

prune_docker_dangling_light() {
  docker_cmd info >/dev/null 2>&1 || return 0
  run_or_dry "prune dangling docker layers" docker_cmd image prune -f || true
}

trim_helm_release_history() {
  local release="$1"
  local keep="$2"
  command -v kubectl >/dev/null 2>&1 || return 0
  export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
  kubectl cluster-info >/dev/null 2>&1 || {
    log "kubernetes api unavailable — skip helm history trim for $release"
    return 0
  }
  mapfile -t versions < <(
    kubectl get secret -n "$OBELISK_NAMESPACE" -l "owner=helm,name=${release}" \
      -o jsonpath='{range .items[*]}{.metadata.labels.version}{"\n"}{end}' 2>/dev/null | sort -n | uniq
  )
  local count="${#versions[@]}"
  if (( count <= keep )); then
    return 0
  fi
  local remove=$((count - keep))
  log "helm $release: $count revisions, removing $remove oldest (keep $keep)"
  local i
  for ((i = 0; i < remove; i++)); do
    local ver="${versions[i]}"
    [[ -z "$ver" ]] && continue
    local sec
    sec="$(kubectl get secret -n "$OBELISK_NAMESPACE" -l "owner=helm,name=${release},version=${ver}" \
      -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
    [[ -n "$sec" ]] || continue
    if [[ "$DRY_RUN" == "true" ]]; then
      log "dry-run: delete helm secret $sec (rev $ver)"
    else
      kubectl delete secret -n "$OBELISK_NAMESPACE" "$sec" --ignore-not-found
    fi
  done
}

prune_helm_history() {
  command -v helm >/dev/null 2>&1 || return 0
  export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
  local releases
  mapfile -t releases < <(helm list -n "$OBELISK_NAMESPACE" -q -a 2>/dev/null || true)
  local rel
  for rel in "${releases[@]}"; do
    [[ -n "$rel" ]] || continue
    trim_helm_release_history "$rel" "$KEEP_HELM_REVISIONS"
  done
}

rotate_install_log() {
  local log="${OBELISK_INSTALL_LOG:-/var/log/obelisk-install.log}"
  [[ -f "$log" ]] || return 0
  local max_mb="${OBELISK_INSTALL_LOG_MAX_MB:-8}"
  local size
  size="$(du -m "$log" 2>/dev/null | awk '{print $1}' || echo 0)"
  if (( size > max_mb )); then
    if [[ "$DRY_RUN" == "true" ]]; then
      log "dry-run: rotate install log ($size MB > ${max_mb} MB)"
    else
      log "rotate install log (${size}MB)"
      mv -f "$log" "${log}.1"
      : >"$log"
    fi
  fi
}

obelisk_post_install_cleanup() {
  if [[ "${OBELISK_SKIP_POST_INSTALL_CLEANUP:-false}" == "true" ]]; then
    log "skipped (OBELISK_SKIP_POST_INSTALL_CLEANUP=true)"
    return 0
  fi
  if [[ "${OBELISK_CLEANUP_DEEP:-false}" == "true" ]]; then
    log "post-install cleanup (deep)"
  else
    log "post-install cleanup"
  fi
  prune_docker_platform_images
  prune_k3s_platform_images
  prune_docker_dangling_light
  if [[ "${OBELISK_CLEANUP_DEEP:-false}" == "true" ]] || [[ "${OBELISK_CLEANUP_BUILD_CACHE:-false}" == "true" ]]; then
    prune_docker_dangling
  fi
  prune_helm_history
  rotate_install_log
  log "done"
}

usage() {
  cat <<'EOF'
Usage: cleanup.sh [--dry-run] [--deep]
  or:  source cleanup.sh && obelisk_post_install_cleanup

Environment:
  OBELISK_ROOT, OBELISK_NAMESPACE, OBELISK_IMAGE_TAG
  OBELISK_HELM_HISTORY_MAX (default 10; deep uses 5 via CLI)
  OBELISK_CLEANUP_DRY_RUN=true
  OBELISK_CLEANUP_DEEP=true          Docker build cache + builder prune
  OBELISK_SKIP_POST_INSTALL_CLEANUP=true
  OBELISK_CLEANUP_ALL_PLATFORM_IMAGES=true   remove all platform images (uninstall)
EOF
}

main() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run)
        DRY_RUN=true
        export OBELISK_CLEANUP_DRY_RUN=true
        shift
        ;;
      --deep)
        export OBELISK_CLEANUP_DEEP=true
        export OBELISK_HELM_HISTORY_MAX="${OBELISK_HELM_HISTORY_MAX:-5}"
        shift
        ;;
      -h | --help)
        usage
        exit 0
        ;;
      *)
        echo "unknown argument: $1" >&2
        usage >&2
        exit 1
        ;;
    esac
  done
  obelisk_post_install_cleanup
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi