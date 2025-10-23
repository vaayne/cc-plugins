---
description: Implement feature from spec with iterative codex review and commits
allowed-tools: Read(*), Write(*), Edit(*), Glob(*), Grep(*), Task, TodoWrite, Bash
---

# Feature Implementation with Codex Review

You are executing a systematic implementation workflow with continuous codex review and incremental commits.

## Your Task

Implement the feature defined in: **$ARGUMENTS**

The argument should be the path to a session directory (e.g., `.agents/sessions/2025-10-12-feature-name/`)

## Workflow Overview

This command follows a 4-phase approach:

1. **Plan Analysis** - Read and understand the implementation plan
2. **Task Breakdown** - Create incremental implementation tasks
3. **Iterative Implementation** - Implement, review, modify, commit each task
4. **Final Validation** - Verify completion and update spec status

## Phase 1: Plan Analysis

### Process

1. **Load Session Docs**:
   - Verify the session directory exists at the provided path
   - Confirm `plan.md` is present; read the complete implementation plan
   - Open `tasks.md` to review or seed the current task list
   - Extract key sections:
     - Feature overview and goals
     - Technical approach
     - Implementation steps
     - Testing strategy
     - Success criteria

2. **Understand Context**:
   - Review codebase structure to identify affected files
   - Identify integration points and dependencies
   - Note existing patterns to follow
   - Locate relevant test files
   - Plan how any temporary artifacts will be stored in the session `tmp/` folder

3. **Confirm Understanding**:
   - Summarize the feature to implement
   - List the main components/files to modify or create
   - Highlight any technical constraints or decisions
   - Ask user: "I've analyzed the plan. Ready to proceed with implementation?"
   - Wait for confirmation before continuing

### Key Guidelines

- Thoroughly understand the plan before coding
- Identify all files that need changes
- Note testing requirements from the spec
- Clarify ambiguities with user before starting

## Phase 2: Task Breakdown

### Process

1. **Create Incremental Tasks**:
   - Break implementation into small, focused changes
   - Each task should be:
     - Independently committable
     - Testable (when applicable)
     - 1-3 files maximum per task
     - Complete (not partial/placeholder code)
   - Order tasks by dependency

2. **Task Structure**:
   Each task should specify:
   - Clear objective (what to build/modify)
   - Files to change
   - Expected outcome
   - Testing approach

3. **Use TodoWrite Tool**:
   - Create todo list with all implementation tasks
   - Mark first task as "pending"
   - Keep other tasks as "pending" initially

4. **Update Session Docs**:
   - Sync the ordered task list into `tasks.md` with checkboxes
   - Update `plan.md` status to "In Progress" (if applicable)
   - Note any session-specific context needed for the upcoming work

### Example Task Breakdown

```
For "User Authentication System":

Task 1: Create user model and database schema
- Files: src/models/user.js, migrations/001_users.sql
- Outcome: User table with email, password_hash, role fields

Task 2: Implement password hashing utility
- Files: src/utils/auth.js, src/utils/auth.test.js
- Outcome: Secure bcrypt hashing with tests

Task 3: Create JWT token service
- Files: src/services/token.js, src/services/token.test.js
- Outcome: Token generation, verification, expiration

Task 4: Implement login endpoint
- Files: src/routes/auth.js, src/controllers/auth.js
- Outcome: POST /auth/login with validation

...
```

## Phase 3: Iterative Implementation

### Process for Each Task

Repeat this cycle for each task in the breakdown:

#### Step 1: Implementation

1. **Mark Task as In Progress**:
   - Use TodoWrite to update current task status
   - Only ONE task should be "in_progress" at a time

2. **Use Sub-Agent for Implementation**:
   - Launch Task tool with appropriate agent
   - Provide clear, detailed instructions
   - Include:
     - Task objective
     - Files to modify/create
     - Technical approach from spec
     - Code patterns to follow
     - Testing requirements

3. **Agent Instructions Format**:
   ```markdown
   Implement [task objective].

   Requirements:

   - [Specific requirement 1]
   - [Specific requirement 2]

   Files to modify/create:

   - [file path 1]: [what to do]
   - [file path 2]: [what to do]

   Technical approach:
   [Relevant details from spec]

   Testing:
   [What tests to write/run]

   Follow existing code patterns in the codebase.
   Make complete, working changes (no placeholders).
   ```

4. **Scratch Space**:
   - Place any temporary files or generated artifacts in the session's `tmp/` directory
   - Clean up unneeded items before marking the task complete

