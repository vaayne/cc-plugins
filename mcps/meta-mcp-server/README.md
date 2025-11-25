# Meta MCP Server

A Meta MCP Server that aggregates multiple external MCP servers and exposes their tools through a unified search and code execution interface.

## Overview

Instead of exposing all downstream MCP tools directly to LLMs, this server:

1. **Collects** tool definitions from multiple external MCP servers
2. **Generates** TypeScript wrapper code for each tool with Zod schemas
3. **Indexes** the generated code for full-text (BM25) and regex search
4. **Exposes** only 3 high-level tools:
   - `meta_search_tools`: Search for tools by keyword or pattern
   - `meta_eval_ts`: Execute TypeScript code that imports and combines tools
   - `meta_refresh_tools`: Refresh tool definitions from external servers

This approach reduces LLM context consumption by collapsing many tools into "search + execute" capabilities.

## Installation

```bash
cd mcps/meta-mcp-server
npm install
npm run build
```

## Usage

### Configuration

Create a configuration file (e.g., `config.json`):

```json
{
  "servers": [
    {
      "id": "serverA",
      "name": "User Service",
      "transport": "http",
      "endpoint": "http://localhost:4001/mcp"
    },
    {
      "id": "serverB",
      "name": "Order Service",
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "order-mcp-server"]
    }
  ],
  "toolsOutputDir": "./src/tools",
  "searchIndexPath": "./search-index"
}
```

### Running

**stdio transport (for local use):**
```bash
node dist/index.js -c config.json
```

**HTTP transport (for remote access):**
```bash
node dist/index.js -c config.json -t http -p 3000
```

### MCP Tools

#### `meta_search_tools`

Search for available TypeScript tool wrappers.

**Input:**
```json
{
  "query": "search user email",
  "mode": "auto",
  "limit": 10
}
```

**Output:**
```json
{
  "items": [
    {
      "id": "serverA.searchUser",
      "serverId": "serverA",
      "toolName": "searchUser",
      "functionName": "searchUser",
      "filePath": "./src/tools/serverA/search-user.ts",
      "description": "Search user by email",
      "snippet": "export async function searchUser(input: SearchUserInput)..."
    }
  ],
  "total": 1
}
```

#### `meta_eval_ts`

Execute TypeScript code that imports and uses tool wrappers.

**Input:**
```json
{
  "code": "import { searchUser } from \"@tools/serverA/searchUser\";\nimport { getOrders } from \"@tools/serverB/getOrders\";\n\nexport default async function main() {\n  const user = await searchUser({ email: \"user@example.com\" });\n  const orders = await getOrders({ userId: user.id });\n  return { user, ordersCount: orders.length };\n}"
}
```

**Output:**
```json
{
  "result": {
    "user": { "id": "123", "email": "user@example.com" },
    "ordersCount": 5
  },
  "logs": []
}
```

## Architecture

```
meta-mcp-server/
├── src/
│   ├── index.ts              # Main entry point and orchestrator
│   ├── mcp-server.ts         # MCP server with tool registrations
│   ├── types.ts              # Type definitions
│   ├── external-servers/     # MCP client management
│   │   └── index.ts
│   ├── generator/            # TS wrapper generation
│   │   └── index.ts
│   ├── search/               # BM25 and Regex search
│   │   └── index.ts
│   ├── runtime/              # TS code execution
│   │   ├── eval-ts.ts
│   │   └── call-mcp-tool.ts  # Generated at runtime
│   └── tools/                # Generated tool wrappers
│       ├── serverA/
│       │   ├── search-user.ts
│       │   └── index.ts
│       └── serverB/
│           ├── get-orders.ts
│           └── index.ts
```

## Development

```bash
# Install dependencies
npm install

# Development mode with auto-reload
npm run dev -- -c config.json

# Build
npm run build

# Clean build artifacts
npm run clean
```

## License

MIT
