---
name: {skill-name}
description: {description}
---
# {Title}

MCP service at `{url}` ({transport}) with {tool-count} tools.

## Tools
{tools-list}

## Usage
List tools: `hub -s {url} -t {transport} list`
Get tool details: `hub -s {url} -t {transport} inspect <tool-name>`
Invoke tool: `hub -s {url} -t {transport} invoke <tool-name> '{"param": "value"}'`

## Notes
- Run `inspect` before invoking unfamiliar tools to get full parameter schema
- Timeout: 30s default, use `--timeout <seconds>` to adjust
