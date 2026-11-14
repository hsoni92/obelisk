# Implementation Readiness — mail_gateway.md (Detailed §264+)

**Verdict:** **FAIL**

## Blockers

- Writable credentials on k3s (read-only `/etc/obelisk` mount); no chosen fix.
- Kong dev vs helm file drift; mail plugins need reconciliation.
- Admin API + RTK contracts incomplete (`gateway_token_configured`, test send, duplicate 409).
- `go.mod` SMTP/IMAP deps and library choice (OQ2) open.

## Major

- `/var/lib/obelisk` cursor volume not in Helm.
- Dual gateway wiring (`:8082`, reload, Makefile dev flags) underspecified.
- Account ID format, mux path prefix, rate limit values undecided.
- Artemis host tables external to repo — need vendored fixtures or copy step.