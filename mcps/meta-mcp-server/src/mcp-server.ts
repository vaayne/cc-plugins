/**
 * MCPAdapter Module
 *
 * The MCP server that exposes search_tools and eval_ts tools.
 * This is the main interface that LLMs interact with.
 */

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { codeSearchIndex } from "./search/index.js";
import { tsEvalRuntime } from "./runtime/eval-ts.js";
import type { SearchResultItem } from "./types.js";

// ============================================================================
// MCP Server Instance
// ============================================================================

export const mcpServer = new McpServer({
  name: "meta-mcp-server",
  version: "1.0.0",
});

// ============================================================================
// Tool: search_tools
// ============================================================================

const SearchToolsInputSchema = {
  query: z.string()
    .min(1, "Query must not be empty")
    .max(500, "Query must not exceed 500 characters")
    .describe("Search keywords or regex pattern to find tools"),
  mode: z.enum(["bm25", "regex", "auto"])
    .default("auto")
    .describe("Search mode: 'bm25' for keyword search, 'regex' for pattern matching, 'auto' to detect automatically"),
  limit: z.number()
    .int()
    .min(1)
    .max(50)
    .default(10)
    .describe("Maximum number of results to return"),
};

mcpServer.registerTool(
  "meta_search_tools",
  {
    title: "Search Tools",
    description: `Search for available TypeScript tool wrappers by keyword or regex pattern.

This tool searches through all generated TypeScript wrappers for external MCP tools.
It returns matching tool definitions with code snippets that can be used with eval_ts.

Search Modes:
- 'bm25': Full-text keyword search with BM25 scoring (best for natural language queries)
- 'regex': Regular expression pattern matching on source code
- 'auto': Automatically detect mode based on query (default)

Returns:
{
  "items": [
    {
      "id": "serverId.toolName",
      "serverId": "server identifier",
      "toolName": "original MCP tool name",
      "functionName": "TypeScript function name to import",
      "filePath": "path to the generated wrapper file",
      "description": "tool description",
      "snippet": "relevant code snippet"
    }
  ],
  "total": number
}

Example Usage:
- Search for user-related tools: query="user search" mode="bm25"
- Find all tools with email parameter: query="email.*string" mode="regex"
- Auto-detect search type: query="createOrder" mode="auto"`,
    inputSchema: SearchToolsInputSchema,
    annotations: {
      readOnlyHint: true,
      destructiveHint: false,
      idempotentHint: true,
      openWorldHint: false,
    },
  },
  async (params) => {
    try {
      const query = params.query;
      const mode = params.mode ?? "auto";
      const limit = params.limit ?? 10;

      const result = codeSearchIndex.search({
        query,
        mode,
        limit,
      });

      const output: Record<string, unknown> = {
        items: result.items.map((item: SearchResultItem) => ({
          fn: item.functionName,
          import: `@tools/${item.serverId}/${item.toolName}`,
          desc: item.description ? item.description.slice(0, 100) : undefined,
          params: item.snippet,
        })),
        total: result.total,
      };

      // Format as compact text for token efficiency
      let textContent: string;
      if (result.items.length === 0) {
        textContent = `No tools found for "${query}"`;
      } else {
        const lines = [`Found ${result.total} tool(s):\n`];

        result.items.forEach((item, idx) => {
          // Truncate long descriptions
          const desc = item.description
            ? item.description.length > 100
              ? item.description.slice(0, 100).trim() + "..."
              : item.description
            : "No description";

          lines.push(`${idx + 1}. ${item.functionName} (@tools/${item.serverId}/${item.toolName})`);
          lines.push(`   ${desc}`);
          lines.push(`   Params: ${item.snippet}`);
          lines.push("");
        });

        textContent = lines.join("\n");
      }

      return {
        content: [{ type: "text" as const, text: textContent }],
        structuredContent: output,
      };
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      return {
        isError: true,
        content: [{
          type: "text" as const,
          text: `Error searching tools: ${errorMessage}. Try a different query or check the search mode.`,
        }],
      };
    }
  }
);