#### Step 2: Codex Review

1. **Submit for Review**:
   - Use Task tool with codex-analyzer agent
   - Provide context about what was implemented
   - Request comprehensive code review
   - Focus areas:
     - Code quality and patterns
     - Security vulnerabilities
     - Performance issues
     - Error handling
     - Test coverage
     - Edge cases

2. **Codex Agent Usage**:
   ```markdown
   Use the Task tool with:

   - subagent_type: codex-analyzer
   - description: "Review implementation of [task]"
   - prompt: "Review the implementation of [task objective].

   Changed files:

   - [list files modified]

   Please analyze:

   1. Code quality - Does it follow best practices and existing patterns?
   2. Security - Are there vulnerabilities or unsafe operations?
   3. Performance - Are there efficiency concerns?
   4. Error handling - Are edge cases and errors properly handled?
   5. Testing - Is test coverage adequate?
   6. Completeness - Is the implementation complete and production-ready?

   Be critical and thorough. Identify any issues that need fixing."
   ```

#### Step 3: Address Feedback

1. **Review Codex Feedback**:
   - Read all suggestions and concerns
   - Categorize by severity (critical, important, nice-to-have)
   - Decide what must be fixed before committing

2. **Make Modifications**:
   - If critical issues found, use sub-agent or direct edits to fix
   - Document what was changed and why
   - Re-review with Codex if changes are significant
   - Maximum 2 review cycles per task

3. **Validation**:
   - Run relevant tests if applicable
   - Verify no regressions
   - Check that task objective is fully met

#### Step 4: Commit Changes

1. **Stage Files**:
   - Review what files changed for this task
   - Use `git status` and `git diff` to verify
   - Stage only files relevant to current task

2. **Craft Commit Message**:
   - Follow conventional commit format with emoji
   - Format: `<emoji> <type>: <description>`
   - Types: feat, fix, refactor, test, docs, chore
   - Emoji examples:
     - ✨ feat: new feature
     - 🐛 fix: bug fix
     - ♻️ refactor: code refactoring
     - ✅ test: adding tests
     - 📝 docs: documentation
     - 🔒 security: security improvements
     - ⚡ performance: performance improvements
   - Description: clear, concise summary (50 chars)
   - Body (if needed): explain why, not what

3. **Create Commit**:
   - Use Bash tool with git commit
   - Include descriptive commit message
   - Verify commit succeeded
   - Example:
     ```bash
     git add src/models/user.js migrations/001_users.sql
     git commit -m "$(cat <<'EOF'
     ✨ feat: add user model and database schema

     - Create User model with email, password_hash, role fields
     - Add migration for users table with proper indexes
     - Include timestamps for created_at and updated_at
     EOF
     )"
     ```

4. **Update Progress**:
   - Mark task as completed in TodoWrite
   - Check off the task inside `tasks.md` and note outcomes
   - Update any status markers in `plan.md` if required

#### Step 5: Continue to Next Task

1. **Move to Next Task**:
   - Mark next task as "in_progress" in TodoWrite
   - Repeat Steps 1-4 for this task
   - Continue until all tasks completed

### Important Guidelines

- **Small, Complete Changes**: Each commit should be small but fully working
- **No Placeholders**: Never commit TODO comments or unfinished code
- **Test as You Go**: Run tests after each task when applicable
- **Stay Focused**: One task at a time, don't mix concerns
- **Keep Context**: Reference the session's `plan.md` and `tasks.md` throughout implementation

## Phase 4: Final Validation

### Process

1. **Run Full Test Suite**:
   - Execute all tests to ensure nothing broke
   - Fix any failing tests immediately
   - Commit fixes with `🐛 fix:` or `✅ test:` prefix

2. **Verify Success Criteria**:
   - Review success criteria from original spec
   - Confirm all requirements met
   - Test the feature end-to-end if applicable

3. **Update Session Docs**:
   - Mark `plan.md` status as "Completed" and add completion date
   - Append an "Implementation Summary" section capturing:
     - What was implemented
     - Number of commits made
     - Any deviations from original plan
     - Testing results
     - Known limitations or future work
   - Ensure `tasks.md` shows all items checked off with relevant notes

4. **Final Summary**:
   - List all commits made
   - Summarize what was built
   - Highlight any important changes or decisions
   - Suggest next steps or related work

### Completion Checklist

Before marking complete:

