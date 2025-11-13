# MCP-FS Server

An MCP server that provides unified access to multiple file systems simultaneously through OpenDAL.

## Installation

```bash
pip install mcp-fs
```

## Quick Start

### Single Backend

```bash
# Local filesystem
mcp-fs "fs://"

# S3
mcp-fs --transport http "s3://bucket?region=us-east-1&access_key_id=xxx&secret_access_key=yyy"

# WebDAV
mcp-fs "webdav://server.com/path?username=user&password=pass"

# Memory (testing)
mcp-fs "memory://"
```

### Multi-Backend with Config File

Create `backends.json`:

```json
{
  "backends": [
    {
      "name": "local",
      "url": "fs://",
      "description": "Local filesystem",
      "default": true
    },
    {
      "name": "s3-prod",
      "url": "s3://bucket?region=us-east-1&access_key_id=...",
      "description": "Production S3"
    }
  ]
}
```

Run with config:

```bash
# Stdio (default)
mcp-fs backends.json

# HTTP
mcp-fs --transport http --config backends.json --port 8080
```

## Usage

### Command Options

```bash
mcp-fs [OPTIONS] [URL_OR_CONFIG]

Options:
  --config FILE         JSON configuration file
  --transport TYPE      stdio (default) or http
  --port PORT          HTTP port (default: 8000)
  --host HOST          HTTP host (default: localhost)
```

### Available Tools

**File Operations:**

- `list_files(path, backend=None)` - List files and directories
- `read_file(path, backend=None)` - Read file contents
- `write_file(path, content, backend=None)` - Write to file
- `copy_file(src, dst, src_backend=None, dst_backend=None)` - Copy files
- `rename_file(src, dst, backend=None)` - Rename/move files
- `create_dir(path, backend=None)` - Create directory
- `stat_file(path, backend=None)` - Get file metadata

**Backend Management:**

- `register_backend(name, url, ...)` - Add new backend
- `list_backends()` - Show all backends
- `set_default_backend(name)` - Set default backend
- `remove_backend(name)` - Remove backend
- `check_backend_health(backend=None)` - Check connectivity

## Supported Backends

| Type   | URL Example                                            |
| ------ | ------------------------------------------------------ |
| Local  | `fs://`                                                |
| S3     | `s3://bucket?region=us-east-1&access_key_id=...`       |
| WebDAV | `webdav://server.com/path?username=user&password=pass` |
| Memory | `memory://`                                            |
| FTP    | `ftp://server.com?username=user&password=pass`         |
| HTTP   | `https://api.example.com`                              |

## Examples

### Cross-Backend Copy

```python
Found 2 errors (2 fixed, 0 remaining).
```

### Runtime Backend Management

```python
All checks passed!
```

## Development

### Development Mode (with uv)

Using uv is recommended for development as it provides fast dependency management and isolated environments:

```bash
# Clone the repository
git clone https://github.com/vaayne/cc-plugins
cd cc-plugins/mcps/mcp-fs

# Install dependencies and create virtual environment
uv sync

# Run in development mode (uses local source code)
uv run mcp-fs "memory://"

# Run with config file
uv run mcp-fs examples/demo_config.json

# Run with HTTP transport
uv run mcp-fs --transport http "memory://"

# Run tests
uv run pytest

# Format and lint code
uv run ruff format src/
uv run ruff check src/

# Type checking
uv run mypy src/
```

### Production Mode (with uv)

For production deployments using uv:

```bash
# Install from PyPI
uv pip install mcp-fs

# Or install from source
uv pip install .

# Build package
uv build

# Run the installed package
mcp-fs "fs://"
mcp-fs --transport http --config production.json --port 8080
```

### Traditional Installation (pip)

```bash
# Install from PyPI
pip install mcp-fs

# Or install from source
git clone https://github.com/vaayne/cc-plugins
cd cc-plugins/mcps/mcp-fs
pip install .
```

## License

MIT
