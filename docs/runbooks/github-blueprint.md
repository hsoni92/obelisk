# GitHub encrypted blueprint

On a **new VPS**, install the platform first with the curl installer in [install.md](./install.md), then use this runbook.

Operators sign in with GitHub. The appliance keeps an encrypted copy of desired state in a private repository and uses S3/R2 only for `data/`.

## OAuth app

Register an OAuth App at GitHub → **Settings → Developer settings → OAuth Apps**. The admin UI redirects to `{origin}/login/callback`; redirect URIs must match **exactly** (no wildcards).

| GitHub field | Value |
|--------------|--------|
| Application name | `obelisk` (or a name operators trust) |
| Homepage URL | Admin UI base URL, e.g. `https://<admin-host>` or `http://localhost:3080` for dev |
| Application description | Optional (shown on consent) |
| **Redirect URI** | `https://<admin-host>/login/callback` — add `http://localhost:3080/login/callback` for local dev (up to 10 URIs) |
| Allow wildcard matching | **Off** |
| Enable Device Flow | **On** (headless VPS / “Use a device code”) |
| Expire user access tokens | **On** (Obelisk stores refresh tokens and renews access in the background) |

Authorization requests scope **`repo`** (hard-coded in the controller) for private blueprint create/read/write.

On the controller (local dev: copy `.env.example` → `.env` at repo root; the controller loads it when `OBELISK_ROOT` is set):

```bash
export OBELISK_GITHUB_CLIENT_ID=...
export OBELISK_GITHUB_CLIENT_SECRET=...
export OBELISK_GITHUB_REDIRECT_URI=http://localhost:3080/login/callback
```

Or write `github-oauth.client.json` (mode 0600) in the credentials directory:

```json
{"client_id":"...","client_secret":"...","redirect_uri":"http://localhost:3080/login/callback"}
```

In cluster, create Secret `obelisk-github-oauth` with keys `client_id`, `client_secret`, and optional `redirect_uri`. The controller stores tokens in Secret `obelisk-blueprint-github` and in a mode-0600 file under the credentials directory. Tokens are never written to `obelisk.yaml`.

## Fresh appliance

1. Open the admin UI. **Sign in with GitHub** (or **Use a device code** on a VPS).
2. Set a passphrase. The browser stores it in local storage so the console can unlock after reboot. The appliance does not write the passphrase to disk. There is no recovery if it is lost.
3. Confirm or create the private repo (default `obelisk-appliance`).
4. The controller pushes generation 1: `manifest.json`, `obelisk.public.yaml`, and `secrets.enc`.

Later saves sync about 5 seconds after the controller is unlocked. If GitHub is down, the local save still succeeds and the push retries.

**Sync failed or “import required” with generation 0:** That import warning is wrong until the first successful push is recorded on disk (`generation` 0, no `last_commit_sha`). Unlock (or re-enter your passphrase), restart `make dev` on a current build, then **Push now** — the appliance overwrites stray commits on GitHub from failed attempts. For `github http 404`, ensure the controller was rebuilt after the `/git/refs` fix.

## After reboot

OAuth tokens remain. The in-memory key does not. The UI shows **push paused** until unlock. If this browser still has the passphrase, unlock runs automatically.

**Forget passphrase on this device** (Platform → Blueprint) clears local storage and locks the controller.

## Import and a different machine id

If the repo already has a bundle, sign-in leads to **Restore blueprint** instead of a new repo. When the host’s `machine.id` differs from the bundle, choose:

- **Adopt blueprint id** — keep that S3 bucket and restore `data/`.
- **Keep this host’s id** — apply settings only. Data from the old bucket is not restored.

If GitHub’s generation is ahead of this appliance, push is refused until you import.

## Chained restore

Disaster Recovery → **Run restore chain**, or:

```bash
obelisk restore --chain --passphrase "$OBELISK_BLUEPRINT_PASSPHRASE" --adopt-machine-id
```

Phases: `pull_blueprint` → `unlock` → `apply_control` → `restore_data` → `reconcile_services` → Funnel forced off. The UI shows settings restored and data restored separately. `restore_data` downloads objects under `data/` only.

`obelisk blueprint status|unlock|lock|push` talks to the controller on `127.0.0.1:8080` with the on-box token in the credentials directory.
