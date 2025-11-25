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
 * Convert JSON Schema type to Zod schema string
 */
function jsonSchemaToZod(schema: Record<string, unknown> | undefined, indent = 0): string {
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
        const values = (schema.enum as string[]).map((v) => `"${v}"`).join(", ");
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
      const properties = schema.properties as Record<string, Record<string, unknown>> | undefined;
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
  const outputZod = tool.outputSchema ? jsonSchemaToZod(tool.outputSchema, 0) : "z.any()";

  const description = tool.description || `Wrapper for ${tool.serverId}.${tool.toolName} MCP tool.`;
  const escapedDescription = description.replace(/\*\//g, "* /").replace(/\\/g, "\\\\");

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

    // Ensure runtime directory exists and generate call-mcp-tool.ts
    const runtimeDir = path.join(this.outputDir, "..", "runtime");
    await fs.mkdir(runtimeDir, { recursive: true });
    await fs.writeFile(
      path.join(runtimeDir, "call-mcp-tool.ts"),
      generateCallMcpToolRuntime()
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
    serverDir: string
  ): Promise<GeneratedToolInfo> {
    const fileName = toFileName(tool.toolName);
    const filePath = path.join(serverDir, `${fileName}.ts`);
    const sourceCode = generateToolWrapper(tool);

    await fs.writeFile(filePath, sourceCode);

    return {
      serverId: tool.serverId,
      toolName: tool.toolName,
      functionName: toFunctionName(tool.toolName),
      filePath,
      sourceCode,
      description: tool.description,
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
