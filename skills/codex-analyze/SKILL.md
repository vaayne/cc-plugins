---
name: codex-analyze
description: Orchestrate Codex CLI for comprehensive code analysis (bugs, security, performance, and quality). Use when deep analysis is requested or when the scope is large; ask for scope if unclear.
---

# Codex Analyzer

## Overview
Drive Codex CLI to perform comprehensive code analysis and return prioritized findings with actionable fixes.

## Workflow
1. Clarify scope (directory, files, or subsystem) and analysis type.
2. Build a detailed prompt with objectives and relevant context.
3. Run: `codex --cd "{dir}" exec "{prompt}"`.
4. Summarize findings by severity and provide fix guidance.
5. Offer follow-up analysis for specific files or issue types.

## Prompt ingredients
- Analysis objectives (bugs, security, performance, quality)
- Tech stack and framework context
- Business logic or domain constraints
- Specific areas of concern from the user
- Request for prioritized findings and severity labels

## Output expectations
- Summarize key findings in order of severity
- Explain impact and recommended fixes
- Note systemic patterns or preventative measures
