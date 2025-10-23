---
description: Implement feature from spec with iterative codex review and commits
allowed-tools: Read(*), Write(*), Edit(*), Glob(*), Grep(*), Task, TodoWrite, Bash
---

# Feature Implementation with Codex Review

You are executing a disciplined implementation workflow that keeps Codex (GPT-5) in the loop and lands focused, validated commits.

## Quickstart Flow

| Phase | What you do | Exit criteria |
| --- | --- | --- |
| 1. Plan Analysis | Open the provided session directory, read `plan.md` + `tasks.md`, understand scope and constraints. | You can summarize the feature, affected areas, constraints, and tests; user confirms readiness. |
| 2. Task Breakdown | Break work into small, independent tasks (1–3 files each), sync TodoWrite and `tasks.md`. | Ordered task list exists, first task marked pending, dependencies noted. |
| 3. Iterative Implementation | For each task: mark in progress, implement via agent, run tests, collect Codex review, commit, update docs. | All tasks complete, Codex feedback addressed, commits passing. |
| 4. Final Validation | Run regression checks, update docs, mark session complete, recap next steps. | Tests green, docs updated, user approves completion. |

### Workflow Loop (Phase 3)

1. Update TodoWrite → current task `in_progress`.
2. Launch implementation agent (usually `general-purpose`) with explicit instructions.
3. Run required tests or checks.
4. Request Codex review (`codex-analyzer`) on the diff or files.
5. Apply feedback, rerun tests, and only then commit (`emoji + conventional` message).
6. Mark task `done` in TodoWrite and mirror status in `tasks.md`.
7. Move to the next task.

## Detailed Reference

### Phase 1 – Plan Analysis

**Goal:** Internalize the session plan before touching code.

**Steps:**
1. Verify the session path passed to `/specs-dev:impl` exists and contains `plan.md`, `tasks.md`, and `tmp/`.
2. Read the entire plan, capturing: feature overview, technical approach, implementation steps, testing strategy, and success criteria.
3. Note impacted files/components, integrations, and testing expectations.
4. Summarize the plan back to the user and confirm readiness to proceed.

**Checklist:**

- [ ] Session documents reviewed end-to-end.
- [ ] Key components/files identified.
- [ ] Constraints, risks, and acceptance criteria captured.
- [ ] User approval explicitly received before moving on.

### Phase 2 – Task Breakdown

**Goal:** Create actionable, incremental tasks mapped to the plan.

**Steps:**
1. Translate the plan’s implementation steps into granular tasks (1–3 files each, independently testable).
2. Record tasks using TodoWrite (one `pending` per task; no concurrent `in_progress`).
3. Mirror the same list in `tasks.md` using checkboxes, including dependencies or testing notes when helpful.
4. Highlight riskier tasks or external dependencies.

**Task Template:**

- Objective
- Files to touch
- Expected output
- Tests to run

**Tips:** Keep tasks vertical (deliver user value) instead of broad refactors; break down complex tasks further rather than parking partial code.

### Phase 3 – Iterative Implementation

Repeat the cycle for each task:

1. **Start** – Mark the TodoWrite item `in_progress`; update `tasks.md` if you track owners or notes.
2. **Implement** – Use the Task tool (usually `general-purpose`) with a detailed prompt: objective, files, acceptance tests, patterns to follow.
3. **Validate** – Run unit/integration tests relevant to the change. Document commands you run.
4. **Codex Review** – Call `codex-analyzer` with the diff or affected files. Request severity-ranked findings covering bugs, security/performance issues, and regressions.
5. **Address Feedback** – Apply fixes, rerun tests, and if changes are significant, re-run the review.
6. **Commit** – Ensure a clean diff, then commit with emoji-prefixed Conventional Commit messages (e.g., `✨ feat: add onboarding API handler`). One logical change per commit.
7. **Close Task** – Set TodoWrite to `done`, tick the checkbox in `tasks.md`, and note any follow-up work.

**Quality Gates per Task:**

- [ ] Tests covering the change are added/updated and passing.
- [ ] Codex review feedback implemented.
- [ ] No TODOs or commented-out code left behind.
- [ ] Commit message follows emoji + Conventional Commit format.

### Phase 4 – Final Validation

1. Run the agreed regression/acceptance suite (from `plan.md`).
2. Update `plan.md`/`tasks.md` with final status and notes (e.g., follow-up work, decisions, known issues).
3. Summarize completed work, outstanding risks, and testing outcomes for the user.
4. Confirm the session is ready for merge/release.

**Completion Checklist:**

- [ ] All tasks marked complete in both TodoWrite and `tasks.md`.
- [ ] Tests (unit/integration/lint) run and passing.
- [ ] No pending Codex feedback or unanswered comments.
- [ ] Session docs reflect final state and any next steps.

## Reference Material

### Agent & Tool Guidance

- **Implementation agent:** `general-purpose` (most tasks), or `debugger` for failing tests.
- **Review agent:** `codex-analyzer`; include stack details, modules touched, and request severity-ranked output.
- **Todo management:** Use TodoWrite to keep one active task. Mirroring status in `tasks.md` avoids drift.
- **Scratch space:** Place temporary artifacts in the session’s `tmp/` directory; it is gitignored.

### Best Practices

- Match existing code patterns and conventions before introducing new ones.
- Keep commits atomic and readable; amend only the latest commit if necessary.
- Document any deviations from the plan inside `plan.md` so future readers understand why.
- Re-run impacted tests after each feedback iteration, not just once at the end.

### Troubleshooting

- **No session folder:** Confirm the path argument and create the folder using the plan command before retrying.
- **Task too large:** Split it into smaller vertical slices; update TodoWrite and `tasks.md` accordingly.
- **Codex requests major rework:** Pause, revisit the plan with the user, and update `plan.md` before continuing.
- **Persistent test failures:** Switch to the `debugger` agent or request deeper Codex analysis targeting the failing module.
- **Messy commit history:** Use `git commit --amend` for the latest commit only. Avoid rewriting history beyond that without user approval.

### Communication Tips

- Narrate progress: after each phase and major task, briefly recap what changed, tests run, and what’s next.
- Escalate risks early (integration blockers, tech debt, missing requirements).
- Encourage treating Codex review comments as blocking until resolved.

Delivering consistent updates and respecting these quality gates keeps `/specs-dev:impl` runs predictable and merge-ready.
