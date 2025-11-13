# cc-plugins

A curated collection of Claude Code plugins and MCP servers for enhanced development workflows.

## Overview

This repository provides two types of development tools:

- **Claude Code Plugins**: Extend Claude Code with custom slash commands and workflows
- **MCP Servers**: Standalone Model Context Protocol servers for tool integration

## Plugin Marketplace

### Installation

Install this plugin marketplace in Claude Code:

```bash
/plugin marketplace add vaayne/cc-plugins
```

### Available Plugins

#### [specs-dev](./plugins/specs-dev/)

Spec-driven development with Codex review. Features AI-reviewed planning, human approval gates, and structured implementation workflow.

**Commands:** `/specs-dev:plan`, `/specs-dev:impl`

## MCP Servers

Independent MCP servers that can be used with any MCP client.

### [mcp-fs](./mcps/mcp-fs/)

Unified file system access across multiple backends (S3, WebDAV, FTP, local).

```bash
uvx mcp-fs "fs://"  # Local filesystem
```

### [mcp-executor](./mcps/mcp-executor/)

MCP tool discovery and execution with CLI interface.

```bash
uvx mcp-executor list -c mcp.json
```

## Development

### Prerequisites

- [mise](https://mise.jdx.dev/) - Development environment management
- [uv](https://github.com/astral-sh/uv) - Python package management

### Quick Start

```bash
git clone https://github.com/vaayne/cc-plugins.git
cd cc-plugins
mise install
```

### Project Structure

```
cc-plugins/
├── plugins/          # Claude Code plugins
├── mcps/            # MCP servers
├── .claude-plugin/  # Marketplace metadata
├── dprint.json      # Code formatting
└── mise.toml        # Task automation
```

See individual directories for detailed development guides.

### Code Quality

```bash
# Format all code
mise run fmt
```

## Contributing

- **Plugins**: Add to `plugins/` with README documentation
- **MCP Servers**: Add to `mcps/` with standalone README
- Run `mise run fmt` before committing
- See subdirectories for specific contribution guidelines

## License

[MIT License](./LICENSE)
