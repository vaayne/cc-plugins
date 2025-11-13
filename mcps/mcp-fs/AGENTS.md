# Repository Guidelines

## Project Structure & Module Organization

- `src/mcp_fs/` hosts the MCP server logic (`cli.py`, `backend_manager.py`, `mcp_server.py`, `dal.py`). Keep new modules scoped here and mirror OpenDAL concepts when possible.
- `examples/` contains runnable configs such as `demo_config.json`; add new samples here so they stay out of production packages.
- `dist/` and `uv.lock` are build outputs/locks; never hand-edit them.
- Root assets like `README.md`, `LICENSE`, and `pyproject.toml` define public docs, licensing, and tooling. Update these in tandem when changing surface behavior.

## Build, Test, and Development Commands

- `uv sync` — create the virtual env and install dev dependencies.
- `uv run mcp-fs "memory://"` — run the server locally against the in-memory backend for fast smoke tests.
- `uv run mcp-fs examples/demo_config.json` — exercise multi-backend routing via the sample config; adjust paths before committing.
- `uv run pytest` — execute the full test suite (add `tests/` first if needed).
- `uv run rr lint` — invokes the `pyproject-runner` task which formats and lints via Ruff in fix mode; run this before every commit.
- `uv run rr build` — runs the configured build task to produce sdist/wheel under `dist/`.
- `PYPI_API_TOKEN=... uv run rr publish` — publishes the package via the task wrapper; use test tokens for dry runs.

## Coding Style & Naming Conventions

- Python 3.12+, 4-space indentation, 88-char line length enforced by Ruff; use double quotes per `[tool.ruff.format]`.
- Favor descriptive module-level functions/classes (`BackendManager`, `McpServer`) and snake_case for functions/variables.
- Keep public CLI options mirrored in `README.md` examples; document new flags immediately.

## Testing Guidelines

- Pytest auto-discovers files under `tests/` matching `test_*.py`; create the folder if adding suites.
- Organize tests to mirror `src/mcp_fs` modules and prefer parametrized cases over loops.
- Aim to cover backend registration, OpenDAL interactions, and CLI parsing logic whenever they change.

## Commit & Pull Request Guidelines

- Follow emoji Conventional Commits (e.g., `✨ feat: add r2 backend`, `🐛 fix: handle empty key`). Scope each commit narrowly.
- Reference related issues in the commit body or PR description and describe user-visible effects plus test evidence.
- PRs should include repro steps, updated docs/configs, and screenshots or logs for UX-facing changes.

## Security & Configuration Tips

- Never commit backend credentials; rely on env vars or redacted `backends.json` examples.
- Validate new transports against `register_backend` safeguards and document any required auth parameters in `examples/`.
