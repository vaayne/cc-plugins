/**
 * ToolIntrospector & Generator Module
 *
 * Generates TypeScript wrapper code for external MCP tools.
 * Creates Zod schemas and typed functions that call the underlying MCP tools.
 */

import * as fs from "node:fs/promises";
import * as path from "node:path";
import type { ExternalToolMeta, GeneratedToolInfo } from "../types.js";

/**
 * Convert a tool name to a valid TypeScript function name (camelCase)
 */
function toFunctionName(toolName: string): string {
	// Replace non-alphanumeric chars with underscores, then convert to camelCase
	const sanitized = toolName
		.replace(/[^a-zA-Z0-9_]/g, "_")
		.replace(/^_+|_+$/g, "");

	// Convert to camelCase
	const parts = sanitized.split("_").filter(Boolean);
	if (parts.length === 0) return "tool";

	return parts
		.map((part, index) => {
			if (index === 0) {
				return part.toLowerCase();
			}
			return part.charAt(0).toUpperCase() + part.slice(1).toLowerCase();
		})
		.join("");
}

/**
 * Convert a tool name to a valid file name (kebab-case)
 */
function toFileName(toolName: string): string {
	return toolName
		.replace(/[^a-zA-Z0-9_-]/g, "-")
		.replace(/^-+|-+$/g, "")
		.toLowerCase();
}

/**
 * Convert JSON Schema to compact TypeScript-like type signature
 * e.g., "{ users: Array<object>, total: number }"
 */
function jsonSchemaToCompactType(
	schema: Record<string, unknown> | undefined,
): string {
	if (!schema) {
		return "unknown";
	}

	const type = schema.type as string | undefined;

	switch (type) {
		case "string": {
			if (schema.enum) {
				const values = (schema.enum as string[]).map((v) => `"${v}"`).join("|");
				return values;
			}
			return "string";
		}
		case "number":
		case "integer":
			return "number";
		case "boolean":
			return "boolean";
		case "array": {
			const items = schema.items as Record<string, unknown> | undefined;
			const itemType = jsonSchemaToCompactType(items);
			return `Array<${itemType}>`;
		}
		case "object": {
			const properties = schema.properties as
				| Record<string, Record<string, unknown>>
				| undefined;
			const required = (schema.required as string[]) || [];

			if (!properties || Object.keys(properties).length === 0) {
				return "object";
			}

			const props: string[] = [];
			for (const [propName, propSchema] of Object.entries(properties)) {
				const propType = jsonSchemaToCompactType(propSchema);
				const isRequired = required.includes(propName);
				props.push(`${propName}${isRequired ? "" : "?"}: ${propType}`);
			}
			return `{ ${props.join(", ")} }`;
		}
		default:
			return "unknown";
	}
}

/**
 * Convert JSON Schema type to Zod schema string
 */
function jsonSchemaToZod(
	schema: Record<string, unknown> | undefined,
	indent = 0,
): string {
	if (!schema) {
		return "z.any()";
	}

	const spaces = "  ".repeat(indent);
	const type = schema.type as string | undefined;
	const description = schema.description as string | undefined;

	let zodStr: string;

	switch (type) {
		case "string": {
			zodStr = "z.string()";
			if (schema.enum) {
				const values = (schema.enum as string[])
					.map((v) => `"${v}"`)
					.join(", ");
				zodStr = `z.enum([${values}])`;
			}
			if (schema.format === "email") {
				zodStr += ".email()";
			}
			if (schema.minLength !== undefined) {
				zodStr += `.min(${schema.minLength})`;
			}
			if (schema.maxLength !== undefined) {
				zodStr += `.max(${schema.maxLength})`;
			}
			break;
		}
		case "number":
		case "integer": {
			zodStr = type === "integer" ? "z.number().int()" : "z.number()";
			if (schema.minimum !== undefined) {
				zodStr += `.min(${schema.minimum})`;
			}
			if (schema.maximum !== undefined) {
				zodStr += `.max(${schema.maximum})`;
			}
			break;
		}
		case "boolean":
			zodStr = "z.boolean()";
			break;
		case "array": {
			const items = schema.items as Record<string, unknown> | undefined;
			const itemsZod = jsonSchemaToZod(items, indent);
			zodStr = `z.array(${itemsZod})`;
			break;
		}
		case "object": {
			const properties = schema.properties as
				| Record<string, Record<string, unknown>>
				| undefined;
			const required = (schema.required as string[]) || [];

			if (!properties || Object.keys(properties).length === 0) {
				zodStr = "z.record(z.unknown())";
			} else {
				const propLines: string[] = [];
				for (const [propName, propSchema] of Object.entries(properties)) {
					const propZod = jsonSchemaToZod(propSchema, indent + 1);
					const isRequired = required.includes(propName);
					const propDesc = propSchema.description as string | undefined;
					let line = `${spaces}  ${propName}: ${propZod}`;
					if (!isRequired) {
						line += ".optional()";
					}
					if (propDesc) {
						line += `.describe("${propDesc.replace(/"/g, '\\"')}")`;
					}
					propLines.push(line);
				}
				zodStr = `z.object({\n${propLines.join(",\n")}\n${spaces}})`;
			}
			break;
		}
		default:
			zodStr = "z.any()";
	}

	if (description && !zodStr.includes(".describe(")) {
		zodStr += `.describe("${description.replace(/"/g, '\\"')}")`;
	}

	return zodStr;
}

