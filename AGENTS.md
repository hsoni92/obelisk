# obelisk — agent instructions

## Elephant-Goldfish Model (EGM)

This project uses the [Elephant-Goldfish Model](https://github.com/google/elephant-goldfish-model) for spec-driven development.

EGM_ROOT: /home/himanshu/.cursor/plugins/local/egm

EGM artifact directory: docs/designs

Approved specifications live under `docs/designs/`. Before running EGM review skills, read `EGM_ROOT/docs/cursor.md`.

Typical workflow:

1. `/egm-brainstorm` — explore a feature idea → `concepts_brief.md`
2. `/egm-draft` — turn the brief into a design doc
3. `/egm-review` — independent review of the draft (uses fresh subagents)
4. Implement against the approved spec in `docs/designs/`

**Product note:** Tailscale is **optional** (local profile only, when internet reachability is needed). **VPS deployments do not require Tailscale.**

For an existing codebase, use `/egm-bootstrap`. Use `/egm-drift-report docs/designs/` to compare docs and code.