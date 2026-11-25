# PR-only main, CI-built releases, and download-only `obelisk update`

* **Author:** hsoni92
* **Date:** 2026-11-25

## Overview

Today anyone with push access can commit to `main`, CI never builds a Docker image, and every `obelisk update` rebuilds both images (and the Go CLI) on the machine. That rebuild is the slow part.

This change has three parts:

1. **PR-only `main`.** Nothing is pushed to `main` directly. Every change is a pull request, and it can merge only when CI passes. The rule applies to the repo admin too.
2. **CI builds the release.** After a squash-merge, CI builds the controller and admin-UI images for `linux/amd64` and `linux/arm64`, plus one CLI binary per architecture. It pushes the images to GHCR (`ghcr.io/<owner>/...`) and then publishes a GitHub Release `build-N`. N is the number of commits on `main` at the merged commit. The Release is created last, so a half-finished build is never visible to machines.
3. **`obelisk update` downloads, never builds.** The command reads the latest release number and compares it with the number recorded on the machine. If the latest is higher, it downloads the source tree for that tag, installs the CLI binary for its architecture, and runs `helm upgrade` with image references for that build. k3s pulls the images. The machine runs no `docker build` and no `go build`, and needs neither Docker nor Go.

Target machines: any CPU architecture, both `local` and `vps` roles, all running k3s. The repo is public and will be open-sourced.

"No builds on the machine" does not mean "no source on the machine". The controller reads Helm charts from the host tree at `/opt/obelisk` (`deploy/helm/services/*`, and the Kong base file). The tree must come from the same release as the images.

## Analysis

### What the code does today

- [ci.yml](../../.github/workflows/ci.yml) runs on pushes to `main`/`master` and on PRs. It has Go, web, helm and gitleaks jobs. It builds and publishes no image.
- `install/install.sh` fetches a GitHub archive of a ref, runs `go build ./cmd/obelisk` on the machine (installing Go through snap if missing), then runs `bootstrap.sh`. `bootstrap.sh` builds both images with Docker and imports them into k3s with `docker save | k3s ctr images import`.
- The image tag is `0.1.0-<ref>-<timestamp>`. It is unique per run so Helm restarts the Deployments. Helm gets a local image name with no registry.
- `.install-meta.json` records `ref` and `image_tag`. The timestamp tag cannot be compared as a number.
- The controller pod mounts `/opt/obelisk` from the host and renders charts from it.
- `install.sh` refuses to run on non-x86_64 and refuses any profile other than `vps`. `Dockerfile.controller` downloads an amd64 `kubectl`, so an arm64 image would build and then fail at runtime.

### Blockers the Goldfish review found

**1. Source tree and images arrive by two paths, so they can disagree.** A crash between swapping the tree and finishing `helm upgrade` leaves a new tree running old images. Fix: fetch the tree at the same `build-N` tag. Stage it, pull the images first, swap the tree last, run `helm upgrade --atomic --wait`, and write the meta file last. Keep the previous tree so a failed update can be undone. A later option is to bake `deploy/helm` and the Kong file into the controller image and drop the host mount. That removes this class of problem.

**2. `Dockerfile.controller` cannot produce a working arm64 image, and its output is not reproducible.** It hardcodes the amd64 `kubectl`, pipes Helm's install script from `main`, installs an unpinned Tailscale, and runs `go mod tidy` inside the build. There is no `.dockerignore`. The admin UI builds on Node 22 while CI tests on Node 20. This must be fixed first. A post-merge arm64 failure means no release.

**3. The build number N is under-specified, and the current CI concurrency would break publishing.** N must be computed from the merged commit with full history (`fetch-depth: 0`), not from the branch head. `cancel-in-progress: true` in `ci.yml` would cancel an in-flight publish when the next PR merges. Publishing needs its own workflow with `cancel-in-progress: false`. GitHub keeps only one pending run per concurrency group, so a middle merge can be skipped. Numbers will have gaps, and that is normal. "Latest" must be the highest N compared as an integer, not the newest Release by creation time. Every publish step must be safe to re-run, and a published `build-N` is never overwritten.

