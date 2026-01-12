# Troubleshooting & Best Practices

## Common Issues

| Problem                      | Solution                                                  |
| ---------------------------- | --------------------------------------------------------- |
| Requirements keep changing   | More time in Phase 1; update summary until sign-off       |
| Plan too high-level          | Add file names, interfaces, data contracts, test outlines |
| Task too large               | Split into smaller vertical slices                        |
| Review requests major rework | Pause, revisit plan with user, update before continuing   |
| Persistent test failures     | Deep-dive analysis on failing module                      |
| Documentation debt           | Pause and catch up if 3+ tasks without doc updates        |

## Best Practices

### Planning

- Spend adequate time in Discovery — better questions reduce rework
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
