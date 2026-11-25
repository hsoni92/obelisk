#!/usr/bin/env bash
# Create or update the repo rulesets in .github/rulesets/*.json using the GitHub CLI.
# Needs `gh` logged in as a repo admin. The JSON files are the reviewed source of truth;
# this script does not run in CI (that would need an admin token, which is itself a bypass).
#
#   scripts/apply-ruleset.sh [owner/repo]     # default: repo of the current directory
#
# Apply the ruleset only after `ci-gate` has run at least once on a PR, or no PR can merge.
set -euo pipefail

REPO="${1:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
DIR="$(cd "$(dirname "$0")/../.github/rulesets" && pwd)"

for f in "$DIR"/*.json; do
  name="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["name"])' "$f")"
  id="$(gh api "repos/$REPO/rulesets" --jq ".[] | select(.name == \"$name\") | .id" || true)"
  if [[ -n "$id" ]]; then
    echo "updating ruleset '$name' (id $id) on $REPO"
    gh api -X PUT "repos/$REPO/rulesets/$id" --input "$f" >/dev/null
  else
    echo "creating ruleset '$name' on $REPO"
    gh api -X POST "repos/$REPO/rulesets" --input "$f" >/dev/null
  fi
done
echo "done. Review at https://github.com/$REPO/settings/rules"
