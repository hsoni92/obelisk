#!/usr/bin/env bash
# Multipass harness: prod-like Obelisk install on a fresh Ubuntu VM.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
VM_NAME="${OBELISK_VM_NAME:-obelisk-vm}"
MOUNT_PATH="${OBELISK_VM_MOUNT:-/mnt/obelisk-src}"
# Sized in resolve_vm_size: manual (test-vm) is 2 CPU / 2G; full automation stays at the design floor.
CPUS=""
MEM=""
DISK="${OBELISK_VM_DISK:-40G}"
CLOUD_INIT_FULL="${ROOT}/tests/integration/multipass/cloud-init.yaml"
CLOUD_INIT_MANUAL="${ROOT}/tests/integration/multipass/cloud-init-manual.yaml"
# manual = no repo mount, minimal cloud-init, SSH password for hand testing
OBELISK_VM_MODE="${OBELISK_VM_MODE:-full}"

log() { printf '[obelisk-harness] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

need() { command -v "$1" >/dev/null || die "missing $1"; }

vm_exists() {
  multipass info "$VM_NAME" &>/dev/null
}

vm_ip() {
  multipass info "$VM_NAME" 2>/dev/null | awk '/^IPv4:/ { print $2; exit }'
}

cloud_init_file() {
  if [[ "$OBELISK_VM_MODE" == "manual" ]]; then
    echo "$CLOUD_INIT_MANUAL"
  else
    echo "${OBELISK_VM_CLOUD_INIT:-$CLOUD_INIT_FULL}"
  fi
}

usage() {
  cat <<EOF
Usage: $0 <command>

Commands:
  up        Launch VM and wait for cloud-init (see OBELISK_VM_MODE)
  ready     up (manual mode) + print SSH / install hints — used by make test-vm
  login     Print SSH address and credentials (VM must be running)
  install   Sync repo in VM and run install/bootstrap.sh (VPS profile; needs mount)
  smoke     Run in-VM smoke (basic or OBELISK_SMOKE_LEVEL=full)
  uninstall Run in-VM uninstall test (after install)
  all       up (full mode) + install + smoke
  url       Print Kong URL on the VM (after install)
  shell     Interactive shell on the VM (Multipass)
  stop      Stop VM
  destroy   Delete VM

Environment:
  OBELISK_VM_NAME, OBELISK_VM_CPUS, OBELISK_VM_MEM, OBELISK_VM_DISK
  Defaults: manual (test-vm) 2 CPU / 2G / 40G; full 4 CPU / 8G / 40G
  OBELISK_VM_MODE=manual|full   manual: minimal OS, SSH password, no repo mount
  OBELISK_VM_CLOUD_INIT         Override cloud-init (full mode only)
  OBELISK_VM_MOUNT=true         Mount repo (full/dev automation only; never default in manual)
  OBELISK_SMOKE_LEVEL=basic|full
  OBELISK_PROFILE=vps|local (install only)

Repo: $ROOT
EOF
}

resolve_vm_size() {
  if [[ -n "${OBELISK_VM_CPUS:-}" ]]; then
    CPUS="$OBELISK_VM_CPUS"
  elif [[ "$OBELISK_VM_MODE" == "manual" ]]; then
    CPUS=2
  else
    CPUS=4
  fi
  if [[ -n "${OBELISK_VM_MEM:-}" ]]; then
    MEM="$OBELISK_VM_MEM"
  elif [[ "$OBELISK_VM_MODE" == "manual" ]]; then
    MEM=2G
  else
    MEM=8G
  fi
}

should_mount() {
  if [[ "$OBELISK_VM_MODE" == "manual" ]]; then
    [[ "${OBELISK_VM_MOUNT:-false}" == "true" ]]
  else
    [[ "${OBELISK_VM_MOUNT:-true}" == "true" ]]
  fi
}

prod_install_url() {
  echo "https://raw.githubusercontent.com/hsoni92/obelisk/main/install/install.sh"
}

prod_install_reachable() {
  curl -fsSI "$(prod_install_url)" | head -1 | grep -q ' 200'
}

cmd_up() {
  need multipass
  local cloud_init
  cloud_init="$(cloud_init_file)"

  resolve_vm_size

  if vm_exists; then
    log "VM $VM_NAME already exists — starting"
    multipass start "$VM_NAME" 2>/dev/null || true
  else
    log "launching $VM_NAME (${CPUS}cpu ${MEM} ram ${DISK} disk) mode=$OBELISK_VM_MODE"
    # shellcheck disable=SC2086
    multipass launch 24.04 --name "$VM_NAME" --cpus "$CPUS" --memory "$MEM" --disk "$DISK" \
      --cloud-init "$cloud_init"
  fi

  if should_mount; then
    log "mounting $ROOT -> ${VM_NAME}:${MOUNT_PATH}"
    if ! multipass info "$VM_NAME" 2>/dev/null | grep -q "$MOUNT_PATH"; then
      multipass mount "$ROOT" "${VM_NAME}:${MOUNT_PATH}"
    fi
  else
    log "repo mount skipped (manual test VM)"
  fi

  log "waiting for cloud-init"
  multipass exec "$VM_NAME" -- cloud-init status --wait || true

  if [[ "$OBELISK_VM_MODE" == "full" ]]; then
    multipass exec "$VM_NAME" -- bash -c 'command -v docker && command -v go && sudo docker info >/dev/null'
  else
    multipass exec "$VM_NAME" -- bash -c 'command -v curl && systemctl is-active --quiet ssh || systemctl is-active --quiet sshd'
  fi

  log "VM ready: $(vm_ip)"
}

cmd_login() {
  local ip url
  ip="$(vm_ip)"
  url="$(prod_install_url)"
  [[ -n "$ip" ]] || die "could not resolve VM IP (is the VM up?)"

  cat <<EOF

=== Obelisk prod-like test VM ===
Fresh Ubuntu (curl + SSH only). No repo mount — install exactly as on a real VPS.

Name:     $VM_NAME
Address:  $ip
User:     ubuntu
Password: obelisk-vm   (local Multipass only)

SSH:
  ssh ubuntu@$ip
  multipass shell $VM_NAME

On the VM — production install:
  curl -fsSL $url | sudo bash

After install, smoke from the host:
  make vm-smoke
  ./tests/integration/multipass/harness.sh url

On the VM — smoke directly:
  sudo /opt/obelisk/tests/integration/vm/smoke.sh

Tear down:
  make vm-destroy

EOF
}

cmd_ready() {
  OBELISK_VM_MODE=manual OBELISK_VM_MOUNT=false cmd_up
  if ! prod_install_reachable; then
    die "production install URL not reachable (expected HTTP 200): $(prod_install_url)
Publish hsoni92/obelisk as a public GitHub repo with install/install.sh on main, then retry make test-vm"
  fi
  cmd_login
}

cmd_install() {
  vm_exists || die "VM missing — run: $0 up"
  should_mount || die "install needs repo mount — run: OBELISK_VM_MODE=full $0 up (or OBELISK_VM_MOUNT=true)"
  multipass exec "$VM_NAME" -- sudo chmod +x "$MOUNT_PATH/tests/integration/vm/install-in-vm.sh"
  multipass exec "$VM_NAME" -- sudo OBELISK_VM_SRC="$MOUNT_PATH" OBELISK_PROFILE="${OBELISK_PROFILE:-vps}" \
    "$MOUNT_PATH/tests/integration/vm/install-in-vm.sh"
}

cmd_smoke() {
  vm_exists || die "VM missing"
  local smoke=""
  if multipass exec "$VM_NAME" -- test -x /opt/obelisk/tests/integration/vm/smoke.sh 2>/dev/null; then
    smoke="/opt/obelisk/tests/integration/vm/smoke.sh"
  elif multipass info "$VM_NAME" 2>/dev/null | grep -q "$MOUNT_PATH"; then
    smoke="$MOUNT_PATH/tests/integration/vm/smoke.sh"
    multipass exec "$VM_NAME" -- sudo chmod +x "$smoke"
  else
    die "no smoke script — run prod install first (curl install.sh) or use OBELISK_VM_MODE=full with mount"
  fi
  multipass exec "$VM_NAME" -- sudo OBELISK_SMOKE_LEVEL="${OBELISK_SMOKE_LEVEL:-basic}" "$smoke"
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

cmd_uninstall() {
  vm_exists || die "VM missing — run install first"
  local script="${ROOT}/tests/integration/vm/uninstall-in-vm.sh"
  [[ -f "$script" ]] || die "missing $script"
  chmod +x "$script"
  multipass exec "$VM_NAME" -- sudo OBELISK_ROOT=/opt/obelisk "$script"
}

main() {
  [[ $# -ge 1 ]] || { usage; exit 1; }
  case "$1" in
    up) cmd_up ;;
    ready) cmd_ready ;;
    login) cmd_login ;;
    install) cmd_install ;;
    smoke) cmd_smoke ;;
    uninstall) cmd_uninstall ;;
    all) OBELISK_VM_MODE=full cmd_up; cmd_install; cmd_smoke; cmd_url ;;
    url) cmd_url ;;
    shell) cmd_shell ;;
    stop) cmd_stop ;;
    destroy) cmd_destroy ;;
    -h|--help|help) usage ;;
    *) die "unknown command: $1" ;;
  esac
}

main "$@"