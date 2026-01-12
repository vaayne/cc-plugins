---
name: { skill-name }
description: { description }
---

# {Title}

MCP service at `{url}` ({transport}) with {tool-count} tools.

## Requirements

- `hub` CLI must be installed. If not available, install with:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/vaayne/cc-plugins/main/mcps/hub/scripts/install.sh | sh
  ```

## Usage

List tools: `hub -s {url} -t {transport} list`
Get tool details: `hub -s {url} -t {transport} inspect <tool-name>`
Invoke tool: `hub -s {url} -t {transport} invoke <tool-name> '{"param": "value"}'`

## Notes

- Run `inspect` before invoking unfamiliar tools to get full parameter schema
- Timeout: 30s default, use `--timeout <seconds>` to adjust

## Tools

{tools-list}