/**
 * Generate TypeScript wrapper code for a single tool
 */
function generateToolWrapper(tool: ExternalToolMeta): string {
	const functionName = toFunctionName(tool.toolName);
	const inputZod = jsonSchemaToZod(tool.inputSchema, 0);
	const outputZod = tool.outputSchema
		? jsonSchemaToZod(tool.outputSchema, 0)
		: "z.any()";

	const description =
		tool.description ||
		`Wrapper for ${tool.serverId}.${tool.toolName} MCP tool.`;
	const escapedDescription = description
		.replace(/\*\//g, "* /")
		.replace(/\\/g, "\\\\");

	return `/**
 * Auto-generated TypeScript wrapper for MCP tool.
 *
 * Server: ${tool.serverId}
 * Tool: ${tool.toolName}
 * Description: ${escapedDescription}
 */

import { z } from "zod";
import { callMcpTool } from "../../runtime/call-mcp-tool.js";

// Input schema
export const ${functionName}InputSchema = ${inputZod};
export type ${functionName}Input = z.infer<typeof ${functionName}InputSchema>;

// Output schema
export const ${functionName}OutputSchema = ${outputZod};
export type ${functionName}Output = z.infer<typeof ${functionName}OutputSchema>;

/**
 * ${escapedDescription}
 *
 * @param input - Tool input parameters (defaults to empty object)
 * @returns Tool execution result
 */
export async function ${functionName}(
  input: ${functionName}Input = {} as ${functionName}Input
): Promise<${functionName}Output> {
  // Validate input
  const parsed = ${functionName}InputSchema.parse(input);

  // Call the underlying MCP tool
  const result = await callMcpTool<${functionName}Input, ${functionName}Output>(
    "${tool.serverId}",
    "${tool.toolName}",
    parsed
  );

  // Return result (skip strict validation to handle various MCP response formats)
  return result as ${functionName}Output;
}
`;
}

/**
 * Generate parallel execution utilities
 */
function generateUtilsModule(): string {
	return `/**
 * Parallel execution utilities for tool orchestration.
 *
 * These helpers make it easy to run multiple tool calls concurrently,
 * reducing latency when operations are independent of each other.
 */

/**
 * Execute multiple promises in parallel and return all results.
 * Fails fast: if any promise rejects, the entire operation fails.
 *
 * @example
 * const [users, orders] = await parallel(
 *   getUsers({ limit: 10 }),
 *   getOrders({ status: "pending" })
 * );
 *
 * @param promises - Promises to execute in parallel
 * @returns Array of resolved values in the same order as input
 */
export async function parallel<T extends readonly unknown[]>(
  ...promises: { [K in keyof T]: Promise<T[K]> }
): Promise<T> {
  return Promise.all(promises) as Promise<T>;
}

/**
 * Map over an array and execute async operations in parallel.
 * Useful for batch processing items with the same operation.
 *
 * @example
 * const results = await parallelMap(
 *   ["user1", "user2", "user3"],
 *   userId => getUser({ id: userId })
 * );
 *
 * @param items - Array of items to process
 * @param fn - Async function to apply to each item
 * @returns Array of results in the same order as input
 */
export async function parallelMap<T, R>(
  items: T[],
  fn: (item: T, index: number) => Promise<R>
): Promise<R[]> {
  return Promise.all(items.map((item, index) => fn(item, index)));
}

/**
 * Execute multiple promises and return all results, including failures.
 * Unlike parallel(), this never fails - it returns the status of each promise.
 *
 * @example
 * const results = await settle(
 *   fetchData({ endpoint: "/users" }),
 *   fetchData({ endpoint: "/orders" })
 * );
 *
 * // Handle results
 * results.forEach(result => {
 *   if (result.status === "fulfilled") {
 *     console.log("Success:", result.value);
 *   } else {
 *     console.log("Failed:", result.reason);
 *   }
 * });
 *
 * @param promises - Promises to execute in parallel
 * @returns Array of PromiseSettledResult objects
 */
export async function settle<T extends readonly unknown[]>(
  ...promises: { [K in keyof T]: Promise<T[K]> }
): Promise<{ [K in keyof T]: PromiseSettledResult<T[K]> }> {
  return Promise.allSettled(promises) as Promise<{
    [K in keyof T]: PromiseSettledResult<T[K]>;
  }>;
}

/**
 * Execute promises in parallel with a concurrency limit.
 * Useful when you need to avoid overwhelming external services.
 *
 * @example
 * const results = await parallelLimit(
 *   urls.map(url => () => fetch(url)),
 *   5 // Max 5 concurrent requests
 * );
 *
 * @param tasks - Array of functions that return promises
 * @param limit - Maximum number of concurrent executions
 * @returns Array of results in the same order as input
 */
export async function parallelLimit<T>(
  tasks: (() => Promise<T>)[],
  limit: number
): Promise<T[]> {
  const results: T[] = new Array(tasks.length);
  let currentIndex = 0;

  async function worker(): Promise<void> {
    while (currentIndex < tasks.length) {
      const index = currentIndex++;
      results[index] = await tasks[index]();
    }
  }

  const workers = Array(Math.min(limit, tasks.length))
    .fill(null)
    .map(() => worker());

  await Promise.all(workers);
  return results;
}
`;
}

/**
 * Generate the runtime helper for calling MCP tools
 */
function generateCallMcpToolRuntime(): string {
	return `/**
 * Runtime helper for calling MCP tools through the ExternalServersManager.
 *
 * This module provides a typed interface for calling external MCP tools
 * from the generated wrapper code.
 */

import { externalServersManager } from "../external-servers/index.js";

/**
 * Call an MCP tool on an external server.
 *
 * @param serverId - The server identifier
 * @param toolName - The tool name
 * @param input - The tool input
 * @returns The tool output
 */
export async function callMcpTool<TInput, TOutput>(
  serverId: string,
  toolName: string,
  input: TInput
): Promise<TOutput> {
  const result = await externalServersManager.callTool(
    serverId,
    toolName,
    input as Record<string, unknown>
  );
  return result as TOutput;
}
`;
}

/**
 * Generate an index file that re-exports all tools for a server
 */
function generateServerIndex(tools: GeneratedToolInfo[]): string {
	const exports = tools.map((t) => {
		const funcName = toFunctionName(t.toolName);
		const fileName = toFileName(t.toolName);
		return `export { ${funcName}, ${funcName}Input, ${funcName}Output, ${funcName}InputSchema, ${funcName}OutputSchema } from "./${fileName}.js";`;
	});

	return `/**
 * Auto-generated index file for server tools.
 * Re-exports all tool wrappers for this server.
 */

${exports.join("\n")}
`;
}

export class ToolGenerator {
	private outputDir: string;

	constructor(outputDir: string) {
		this.outputDir = outputDir;
	}

	/**
	 * Generate TypeScript wrappers for all provided tools
	 */
	async generateAll(tools: ExternalToolMeta[]): Promise<GeneratedToolInfo[]> {
		const generatedTools: GeneratedToolInfo[] = [];

		// Ensure output directory exists
		await fs.mkdir(this.outputDir, { recursive: true });

		// Ensure runtime directory exists and generate call-mcp-tool.ts
		const runtimeDir = path.join(this.outputDir, "..", "runtime");
		await fs.mkdir(runtimeDir, { recursive: true });
		await fs.writeFile(
			path.join(runtimeDir, "call-mcp-tool.ts"),
			generateCallMcpToolRuntime(),
		);

		// Generate utils.ts with parallel execution helpers
		await fs.writeFile(
			path.join(this.outputDir, "utils.ts"),
			generateUtilsModule(),
		);

		// Group tools by server
		const toolsByServer = new Map<string, ExternalToolMeta[]>();
		for (const tool of tools) {
			const serverTools = toolsByServer.get(tool.serverId) || [];
			serverTools.push(tool);
			toolsByServer.set(tool.serverId, serverTools);
		}

		// Generate tools for each server
		for (const [serverId, serverTools] of toolsByServer) {
			const serverDir = path.join(this.outputDir, serverId);
			await fs.mkdir(serverDir, { recursive: true });

			const serverGeneratedTools: GeneratedToolInfo[] = [];

			for (const tool of serverTools) {
				const generated = await this.generateTool(tool, serverDir);
				generatedTools.push(generated);
				serverGeneratedTools.push(generated);
			}

			// Generate index file for this server
			const indexContent = generateServerIndex(serverGeneratedTools);
			await fs.writeFile(path.join(serverDir, "index.ts"), indexContent);
		}

		return generatedTools;
	}

	/**
	 * Generate a single tool wrapper
	 */
	private async generateTool(
		tool: ExternalToolMeta,
		serverDir: string,
	): Promise<GeneratedToolInfo> {
		const fileName = toFileName(tool.toolName);
		const filePath = path.join(serverDir, `${fileName}.ts`);
		const sourceCode = generateToolWrapper(tool);

		await fs.writeFile(filePath, sourceCode);

		// Extract compact return type signature from output schema
		const returns = jsonSchemaToCompactType(tool.outputSchema);

		return {
			serverId: tool.serverId,
			toolName: tool.toolName,
			functionName: toFunctionName(tool.toolName),
			filePath,
			sourceCode,
			description: tool.description,
			returns,
		};
	}

	/**
	 * Clear all generated files
	 */
	async clearAll(): Promise<void> {
		try {
			await fs.rm(this.outputDir, { recursive: true, force: true });
		} catch {
			// Directory may not exist
		}
	}
}
