/**
 * Type definitions for the Meta MCP Server
 */

import { z } from "zod";

// ============================================================================
// Configuration Types
// ============================================================================

/**
 * Server configuration schema matching Claude Desktop's MCP config format.
 * The server ID is derived from the key in the mcpServers object.
 */
export const McpServerEntrySchema = z.object({
  transport: z
    .enum(["http", "sse", "stdio"])
    .optional()
    .describe(
      "Transport protocol (defaults to http/sse if url is present, stdio if command is present)",
    ),
  url: z.string().optional().describe("HTTP/SSE endpoint URL"),
  command: z
    .string()
    .optional()
    .describe("Command to run (required for stdio transport)"),
  args: z.array(z.string()).optional().describe("Arguments for stdio command"),
  env: z
    .record(z.string())
    .optional()
    .describe("Environment variables for stdio command"),
  required: z
    .boolean()
    .default(false)
    .describe("If true, server must connect successfully at startup"),
});

export type McpServerEntry = z.infer<typeof McpServerEntrySchema>;

export const MetaServerConfigSchema = z.object({
  mcpServers: z
    .record(McpServerEntrySchema)
    .describe("Map of server ID to server configuration"),
});

export type MetaServerConfig = z.infer<typeof MetaServerConfigSchema>;

/**
 * Internal server config with resolved ID and transport type.
 */
export interface ServerConfig {
  id: string;
  transport: "http" | "sse" | "stdio";
  url?: string;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  required?: boolean;
}

// ============================================================================
// External Tool Types
// ============================================================================

export interface ExternalToolMeta {
  serverId: string;
  toolName: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
  outputSchema?: Record<string, unknown>;
}

// ============================================================================
// Generated Tool Types
// ============================================================================

export interface GeneratedToolInfo {
  serverId: string;
  toolName: string;
  functionName: string;
  filePath: string;
  sourceCode: string;
  description?: string;
  returns?: string;
}

// ============================================================================
// Search Types
// ============================================================================

export const SearchQuerySchema = z.object({
  query: z.string().min(1).describe("Search keywords or regex pattern"),
  mode: z
    .enum(["bm25", "regex", "auto"])
    .default("auto")
    .describe("Search mode"),
  limit: z
    .number()
    .int()
    .min(1)
    .max(50)
    .default(10)
    .describe("Maximum results to return"),
});

export type SearchQuery = z.infer<typeof SearchQuerySchema>;

export interface SearchResultItem {
  id: string;
  serverId: string;
  toolName: string;
  functionName: string;
  filePath: string;
  description?: string;
  snippet: string;
  returns?: string;
}

export interface SearchResult {
  items: SearchResultItem[];
  total: number;
}

// ============================================================================
// Eval Types
// ============================================================================

export const EvalTsInputSchema = z.object({
  code: z
    .string()
    .min(1)
    .describe(
      "TypeScript source code to execute. Must export a default async function.",
    ),
});

export type EvalTsInput = z.infer<typeof EvalTsInputSchema>;

export interface EvalTsOutput {
  result: unknown;
  logs?: string[];
  error?: string;
}

// ============================================================================
// MCP Tool Schemas for registration
// ============================================================================

export const SearchToolsInputSchema = z
  .object({
    query: z
      .string()
      .min(1, "Query must not be empty")
      .max(500, "Query must not exceed 500 characters")
      .describe("Search keywords or regex pattern to find tools"),
    mode: z
      .enum(["bm25", "regex", "auto"])
      .default("auto")
      .describe(
        "Search mode: 'bm25' for keyword search, 'regex' for pattern matching, 'auto' to detect",
      ),
    limit: z
      .number()
      .int()
      .min(1)
      .max(50)
      .default(10)
      .describe("Maximum number of results to return"),
  })
  .strict();

export const EvalTsToolInputSchema = z
  .object({
    code: z
      .string()
      .min(1, "Code must not be empty")
      .describe(
        "TypeScript source code to execute. Must export a default async function that returns the result.",
      ),
  })
  .strict();