**4. "Upgrade if the number differs" allows silent downgrades.** Controller state and files such as `obelisk.yaml` may have moved forward. Default rule: upgrade only when latest is greater than installed. Add `--to build-N` for deliberate pinning or rollback, with a warning, and `--check` that reports without changing anything.

**5. Existing machines cannot reach the new flow on their own.** They have timestamp tags, locally imported images, a meta file with no build number, and an old `install.sh` that builds on the host. Treat a missing build number as build 0. Plan a one-time bridge: the first release comes from a `main` that already holds the new scripts, and the old updater (default ref `main`) lands the new CLI and meta once. The old script is already loaded in memory when it replaces itself, so it must re-run the new one after extraction. Test an old install updating to the new tree in a VM.

**6. GHCR packages start private.** The first push creates a private package, and every machine's first pull would fail. This needs a one-time manual step: make each package public and link it to the repo. Add a CI gate that checks an anonymous pull of both platforms succeeds before the Release is published. Derive the registry owner from `github.repository_owner`, not a literal.

**7. The installers contradict the goal.** `install.sh` rejects non-x86_64, defaults to building images, installs Docker and Go, and runs only for the `vps` profile. `update.sh` and the bootstrap-only fallback hardcode `vps`. For this change: map `uname -m` to the right binary, default to no builds, download and verify the CLI for the machine's architecture, and read profile and role from the live config or the meta file. The CLI binary must exist before bootstrap, because `write_config` calls `obelisk init-config`.

**8. "No admin bypass" with one maintainer is not literally "no escape".** If CI itself breaks, the fix needs a passing CI. The repo owner can always edit or disable the ruleset in Settings. That is the real escape hatch. Write it down as a logged procedure: disable the ruleset, merge the fix, re-enable it. Use one aggregate required check (`ci-gate`, which depends on all other jobs) so renaming a job does not strand the ruleset. Add the new Docker jobs to the required list only after they have run once.

### Major findings

