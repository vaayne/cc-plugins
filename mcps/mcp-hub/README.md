# MCP Hub

An MCP server that aggregates multiple external MCP servers and exposes their tools through a unified search and code execution interface.

## Overview

Instead of exposing all downstream MCP tools directly to LLMs, this server:

1. **Collects** tool definitions from multiple external MCP servers
2. **Generates** TypeScript wrapper code for each tool with Zod schemas
3. **Indexes** the generated code for full-text (BM25) and regex search
4. **Exposes** only 2 high-level tools:
   - `search`: Search for tools by keyword or pattern
   - `exec`: Execute TypeScript code that imports and combines tools

This approach reduces LLM context consumption by collapsing many tools into "search + execute" capabilities.

## Installation

```bash
cd mcps/mcp-hub
bun install
bun run build
```

## Usage

### Configuration

Create a configuration file (e.g., [config.json](./config.example.json)`):

### Running

**stdio transport (for local use):**
```bash
bun start -- -c config.json
```

**HTTP transport (for remote access):**
```bash
bun start -- -c config.json -t http -p 3000
```

### MCP Tools

#### `search`

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
      "fn": "searchUser",
      "import": "@tools/serverA/searchUser",
      "desc": "Search user by email",
      "params": "{ email: string }",
      "returns": "{ id: string, email: string }"
    }
  ],
  "total": 1
}
```

#### `exec`

Execute TypeScript code that imports and uses tool wrappers.

**Input:**
```json
{
  "code": "import { searchUser } from \"@tools/serverA/searchUser\";\nimport { getOrders } from \"@tools/serverB/getOrders\";\n\nexport default async function() {\n  const user = await searchUser({ email: \"user@example.com\" });\n  const orders = await getOrders({ userId: user.id });\n  return { user, ordersCount: orders.length };\n}"
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
mcp-hub/
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
bun install

# Development mode with auto-reload
bun run dev -- -c config.json

# Build
bun run build

# Clean build artifacts
bun run clean
```

## License

MIT
