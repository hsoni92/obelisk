#!/usr/bin/env bash
# VPS Obelisk teardown — inverse of bootstrap (see docs/designs/obelisk_uninstaller.md).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=install/obelisk-paths.sh
source "$SCRIPT_DIR/obelisk-paths.sh"

DRY_RUN=false
ASSUME_YES=false
TIER="full"
KEEP_DATA=false
REMOVE_K3S=false
FORCE=false
RESUME=false
PARTIAL=false

log() {
  local pfx="[obelisk-uninstall]"
  [[ "$DRY_RUN" == "true" ]] && pfx="[obelisk-uninstall] [dry-run]"
  printf '%s %s\n' "$pfx" "$*"
  if [[ -w "$(dirname "$OBELISK_UNINSTALL_LOG")" ]] 2>/dev/null; then
    printf '%s %s\n' "$pfx" "$*" >>"$OBELISK_UNINSTALL_LOG"
  fi
}
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<EOF
Usage: uninstall.sh [options]

Options:
  --dry-run           inventory only, no mutations
  --yes               skip interactive confirm
  --tier full|platform   full (default) removes PVCs and /var/lib/obelisk
  --keep-data         skip PVC delete and keep /var/lib/obelisk
  --remove-k3s        run k3s-uninstall.sh after Obelisk cleanup
  --force             allow uninstall when profile is local
  --resume            continue from .uninstall-meta.json last stage
  --root PATH         install tree (default $OBELISK_ROOT)
  --namespace NAME    kubernetes namespace (default $OBELISK_NAMESPACE)
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run) DRY_RUN=true ;;
      --yes) ASSUME_YES=true ;;
      --tier)
        shift
        TIER="${1:-full}"
        ;;
      --keep-data) KEEP_DATA=true ;;
      --remove-k3s) REMOVE_K3S=true ;;
      --force) FORCE=true ;;
      --resume) RESUME=true ;;
      --root)
        shift
        OBELISK_ROOT="${1:-/opt/obelisk}"
        ;;
      --namespace)
        shift
        OBELISK_NAMESPACE="${1:-obelisk}"
        ;;
      -h|--help) usage; exit 0 ;;
      *) die "unknown option: $1" ;;
    esac
    shift
  done
  if [[ "$TIER" == "platform" ]]; then
    KEEP_DATA=true
  fi
  if [[ "${OBELISK_UNINSTALL_DRY_RUN:-}" == "true" ]]; then
    DRY_RUN=true
  fi
  if [[ "${OBELISK_UNINSTALL_TIER:-}" == "platform" ]]; then
    TIER="platform"
    KEEP_DATA=true
  fi
  if [[ "${OBELISK_UNINSTALL_KEEP_DATA:-}" == "true" ]]; then
    KEEP_DATA=true
  fi
  if [[ "${OBELISK_REMOVE_K3S:-}" == "true" ]]; then
    REMOVE_K3S=true
  fi
  if [[ "${OBELISK_UNINSTALL_YES:-}" == "true" ]]; then
    ASSUME_YES=true
  fi
  if [[ "${OBELISK_UNINSTALL_FORCE:-}" == "true" ]]; then
    FORCE=true
  fi
}

require_root() {
  [[ "$(id -u)" -eq 0 ]] || die "run as root (sudo obelisk uninstall)"
}

write_meta() {
  local stage="$1"
  mkdir -p "$(dirname "$OBELISK_UNINSTALL_META")"
  printf '{"last_stage":"%s","tier":"%s","at":"%s"}\n' \
    "$stage" "$TIER" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$OBELISK_UNINSTALL_META"
}