- **Finding the latest release.** The GitHub API allows 60 unauthenticated calls per hour per IP, and "latest" there means newest created, not highest N. Publish one `release.json` asset (build number, commit, image references, per-architecture CLI URL and sha256) and read `releases/latest/download/release.json`, which has no API quota. A 403 or network error must not be reported as "up to date".
- **Publish order.** Checks, per-architecture image push, manifest list, anonymous-pull test, CLI cross-builds and checksums, draft Release with all assets, publish. A published Release must imply pullable images. Tag images `build-N` and `sha-<short>`; never publish or consume `latest`.
- **Update order and recovery.** Resolve the release, stage and verify checksums, pre-pull images, swap the tree, `helm upgrade --atomic --wait`, write meta atomically, then rename the CLI into place. Today meta is written after bootstrap, so a rollout failure leaves the cluster on N and the meta on N-1. `helm --atomic` reverts only the chart, so keep the previous tree and CLI. Add a lock file against concurrent updates. Never overwrite a running binary in place.
- **Supply chain.** Machines run the downloaded binary as root, so the maintainer's GitHub account and the workflow become the trust root for the whole fleet. Publish `SHA256SUMS` and verify before replacing anything. Pin GitHub Actions by commit SHA before any job gets write permission. Use `pull_request`, never `pull_request_target`, and give `packages: write` and `contents: write` only to the publish job. Signing and provenance can be deferred, but record that decision. Run gitleaks over the full history before the repo goes public.
- **arm64 build time.** QEMU emulation of the Node and Go stages is slow. Native `ubuntu-24.04-arm` runners are free for public repos. Build each architecture natively, push by digest, then merge a manifest list. Cross-compile the CLI with `CGO_ENABLED=0`. Open question: whether PR checks also build arm64. Without it, an arm64-only break shows up after merge as a missing release.
- **Third-party images.** `versions.lock` pins `rustfs`, `searxng` and `unimeter` as `:latest`, so a machine's behavior can change with no new build. Docker Hub anonymous pull limits apply on shared VPS IPs. Some service images may lack arm64. Pin versions, check arm64 manifests, and document gaps.
- **Version and meta contract.** The words `ref`, `tag`, `image_tag`, `build` and `latest` currently overlap. Proposed meaning: N is the build number, the release tag and image tag are both `build-N`, and `ref` leaves the user-facing update. Meta schema v2: `schema`, `build`, `repo`, `tag`, `installed_at`, `profile`, `arch`, `source` (`release` or `dev`), `images`. Embed N in the CLI and the controller so `obelisk version` can show the embedded, recorded, tree and running-image builds and flag any mismatch. Have the Go CLI write the meta file, atomically, instead of the shell heredoc. There are no Go tests under `cmd/`, so add tests for version comparison, asset selection, checksum failure, legacy meta and `--check`.
- **`cleanup.sh` and uninstall.** Image-name patterns assume local names. Cleanup removes every non-current image, so a rollback needs a re-pull unless it keeps N and N-1. `uninstall.sh` and the uninstaller design reference old tag patterns.
- **Developer flow.** Contributors, `k3d-dev.sh`, the Makefile and the VM integration test still need local builds. Keep one explicit dev mode, recorded as `source: dev`, which `update` will not overwrite without `--force`. A fresh repo has no Release until the first merge, so the first release needs a one-time bootstrap run.
- **Failure visibility.** A failed post-merge build leaves `main` ahead of the last release with no signal. Add a failure notification, a manual re-run for the same commit, and `actionlint` so the release path is not first tested by a live merge. On the machine, log each stage to `/var/log/obelisk-install.log`, and on a rollout timeout print pod status, events and previous logs. The 300 s default may be too short for a first pull of about 1 GB.
- **Workflow shape.** `ci.yml` also runs on `push` to `main`, so a merge would run checks twice. Make `ci.yml` PR-only and reusable, and have `release.yml` call it before building. Required check names must match job names exactly. ESLint is `continue-on-error`, so "CI passes" has a hole there; keep it non-required or decide deliberately. Add shellcheck for `install/*.sh`.
- **Ruleset as code.** Keep `.github/rulesets/*.json` and a `gh api` apply script for review and re-creation. Settings: squash merges only, zero required approvals, required aggregate check, no force-push or deletion, no bypass actors, and a tag rule making `build-*` immutable. These files document intent; they do not enforce it. Do not add a drift job that needs an admin token, because that token would itself be a bypass.

### Scope risk

The review flagged that "everything" bundles separable work. Only image publishing plus a small `update` change serves the stated pain (slow builds). Governance, multi-arch, and CLI binaries can ship in phases. You asked for no Go on machines, so CLI binaries stay in scope. Suggested phases:

1. Governance: `ci-gate`, PR image builds, ruleset. No effect on machines.
2. Release workflow that publishes images and the tag, while machines still build the CLI. This proves the pipeline over several merges.
3. Update client with meta v2 and the shared release-resolution code. This is the switchover.
4. First install from released images.
5. Remove the on-host build and the old tag format.
6. Later, if measured need: digest pinning, signing, offline install.

### Docs that conflict with this change

- `curl_vps_installer.md` and its concepts brief list GHCR and registry-first as non-goals and rejected alternatives, require Docker and Go on the machine, default to building images, and state amd64 only.
- `obelisk.md` (around lines 270 and 468) describes on-host image builds.
- `precommit_ci_guardrails_concepts_brief.md` already proposes branch protection; this change takes that over.
- README and `docs/runbooks/install.md` document `OBELISK_BUILD_IMAGES=true`, `--ref v0.2.0` and `--skip-images`.
- `obelisk_uninstaller.md` (line ~97) has an old tag pattern.
- `multi_machine_blueprint.md` concerns a different repo's `main` and does not conflict.

