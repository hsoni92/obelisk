# obelisk — agent instructions

## Elephant-Goldfish Model (EGM)

This project uses the [Elephant-Goldfish Model](https://github.com/google/elephant-goldfish-model) for spec-driven development.

EGM_ROOT: /home/himanshu/.cursor/plugins/local/egm

EGM artifact directory: docs/designs

Approved specifications live under `docs/designs/`. Before running EGM review skills, read `EGM_ROOT/docs/cursor.md`.

### Cursor: Goldfish subagents and usage limits

EGM reviews spawn isolated `Task` subagents. They bill **per subagent** and may use a **different usage pool** than the parent chat model unless they inherit it.

- **Never pass `model` on `Task`** for EGM reviewers unless the user explicitly requests a model. Omit `model` so subagents inherit the parent (e.g. Composer 2.5 Fast).
- **Do not** default Goldfish tasks to Claude, Opus, Sonnet, or other premium slugs—that pattern burns “Other Models” quota and fails with usage limits while direct Composer chat still works.
- On usage-limit failure: stop parallel launches, retry reviewers **sequentially** without `model`, then mark the stage incomplete in the artifact if tasks still cannot start (see `EGM_ROOT/docs/cursor.md`).

Workspace rule: `.cursor/rules/egm-cursor-usage.mdc`.

Typical workflow:

1. `/egm-brainstorm` — explore a feature idea → `concepts_brief.md`
2. `/egm-draft` — turn the brief into a design doc
3. `/egm-review` — independent review of the draft (uses fresh subagents)
4. Implement against the approved spec in `docs/designs/`

**Product note:** Tailscale is **optional** on `deployment.role: local` when internet reachability is needed. Standalone `profile: vps` still skips Tailscale. **`deployment.role: gateway`** requires Tailscale for mesh backhaul to local peers.

For an existing codebase, use `/egm-bootstrap`. Use `/egm-drift-report docs/designs/` to compare docs and code.