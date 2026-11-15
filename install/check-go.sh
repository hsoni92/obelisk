#!/usr/bin/env bash
# Verify Go 1.22+ is on PATH (used by Makefile dev targets).
set -euo pipefail

GO="${GO:-go}"

if ! command -v "$GO" >/dev/null 2>&1; then
  cat <<'EOF' >&2
[obelisk] Go is not on PATH. This repo needs Go 1.22 or newer.

Install (pick one):

  Official tarball: https://go.dev/dl/  (extract to /usr/local/go, add /usr/local/go/bin to PATH)

  Ubuntu snap:
    sudo snap install go --classic
    # or a specific track: sudo snap install go --channel=1.22/stable --classic

  Fedora: sudo dnf install golang

Then open a new shell and run:
  go version
  make install-air

If you use the official tarball, also add to ~/.profile or ~/.bashrc:
  export PATH=$PATH:/usr/local/go/bin
  export PATH=$PATH:$(go env GOPATH)/bin   # for air, after install-air
EOF
  exit 127
fi

ver="$("$GO" env GOVERSION 2>/dev/null || "$GO" version | awk '{print $3}')"
ver="${ver#go}"
need="1.22"
if [ -z "$ver" ] || [ "$(printf '%s\n' "$need" "$ver" | sort -V | head -n1)" != "$need" ]; then
  echo "[obelisk] found go${ver:-?}; need Go ${need}+" >&2
  exit 1
fi