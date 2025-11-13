# Repository Guidelines

## Project Structure & Module Organization

- Core package lives in `src/mcp_executor`, with `cli.py` exposing the Click commands and `runtime.py` wiring FastMCP transports.
- Distributable metadata and tasks sit in `pyproject.toml`; sample client config ships as `mcp.json.example` and should be copied into your own config dir before invoking the CLI.
- Build artifacts drop into `dist/`; keep temporary configs or notebooks outside `src/` to avoid polluting the wheel.
- Tests default to `tests/` per `pyproject.toml`; mirror the module path (e.g., `tests/test_cli.py`) when adding coverage.

## Build, Test, and Development Commands

- `uv venv --seed && uv sync` — create the Python 3.12+ environment and install deps.
- `uv run mcp-executor list -c mcp.json.example` — smoke-test tool discovery against the sample config.
- `uv run pytest` — execute the full test suite (add `tests/` first if needed).
- `uv run rr lint` — invokes the `pyproject-runner` task which formats and lints via Ruff in fix mode; run this before every commit.
- `uv run rr build` — runs the configured build task to produce sdist/wheel under `dist/`.
- `PYPI_API_TOKEN=... uv run rr publish` — publishes the package via the task wrapper; use test tokens for dry runs.

## Coding Style & Naming Conventions

- Follow Ruff’s defaults: 4-space indentation, 88-char lines, double quotes, and `snake_case` for functions/variables; keep Click command names kebab-cased (`serve-http`).
- Maintain type hints throughout (`mypy` runs in strict-ish mode with `disallow_untyped_defs`).
- Isolate FastMCP helpers into small, composable functions; favor module-level `async` helpers for transport orchestration.

## Testing Guidelines

- Author pytest modules named `test_<feature>.py` with functions `test_<behavior>`; mirror CLI command names for clarity.
- Use fixtures to stub MCP tool registries; assert both stdout text and HTTP responses where applicable.
- Target meaningful coverage for CLI parsing, FastMCP server lifecycle, and config loading; prefer black-box tests that exercise Click entrypoints.

## Commit & Pull Request Guidelines

- Use emoji-prefixed Conventional Commits (e.g., `✨ feat: add websocket transport`, `🐛 fix: handle empty tool list`).
- Reference related issues in the body (`Fixes #123`) and summarize validation (`uv run pytest`, `uv build`).
- PRs should describe the scenario, list testing evidence, and include screenshots or terminal captures for user-facing changes.
- Keep diffs focused; open follow-up PRs for refactors unrelated to the main fix.
