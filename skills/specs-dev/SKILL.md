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

| Phase | Purpose | Exit Criteria |
|-------|---------|---------------|
| 1. Discovery | Understand requirements | User approves summary |
| 2. Planning | Create reviewed plan | Plan reviewed and approved |
| 3. Implementation | Iterative coding with reviews | All tasks complete, reviewed |
| 4. Completion | Final validation | Tests pass, docs updated |

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

**Goal:** Implement tasks iteratively with review gates.

### Task Loop

For each task in `tasks.md`:

#### 1. Start Task
- Mark task as in-progress
- Read task context from `plan.md` and `tasks.md`

#### 2. Implement
Delegate to implementer subagent:
- Load `references/implementer-agent.md` for implementation context
- Provide: task objective, files to modify, acceptance criteria
- Implementer follows pattern-first, test-driven approach

#### 3. Validate
- Run relevant tests
- Verify no linting/type errors

#### 4. Review
Delegate to analyzer subagent:
- Request severity-ranked review of changes
- Focus: bugs, security, performance, patterns

#### 5. Address Feedback
- Apply fixes from review
- Re-run tests
- Re-review if changes significant

#### 6. Commit
Create clean commit with emoji + conventional format:
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
  - **Approach:** Brief description
  - **Gotchas:** Any surprises discovered
  - **Commit:** {hash}
```

Update `plan.md` only if:
- Implementation deviated from original plan
- New architectural decisions made
- Risks discovered affecting future work

#### 8. Complete
- Mark task done
- Move to next task

### Quality Gates Per Task

- [ ] Tests covering change added/updated and passing
- [ ] Review feedback addressed
- [ ] No TODOs or commented-out code
- [ ] Commit follows emoji + conventional format
- [ ] `tasks.md` updated with implementation notes

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

| Problem | Solution |
|---------|----------|
| Requirements keep changing | Spend more time in Phase 1; update summary until sign-off |
| Plan too high-level | Add file names, interfaces, data contracts, test outlines |
| Task too large | Split into smaller vertical slices |
| Review requests major rework | Pause, revisit plan with user, update before continuing |
| Persistent test failures | Deep-dive analysis on failing module |
| Documentation debt | Pause and catch up if 3+ tasks without doc updates |
