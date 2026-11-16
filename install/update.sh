#!/usr/bin/env bash
# In-place upgrade: fetch GitHub source archive, rebuild images, helm upgrade.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export OBELISK_ROOT="${OBELISK_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"
export OBELISK_INSTALL_MODE=upgrade
export OBELISK_INSTALL_UPGRADE=true
export OBELISK_PROFILE="${OBELISK_PROFILE:-vps}"

exec "$SCRIPT_DIR/install.sh"