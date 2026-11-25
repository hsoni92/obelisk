#!/usr/bin/env bash
# Checks for the fresh-install profile menu and the closed-by-default admin allowlist.
# Runs on the host, needs bash, python3 (for a pseudo-terminal) and go. No root, no k3s.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
fail() { echo "FAIL: $*"; exit 1; }

(cd "$ROOT" && go build -o "$WORK/obelisk" ./cmd/obelisk)

# --- run choose_profile on a real pseudo-terminal, feeding it keystrokes -------------------
menu() { # $1 = keystrokes (python escape string); prints the profile the menu chose
  python3 - "$ROOT/install/install.sh" "$1" <<'PY'
import os, pty, sys, select, time
script, keys = sys.argv[1], sys.argv[2].encode().decode('unicode_escape')
cmd = ['bash', '-c', f'OBELISK_INSTALL_SOURCE_ONLY=1 source {script}; choose_profile; echo CHOSEN=$OBELISK_PROFILE']
pid, fd = pty.fork()
if pid == 0:
    os.environ['TERM'] = 'xterm'
    os.execvp(cmd[0], cmd)
out = b''
def drain(t=0.4):
    global out
    end = time.time() + t
    while time.time() < end:
        r, _, _ = select.select([fd], [], [], 0.05)
        if r:
            try: out += os.read(fd, 4096)
            except OSError: return
drain(0.8)
for ch in keys:
    os.write(fd, ch.encode()); drain(0.2)
drain(1.0)
text = out.decode(errors='replace')
for line in text.splitlines():
    if 'CHOSEN=' in line:
        print(line.split('CHOSEN=')[1].strip()); break
else:
    print('NONE'); sys.stderr.write(text)
PY
}

echo "== 1. radio menu"
[[ "$(menu '\r')" == "vps" ]]            || fail "Enter alone must keep the default (vps)"
[[ "$(menu '\x1b[B\r')" == "local" ]]    || fail "Down then Enter must choose local"
[[ "$(menu '\x1b[B\x1b[A\r')" == "vps" ]] || fail "Down then Up then Enter must go back to vps"
[[ "$(menu '2\r')" == "local" ]]         || fail "key 2 must choose local"
[[ "$(menu 'j \r')" == "local" ]]        || fail "j then Space then Enter must choose local"

echo "== 2. no terminal: falls back to vps with a note, and does not hang"
out="$(OBELISK_INSTALL_SOURCE_ONLY=1 OBELISK_ROOT="$WORK/none" OBELISK_LIB_DIR="$WORK/lib" OBELISK_ETC_DIR="$WORK/etc" \
  bash -c "source '$ROOT/install/install.sh'; unset OBELISK_PROFILE; resolve_profile; echo PROFILE=\$OBELISK_PROFILE" </dev/null 2>&1 | cat)"
echo "$out" | grep -q 'PROFILE=vps' || fail "non-interactive default: $out"
echo "$out" | grep -q 'no terminal' || fail "note missing: $out"

echo "== 3. profile precedence: env > recorded meta > existing config"
mkdir -p "$WORK/root3" "$WORK/lib3" "$WORK/etc3"
echo '{"schema":2,"profile":"local"}' >"$WORK/root3/.install-meta.json"
printf 'deployment:\n  profile: vps\nresources:\n  profile: default\n' >"$WORK/lib3/obelisk.yaml"
run_resolve() {
  OBELISK_INSTALL_SOURCE_ONLY=1 OBELISK_ROOT="$WORK/root3" OBELISK_LIB_DIR="$WORK/lib3" OBELISK_ETC_DIR="$WORK/etc3" \
    bash -c "source '$ROOT/install/install.sh'; $1; resolve_profile >/dev/null; echo \$OBELISK_PROFILE" </dev/null
}
[[ "$(run_resolve 'export OBELISK_PROFILE=vps')" == "vps" ]]   || fail "env must win"
[[ "$(run_resolve 'unset OBELISK_PROFILE')" == "local" ]]      || fail "meta must beat config"
rm "$WORK/root3/.install-meta.json"
[[ "$(run_resolve 'unset OBELISK_PROFILE')" == "vps" ]]        || fail "config profile (vps), not 'resources.profile: default'"
printf 'deployment:\n  profile: local\n' >"$WORK/lib3/obelisk.yaml"
[[ "$(run_resolve 'unset OBELISK_PROFILE')" == "local" ]]      || fail "config profile local"
if run_resolve 'export OBELISK_PROFILE=bogus' >/dev/null 2>&1; then fail "invalid profile accepted"; fi

echo "== 4. fresh install closes the admin allowlist; upgrades and re-runs do not touch it"
LIB="$WORK/lib4"; mkdir -p "$LIB" "$WORK/root4"; cp "$WORK/obelisk" "$WORK/root4/obelisk"
boot() { # $1 = function to call
  OBELISK_BOOTSTRAP_SOURCE_ONLY=1 OBELISK_ROOT="$WORK/root4" OBELISK_LIB_DIR="$LIB" CONFIG_PATH="$LIB/obelisk.yaml" \
    bash -c "source '$ROOT/install/bootstrap.sh'; CONFIG_PATH='$LIB/obelisk.yaml'; $1"
}
boot init_admin_allowlist >/dev/null
f="$LIB/credentials/admin-github-allowlist.json"
[[ -f "$f" ]] || fail "allowlist file not created at $f"
grep -q '"logins":\[\]' "$f" || fail "allowlist must start empty: $(cat "$f")"
[[ "$(stat -c %a "$f")" == "600" ]] || fail "allowlist must be mode 600"
OBELISK_CREDENTIALS_DIR="$LIB/credentials" "$WORK/obelisk" admin allow list --config "$LIB/obelisk.yaml" | grep -q 'enforced, empty' || fail "CLI must report an enforced, empty list"
OBELISK_CREDENTIALS_DIR="$LIB/credentials" "$WORK/obelisk" admin allow add --login alice --config "$LIB/obelisk.yaml" >/dev/null
boot init_admin_allowlist >/dev/null
grep -q '"alice"' "$f" || fail "re-running init must keep existing users: $(cat "$f")"

echo "== 5. machines without an allowlist get a warning, machines with one do not"
rm -rf "$LIB/credentials"
boot warn_open_admin_allowlist 2>&1 | grep -q 'ANY GitHub user' || fail "open install must warn"
boot init_admin_allowlist >/dev/null
if boot warn_open_admin_allowlist 2>&1 | grep -q 'ANY GitHub user'; then fail "closed install must not warn"; fi

echo "ALL PROFILE AND ALLOWLIST CHECKS PASSED"
