# Specs-Dev

**Plan, implement, and review features with spec-first automation.**

## Why Specs-Dev

- Align requirements before coding to avoid rework.
- Capture decisions in living specs reviewed by Codex.
- Ship focused, validated commits using emoji-friendly conventional messages.

## Getting Started

1. **Add marketplace** – In Claude Code, run `/plugin marketplace add  vaayne/ccplugins`.
2. **Install** – Open `/plugin`, choose “Browse Plugins,” and install `specs-dev`. Enable it if it’s listed as disabled.
3. **Verify** – Run `/plugin` to confirm the `specs-dev:*` commands are registered.
4. **First run** – In any project, execute `/specs-dev:plan onboarding-flow`. Answer the guided questions, approve the summary, and let the command generate a session at `.agents/sessions/{date}-onboarding-flow/`.
5. **Review artifacts** – Open the session’s `plan.md` and `tasks.md`, adjust as needed, then step into implementation with `/specs-dev:impl .agents/sessions/{date}-onboarding-flow/`.

Requirements: Claude Code with Codex CLI access and Codex enabled.

For iterative development, keep this repository registered as a local marketplace and rebuild with `/plugin marketplace add  ./` whenever you update the commands or agents.

## Directory Overview

- `agents/` – Task agents that wrap Codex behaviors (e.g., `codex-analyzer`).
- `commands/` – Markdown specs that power the `/spec:*` commands.
- `hooks/` – Optional automation hooks (empty placeholder today).
- `skills/` – Shared snippets for future cross-command reuse (currently empty).

## Workflow at a Glance

1. `/specs-dev:plan {feature}` captures requirements, writes a session folder, and blocks until the plan is approved.
2. Review/refine `plan.md` + `tasks.md`; keep them accurate as requirements evolve.
3. `/specs-dev:impl {session-folder}` walks the task list, keeps TodoWrite in sync, and enforces commit hygiene.
4. Iterate until `tasks.md` is complete; the session directory becomes the living source of truth.

## Tips for Smooth Runs

- Keep implementation tasks scoped to 1–3 files so each commit stays focused.
- Update `plan.md` and `tasks.md` whenever requirements shift; the commands expect them to be current.
- Treat Codex feedback as a blocking review before merging any change set.
- Confirm the session docs are accurate before starting `/specs-dev:impl` to avoid redo loops.

Built for Claude Code · Powered by Codex
