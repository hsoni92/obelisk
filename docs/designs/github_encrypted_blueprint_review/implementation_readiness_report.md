# Stage 9 — Implementation Readiness (lines 232–end)

## Reviewer Status

| Reviewer | Status |
| --- | --- |
| egm-implementation-readiness | completed |
| egm-codebase-details | completed |
| egm-style | completed |

## Verdict

**Does not pass** implementation-readiness standard (11+ blockers).

## Blockers (consolidated)

1. `config.Save` hook: central vs wrappers; 15+ call sites; suppress on restore/bootstrap  
2. P4b S3 upload/restore unspecified; `runner.go` path wrong (`backup.go`)  
3. Restore orchestrator: no phase enum, persistence, `controller.go` does not exist  
4. Crypto: Argon2id params, manifest schema, `RedactedSpec()` rules  
5. OAuth: client ownership, Secret schema, git library choice  
6. Session unlock: which process, TTL, locked mid-queue  
7. `machine.id` restore flag unnamed in `Load()`  
8. Wrong file paths (helm/platform, obelisk-api.ts, routes)  
9. Verification tests without file targets  
10. Phase table not buildable PR checkpoints  
11. obelisk.md CUJ 5 replacement text not in plan  

## Codebase corrections for plan

* `internal/backup/backup.go` not `runner.go`  
* `internal/controller/reconcile.go` not `controller.go`  
* `deploy/helm/platform/` not `obelisk-platform`  
* RTK in `web/admin/lib/store/obelisk-api.ts`  
* Credential path asymmetry (mail vs backup/openrouter)  
* No k8s client-go for Secret `obelisk-blueprint-github`