Inspect a specific MCP tool to get its full schema.

Use this tool when you need detailed information about a single tool, including its complete input schema with all parameters, types, and constraints.

## When To Use

- When you know the tool name and need its full schema before calling it
- When `list` output is truncated and you need complete parameter details
- To verify exact parameter names and types before using `exec`

## Usage

Provide the namespaced tool name in `serverID__toolName` format:

```json
{ "name": "github__search_repos" }
```

## Output Format

Returns JSON with full tool details:

```json
{
  "name": "github__search_repos",
  "description": "Search GitHub repositories",
  "server": "github",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Search query"
      },
      "limit": {
        "type": "number",
        "description": "Max results to return"
      }
    },
    "required": ["query"]
  }
}
```
