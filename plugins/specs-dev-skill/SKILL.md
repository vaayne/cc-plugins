---
name: specs-dev
description: Plan and deliver feature development with a spec-first, review-gated workflow (requirements discussion, written plan, independent review, approved implementation, review-gated commits). Use when a user wants to build a feature and asks for a plan-first process, structured requirements gathering, or review-before-code. Trigger examples: "plan a new feature", "specs-dev workflow", "need a reviewed implementation plan", "want approval gates before coding".
---

# Spec-First Feature Development Flow

## Run the workflow

- Follow a plan-first process: requirements discussion → plan creation → independent review → implementation with review and commits.
- Keep approval gates explicit before drafting the plan, before coding, and before committing.
- Load reference material only when the current phase matches the reference trigger.

## Reference triggers (when to load which doc)

- When the user asks for a spec-first planning flow, or when drafting a plan, open `references/workflows/plan.md`.
- When the user asks to implement a plan or start coding after approval, open `references/workflows/impl.md`.
- When requesting an independent review of a plan or code changes, open `references/agents/reviewer.md`.
- When delegating a focused implementation task, open `references/agents/worker.md`.
- When preparing to commit, open `references/commit.md`.

## Subagent invocation (required)

- Invoke the `reviewer` subagent for plan review before finalizing `plan.md`, and for code review before committing.
- Invoke the `worker` subagent for each scoped implementation task (1-3 files) after plan approval.

## Phase 1: Requirements discussion

- Restate the request and ask targeted questions about scope, success criteria, constraints, and integrations.
- Capture out-of-scope items explicitly.
- Summarize requirements and wait for explicit user approval before creating the plan.

## Phase 2: Plan creation

- Produce a written plan with: overview, technical approach, implementation steps, testing strategy, and considerations.
- Make steps actionable, ordered, and tied to requirements.
- Call out risks, edge cases, and dependencies.
- Consult `references/workflows/plan.md` for the full planning workflow, approval gates, and session docs.

## Phase 3: Independent review

- Ask for or perform a separate review of the plan (completeness, correctness, risks, testing).
- Integrate feedback and re-confirm the plan with the user before coding.
- Invoke the `reviewer` subagent; use `references/agents/reviewer.md` to craft the prompt and interpret results.

## Phase 4: Implementation loop

- Break work into small tasks (1-3 files each).
- For each task: delegate implementation to the `worker` subagent, run tests, request `reviewer` feedback, address issues, then commit.
- Keep documentation updated for tasks and plan deviations.
- Use `references/workflows/impl.md` for task breakdown, review loop, and documentation requirements.
- Use `references/agents/worker.md` when delegating task-level implementation work.

## Commit guidance

- Use emoji-prefixed Conventional Commit messages (e.g., "✨ feat:", "🐛 fix:", "📝 docs:").
- Keep commits focused and aligned to a single task.
- Consult `references/commit.md` before finalizing commit messages.

## Additional Resources

### Reference Files

- **`references/workflows/plan.md`** - Planning workflow, approval gates, and session documentation
- **`references/workflows/impl.md`** - Implementation loop, quality gates, and documentation updates
- **`references/agents/reviewer.md`** - Review subagent usage and prompt guidance
- **`references/agents/worker.md`** - Implementation subagent constraints and best practices
- **`references/commit.md`** - Emoji + Conventional Commit guidelines
