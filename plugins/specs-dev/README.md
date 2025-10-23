# Specs-Dev

**Plan, implement, and review features with spec-first automation.**

## Why Specs-Dev

- Align requirements before coding to avoid rework.
- Capture decisions in living specs reviewed by GPT-5 (Codex).
- Ship focused, validated commits using emoji-friendly conventional messages.

## Session Layout

- Each run creates `.agents/sessions/{YYYY-MM-DD-feature-name}/` at repo root.
- `plan.md` holds the approved plan.
- `tasks.md` tracks implementation tasks and status.
- `tmp/` stores scratch data; it is gitignored automatically.

## How It Works

1. `/spec:plan` guides requirements, captures decisions, and writes `plan.md` in the session folder.
2. Review `plan.md` and `tasks.md`; edit them directly or ask an agent to refine the plan and task list.
3. `/spec:impl` consumes that plan, updates `tasks.md`, and uses `tmp/` for intermediates.
4. Iterate until tasks are complete; progress lives inside the session directory.

## Commands

- `/spec:plan {feature}` — create or refine a spec before coding.
- `/spec:impl {session-folder}` — implement using a saved session directory.

## Tips

- Keep tasks to 1-3 files so commits stay clean.
- Update `plan.md` and `tasks.md` when requirements shift; they are living docs.
- Treat Codex feedback as a blocking review to maintain quality.
- Make sure the session docs are accurate before launching `/spec:impl` for smoother implementation.

Requires Claude Code with the Codex CLI installed. Once the plugin is in your Claude Code plugins directory, it loads automatically; run `/help` to confirm the commands are available.

Built for Claude Code · Powered by GPT-5 (Codex)
