#!/usr/bin/env bash
# In-place upgrade (see install.sh): release mode downloads the CLI and pulls CI-built images;
# dev mode (OBELISK_SOURCE=dev) fetches a git ref and builds on this machine.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export OBELISK_ROOT="${OBELISK_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"
export OBELISK_INSTALL_MODE=upgrade
export OBELISK_INSTALL_UPGRADE=true
# Profile comes from OBELISK_PROFILE, else the previous .install-meta.json (install.sh), else vps.

exec "$SCRIPT_DIR/install.sh"