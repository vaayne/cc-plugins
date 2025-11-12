# MCP Tools Executor Skill

Generate and execute CLI tools from MCP (Model Context Protocol) servers.

**What it does:**
- Transforms MCP server tool definitions into standalone Python CLI scripts with Click-based interfaces
- Provides systematic workflow for tool discovery, generation, and execution
- Generates a `tools.md` documentation file listing all available MCP tools

**Key workflow:**
1. Configure MCP servers in `scripts/mcp.json`
2. Run `uv run scripts/init_tools.py` to generate CLI tools
3. Load `@tools/tools.md` to discover available tools
4. Run `uv run tools/{filename}.py --help` to learn tool usage
5. Execute tools with proper arguments

**Use when:**
- Needing to interact with MCP tools via command-line interfaces
- Setting up new MCP servers and their corresponding CLI tools
- Discovering what MCP tools are available in the environment
- Executing MCP tool operations through Python scripts

See `mcp-tools-executor/SKILLS.md` for complete documentation.