### Ideal Final Result

A merged PR is the only way code changes. From it, with no manual step, CI produces one immutable, numbered release: per-architecture images, per-architecture CLI binaries with checksums, and one manifest. The tag is the exact source tree the images were built from, so tree, CLI, chart and images agree. On any machine, one command reads the manifest, verifies checksums, pulls images before changing anything, upgrades atomically, and records the result. It needs no compiler and no Docker, shows which build is running, and can go back to the previous one. A failed build leaves machines untouched and notifies the maintainer. A failed update leaves the machine on its previous working build.

### Contradictions this design accepts

1. **No escape hatch vs. repairing a broken pipeline.** The ruleset edit in Settings is the hatch; it is a logged procedure.
2. **Merge only when CI passes vs. a post-merge build that can still fail.** PR CI predicts the release but does not guarantee it. Gaps in N are expected.
3. **"Release last, tag equals source" vs. "N = commit count".** Both hold only if N and the tag come from the merged commit and publishing is serialized. A skipped middle merge is still possible.
4. **No builds on the machine vs. charts read from the host tree.** Pin the tree to the same build, or move charts into the image.
5. **Upgrade-if-different vs. forward-only safety.** Newer-only by default, `--to` for explicit moves.
6. **Anonymous public pulls vs. GHCR private by default, and a fleet that trusts one account and workflow.** One manual visibility step; the trust root is stated openly.
7. **Any architecture vs. amd64 hardcoding in the Dockerfile and `install.sh`, and possible service images without arm64.**
8. **Local and VPS vs. installer and updater that only support `vps`.**
9. **Reproducible release vs. unpinned Helm, Tailscale, k3s installer and `:latest` service images.**
10. **Understandable build number vs. gaps and no compatibility signal.**
11. **Everything from Releases vs. the curl installer served from raw `main`.** Serve it as a Release asset instead.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * PR-only `main` enforced by a repo ruleset with no bypass actors, with the ruleset kept as reviewed files in the repo.
  * PR checks: existing checks plus no-push builds of both images, behind one aggregate required check.
  * Post-merge workflow that builds multi-arch images and per-arch CLI binaries, pushes to GHCR, and publishes `build-N` last with a `release.json` manifest and checksums.
  * `obelisk update` that compares build numbers, upgrades only forward by default, downloads and verifies, and never builds. Includes `--check` and `--to build-N`.
  * First install (`install.sh`/bootstrap) from released images and binaries for amd64 and arm64, for `local` and `vps` roles.
  * Fix `Dockerfile.controller` and `Dockerfile.admin-ui` so arm64 works and builds are pinned.
  * Meta schema v2, build number embedded in CLI and controller, migration path for existing machines.
  * Update the conflicting design docs, runbooks and README, and add a release runbook and a PR-only note in CONTRIBUTING.md.
* **Out-of-Scope Non-Goals:**
  * Offline or air-gapped installs (current installs already need the network).
  * Image signing, provenance and SBOM, in the first version (recorded as deferred).
  * Required PR approvals (solo maintainer).
  * Merge queue.
  * A bot or script that commits to `main` (for example to bump a version file).
  * Auto-update on a timer.

## Alternatives Weighed

