#!/usr/bin/env bash
# Prod-like install via install/install.sh + archive (no rsync).
set -euo pipefail

SRC_MOUNT="${OBELISK_VM_SRC:-/mnt/obelisk-src}"
OBELISK_ROOT="${OBELISK_ROOT:-/opt/obelisk}"

log() { printf '[obelisk-curl-install] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

[[ "$(id -u)" -eq 0 ]] || die "run with sudo"

[[ -x "$SRC_MOUNT/install/install.sh" ]] || die "missing $SRC_MOUNT/install/install.sh"

if [[ -z "${OBELISK_ARCHIVE_URL:-}" ]]; then
  parent="$(dirname "$SRC_MOUNT")"
  base="$(basename "$SRC_MOUNT")"
  archive="$(mktemp /tmp/obelisk-archive.XXXXXX.tar.gz)"
  log "packing $SRC_MOUNT -> $archive"
  tar -czf "$archive" -C "$parent" "$base"
  export OBELISK_ARCHIVE_URL="file://${archive}"
  export OBELISK_REF="${OBELISK_REF:-local-test}"
fi

export OBELISK_ROOT
exec "$SRC_MOUNT/install/install.sh"