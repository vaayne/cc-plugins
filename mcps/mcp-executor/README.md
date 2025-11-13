# mcp-executor

`mcp-executor` is a lightweight FastMCP driver that discovers every MCP tool
exposed by your configured backends, prints Python-like signatures, and lets you
invoke them either from the command line or through a minimal FastMCP server.

## Why use it?

- 📋 **Discoverability**: Generate readable signatures for every tool so you know
  which arguments to pass.
- ⚡ **Fast iteration**: Jump between running the HTTP/stdio server and issuing
  one-off tool calls without reconfiguring clients.
- 🧰 **Batteries included**: Ships with a `uv`-first workflow for building,
  testing, and publishing to PyPI.

## Installation

```bash
pip install mcp-executor
```

or, if you prefer the `uv` toolchain:

```bash
uv tool install mcp-executor
```

## Configuration

Point the CLI at an MCP client definition. Copy `mcp.json.example` into your
workspace, adjust the upstream server definitions, and pass the path with
`-c/--config`.

```bash
cp mcp.json.example ~/.config/mcp-executor/mcp.json
```

## Usage

List available tools and inspect their signatures:

```bash
mcp-executor list -c ~/.config/mcp-executor/mcp.json
```

Call a tool directly:

```bash
mcp-executor call -c ~/.config/mcp-executor/mcp.json weather --arg city="Lisbon"
```

Run the FastMCP server (HTTP by default):

```bash
mcp-executor serve -c ~/.config/mcp-executor/mcp.json --transport http --host 0.0.0.0 --port 23456
```

Every command shares the `--config` option so you can point at different MCP
client definitions per invocation.

## Local development

```bash
uv venv --seed
uv sync
uv run mcp-executor list -c mcp.json.example
```

The repository still ships `main.py` so you can run `./main.py list ...` directly
with `uv run` if you prefer scripting locally.

## Releasing with `uv`

1. Bump `version` inside `pyproject.toml`.
2. Build and verify the artifacts:
   ```bash
   uv build
   uv run python -m mcp_executor.cli --help
   ```
3. Publish to PyPI with an API token (create one under your PyPI account and
   store it securely):
   ```bash
   export PYPI_API_TOKEN="pypi-xxxxxxxxxxxxxxxxxxxxxxxx"
   uv publish --token "$PYPI_API_TOKEN"
   ```
4. Tag the release in git and push (`git tag v0.1.1 && git push --tags`).

## GitHub Action publish workflow

The repository includes `.github/workflows/publish.yml`, which builds the wheel
and sdist via `uv build` and calls `uv publish --token $PYPI_API_TOKEN`. Add a
`PYPI_API_TOKEN` secret (scoped to “Publish to PyPI”) in your repository settings
and trigger the workflow from the Actions tab or by creating a GitHub Release.
