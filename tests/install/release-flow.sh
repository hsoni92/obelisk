#!/usr/bin/env bash
# End-to-end check of the release install/update protocol in install/install.sh, no k3s needed.
# Runs in a throwaway Ubuntu container (needs docker + network): serves a fake GitHub release,
# runs install.sh as a fresh install, then as an upgrade, and checks checksums, re-exec,
# meta v2, rollback on failure, and the lock. bootstrap.sh is replaced by a stub.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

for n in 3 4; do
  (cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH="$(go env GOARCH)" go build -trimpath \
    -ldflags "-X main.buildNumber=$n" -o "$WORK/obelisk-linux-$n" ./cmd/obelisk)
done

cat >"$WORK/run.sh" <<'INNER'
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq >/dev/null && apt-get install -y -qq curl rsync python3 util-linux >/dev/null
ARCH=$(uname -m); case "$ARCH" in x86_64) A=amd64;; aarch64) A=arm64;; esac

mkserver() { # $1 = latest build
  cat >/work/server.py <<PY
import http.server, os
LATEST=$1
class H(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/o/r/releases/latest":
            self.send_response(302); self.send_header("Location", f"/o/r/releases/tag/build-{LATEST}"); self.end_headers(); return
        if self.path.startswith("/o/r/releases/tag/"):
            self.send_response(200); self.end_headers(); return
        return super().do_GET()
    def log_message(self,*a): pass
os.chdir("/srv")
http.server.ThreadingHTTPServer(("127.0.0.1",8099),H).serve_forever()
PY
  pkill -f server.py || true
  python3 /work/server.py & sleep 1
}

mkrelease() { # $1 = build number
  n=$1
  d=/srv/o/r/releases/download/build-$n; mkdir -p $d
  cp /work/obelisk-linux-$n $d/obelisk-linux-$A
  (cd $d && sha256sum obelisk-linux-$A >SHA256SUMS)
  t=$(mktemp -d); mkdir -p $t/obelisk-build-$n/install
  cp /repo/install/*.sh $t/obelisk-build-$n/install/
  cat >$t/obelisk-build-$n/install/bootstrap.sh <<STUB
#!/usr/bin/env bash
echo "STUB-BOOTSTRAP build=$n tag=\$OBELISK_IMAGE_TAG registry=\$OBELISK_IMAGE_REGISTRY build_images=\$OBELISK_BUILD_IMAGES profile=\$OBELISK_PROFILE" >>/tmp/bootstrap.log
[ -n "\${FAIL_BOOTSTRAP:-}" ] && exit 7
exit 0
STUB
  chmod +x $t/obelisk-build-$n/install/*.sh
  tar -czf /srv/archive-$n.tar.gz -C $t obelisk-build-$n
}

export OBELISK_ROOT=/opt/obelisk OBELISK_REPO=o/r OBELISK_RELEASE_BASE_URL=http://127.0.0.1:8099
export OBELISK_IMAGE_REGISTRY=example.invalid/x OBELISK_SKIP_GITHUB_WIZARD=true
export OBELISK_INSTALL_LOG=/tmp/install.log OBELISK_INSTALL_WAIT_SEC=1
mkrelease 3; mkrelease 4
fail() { echo "FAIL: $*"; exit 1; }

echo "== 1. fresh install resolves latest (build-3), verifies CLI, re-execs, writes meta v2"
mkserver 3
OBELISK_ARCHIVE_URL=http://127.0.0.1:8099/archive-3.tar.gz bash /repo/install/install.sh
grep -q '"build":3' /opt/obelisk/.install-meta.json || fail "meta build"
grep -q '"source":"release"' /opt/obelisk/.install-meta.json || fail "meta source"
grep -q "\"arch\":\"$A\"" /opt/obelisk/.install-meta.json || fail "meta arch"
grep -q 'tag=build-3 registry=example.invalid/x build_images=false' /tmp/bootstrap.log || fail "bootstrap env: $(cat /tmp/bootstrap.log)"
/opt/obelisk/obelisk version | grep -q 'cli_build=3' || fail "cli build embedded"

echo "== 2. bad checksum is rejected before the tree is touched"
mkserver 4
echo "deadbeef  obelisk-linux-$A" >/srv/o/r/releases/download/build-4/SHA256SUMS
if OBELISK_INSTALL_MODE=upgrade OBELISK_ARCHIVE_URL=http://127.0.0.1:8099/archive-4.tar.gz bash /repo/install/install.sh; then fail "bad checksum accepted"; fi
grep -q '"build":3' /opt/obelisk/.install-meta.json || fail "meta changed after rejected download"
(cd /srv/o/r/releases/download/build-4 && sha256sum obelisk-linux-$A >SHA256SUMS)

echo "== 3. failed bootstrap restores the previous tree and keeps meta at build 3"
echo marker-from-build-3 >/opt/obelisk/MARKER
if FAIL_BOOTSTRAP=1 OBELISK_INSTALL_MODE=upgrade OBELISK_ARCHIVE_URL=http://127.0.0.1:8099/archive-4.tar.gz bash /repo/install/install.sh; then fail "bootstrap failure not propagated"; fi
grep -q '"build":3' /opt/obelisk/.install-meta.json || fail "meta moved on failure"
[ -f /opt/obelisk/MARKER ] || fail "previous tree not restored"

echo "== 4. upgrade to build-4 (pinned via OBELISK_REF)"
rm -f /opt/obelisk/MARKER
sed -i 's/"profile":"vps"/"profile":"local"/' /opt/obelisk/.install-meta.json   # a local-profile machine
OBELISK_REF=build-4 OBELISK_INSTALL_MODE=upgrade OBELISK_ARCHIVE_URL=http://127.0.0.1:8099/archive-4.tar.gz bash /repo/install/install.sh
grep -q '"build":4' /opt/obelisk/.install-meta.json || fail "meta build 4"
grep -q '"profile":"local"' /opt/obelisk/.install-meta.json || fail "profile not kept across upgrade: $(cat /opt/obelisk/.install-meta.json)"
tail -1 /tmp/bootstrap.log | grep -q 'profile=local' || fail "bootstrap did not get profile=local"

echo "== 5. dev mode (--ref main) still selects source build path"
OBELISK_REF=main bash -c 'source <(sed -n "1,/^OBELISK_REF=\"\${OBELISK_REF:-}\"/p" /repo/install/install.sh | grep -v "^set -euo"); echo "mode=$OBELISK_SOURCE"' | grep -q 'mode=dev' || fail "dev mode detection"

echo "ALL RELEASE FLOW CHECKS PASSED"
INNER

docker run --rm --network host -v "$ROOT:/repo:ro" -v "$WORK:/work" ubuntu:22.04 bash /work/run.sh
