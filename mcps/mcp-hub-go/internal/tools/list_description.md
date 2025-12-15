List available tools from connected MCP servers.

Returns JavaScript function stubs with JSDoc for each tool, making it easy to understand the API when writing code for the exec tool.

## Usage

1. Call list to discover available tools (optionally filter by server or keywords)
2. Review the JSDoc comments to understand required parameters
3. Use exec with mcp.callTool("serverID__toolName", params) to call tools

## Examples

List all tools:
  {} (no parameters)

Filter by server:
  {"server": "github"}

Search with keywords:
  {"query": "file,read"}  // matches tools containing either "file" or "read" in name or description

Combine filters:
  {"server": "fs", "query": "write,delete"}

## Avaliable Tools

{{AVAILABLE_TOOLS}}

## Output Format

The output is JavaScript function stubs with JSDoc comments:

```javascript
// Total: 2 tools

/**
 * List files in a directory
 * @param {Object} params - Parameters
 * @param {string} params.path - Directory path to list
 */
function filesystem__list_directory(params) {}

/**
 * Read file contents
 * @param {Object} params - Parameters
 * @param {string} params.path - File path to read
 */
function filesystem__read_file(params) {}
```
