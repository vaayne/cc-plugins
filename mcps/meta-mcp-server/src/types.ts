/**
 * Type definitions for the Meta MCP Server
 */

import { z } from "zod";

// ============================================================================
// Configuration Types
// ============================================================================

export const ServerConfigSchema = z.object({
  id: z.string().describe("Unique identifier for the server"),
  name: z.string().describe("Human-readable name for the server"),
  transport: z.enum(["http", "stdio"]).describe("Transport protocol"),
  endpoint: z.string().optional().describe("HTTP endpoint URL (required for http transport)"),
  command: z.string().optional().describe("Command to run (required for stdio transport)"),
  args: z.array(z.string()).optional().describe("Arguments for stdio command"),
  env: z.record(z.string()).optional().describe("Environment variables for stdio command"),
});

export type ServerConfig = z.infer<typeof ServerConfigSchema>;

export const MetaServerConfigSchema = z.object({
  servers: z.array(ServerConfigSchema),
  toolsOutputDir: z.string().default("./src/tools").describe("Output directory for generated TS wrappers"),
  searchIndexPath: z.string().default("./search-index").describe("Path for search index storage"),
});

export type MetaServerConfig = z.infer<typeof MetaServerConfigSchema>;

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
}

// ============================================================================
// Search Types
// ============================================================================

export const SearchQuerySchema = z.object({
  query: z.string().min(1).describe("Search keywords or regex pattern"),
  mode: z.enum(["bm25", "regex", "auto"]).default("auto").describe("Search mode"),
  limit: z.number().int().min(1).max(50).default(10).describe("Maximum results to return"),
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
}

export interface SearchResult {
  items: SearchResultItem[];
  total: number;
}

// ============================================================================
// Eval Types
// ============================================================================

export const EvalTsInputSchema = z.object({
  code: z.string().min(1).describe("TypeScript source code to execute. Must export a default async function."),
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

export const SearchToolsInputSchema = z.object({
  query: z.string()
    .min(1, "Query must not be empty")
    .max(500, "Query must not exceed 500 characters")
    .describe("Search keywords or regex pattern to find tools"),
  mode: z.enum(["bm25", "regex", "auto"])
    .default("auto")
    .describe("Search mode: 'bm25' for keyword search, 'regex' for pattern matching, 'auto' to detect"),
  limit: z.number()
    .int()
    .min(1)
    .max(50)
    .default(10)
    .describe("Maximum number of results to return"),
}).strict();

export const EvalTsToolInputSchema = z.object({
  code: z.string()
    .min(1, "Code must not be empty")
    .describe("TypeScript source code to execute. Must export a default async function that returns the result."),
}).strict();
