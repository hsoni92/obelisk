# Release runbook

How a merged pull request becomes something machines can install, and how to recover when it does not.

## How it works

1. Nobody pushes to `main`. A PR can merge only when the `ci-gate` check passes, using **squash** merge.
2. On merge, [release.yml](../../.github/workflows/release.yml) re-runs the CI checks, then:
   1. computes **N** = number of commits reachable from the merged commit (`build-N` is the tag; gaps are normal),
   2. builds `obelisk-controller` and `obelisk-admin-ui` for `linux/amd64` and `linux/arm64` on native runners and pushes them to `ghcr.io/<owner>/` as `build-N` and `sha-<short>`,
   3. checks both images can be pulled **without logging in**,
   4. cross-compiles the `obelisk` CLI for amd64 and arm64,
   5. publishes GitHub Release `build-N` **last**, with the two CLI binaries, `install.sh` and `SHA256SUMS`.
3. Machines run `sudo obelisk update`. It reads the latest Release, compares N with the installed build, and upgrades only if the latest is higher. Machines never build.

If any step before 2.5 fails, no Release exists, so machines see nothing new. The workflow opens an issue named `Release failed for <sha>`.

## One-time setup

1. **Merge settings.** Settings → General → Pull Requests: allow **squash merging only**; enable "Automatically delete head branches".
2. **Land the workflows first.** Open a PR with these files; let `ci-gate` run once. The ruleset can only require a check that has run before.
3. **Apply the rulesets.** With `gh` logged in as a repo admin: `scripts/apply-ruleset.sh`. This requires a PR and `ci-gate` for `main`, squash only, no force-push or deletion, no bypass actors, and makes `build-*` tags immutable. Review the result under Settings → Rules.
4. **Actions policy.** Settings → Actions → General: "Require approval for all outside collaborators". Workflow permissions can stay read-only (jobs request what they need).
5. **First release and GHCR visibility.** New GHCR packages start **private**. After the first merge the `manifest` job fails at "Verify anonymous pull". Then, for `obelisk-controller` and `obelisk-admin-ui`: GitHub profile → Packages → package → Package settings → Change visibility → **Public**. Confirm the repo shows under "Manage Actions access" with write role. Re-run the failed workflow (Actions → Release → Run workflow on `main`). All steps are safe to repeat.

## Recovering

- **Release workflow failed.** Fix the cause in a PR (or, for a transient error, re-run: Actions → Release → Run workflow, branch `main`). The numbering does not change for a given commit.
- **CI itself is broken, so no PR can merge.** The repo owner can edit rulesets in Settings → Rules; there is no other bypass. Procedure: set the `main` ruleset to Disabled, merge the fix PR, set it back to Active, and note it in the PR description. Do not push to `main`.
- **A bad build shipped.** Do not delete the Release. Merge a fix (higher N). Machines that must go back now: `sudo obelisk update --to build-<good> --force` (downgrades can fail if controller state moved forward).
- **Machines see "cannot determine the release" (exit 11).** The machine cannot reach github.com, or GitHub returned an error. It says nothing about being up to date.

## What a release contains

| Asset | Used by |
| --- | --- |
| `obelisk-linux-amd64`, `obelisk-linux-arm64` | `install.sh` / `obelisk update` (checksum checked against `SHA256SUMS`) |
| `install.sh` | `curl -fsSL https://github.com/<owner>/obelisk/releases/latest/download/install.sh \| sudo bash` |
| `SHA256SUMS` | integrity check for the above |
| Source tarball at tag `build-N` (GitHub-generated) | `/opt/obelisk` tree: Helm charts, install scripts (the controller reads charts from here) |
| `ghcr.io/<owner>/obelisk-{controller,admin-ui}:build-N` | pulled by k3s |

## Known limits

- Images pin Helm and kubectl with checksums, but Tailscale (apt), the k3s installer, and third-party service images (`install/versions.lock`, some `:latest`) are not pinned. A release is only as repeatable as its least-pinned input.
- A machine upgrading runs the install script of the build it is on for the first steps (fetch, verify, pull), then hands over to the new release's `install.sh`. Keep those first steps compatible across builds.
- Release assets are not signed (checksums only). Trust root: this GitHub repo and its Actions workflow.
- Needs github.com and ghcr.io. Offline installs are not supported.
