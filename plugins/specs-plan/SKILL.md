---
name: specs-plan
description: Spec-first planning workflow that produces an approved plan.md before implementation, with reviewer subagent feedback and progress tracking. Use when a user asks for a plan-before-code process, a review-gated implementation plan, or a single plan.md that includes tasks and ongoing progress updates, with commits after each phase.
---

# Specs Plan

## Overview

Create a spec-first plan before implementation, with explicit approval gates and reviewer subagent feedback. Maintain a single plan.md that includes tasks and progress tracking.

## Workflow

### Phase 1 - Requirements discussion

- Interpret the request and restate goals, scope boundaries, success criteria, constraints, and risks.
- Ask targeted questions to resolve ambiguity.
- Summarize the final requirements and ask for approval to proceed to planning.
- Gate A: Do not create or edit files until the user approves the requirements summary.

### Phase 2 - Plan drafting and reviewer approval (no files yet)

- Draft the plan in chat using the template below.
- Split work into concrete tasks in the draft plan.
- Send the full draft plan to the reviewer subagent.
- Incorporate reviewer feedback and iterate until the reviewer approves (max three passes).
- Do not create or edit files in this phase.

### Phase 3 - User approval and plan.md creation

- Present the reviewer-approved plan to the user.
- Ask the user to approve the plan.
- Gate B: Only after user approval, create or update plan.md at `.agents/sessions/{YYYY-MM-DD}-{feature}/plan.md` (use the current date, e.g., run `date +%Y-%m-%d` to obtain it).

## plan.md template

Use this structure and keep it concise. Prefer subsections and short paragraphs over bullet-only sections when detail is needed.

```markdown
# Plan: <feature name>

## Implementation rules

> **MUST follow these rules strictly during implementation.**

1. Implement each phase in a dedicated subagent to preserve context.
2. After each implementation phase, request reviewer subagent feedback.
3. Incorporate reviewer feedback, then ask for reviewer approval.
4. Once reviewer-approved, commit code changes and update this plan (status + notes).

## Overview

### Goal

<paragraph>

### Success criteria

- ...

### Non-goals

- ...

## Requirements

- ...

## Technical approach

### Architecture

<paragraphs and subheadings as needed>

### Data model or APIs

<paragraphs, schemas, or tables as needed>

### Integrations

<paragraphs and diagrams/links if available>

### Risks and mitigations

- Risk:
  - Mitigation:

## Implementation phases

### Phase 1 - <phase name>

**Status:** not started | in progress | done

**Tasks**

- [ ] T1:

**Progress log**

- YYYY-MM-DD: ...

**Notes**

- ...

### Phase 2 - <phase name>

**Status:** not started | in progress | done

**Tasks**

- [ ] T1:

**Progress log**

- YYYY-MM-DD: ...

**Notes**

- ...

### Phase 3 - <phase name>

**Status:** not started | in progress | done

**Tasks**

- [ ] T1:

**Progress log**

- YYYY-MM-DD: ...

**Notes**

- ...

### Phase xxx

...
```
