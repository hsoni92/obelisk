#!/usr/bin/env bash
# Build controller + admin UI images and import into the local k3s containerd.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAG="${OBELISK_IMAGE_TAG:-0.1.0}"

log() { printf '[obelisk] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

docker_cmd() {
  if docker info >/dev/null 2>&1; then
    docker "$@"
  elif sudo docker info >/dev/null 2>&1; then
    sudo docker "$@"
  else
    die "docker required (install docker.io and ensure the daemon is running)"
  fi
}

k3s_import() {
  local ref="$1"
  log "importing $ref into k3s"
  docker_cmd save "$ref" | sudo k3s ctr images import -
}

command -v k3s >/dev/null || die "k3s must be installed before importing images"

log "building obelisk-controller:${TAG}"
docker_cmd build -f "$ROOT/Dockerfile.controller" -t "obelisk-controller:${TAG}" "$ROOT"

log "building obelisk-admin-ui:${TAG}"
docker_cmd build -f "$ROOT/Dockerfile.admin-ui" -t "obelisk-admin-ui:${TAG}" "$ROOT"

k3s_import "obelisk-controller:${TAG}"
k3s_import "obelisk-admin-ui:${TAG}"

log "images ready: obelisk-controller:${TAG}, obelisk-admin-ui:${TAG}"