- [ ] All tasks from breakdown are implemented
- [ ] All commits follow conventional format with emoji
- [ ] Codex reviewed all significant changes
- [ ] Tests pass (if applicable)
- [ ] Success criteria from `plan.md` are met
- [ ] Session docs updated with completion status
- [ ] No TODO or placeholder code committed

## Example Workflow

```
User: /impl .agents/sessions/2025-10-12-user-authentication/

Assistant (Phase 1):
[Reads `plan.md` and `tasks.md` from session folder]
"I've analyzed the authentication system plan. The implementation involves:
- User model with JWT authentication
- Login/logout endpoints
- Password hashing with bcrypt
- Token middleware
- Role-based access control

I'll need to modify/create:
- src/models/user.js
- src/services/token.js
- src/routes/auth.js
- src/middleware/auth.js
- Tests for all components

Ready to proceed with implementation?"

User: "Yes"

Assistant (Phase 2):
[Creates task breakdown with TodoWrite]
[Syncs tasks into `tasks.md` and marks `plan.md` as In Progress]

"Created 8 implementation tasks. Starting with Task 1: User model and schema..."

Assistant (Phase 3 - Task 1):
[Uses Task tool with implementation agent]
[Agent implements user model]
[Uses Task tool with codex-analyzer for review]

Codex: "Consider adding email validation and indexing the email field for performance."

[Makes modifications based on feedback]
[Commits with: "✨ feat: add user model with email validation and indexes"]
[Updates TodoWrite and `tasks.md` progress]

Assistant (Phase 3 - Task 2):
[Repeats cycle for password hashing utility]
[Commits with: "🔒 feat: implement secure password hashing with bcrypt"]
...

Assistant (Phase 4):
[Runs full test suite]
[Updates session docs to "Completed"]

"Implementation complete! Made 8 commits:
1. ✨ feat: add user model with email validation
2. 🔒 feat: implement secure password hashing
3. ✨ feat: create JWT token service
4. ✨ feat: implement login endpoint
5. ✨ feat: add logout functionality
6. ✨ feat: create auth middleware
7. ✨ feat: implement role-based access control
8. ✅ test: add comprehensive auth tests

All success criteria met. The authentication system is ready for use."
```

## Sub-Agent Selection Guide

Choose the appropriate agent based on task type:

- **general-purpose**: Complex multi-file changes, new features
- **debugger**: Fixing bugs, test failures
- **code-reviewer**: Quick review before codex (optional)
- **python-runner**: Tasks needing complex scripts or automation

Most implementation tasks should use **general-purpose** agent.

## Best Practices

### Implementation Quality

- Follow existing code patterns in the codebase
- Write comprehensive tests alongside code
- Handle errors gracefully with proper messages
- Add comments for complex logic
- Keep functions small and focused

### Commit Hygiene

- One logical change per commit
- Test before committing
- Write clear, descriptive messages
- Use appropriate emoji and conventional commit type
- Group related file changes together

### Codex Review

- Always review significant changes
- Take feedback seriously
- Fix critical issues before committing
- Don't skip review to save time
- Use review to learn and improve

### Progress Tracking

- Keep TodoWrite updated in real-time
- Update session docs (`tasks.md`, `plan.md`) after each commit
- Maintain clear status in both places
- Help user understand progress

## Troubleshooting

**Issue**: Session folder not found
**Solution**: Verify path is correct; confirm `.agents/sessions/{date-feature}/` exists with `plan.md`

**Issue**: Task is too large
**Solution**: Break it down further into smaller subtasks

**Issue**: Codex review suggests major refactor
**Solution**: Discuss with user, may need to update plan and restart

**Issue**: Tests keep failing
**Solution**: Use debugger agent, fix root cause before continuing

**Issue**: Commit history is messy
**Solution**: Use git commit --amend for last commit only, or rebase carefully

**Issue**: Lost track of progress
**Solution**: Check `tasks.md` and TodoWrite, sync them up

## Important Notes

### Context Management

- Keep `plan.md` and `tasks.md` open for reference
- Review related code before implementing
- Maintain consistency with existing code
- Don't duplicate functionality

### Agent Usage

- Provide detailed instructions to agents
- Include context about the codebase
- Specify files to modify explicitly
- Review agent output before proceeding

### Quality Gates

- Don't skip codex review
- Don't commit failing code
- Don't leave TODOs in commits
- Don't mix unrelated changes

### Communication

- Keep user informed of progress
- Ask for clarification when needed
- Report issues promptly
- Celebrate milestones

Now begin the implementation process for: **$ARGUMENTS**
