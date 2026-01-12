---
name: specs-dev
description: Plan-first development workflow with review gates. Use when implementing features, refactoring, or any task requiring structured planning, iterative implementation with reviews, and clean commits. Triggers on requests like "implement feature X", "plan and build", "spec-driven development", or when user wants disciplined, reviewed code changes.
---

# Specs-Dev Workflow

A disciplined, review-gated development workflow that ensures quality through structured planning, iterative implementation, and continuous review.

## When to Use

- Implementing new features
- Complex refactoring
- Any task requiring planning before coding
- When user requests "plan first" or "spec-driven" approach
- Multi-file changes that benefit from review gates

## Workflow Overview

| Phase             | Purpose                       | Exit Criteria                |
| ----------------- | ----------------------------- | ---------------------------- |
| 1. Discovery      | Understand requirements       | User approves summary        |
| 2. Planning       | Create reviewed plan          | Plan reviewed and approved   |
| 3. Implementation | Iterative coding with reviews | All tasks complete, reviewed |
| 4. Completion     | Final validation              | Tests pass, docs updated     |

## Phase 1: Discovery & Requirements

**Goal:** Reach shared understanding before planning.

### Steps

1. **Interpret the request** - State your initial understanding
2. **Ask clarifying questions** about:
   - Core functionality and user goals
   - Constraints and dependencies
   - Success criteria and acceptance tests
   - Out-of-scope items
3. **Iterate** - Reflect answers, tighten understanding, ask follow-ups
4. **Summarize** - Present final requirements summary

### Approval Gate A

> "Do I understand correctly? Should I proceed to create the plan?"

**Stop and wait for explicit approval before proceeding.**

## Phase 2: Planning

**Goal:** Create a comprehensive, reviewed implementation plan.

### Plan Contents

1. **Overview** - Feature summary, goals, success criteria
2. **Technical Approach** - Architecture, design decisions, components
3. **Implementation Steps** - Ordered tasks with file scopes (1-3 files each)
4. **Testing Strategy** - Unit/integration tests, edge cases
5. **Considerations** - Security, performance, risks, open questions

### Plan Quality Checklist

- [ ] Every requirement from Phase 1 addressed
- [ ] Tasks are actionable and logically ordered
- [ ] Key decisions have documented rationale
- [ ] Testing and edge cases specified
- [ ] Risks and mitigations captured

### Review Loop

1. **Draft the plan** using the template from `references/plan-template.md`
2. **Delegate to analyzer subagent** for review:
   - Load `references/analyzer-agent.md` for review prompt context
   - Request feedback on completeness, technical soundness, risks
3. **Integrate feedback** - Adjust plan based on review
4. **Iterate** (max 3 rounds) until analyzer approves

### Approval Gate B

Present the reviewed plan to user. Only after explicit approval:

1. Create session directory: `.agents/sessions/{YYYY-MM-DD}-{feature-name}/`
2. Save `plan.md` (use `references/plan-template.md`)
3. Save `tasks.md` (use `references/tasks-template.md`)

## Phase 3: Implementation

**Goal:** Implement tasks iteratively with approval-gated review loops.

### Task Loop State Machine

```
┌──────────────────────────────────────────────────────────┐
│  IMPLEMENTING ◄───────────────────┐                      │
│  (subagent)                       │                      │
│       │                           │                      │
│       ▼                           │                      │
│  VALIDATING ──── fail ────────────┤                      │
│  (tests/lint)                     │                      │
│       │ pass                      │                      │
│       ▼                           │                      │
│  REVIEWING ───── not approved ────┤ (iteration < 3)      │
│  (subagent)                       │                      │
│       │                           │                      │
│       │ approved       iteration >= 3                    │
│       │                           │                      │
│       │                    ┌──────┴──────┐               │
│       │                    │  ESCALATE   │               │
│       │                    │ (ask user)  │               │
│       │                    └──────┬──────┘               │
│       │◄──────────────────────────┘                      │
│       ▼                                                  │
│  COMMITTING                                              │
│       │                                                  │
│       ▼                                                  │
│  DOCUMENTING ────► NEXT TASK                             │
└──────────────────────────────────────────────────────────┘
```

### Loop Steps

For each task in `tasks.md`:

#### 1. Start

- Set task state: `IMPLEMENTING`
- Read context from `plan.md` and `tasks.md`
- Initialize iteration counter: `0`

#### 2. Implement (Subagent)

Delegate to implementer subagent:

- Context: `references/implementer-agent.md`
- Input: task objective, files, acceptance criteria
- Input (if iteration > 0): previous feedback to address

#### 3. Validate

