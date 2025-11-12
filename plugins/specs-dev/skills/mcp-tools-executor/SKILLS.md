---
name: mcp-tools-executor
description: Generate and execute CLI tools from MCP (Model Context Protocol) servers. Use this skill when needing to interact with MCP tools via command-line interfaces, including generating tools from server configurations and executing them with proper arguments.
---

# MCP Tool Executor

## Purpose

This skill provides a systematic workflow for generating executable CLI tools from MCP server configurations and guides their proper usage. It transforms MCP server tool definitions into standalone Python scripts with Click-based command-line interfaces, making MCP tools accessible through standard shell commands.

## When to Use This Skill

Use this skill when:

1. Needing to interact with MCP (Model Context Protocol) tools via command-line interfaces
2. Setting up a new MCP server and generating its corresponding CLI tools
3. Discovering what MCP tools are available in the current environment
4. Determining correct arguments and usage patterns for MCP tools
5. Executing MCP tool operations through Python CLI scripts

## Tool Generation Workflow

### Initial Setup

Before using any MCP tools, verify that CLI tools have been generated:

1. Check for the existence of the `tools/` directory in the skill folder
2. Check if `tools/tools.md` exists and contains tool documentation
3. If either is missing or empty, tools must be generated first

### Generating Tools from MCP Servers

To generate CLI tools from MCP server configurations:

1. **Configure MCP Servers**: Instruct the user to edit `scripts/mcp.json` with their MCP server configurations. Reference `scripts/mcp.json.example` for the configuration format, which supports:
   - HTTP transport with URL and authentication headers
   - STDIO transport with command, arguments, and environment variables

2. **Run Generation Script**: Execute the tool generation script:
   ```bash
   uv run scripts/init_tools.py
   ```

3. **Verify Generation**: After successful execution, the script will:
   - Generate individual Python CLI scripts in the `tools/` directory (named `{server}__{tool}.py`)
   - Create a `tools.md` file documenting all available tools
   - Make all generated scripts executable
   - Apply code formatting via ruff

### Tool Discovery

To discover available MCP tools, load the tools documentation into context:

```
@tools/tools.md
```

This file contains:
- Complete list of all available MCP tools organized by server
- Tool names, descriptions, and corresponding script filenames
- General usage instructions

**Important**: Always reference `tools.md` to understand which tools are available before attempting to use them.

## Tool Execution Workflow

### Understanding Tool Usage

To learn how to use a specific MCP tool:

1. **Check Help Documentation**: Run the tool's help command:
   ```bash
   uv run tools/{server}__{tool}.py --help
   ```

2. **Analyze Output**: The help output reveals:
   - Tool description and purpose
   - Required arguments (positional parameters)
   - Optional flags and their default values
   - Parameter types and constraints
   - Boolean flags (--flag/--no-flag syntax)

### Executing Tools

To execute an MCP tool with the discovered arguments:

```bash
uv run tools/{server}__{tool}.py [ARGUMENTS] [OPTIONS]
```

**Examples**:
```bash
# Tool with a single required argument
uv run tools/chrome-devtools__click.py element_123

# Tool with arguments and options
uv run tools/chrome-devtools__click.py element_123 --dbl_click

# Tool with multiple options
uv run tools/jetbrains__search_in_files_by_text.py "search query" --case_sensitive
```

### Tool Script Structure

Each generated tool script:
- Is a standalone Python executable using uv's inline script metadata
- Uses Click for command-line argument parsing
- Contains embedded MCP server configuration
- Connects to the MCP server and invokes the specified tool
- Returns results as JSON output

## Best Practices

1. **Always Generate First**: Ensure tools are generated before attempting to use them
2. **Reference Documentation**: Load `@tools/tools.md` to see available tools
3. **Check Help First**: Run `--help` before using an unfamiliar tool to understand its interface
4. **Update When Needed**: Regenerate tools after modifying `scripts/mcp.json`
5. **Handle Errors**: If a tool fails, verify:
   - The MCP server is properly configured and accessible
   - Required arguments are provided with correct types
   - The server is running (for STDIO-based servers)

## Workflow Summary

**Complete workflow for using MCP tools:**

1. Check if `tools/` directory and `tools.md` exist
2. If missing, instruct user to configure `scripts/mcp.json` and run `uv run scripts/init_tools.py`
3. Load `@tools/tools.md` to discover available tools
4. For any tool to be used, run `uv run tools/{filename}.py --help` to understand arguments
5. Execute the tool with appropriate arguments: `uv run tools/{filename}.py [args] [options]`
6. Parse and utilize the JSON output returned by the tool

This systematic approach ensures proper tool discovery, understanding, and execution of MCP tools through their generated CLI interfaces.