* **Smallest possible change (images only):** CI pushes images; machines keep the archive fetch and a local `go build` of the CLI. Rejected by author: the requirement is no builds on machines. Kept as phase 2 to prove the pipeline.
* **`github.run_number` as the build number:** simpler, but it cannot be derived from the repo and resets if the workflow is renamed. Not chosen; commit count on the merged commit is used.
* **GitHub Releases API `/releases/latest` for discovery:** rate-limited and ordered by creation time. Replaced by a `release.json` asset fetched by redirect.
* **Moving `latest` tag or a version file in git:** a version file needs a bot commit to `main`, which breaks PR-only. A moving tag is mutable and cannot be told apart from a failed build. Not chosen.
* **Digest-pinned image references in Helm:** stronger for supply chain, but `cleanup.sh` parses tags. Open: choose tag or digest and fix cleanup to match.
* **Charts inside the controller image, or published as an OCI artifact:** removes the tree-versus-image skew. Not chosen for the first version because it changes how the controller loads charts; same-tag tree with staged swap is used. Revisit later.
* **Separate base image for apt, Helm, kubectl and Tailscale layers:** faster per-merge builds, but a second image to maintain. Optional later.
* **QEMU arm64 builds vs. native arm runners:** native runners are free for public repos and faster; QEMU is the fallback.
* **Strict "branch up to date" before merge:** safer, but adds rebase work for one maintainer. Open.
* **Full source archive vs. a slim deterministic deploy asset (`obelisk-deploy-N.tar.gz`):** the codeload archive is simple but not byte-stable. Open.

## Key Decisions & Staging for Next Steps

Decided during ideation:

* Build number is the commit count at the merged commit; release and image tag are `build-N`; Release is created last.
* Update direction is forward-only by default; `--to build-N` for explicit moves.
* No failed-build visibility for machines: no Release means nothing new.
* Registry is GHCR, public, owner derived from the repo.
* Multi-arch (`amd64`, `arm64`) for images and CLI binaries.
* No direct-push escape hatch; the ruleset-edit recovery is documented instead.
* Scope is the whole path (CI, release, update, first install), delivered in phases.

Open for `/egm-draft`:

* Tag or digest in Helm values.
* Slim deploy asset vs. full source archive.
* Whether PR checks build arm64.
* Strict up-to-date merge mode.
* Whether to move charts into the controller image now or later.
* How existing machines bridge to the new updater (one-time curl installer vs. re-exec).
* What replaces the current direct-to-`main` commit script, and how the first PR is bootstrapped.
* Which design docs to amend and how to mark them superseded.

## Implementation Notes (decisions and deviations)

Implemented on branch `release-pipeline`. Where the code differs from the brief above:

* **CI platform: GitHub Actions on a public repo.** Buildkite was rejected by author: no arm64 agent, a stored publish token, no commit status posted to GitHub, and PRs would run on a self-hosted laptop.
* **Registry: GHCR**, owner derived from `github.repository_owner`. Tags, not digests, in Helm values (immutable `build-N`).
* **No `release.json`.** The build number is the tag (`build-N`), found through the `github.com/<repo>/releases/latest` redirect (no API quota). Per-architecture CLI binaries are verified against `SHA256SUMS` in the same Release. This removes a second place the version could be written.
* **Source tree: full GitHub archive at the tag**, not a slim deploy asset. The controller still reads charts from `/opt/obelisk`.
* **PR checks build both architectures** (native arm64 runners are free for public repos). Strict up-to-date merge mode is off.
* **Upgrade order:** resolve release, check anonymous image pull, download and verify CLI, pre-pull images, back up the tree, extract the tree, hand over to the new `install.sh`, `helm upgrade --atomic --wait`, write meta last. A failed update restores the previous tree.
* **Existing machines:** counted as build 0. Their first update runs the old on-host build path once; later updates are download-only.
* **Not done:** moving charts into the controller image, image signing, offline installs, pinning Tailscale/k3s/third-party images, a `local`-profile first install through `install.sh` (upgrades keep the profile from `.install-meta.json`).
* **Files:** `.github/workflows/{ci,release}.yml`, `.github/rulesets/*.json`, `scripts/apply-ruleset.sh`, `Dockerfile.*`, `.dockerignore`, `install/{install,bootstrap,update,cleanup}.sh`, `cmd/obelisk/{update_cmd,release,version_cmd}.go`, `tests/install/release-flow.sh`, `docs/runbooks/release.md`.
