---
name: gemini-analyze
description: Manage Gemini CLI for large codebase analysis and pattern detection. Use when a broad architectural overview or large-scale pattern search is needed; run Gemini CLI and return raw output.
---

# Gemini Analyzer

## Overview
Act as a CLI wrapper for Gemini: construct prompts, run the command, and return raw results without interpretation.

## Workflow
1. Clarify the analysis goal and scope.
2. Choose appropriate flags (use `--all-files` for broad scans; consider `--yolo` for non-destructive analysis).
3. Craft a focused prompt with the exact patterns or concerns.
4. Execute Gemini CLI and return the full output verbatim.
5. Do not interpret or analyze the results.

## Command pattern
- `gemini --all-files -p "{prompt}"`

## Example prompts
- "Identify authentication flows, token handling, and access control patterns."
- "List all database query patterns and connection handling."

## Rules
- Act only as the CLI runner.
- Always return complete, unfiltered output.
- Defer interpretation to the caller.
