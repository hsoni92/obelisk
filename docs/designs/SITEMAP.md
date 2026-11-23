# Design sitemap

Index of specification documents for obelisk. Add a row when a design is approved.

| Document | Status | Summary |
| --- | --- | --- |
| [concepts_brief.md](./concepts_brief.md) | Ideation | Brainstorm output; locked v1 decisions |
| [obelisk.md](./obelisk.md) | Proposed | Middleware appliance: k3s, lazy data services, Kong, Tailscale, DR |
| [design_draft.md](./design_draft.md) | Draft | Working copy synced from drafting session |
| [email_gateway_concepts_brief.md](./email_gateway_concepts_brief.md) | Ideation | Mail gateway brainstorm (HTTP send/read pool, OpenRouter pattern) |
| [mail_gateway.md](./mail_gateway.md) | Proposed | HTTP mail gateway (OpenRouter pattern; send v1.0, read v1.1) |
| [github_encrypted_blueprint_concepts_brief.md](./github_encrypted_blueprint_concepts_brief.md) | Ideation | Goldfish brainstorm: GitHub encrypted control-plane mirror + S3 data |
| [github_encrypted_blueprint.md](./github_encrypted_blueprint.md) | Proposed | Encrypted blueprint mirror: auto-push, user key, chained DR rebirth |
| [blueprint_github_repo_growth_concepts_brief.md](./blueprint_github_repo_growth_concepts_brief.md) | RFC | 30-day daily blueprint history (1 commit/day) + verified latest tip |
| [multi_machine_blueprint_concepts_brief.md](./multi_machine_blueprint_concepts_brief.md) | RFC | One repo, branch-per-machine upstream + manual follows; amends blueprint |
| [multi_machine_blueprint.md](./multi_machine_blueprint.md) | Proposed | Design: fleet branches, follows[], migration off main |
| [curl_vps_installer_concepts_brief.md](./curl_vps_installer_concepts_brief.md) | Ideation | Brainstorm: curl VPS prod install without git clone |
| [curl_vps_installer.md](./curl_vps_installer.md) | Proposed | Thin `install/install.sh` + GitHub archive → existing `bootstrap.sh` |
| [obelisk_uninstaller_concepts_brief.md](./obelisk_uninstaller_concepts_brief.md) | Ideation | VPS uninstall: full local wipe, optional k3s, dry-run |
| [obelisk_uninstaller.md](./obelisk_uninstaller.md) | Proposed | `obelisk uninstall` + `install/uninstall.sh` teardown |
| [precommit_ci_guardrails_concepts_brief.md](./precommit_ci_guardrails_concepts_brief.md) | Ideation | Public-repo CI + optional pre-commit (lint, gitleaks, deps) |
| [backup_lineage_concepts_brief.md](./backup_lineage_concepts_brief.md) | Ideation | First-class backups: per-service lineage, orphans, deploy-from-backup |
| [backup_lineage.md](./backup_lineage.md) | RFC | Backup lineage brainstorm published for team feedback |
| [projects_discovery_concepts_brief.md](./projects_discovery_concepts_brief.md) | Ideation | Projects as connection discovery groups (Goldfish 2026-11-19) |
| [projects_discovery.md](./projects_discovery.md) | Proposed | GET /discover/v1/projects/{id} with scoped bearer; Services UI Projects |
| [peer_gateway.md](./peer_gateway.md) | Proposed | Gateway role: VPS fronts clients; locals advertise over tailnet; federated proxy |
| [cloudflare_tunnel_concepts_brief.md](./cloudflare_tunnel_concepts_brief.md) | Ideation | Standalone local: CF Tunnel UI, HTTPS Kong edge, no Tailscale/gateway |
| [cloudflare_tunnel.md](./cloudflare_tunnel.md) | Proposed | Cloudflare Tunnel local ingress; Connectivity nav page; HTTPEdgeEnabled; cloudflared lifecycle |
| [../concepts_brief.md](../concepts_brief.md) | Ideation | Peer gateway brainstorm brief (repo root) |