- Set task state: `VALIDATING`
- Run tests, check lint/type errors
- **If fail:** Increment iteration, loop to step 2 with error output
- **If pass:** Continue to step 4

#### 4. Review (Subagent)

- Set task state: `REVIEWING`
- Delegate to analyzer subagent
- Context: `references/analyzer-agent.md`
- Request structured verdict (approved/blockers/suggestions)

#### 5. Evaluate Verdict

Check analyzer verdict:

- **If `approved: true`:** Continue to step 6
- **If `approved: false` and iteration < 3:** Increment iteration, loop to step 2 with blockers
- **If `approved: false` and iteration >= 3:** Escalate to user

### Fix Routing

| Condition | Action |
|-----------|--------|
| Validation failure | Implementer subagent with error output |
| Review blockers (critical/important) | Implementer subagent with feedback |
| Review suggestions only | Orchestrator quick-fix or defer |
| Iteration >= 3 | Pause, ask user for guidance |

#### 6. Commit

- Set task state: `APPROVED`
- Create commit with emoji + conventional format:
  - `✨ feat:` - New features
  - `🐛 fix:` - Bug fixes
  - `♻️ refactor:` - Code restructuring
  - `📝 docs:` - Documentation
  - `✅ test:` - Tests
  - `⚡️ perf:` - Performance

#### 7. Document

Update `tasks.md`:

```markdown
- [x] Task name
  - **Files:** `file1.ts`, `file2.ts`
  - **State:** APPROVED
  - **Iterations:** 2
  - **Approach:** Brief description
  - **Gotchas:** Any surprises discovered
  - **Commit:** {hash}
```

Update `plan.md` only if:

- Implementation deviated from original plan
- New architectural decisions made
- Risks discovered affecting future work

#### 8. Next Task

- Mark task checkbox complete
- Move to next task, repeat from step 1

### Quality Gates Per Task

- [ ] Tests covering change added/updated and passing
- [ ] Analyzer verdict: `approved: true`
- [ ] No TODOs or commented-out code
- [ ] Commit follows emoji + conventional format
- [ ] `tasks.md` updated with state, iterations, notes

## Phase 4: Completion

**Goal:** Final validation and wrap-up.

### Steps

1. **Run full test suite** - Regression/acceptance tests from plan
2. **Update plan.md** with:
   - Overall testing results
   - Final status
   - Known issues
   - Follow-up work / next steps
3. **Verify tasks.md** - All tasks marked complete with notes
4. **Summarize** - Recap completed work, risks, outcomes
5. **Confirm** - Session ready for merge/release

### Completion Checklist

- [ ] All tasks complete in `tasks.md`
- [ ] All tests passing
- [ ] No pending review feedback
- [ ] Session docs reflect final state
- [ ] User confirms completion

## Subagent Delegation

### Analyzer Subagent

Use for plan reviews and code reviews.

```
Task: Review the following [plan/code changes] for:
- Completeness and correctness
- Security and performance issues
- Edge cases and error handling
- Adherence to patterns and conventions

Context: [Load references/analyzer-agent.md]

[Content to review]
```

### Implementer Subagent

Use for focused implementation tasks.

```
Task: Implement the following task:
- Objective: [task description]
- Files: [1-3 files to modify]
- Acceptance criteria: [what success looks like]
- Patterns to follow: [existing code references]

Context: [Load references/implementer-agent.md]

Session plan: [summary from plan.md]
```

## Session Structure

```
.agents/sessions/{YYYY-MM-DD}-{feature-name}/
├── plan.md      # Strategic plan (template: references/plan-template.md)
└── tasks.md     # Tactical tasks (template: references/tasks-template.md)
```

## Best Practices

### Planning

- Spend adequate time in Discovery - better questions reduce rework
- Keep tasks scoped to 1-3 files for focused commits
- Document decisions and rationale for future reference

### Implementation

- Match existing code patterns before introducing new ones
- Keep commits atomic and readable
- Document while context is fresh
- Treat review feedback as blocking until resolved

### Communication

- Narrate progress after each phase and major task
- Escalate risks early (blockers, tech debt, missing requirements)
- Checkpoint with user at major milestones

## Troubleshooting

| Problem                      | Solution                                                  |
| ---------------------------- | --------------------------------------------------------- |
| Requirements keep changing   | Spend more time in Phase 1; update summary until sign-off |
| Plan too high-level          | Add file names, interfaces, data contracts, test outlines |
| Task too large               | Split into smaller vertical slices                        |
| Review requests major rework | Pause, revisit plan with user, update before continuing   |
| Persistent test failures     | Deep-dive analysis on failing module                      |
| Documentation debt           | Pause and catch up if 3+ tasks without doc updates        |