resume_after() {
  local stage="$1"
  [[ "$RESUME" != "true" ]] && return 0
  [[ -f "$OBELISK_UNINSTALL_META" ]] || return 0
  local last
  last="$(grep -o '"last_stage":"[^"]*"' "$OBELISK_UNINSTALL_META" | cut -d'"' -f4 || true)"
  [[ -z "$last" ]] && return 0
  # stage ids S0..S11 — skip stages at or before last completed
  local cur="$stage"
  local last_num="${last#S}"
  local cur_num="${cur#S}"
  [[ "$cur_num" -le "$last_num" ]] && return 1
  return 0
}

profile_ok() {
  [[ "$FORCE" == "true" ]] && return 0
  local cfg="$OBELISK_CONFIG"
  [[ -f "$cfg" ]] || return 0
  if grep -qE 'profile:\s*local' "$cfg" 2>/dev/null; then
    die "config profile is local — use --force if you mean to uninstall this host"
  fi
}

kube_ready() {
  export KUBECONFIG="${KUBECONFIG:-$OBELISK_KUBECONFIG_DEFAULT}"
  [[ -r "$KUBECONFIG" ]] || return 1
  kubectl cluster-info >/dev/null 2>&1
}

stage_s0_preflight() {
  resume_after S0 || { log "skip S0 (resume)"; return 0; }
  require_root
  profile_ok
  export KUBECONFIG="${KUBECONFIG:-$OBELISK_KUBECONFIG_DEFAULT}"
  log "preflight tier=$TIER namespace=$OBELISK_NAMESPACE root=$OBELISK_ROOT"
  write_meta S0
}

stage_s1_inventory() {
  resume_after S1 || { log "skip S1 (resume)"; return 0; }
  log "=== inventory ==="
  export KUBECONFIG="${KUBECONFIG:-$OBELISK_KUBECONFIG_DEFAULT}"
  if kube_ready; then
    log "Helm releases (namespace $OBELISK_NAMESPACE):"
    helm list -n "$OBELISK_NAMESPACE" 2>/dev/null || log "(helm list failed or empty)"
    log "PVCs:"
    kubectl get pvc -n "$OBELISK_NAMESPACE" 2>/dev/null || log "(no pvc)"
    log "ClusterRole obelisk-controller:"
    kubectl get clusterrole obelisk-controller 2>/dev/null || log "(absent)"
  else
    log "kubernetes API not reachable (kubeconfig=$KUBECONFIG)"
  fi
  log "Host paths:"
  for p in "$OBELISK_ETC_DIR" "$OBELISK_LIB_DIR" "$OBELISK_ROOT" "$OBELISK_CLI_PATH"; do
    if [[ -e "$p" ]]; then log "  exists: $p"; else log "  missing: $p"; fi
  done
  log "remove-k3s requested: $REMOVE_K3S"
  write_meta S1
  if [[ "$DRY_RUN" == "true" ]]; then
    log "dry-run complete — no changes made"
    exit 0
  fi
}

confirm_apply() {
  [[ "$ASSUME_YES" == "true" ]] && return 0
  log "This will remove Obelisk from this host (tier=$TIER)."
  log "Remote GitHub blueprint and backup buckets are NOT touched."
  printf 'Type "uninstall" to continue: '
  read -r ans
  [[ "$ans" == "uninstall" ]] || die "aborted"
}

confirm_k3s() {
  [[ "$REMOVE_K3S" != "true" ]] && return 0
  if [[ "${OBELISK_UNINSTALL_K3S_CONFIRM:-}" == "1" ]]; then
    return 0
  fi
  log "WARNING: --remove-k3s deletes the entire k3s cluster on this host."
  printf 'Type "remove-k3s" to continue: '
  read -r ans
  [[ "$ans" == "remove-k3s" ]] || die "aborted (k3s removal)"
}

stage_s2_quiesce() {
  resume_after S2 || { log "skip S2 (resume)"; return 0; }
  kube_ready || { log "skip S2 (no api)"; write_meta S2; return 0; }
  if kubectl get deployment obelisk-controller -n "$OBELISK_NAMESPACE" >/dev/null 2>&1; then
    log "scaling obelisk-controller to 0"
    kubectl scale deployment/obelisk-controller -n "$OBELISK_NAMESPACE" --replicas=0
    sleep 3
  else
    log "skip S2 (no controller deployment)"
  fi
  write_meta S2
}

stage_s3_helm() {
  resume_after S3 || { log "skip S3 (resume)"; return 0; }
  kube_ready || { log "skip S3 (no api)"; write_meta S3; return 0; }
  local rels
  rels="$(helm list -q -n "$OBELISK_NAMESPACE" 2>/dev/null || true)"
  if [[ -z "$rels" ]]; then
    log "no helm releases in $OBELISK_NAMESPACE"
  else
    while IFS= read -r rel; do
      [[ -z "$rel" ]] && continue
      log "helm uninstall $rel"
      helm uninstall "$rel" -n "$OBELISK_NAMESPACE" || { log "warn: helm uninstall $rel failed"; PARTIAL=true; }
    done <<<"$rels"
  fi
  write_meta S3
}

stage_s4_pvcs() {
  resume_after S4 || { log "skip S4 (resume)"; return 0; }
  if [[ "$KEEP_DATA" == "true" ]]; then
    log "skip S4 (keep-data / platform tier)"
    write_meta S4
    return 0
  fi
  kube_ready || { log "skip S4 (no api)"; write_meta S4; return 0; }
  local pvcs
  pvcs="$(kubectl get pvc -n "$OBELISK_NAMESPACE" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "${pvcs:-0}" -gt 0 ]]; then
    log "deleting PVCs in $OBELISK_NAMESPACE"
    kubectl delete pvc -n "$OBELISK_NAMESPACE" --all --wait=true || { PARTIAL=true; log "warn: pvc delete incomplete"; }
  else
    log "no PVCs to delete"
  fi
  write_meta S4
}

stage_s5_namespace() {
  resume_after S5 || { log "skip S5 (resume)"; return 0; }
  kube_ready || { log "skip S5 (no api)"; write_meta S5; return 0; }
  if kubectl get namespace "$OBELISK_NAMESPACE" >/dev/null 2>&1; then
    log "deleting namespace $OBELISK_NAMESPACE"
    kubectl delete namespace "$OBELISK_NAMESPACE" --wait=true --timeout=120s || {
      PARTIAL=true
      log "warn: namespace delete timed out or failed (check stuck PVC finalizers)"
    }
  fi
  write_meta S5
}

stage_s6_rbac() {
  resume_after S6 || { log "skip S6 (resume)"; return 0; }
  kube_ready || { log "skip S6 (no api)"; write_meta S6; return 0; }
  log "deleting cluster RBAC obelisk-controller"
  kubectl delete clusterrole,clusterrolebinding obelisk-controller --ignore-not-found || PARTIAL=true
  write_meta S6
}

stage_s7_images() {
  resume_after S7 || { log "skip S7 (resume)"; return 0; }
  local cleanup="$OBELISK_ROOT/install/cleanup.sh"
  if [[ -f "$cleanup" ]]; then
    OBELISK_CLEANUP_ALL_PLATFORM_IMAGES=true OBELISK_ROOT="$OBELISK_ROOT" bash "$cleanup" || PARTIAL=true
  else
    if command -v k3s >/dev/null; then
      while IFS= read -r ref; do
        [[ -z "$ref" ]] && continue
        log "k3s ctr rm $ref"
        k3s ctr images rm "$ref" 2>/dev/null || true
      done < <(k3s ctr images ls 2>/dev/null | awk '{print $1}' | grep -E 'obelisk-(controller|admin-ui)' || true)
    fi
    if docker info >/dev/null 2>&1; then
      docker image rm -f obelisk-controller:0.1.0 obelisk-admin-ui:0.1.0 2>/dev/null || true
    fi
  fi
  write_meta S7
}

stage_s8_host_config() {
  resume_after S8 || { log "skip S8 (resume)"; return 0; }
  if [[ -d "$OBELISK_ETC_DIR" ]]; then
    log "removing $OBELISK_ETC_DIR"
    rm -rf "$OBELISK_ETC_DIR"
  fi
  if [[ "$KEEP_DATA" != "true" && -d "$OBELISK_LIB_DIR" ]]; then
    log "removing $OBELISK_LIB_DIR"
    rm -rf "$OBELISK_LIB_DIR"
  elif [[ "$KEEP_DATA" == "true" ]]; then
    log "keeping $OBELISK_LIB_DIR (keep-data)"
  fi
  write_meta S8
}

stage_s9_cli() {
  resume_after S9 || { log "skip S9 (resume)"; return 0; }
  if [[ -f "$OBELISK_CLI_PATH" ]]; then
    log "removing $OBELISK_CLI_PATH"
    rm -f "$OBELISK_CLI_PATH"
  fi
  write_meta S9
}

stage_s10_tree() {
  resume_after S10 || { log "skip S10 (resume)"; return 0; }
  if [[ -d "$OBELISK_ROOT" ]]; then
    log "removing install tree $OBELISK_ROOT"
    rm -rf "$OBELISK_ROOT"
  fi
  write_meta S10
}

stage_s11_k3s() {
  resume_after S11 || { log "skip S11 (resume)"; return 0; }
  [[ "$REMOVE_K3S" == "true" ]] || { write_meta S11; return 0; }
  if [[ -x /usr/local/bin/k3s-uninstall.sh ]]; then
    log "running k3s-uninstall.sh"
    /usr/local/bin/k3s-uninstall.sh || { PARTIAL=true; log "warn: k3s-uninstall failed"; }
  else
    PARTIAL=true
    log "warn: k3s-uninstall.sh not found"
  fi
  write_meta S11
}

main() {
  parse_args "$@"
  : >>"$OBELISK_UNINSTALL_LOG" 2>/dev/null || true
  log "=== uninstall started tier=$TIER ==="
  stage_s0_preflight
  stage_s1_inventory
  confirm_apply
  confirm_k3s
  stage_s2_quiesce
  stage_s3_helm
  stage_s4_pvcs
  stage_s5_namespace
  stage_s6_rbac
  stage_s7_images
  stage_s8_host_config
  stage_s9_cli
  stage_s10_tree
  stage_s11_k3s
  if [[ "$PARTIAL" == "true" ]]; then
    log "REMAINING: see warnings above — re-run obelisk uninstall"
    exit 2
  fi
  log "uninstall complete — remote blueprint/backups unchanged"
}

main "$@"