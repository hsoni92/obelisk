# shellcheck shell=bash
# Source repo-root .env if present (secrets; gitignored).
load_dotenv() {
  local root="${1:?repo root}"
  local f="$root/.env"
  if [ -f "$f" ]; then
    set -a
    # shellcheck disable=SC1090
    source "$f"
    set +a
  fi
}