// ============================================================================
// Tool: eval_ts
// ============================================================================

const EvalTsInputSchema = {
  code: z.string()
    .min(1, "Code must not be empty")
    .describe("TypeScript source code to execute. Must export a default async function that returns the result."),
};

mcpServer.registerTool(
  "meta_eval_ts",
  {
    title: "Evaluate TypeScript",
    description: `Execute TypeScript code that can import and use the generated tool wrappers.

This tool compiles and runs TypeScript code in a sandboxed environment.
The code can import tools using the @tools/* alias pattern.

Code Requirements:
- Must export a default async function that returns the result
- Can import tool wrappers using: import { functionName } from "@tools/serverId/toolName";
- Console output (log, warn, error) is captured and returned in 'logs'

Example Code:
\`\`\`typescript
import { searchUser } from "@tools/serverA/searchUser";
import { getOrders } from "@tools/serverB/getOrders";

export default async function main() {
  const user = await searchUser({ email: "user@example.com" });
  const orders = await getOrders({ userId: user.id });
  return { user, ordersCount: orders.length };
}
\`\`\`

Returns:
{
  "result": <return value of the default function>,
  "logs": ["captured console output"],
  "error": "error message if execution failed"
}

Security Notes:
- Code runs in a sandboxed environment
- File system access is restricted
- Network access is limited to MCP tool calls
- Execution timeout: 30 seconds`,
    inputSchema: EvalTsInputSchema,
    annotations: {
      readOnlyHint: false,
      destructiveHint: true,
      idempotentHint: false,
      openWorldHint: true,
    },
  },
  async (params) => {
    try {
      const result = await tsEvalRuntime.execute({
        code: params.code,
      });

      const structuredResult: Record<string, unknown> = {
        result: result.result,
        logs: result.logs,
        error: result.error,
      };

      if (result.error) {
        return {
          isError: true,
          content: [{
            type: "text" as const,
            text: `Execution Error: ${result.error}\n\nLogs:\n${(result.logs || []).join("\n")}`,
          }],
          structuredContent: structuredResult,
        };
      }

      // Format output
      const lines = ["# Execution Result", ""];

      if (result.logs && result.logs.length > 0) {
        lines.push("## Console Output");
        lines.push("```");
        lines.push(...result.logs);
        lines.push("```");
        lines.push("");
      }

      lines.push("## Return Value");
      lines.push("```json");
      lines.push(JSON.stringify(result.result, null, 2));
      lines.push("```");

      return {
        content: [{ type: "text" as const, text: lines.join("\n") }],
        structuredContent: structuredResult,
      };
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      return {
        isError: true,
        content: [{
          type: "text" as const,
          text: `Failed to execute code: ${errorMessage}. Check your TypeScript syntax and imports.`,
        }],
      };
    }
  }
);

// ============================================================================
// Tool: refresh_tools (optional management tool)
// ============================================================================

const RefreshToolsInputSchema = {};

mcpServer.registerTool(
  "meta_refresh_tools",
  {
    title: "Refresh Tools",
    description: `Refresh the tool index by re-scanning all external MCP servers.

This tool will:
1. Re-connect to all configured external MCP servers
2. Fetch updated tool definitions
3. Regenerate TypeScript wrappers
4. Rebuild the search index

Use this when:
- External servers have been updated with new tools
- You want to ensure the tool index is up to date
- After configuration changes

Returns:
{
  "refreshed": true,
  "toolCount": number,
  "servers": ["list of server IDs"]
}`,
    inputSchema: RefreshToolsInputSchema,
    annotations: {
      readOnlyHint: false,
      destructiveHint: false,
      idempotentHint: true,
      openWorldHint: true,
    },
  },
  async () => {
    // This will be wired up in the main entry point
    // For now, return a placeholder
    return {
      content: [{
        type: "text" as const,
        text: "Tool refresh is handled by the orchestrator. Please use the main refresh mechanism.",
      }],
      structuredContent: {
        refreshed: false,
        message: "Use orchestrator refresh",
      } as Record<string, unknown>,
    };
  }
);
