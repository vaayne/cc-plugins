/**
 * MCPAdapter Module
 *
 * The MCP server that exposes search_tools and eval_ts tools.
 * This is the main interface that LLMs interact with.
 */

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { tsEvalRuntime } from "./runtime/eval-ts.js";
import { codeSearchIndex } from "./search/index.js";
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
	query: z
		.string()
		.min(1, "Query must not be empty")
		.max(500, "Query must not exceed 500 characters")
		.describe("Search keywords or regex pattern to find tools"),
	mode: z
		.enum(["bm25", "regex", "auto"])
		.default("auto")
		.describe(
			"Search mode: 'bm25' for keyword search, 'regex' for pattern matching, 'auto' to detect automatically",
		),
	limit: z
		.number()
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

This tool discovers available MCP tools from connected servers. Use it to find tools before calling them with meta_eval_ts.

## Workflow

1. Search for tools using keywords or patterns
2. Review the results to find the right tool(s)
3. Use meta_eval_ts to execute code that imports and calls the discovered tools

## Search Modes

- 'bm25': Full-text keyword search with BM25 scoring. Best for natural language queries like "search user" or "send email".
- 'regex': Regular expression pattern matching. Best for structural queries like "email.*string" or finding specific patterns.
- 'auto' (default): Automatically detects mode. Uses regex if query contains special characters (.*+?^$|[]), otherwise uses bm25.

## Output Format

Each result includes:
- fn: Function name to import (camelCase)
- import: Import path, e.g., "@tools/serverId/toolName"
- desc: Tool description (truncated to 100 chars)
- params: Input parameters signature, e.g., "{ query: string, limit?: number }"
- returns: Return type signature, e.g., "{ users: Array<object>, total: number }"

## Examples

Search for user-related tools:
  query: "user search"
  mode: "bm25"

Find tools with email parameter:
  query: "email.*string"
  mode: "regex"

Find tools by server:
  query: "github"
  mode: "bm25"

## Tips

- Start broad, then narrow down: search "file" before "file upload aws s3"
- Use the returns field to understand what data you'll get back
- Combine multiple tools in meta_eval_ts for complex workflows`,
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
					returns: item.returns,
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
							? `${item.description.slice(0, 100).trim()}...`
							: item.description
						: "No description";

					lines.push(
						`${idx + 1}. ${item.functionName} (@tools/${item.serverId}/${item.toolName})`,
					);
					lines.push(`   ${desc}`);
					lines.push(`   Params: ${item.snippet}`);
					if (item.returns) {
						lines.push(`   Returns: ${item.returns}`);
					}
					lines.push("");
				});

				textContent = lines.join("\n");
			}

			return {
				content: [{ type: "text" as const, text: textContent }],
				structuredContent: output,
			};
		} catch (error) {
			const errorMessage =
				error instanceof Error ? error.message : String(error);
			return {
				isError: true,
				content: [
					{
						type: "text" as const,
						text: `Error searching tools: ${errorMessage}. Try a different query or check the search mode.`,
					},
				],
			};
		}
	},
);

// ============================================================================
// Tool: eval_ts
// ============================================================================

const EvalTsInputSchema = {
	code: z
		.string()
		.min(1, "Code must not be empty")
		.describe(
			"TypeScript source code to execute. Must export a default async function that returns the result.",
		),
};

mcpServer.registerTool(
	"meta_eval_ts",
	{
		title: "Evaluate TypeScript",
		description: `Execute TypeScript code that imports and calls discovered tool wrappers.

Use meta_search_tools first to discover available tools, then use this tool to execute code that calls them.

## Code Requirements

- Must export a default async function
- Import tools using: import { fn } from "@tools/serverId/toolName"
- Import helpers using: import { parallel, settle } from "@tools/utils"
- Console output is captured in 'logs'
- Timeout: 30 seconds

## Usage Patterns

### 1. Single Tool Call
\`\`\`typescript
import { searchUser } from "@tools/grep/searchUser";

export default async function() {
  return await searchUser({ pattern: "admin" });
}
\`\`\`

### 2. Sequential Calls (when results depend on each other)
\`\`\`typescript
import { getUser } from "@tools/db/getUser";
import { getOrders } from "@tools/db/getOrders";

export default async function() {
  const user = await getUser({ email: "user@example.com" });
  const orders = await getOrders({ userId: user.id });
  return { user, orders };
}
\`\`\`

### 3. Parallel Calls (independent operations - faster!)
\`\`\`typescript
import { parallel } from "@tools/utils";
import { searchUsers } from "@tools/db/searchUsers";
import { getStats } from "@tools/analytics/getStats";

export default async function() {
  const [users, stats] = await parallel(
    searchUsers({ role: "admin" }),
    getStats({ period: "week" })
  );
  return { users, stats };
}
\`\`\`

### 4. Parallel with Error Handling
\`\`\`typescript
import { settle } from "@tools/utils";
import { fetchData } from "@tools/api/fetchData";

export default async function() {
  const results = await settle(
    fetchData({ endpoint: "/users" }),
    fetchData({ endpoint: "/orders" }),
    fetchData({ endpoint: "/products" })
  );

  // Filter successful results
  const data = results
    .filter(r => r.status === "fulfilled")
    .map(r => r.value);
  return data;
}
\`\`\`

### 5. Data Transformation (reduce context usage)
\`\`\`typescript
import { listFiles } from "@tools/fs/listFiles";

export default async function() {
  const allFiles = await listFiles({ path: "/logs", recursive: true });
  // Only return what's needed - saves tokens!
  return allFiles
    .filter(f => f.name.endsWith(".error.log"))
    .map(f => ({ name: f.name, size: f.size }));
}
\`\`\`

### 6. Batch Operations
\`\`\`typescript
import { parallelMap } from "@tools/utils";
import { processItem } from "@tools/worker/processItem";

export default async function() {
  const items = ["a", "b", "c", "d", "e"];
  const results = await parallelMap(items, item =>
    processItem({ id: item })
  );
  return results;
}
\`\`\`

## Output

- result: Return value of your function
- logs: Array of console.log/warn/error output
- error: Error message if execution failed

## Tips

- Use parallel() for independent calls - significantly faster
- Transform/filter data before returning to save context tokens
- Use settle() when some calls might fail but you want partial results
- Console.log for debugging - output appears in logs`,
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
					content: [
						{
							type: "text" as const,
							text: `Execution Error: ${result.error}\n\nLogs:\n${(result.logs || []).join("\n")}`,
						},
					],
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
			const errorMessage =
				error instanceof Error ? error.message : String(error);
			return {
				isError: true,
				content: [
					{
						type: "text" as const,
						text: `Failed to execute code: ${errorMessage}. Check your TypeScript syntax and imports.`,
					},
				],
			};
		}
	},
);

