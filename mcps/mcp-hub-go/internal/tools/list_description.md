List available tools from connected MCP servers.

Returns tool name, description, server, and inputSchema for each tool.

## Usage

1. Call list to discover available tools (optionally filter by server or keywords)
2. Review inputSchema to understand required parameters
3. Use exec with mcp.callTool("serverID.toolName", params) to call tools

## Examples

List all tools:
  {} (no parameters)

Filter by server:
  {"server": "github"}

Search with keywords:
  {"query": "file,read"}  // matches tools containing either "file" or "read" in name or description

Combine filters:
  {"server": "fs", "query": "write,delete"}
