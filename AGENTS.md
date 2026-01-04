# Agent Instructions

## Build/Lint/Test Commands

- `mise run format` — Format all code (ruff for Python, dprint for TS/JSON/YAML/MD)

## Architecture

- `plugins/` — Claude Code plugins
- `mcps/` — MCP servers
- `skills/` — Agent skills (codex-analyze, gemini-analyze, python-script, specs-dev)

## Marketplace

- When adding new plugins or skills, update `.claude-plugin/marketplace.json` to register them

## Code Style

- **Python:** Python 3.12+, ruff (88-char lines, double quotes), snake_case, type hints required
- **TypeScript:** Strict mode, camelCase functions, PascalCase types, Zod for validation, biome for linting
- **Go:** Standard gofmt, internal/ for private packages
- **Commits:** Emoji Conventional Commits (e.g., `✨ feat:`, `🐛 fix:`, `♻️ refactor:`)
- Never commit secrets; use env vars for credentials
