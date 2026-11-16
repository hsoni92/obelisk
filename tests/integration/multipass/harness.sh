#!/usr/bin/env bash
# Multipass harness: prod-like Obelisk install on a fresh Ubuntu VM.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
VM_NAME="${OBELISK_VM_NAME:-obelisk-vm}"
MOUNT_PATH="${OBELISK_VM_MOUNT:-/mnt/obelisk-src}"
CPUS="${OBELISK_VM_CPUS:-4}"
MEM="${OBELISK_VM_MEM:-8G}"
DISK="${OBELISK_VM_DISK:-40G}"
CLOUD_INIT="${OBELISK_VM_CLOUD_INIT:-$ROOT/tests/integration/multipass/cloud-init.yaml}"

log() { printf '[obelisk-harness] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

need() { command -v "$1" >/dev/null || die "missing $1"; }

vm_exists() {
  multipass info "$VM_NAME" &>/dev/null
}

vm_ip() {
  multipass info "$VM_NAME" 2>/dev/null | awk '/^IPv4:/ { print $2; exit }'
}

usage() {
  cat <<EOF
Usage: $0 <command>

Commands:
  up        Launch VM, mount repo, wait for cloud-init
  install   Sync repo in VM and run install/bootstrap.sh (VPS profile)
  smoke     Run in-VM smoke (basic or OBELISK_SMOKE_LEVEL=full)
  all       up + install + smoke
  url       Print Kong URL on the VM (after install)
  shell     Interactive shell on the VM
  stop      Stop VM
  destroy   Delete VM

Environment:
  OBELISK_VM_NAME, OBELISK_VM_CPUS, OBELISK_VM_MEM, OBELISK_VM_DISK
  OBELISK_SMOKE_LEVEL=basic|full
  OBELISK_PROFILE=vps|local (install only)

Repo: $ROOT
EOF
}

cmd_up() {
  need multipass
  if vm_exists; then
    log "VM $VM_NAME already exists — starting"
    multipass start "$VM_NAME" 2>/dev/null || true
  else
    log "launching $VM_NAME (${CPUS}cpu ${MEM} ram ${DISK} disk)"
    # shellcheck disable=SC2086
    multipass launch 24.04 --name "$VM_NAME" --cpus "$CPUS" --memory "$MEM" --disk "$DISK" \
      --cloud-init "$CLOUD_INIT"
  fi
  log "mounting $ROOT -> ${VM_NAME}:${MOUNT_PATH}"
  if ! multipass info "$VM_NAME" 2>/dev/null | grep -q "$MOUNT_PATH"; then
    multipass mount "$ROOT" "${VM_NAME}:${MOUNT_PATH}"
  fi
  log "waiting for cloud-init"
  multipass exec "$VM_NAME" -- cloud-init status --wait || true
  multipass exec "$VM_NAME" -- bash -c 'command -v docker && command -v go && sudo docker info >/dev/null'
  log "VM ready: $(vm_ip)"
}

cmd_install() {
  vm_exists || die "VM missing — run: $0 up"
  multipass exec "$VM_NAME" -- sudo chmod +x "$MOUNT_PATH/tests/integration/vm/install-in-vm.sh"
  multipass exec "$VM_NAME" -- sudo OBELISK_VM_SRC="$MOUNT_PATH" OBELISK_PROFILE="${OBELISK_PROFILE:-vps}" \
    "$MOUNT_PATH/tests/integration/vm/install-in-vm.sh"
}

cmd_smoke() {
  vm_exists || die "VM missing"
  multipass exec "$VM_NAME" -- sudo chmod +x "$MOUNT_PATH/tests/integration/vm/smoke.sh"
  multipass exec "$VM_NAME" -- sudo OBELISK_SMOKE_LEVEL="${OBELISK_SMOKE_LEVEL:-basic}" \
    "$MOUNT_PATH/tests/integration/vm/smoke.sh"
}

cmd_url() {
  local ip
  ip="$(vm_ip)"
  [[ -n "$ip" ]] || die "could not resolve VM IP"
  echo "http://${ip}/"
  echo "Admin API health: http://${ip}/api/v1/health"
}

cmd_shell() {
  vm_exists || die "VM missing"
  multipass shell "$VM_NAME"
}

cmd_stop() {
  multipass stop "$VM_NAME" 2>/dev/null || true
}

cmd_destroy() {
  multipass delete "$VM_NAME" --purge 2>/dev/null || true
}

main() {
  [[ $# -ge 1 ]] || { usage; exit 1; }
  case "$1" in
    up) cmd_up ;;
    install) cmd_install ;;
    smoke) cmd_smoke ;;
    all) cmd_up; cmd_install; cmd_smoke; cmd_url ;;
    url) cmd_url ;;
    shell) cmd_shell ;;
    stop) cmd_stop ;;
    destroy) cmd_destroy ;;
    -h|--help|help) usage ;;
    *) die "unknown command: $1" ;;
  esac
}

main "$@"