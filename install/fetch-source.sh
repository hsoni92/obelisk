#!/usr/bin/env bash
# GitHub source archive helpers (sourced by install.sh or tests).
set -euo pipefail

obelisk_default_ref() {
  echo "${OBELISK_REF:-main}"
}

obelisk_github_archive_url() {
  local repo="${1:-hsoni92/obelisk}"
  local ref="${2:-main}"
  echo "https://codeload.github.com/${repo}/tar.gz/${ref}"
}

# Extract GitHub tarball into OBELISK_ROOT (single top-level directory).
obelisk_extract_archive() {
  local archive="$1"
  local dest="$2"
  local tmp
  tmp="$(mktemp -d)"

  tar -xzf "$archive" -C "$tmp"
  local top
  top="$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -1)"
  [[ -n "$top" && -d "$top" ]] || {
    echo "obelisk: archive has no top-level directory" >&2
    return 1
  }

  if [[ "${OBELISK_INSTALL_UPGRADE:-false}" == "true" ]]; then
    mkdir -p "$dest"
    rsync -a --delete \
      --exclude node_modules --exclude .next --exclude bin --exclude dev \
      "$top/" "$dest/"
  else
    rm -rf "$dest"
    mkdir -p "$dest"
    rsync -a \
      --exclude node_modules --exclude .next --exclude bin --exclude dev \
      "$top/" "$dest/"
  fi
  rm -rf "$tmp"
}