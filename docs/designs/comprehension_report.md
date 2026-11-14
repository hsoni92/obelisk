# Comprehension Test — mail_gateway.md

**Target:** `docs/designs/mail_gateway.md` (Feature 1–263, Detailed 264+)

## What this proposes

HTTP Mail gateway on Obelisk mirroring OpenRouter: pooled operator mailboxes, bearer token, Kong `/mail/api/v1`, send v1.0 then read v1.1.

## How the existing system works, per this document

OpenRouter already uses yaml + credentials sidecar + dedicated listener + Exposure admin panel; mail reuses that Exposure pattern on `obelisk-controller`.

## Findings (Major)

- **Artemis** undefined (external reference repo for host tables).
- **Exposure** admin nav term undefined.
- **`{origin}`** client base URL undefined.
- **Platform ready** CUJ precondition undefined.
- **`wait-for-mail`** mentioned without API shape.