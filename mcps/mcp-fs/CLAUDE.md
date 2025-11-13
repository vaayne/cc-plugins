# Development Guide

## Commands

```bash
# Install dependencies
uv sync

# Run tests (NOTE: No tests currently exist)
uv run pytest

# Run server locally
uv run mcp-fs "memory://"
uv run mcp-fs --transport http "memory://"

# Run with config
uv run mcp-fs examples/demo_config.json
uv run mcp-fs --transport http --config examples/demo_config.json --port 8080

# Build package
uv build

# Code quality (NOTE: No linting/formatting configured yet)
# Consider adding: ruff check src/ && ruff format src/ && mypy src/
```

## Architecture

**Core Components:**
- `mcp_server.py` - MCP server with 13 tools (7 file ops + 6 backend management)
- `backend_manager.py` - Multi-backend management with health monitoring
- `dal.py` - OpenDAL abstraction layer

**Key Features:**
- Multi-backend support (local, S3, WebDAV, FTP, HTTP, memory, etc.)
- Dual transport: stdio (local) and HTTP (remote)
- Backward compatible single-backend mode
- Read-only backend protection
- Runtime backend registration/removal

## Testing

```bash
# Memory backend for testing
uv run mcp-fs "memory://"

# Multi-backend test config
echo '{
  "backends": [
    {"name": "memory1", "url": "memory://", "default": true},
    {"name": "memory2", "url": "memory://"}
  ]
}' > test_config.json

uv run mcp-fs test_config.json
```

## Development Tasks

**Add Backend Types:** Ensure OpenDAL supports it, test connection validation

**Debug Issues:** Set `logging.basicConfig(level=logging.DEBUG)` for detailed logs

**Build System:** Uses hatchling (modern PEP 517/621 compliant)