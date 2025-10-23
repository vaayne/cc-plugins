---
description: Collaborative planning workflow with Codex review before implementation
allowed-tools: Read(*), Write(*), Edit(*), Glob(*), Grep(*), Task, TodoWrite
---

# Feature Planning with Codex Review

You are facilitating a collaborative planning workflow that includes Codex review before implementation.

## Your Task

Create a comprehensive implementation plan for: **$ARGUMENTS**

## Workflow Overview

This command follows a 4-phase approach:

1. **Requirements Discussion** - Iterative conversation to understand needs
2. **Plan Creation** - Draft detailed implementation plan
3. **Codex Review** - Expert review and refinement
4. **Plan Documentation** - Write final approved plan to specs/

## Phase 1: Requirements Discussion

### Process

1. **Initial Understanding**:
   - Present your interpretation of the feature request
   - Ask clarifying questions about:
     - Core functionality and user goals
     - Technical constraints and preferences
     - Success criteria and acceptance
     - Scope boundaries (what's in/out)
     - Integration points and dependencies

2. **Iterative Refinement**:
   - Listen to user responses carefully
   - Refine your understanding based on feedback
   - Ask follow-up questions when unclear
   - Summarize understanding after each iteration
   - Continue until user says "OK", "ready", "looks good", or similar approval

3. **Approval Gate**:
   - Provide final comprehensive summary of requirements
   - Ask explicitly: "Do I understand the requirements correctly? Should I proceed to create the plan?"
   - Wait for user confirmation before proceeding

### Key Guidelines

- Be thorough but concise in questions
- Focus on technical feasibility and implementation details
- Identify potential challenges early
- Don't assume - always clarify ambiguities
- Use specific examples to validate understanding

## Phase 2: Plan Creation

### Process

Once requirements are approved, create a detailed implementation plan covering:

1. **Overview**:
   - Feature summary and goals
   - Key requirements recap
   - Success criteria

2. **Technical Approach**:
   - Architecture and design decisions
   - Technology stack and tools
   - Component breakdown
   - Data models and schemas
   - API design (if applicable)

3. **Implementation Steps**:
   - Ordered task breakdown
   - Dependencies between tasks
   - Estimated complexity/effort
   - Risk areas and mitigation

4. **Testing Strategy**:
   - Unit test approach
   - Integration test scenarios
   - Edge cases and error handling
   - Validation criteria

5. **Considerations**:
   - Security implications
   - Performance concerns
   - Scalability factors
   - Maintenance and documentation

### Plan Quality Criteria

Ensure the plan:

- [ ] Addresses all discussed requirements
- [ ] Provides clear, actionable steps
- [ ] Identifies technical decisions and rationale
- [ ] Includes comprehensive testing approach
- [ ] Considers edge cases and error handling
- [ ] Is implementable with available tools/stack

## Phase 3: Codex Review and Refinement

### Process

1. **Submit for Review**:
   - Use the Task tool with codex-analyzer agent
   - Provide the complete plan for comprehensive review
   - Request feedback on:
     - Completeness and clarity
     - Technical soundness
     - Potential issues or gaps
     - Security considerations
     - Performance implications
     - Implementation approach

2. **Review Integration**:
   - Carefully read Codex's feedback
   - Integrate suggested improvements
   - Address identified concerns
   - Refine ambiguous sections
   - Add missing considerations

3. **Iterative Refinement**:
   - Submit revised plan for follow-up review if significant changes made
   - Continue until Codex indicates plan is comprehensive and ready
   - Maximum 3 review iterations (typically 1-2 sufficient)

4. **Final Validation**:
   - Present refined plan to user
   - Highlight key changes from Codex review
   - Ask: "Plan has been reviewed and refined with Codex. Ready to finalize and save?"

### Codex Agent Usage

```markdown
Use the Task tool with:

- subagent_type: codex-analyzer
- description: "Review implementation plan"
- prompt: "Please review this implementation plan for [feature name]:

[FULL PLAN TEXT]

Provide feedback on:

1. Completeness - Are there any missing considerations or steps?
2. Technical soundness - Are the architectural decisions appropriate?
3. Security - Are there security implications that need addressing?
4. Performance - Are there potential performance issues?
5. Implementation approach - Are the steps clear and actionable?
6. Edge cases - Are error scenarios properly handled?
7. Testing - Is the test strategy comprehensive?

Be thorough and critical. Identify any issues, gaps, or improvements needed."
```

## Phase 4: Plan Documentation

### Process

1. **Generate Filename**:
   - Format: `specs/{YYYY-MM-DD}-{feature-slug}.md`
   - Use today's date from system
   - Convert feature name to kebab-case slug
   - Example: `specs/2025-10-12-user-authentication.md`

2. **Create specs/ Directory**:
   - Check if specs/ exists in project root
   - Create if missing using Write tool

3. **Write Plan Document**:
   - Use Write tool to create the markdown file
   - Include complete plan with all sections
   - Add metadata header:
     ```markdown
     # [Feature Name]

     **Date**: YYYY-MM-DD
     **Status**: Planned
     **Reviewed by**: Codex

     ---

     [Plan content...]
     ```

4. **Confirmation**:
   - Provide full path to created file
   - Summarize next steps for implementation
   - Suggest relevant implementation commands if applicable

## Example Workflow

```
User: /plan user authentication system

Assistant (Phase 1):
"I understand you want to implement user authentication. Let me clarify the requirements:

1. What authentication methods? (email/password, OAuth, JWT tokens?)
2. What user roles/permissions are needed?
3. Should it integrate with existing systems?
4. What are the security requirements?

Please provide details on these aspects."

User: "JWT-based auth with email/password. Just basic user/admin roles. No external integrations."
Assistant (continued):
"Got it. So to summarize:
- JWT-based authentication with email/password login
- Two role types: user and admin
- No external OAuth providers
- Standard security practices (hashing, token expiration)

Do I understand correctly? Should I proceed to create the plan?"

User: "OK"
Assistant (Phase 2):
[Creates detailed implementation plan with all sections...]

Assistant (Phase 3):
[Uses Task tool with codex-analyzer to review plan]

Codex: "Plan looks solid. Consider adding:
- Rate limiting for login attempts
- Email verification step
- Password reset functionality
..."
Assistant (continued Phase 3):
[Integrates feedback, creates refined plan]

"Plan has been reviewed and refined with Codex. Added rate limiting, email verification, and password reset. Ready to finalize and save?"

User: "Yes"

Assistant (Phase 4):
[Creates specs/2025-10-12-user-authentication.md with complete plan]

"Plan saved to specs/2025-10-12-user-authentication.md

Next steps:
- Review the plan in the specs/ directory
- Use /spec:init or begin implementation directly
- Consider breaking into smaller tasks if needed"
```

## Important Notes

### Context Awareness

- Review codebase structure to align with existing patterns
- Consider project's tech stack and conventions
- Identify reusable components or libraries

### Quality Checklist

Before sending plan to Codex:

- [ ] All requirements from Phase 1 are addressed
- [ ] Technical approach is clearly explained
- [ ] Implementation steps are ordered logically
- [ ] Security considerations are included
- [ ] Testing strategy is comprehensive
- [ ] Edge cases and errors are handled

### Codex Review Best Practices

- Provide complete plan text, not summaries
- Be specific in review request
- Take feedback seriously - Codex often catches important issues
- Don't skip review even if plan seems solid
- If multiple iterations needed, show what changed

### File Organization

- Plans go in `specs/` at project root
- Use consistent date format (YYYY-MM-DD)
- Use kebab-case for feature names
- Include status metadata for tracking

## Tips

1. **Be Patient in Phase 1**: Better requirements = better plan
2. **Don't Rush Plan Creation**: Take time to think through all aspects
3. **Trust Codex Feedback**: Expert review catches issues you might miss
4. **Keep Plans Actionable**: Avoid vague statements, be specific
5. **Update Plans**: If implementation deviates, update the spec file

## Troubleshooting

**Issue**: User keeps changing requirements
**Solution**: Take more time in Phase 1, ask comprehensive questions upfront

**Issue**: Codex review is too broad
**Solution**: Be more specific in review prompt about areas of concern

**Issue**: Plan is too high-level
**Solution**: Add more technical detail in Phase 2, specify exact libraries/approaches

**Issue**: Specs directory doesn't exist
**Solution**: Command will create it automatically when saving plan

Now begin the planning process for: **$ARGUMENTS**
