#!/usr/bin/env bash
# Fresh VPS-like install inside a VM (Multipass or any Ubuntu host with sudo).
set -euo pipefail

SRC_MOUNT="${OBELISK_VM_SRC:-/mnt/obelisk-src}"
OBELISK_ROOT="${OBELISK_ROOT:-/opt/obelisk}"
PROFILE="${OBELISK_PROFILE:-vps}"

log() { printf '[obelisk-vm-install] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

[[ "$(id -u)" -eq 0 ]] || die "run with sudo"

[[ -d "$SRC_MOUNT" ]] || die "source mount missing: $SRC_MOUNT (multipass mount repo first)"

if ! command -v go >/dev/null 2>&1; then
  log "installing Go via snap"
  snap install go --classic
fi

if ! docker info >/dev/null 2>&1; then
  systemctl enable --now docker
fi

log "syncing tree to $OBELISK_ROOT"
mkdir -p "$OBELISK_ROOT"
rsync -a --delete \
  --exclude node_modules \
  --exclude .next \
  --exclude bin \
  --exclude dev \
  "$SRC_MOUNT/" "$OBELISK_ROOT/"

chmod +x "$OBELISK_ROOT/install/"*.sh

log "building obelisk CLI"
(
  cd "$OBELISK_ROOT"
  go build -o obelisk ./cmd/obelisk
)

export OBELISK_ROOT
export OBELISK_PROFILE="$PROFILE"
export OBELISK_BUILD_IMAGES=true
export OBELISK_HELM_EXTRA_ARGS="${OBELISK_HELM_EXTRA_ARGS:---set kong.service.type=LoadBalancer}"

log "running platform bootstrap (profile=$PROFILE)"
"$OBELISK_ROOT/install/bootstrap.sh"

log "install complete — root=$OBELISK_ROOT profile=$PROFILE"