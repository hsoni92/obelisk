# Contributing

The **obelisk** repo is public; CI on pull requests is the merge gate. Local git hooks are optional speed-ups.

## Prerequisites

| Tool | Version | Notes |
|------|---------|--------|
| Go | 1.22+ | `make check-go` |
| Node.js | 20 LTS | Admin UI in `web/admin` |
| Helm | 3.14+ | Only when changing charts |
| pre-commit | 3.x | Optional; `make hooks-install` |

## Verify before you push

```bash
make ci
```

Same work as GitHub Actions (Go, Helm, admin build; gitleaks if installed locally).

## Optional git hooks

```bash
make hooks-install   # pre-commit + gitleaks on staged files
```

`git commit --no-verify` still works; required checks on `main` must pass in GitHub.

## Guardrail layers

1. **This repo (CI / hooks)** — lint, build, tests, gitleaks on committed files.
2. **Appliance blueprint push** — `ScanPublicYAML` before encrypted bundle sync (operator private repo; see [github-blueprint runbook](docs/runbooks/github-blueprint.md)).

Do not commit real tokens, `.env` files, or `github-oauth.client.json`. Examples in `docs/` use placeholders; production secrets stay on the host or in the operator blueprint.