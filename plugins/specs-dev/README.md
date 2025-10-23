# Specs-Dev

**Plan, implement, and review features with spec-first automation.**

## Why Specs-Dev
- Align requirements before coding to avoid rework.
- Capture decisions in living specs reviewed by GPT-5 (Codex).
- Ship focused, validated commits using emoji-friendly conventional messages.

## How It Works
1. `/spec:plan` starts a guided requirements chat, captures decisions, and saves a spec in `specs/`.
2. `/spec:impl` turns that spec into bite-sized tasks, runs Codex reviews, and commits safely.
3. Iterate until the spec is complete; progress stays tracked inside the spec file.

## Commands
- `/spec:plan {feature}` — create or refine a spec before coding.
- `/spec:impl {spec-file}` — implement from an existing spec with continuous review.

## Tips
- Keep tasks to 1-3 files so commits stay clean.
- Update the spec when requirements change; it is meant to stay living documentation.
- Treat Codex feedback as a blocking review to maintain quality.

Requires Claude Code with the Codex CLI installed. Once the plugin is in your Claude Code plugins directory, it loads automatically.

Built for Claude Code · Powered by GPT-5 (Codex)
