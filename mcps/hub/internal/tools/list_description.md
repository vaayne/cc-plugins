List MCP tools available through this hub.

Use this tool when you want the **full tool signature** (name, description, and input params) in JavaScript/JSDoc format so you can call tools correctly via `exec`.

## When To Use

- After picking a tool from **Avaliable Tools**, call `list` to get full JSDoc + parameter details.
- Call `list` once to get **multiple tools** at the same time (it returns a combined set of stubs). Use `server` / `query` to fetch the batch you need.
- If tools may have changed, call `refreshTools` first, then `list`.

## Usage

- `{}` lists tools from all connected servers.
- `{"server":"github"}` filters to a single server ID.
- `{"query":"file,read"}` filters by keywords (matches name or description) and returns all matching tools in one response.

## Examples

List all tools:
{} (no parameters)

Filter by server:
{"server": "github"}

Search with keywords:
{"query": "file,read"} // matches tools containing either "file" or "read" in name or description

Combine filters:
{"server": "fs", "query": "write,delete"}

## Avaliable Tools

{{AVAILABLE_TOOLS}}

## Output Format

The output is JavaScript function stubs with JSDoc comments. Use the namespaced tool name (`serverID__toolName`) with `mcp.callTool(...)`:

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
