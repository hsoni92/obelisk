#!/usr/bin/env bash
# Resolve the platform image tag for build + helm (sourced by bootstrap.sh).
set -euo pipefail

# Each bootstrap/upgrade must use a distinct tag so Helm rolls Deployments even
# though images are loaded locally into k3s (no registry pull).
obelisk_resolve_image_tag() {
  if [[ -n "${OBELISK_IMAGE_TAG:-}" ]]; then
    printf '%s' "$OBELISK_IMAGE_TAG"
    return
  fi
  local ref="${OBELISK_REF:-main}"
  ref="$(printf '%s' "$ref" | tr '/+' '--' | tr -cd '[:alnum:]._-' | cut -c1-48)"
  [[ -n "$ref" ]] || ref="main"
  local ts
  ts="$(date -u +%Y%m%dT%H%M%SZ)"
  printf '0.1.0-%s-%s' "$ref" "$ts"
}