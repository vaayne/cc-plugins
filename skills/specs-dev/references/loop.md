# Implementation Loop

Task loop state machine for Phase 3.

## State Machine

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

## Loop Steps

For each task in `tasks.md`:

### 1. Start

- Set task state: `IMPLEMENTING`
- Read context from `plan.md` and `tasks.md`
- Initialize iteration counter: `0`

### 2. Implement (Subagent)

Delegate to worker subagent:

- Context: `references/agents/worker.md`
- Input: task objective, files, acceptance criteria
- Input (if iteration > 0): previous feedback to address

### 3. Validate

- Set task state: `VALIDATING`
- Run tests, check lint/type errors
- **If fail:** Increment iteration, loop to step 2 with error output
- **If pass:** Continue to step 4

### 4. Review (Subagent)

- Set task state: `REVIEWING`
- Delegate to reviewer subagent
- Context: `references/agents/reviewer.md`
- Request structured verdict (approved/blockers/suggestions)

### 5. Evaluate Verdict

- **`approved: true`:** Continue to step 6
- **`approved: false` and iteration < 3:** Increment iteration, loop to step 2
- **`approved: false` and iteration >= 3:** Escalate to user

### 6. Commit

- Set task state: `APPROVED`
- Commit with emoji + conventional format

### 7. Document

Update `tasks.md`:

```markdown
- [x] Task name
  - **Files:** `file1.ts`, `file2.ts`
  - **State:** APPROVED
  - **Iterations:** 2
  - **Approach:** Brief description
  - **Gotchas:** Any surprises
  - **Commit:** {hash}
```

Update `plan.md` only if implementation deviated or new decisions made.

### 8. Next Task

- Mark task complete
- Move to next task, repeat from step 1

## Fix Routing

| Condition                            | Action                                 |
| ------------------------------------ | -------------------------------------- |
| Validation failure                   | Implementer subagent with error output |
| Review blockers (critical/important) | Implementer subagent with feedback     |
| Review suggestions only              | Orchestrator quick-fix or defer        |
| Iteration >= 3                       | Pause, ask user for guidance           |

## Commit Format

- `✨ feat:` — New features
- `🐛 fix:` — Bug fixes
- `♻️ refactor:` — Code restructuring
- `📝 docs:` — Documentation
- `✅ test:` — Tests
- `⚡️ perf:` — Performance
