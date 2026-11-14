# Critic Report — mail_gateway.md (L1–263)

## Blockers

1. k3s writable credentials path unresolved (OQ1).
2. Missing consolidated Risks & Mitigations / NFR section.
3. Duplicate-send policy on SMTP failover undecided (OQ3).

## Major

4. Consumer mailbox quota economics not modeled.
5. Selector/cooldown durability vs single-replica assumption unstated.
6. Abuse blast radius under-specified for one bearer token.
7. Direct `:8082` bypass not in feature-range architecture.
8. Alternatives rows use weak "Rejected by author" rationales.
9. No timeout/latency SLOs for CUJs.

## Structural/Style

- Add `## Non-Functional Requirements` before Detailed Implementation.
- Calibrate Kong rate limits vs OpenRouter with numbers or new OQ.