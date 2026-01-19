/**
 * TSEval Runtime Module
 *
 * Executes TypeScript code using Bun's native runtime.
 * The code can import generated tool wrappers using @tools/* aliases.
 *
 * By using Bun's native dynamic import instead of esbuild bundling,
 * we ensure the executed code shares the same module context as the
 * main server, allowing access to the connected MCP clients.
 */

import * as fs from "node:fs/promises";
import * as path from "node:path";
import * as url from "node:url";
import type { EvalTsInput, EvalTsOutput } from "../types.js";

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

/**
 * Configuration for the eval runtime
 */
interface EvalConfig {
  toolsDir: string;
  timeout: number;
}

/**
 * TSEval Runtime class
 */
export class TsEvalRuntime {
  private config: EvalConfig;
  private logs: string[] = [];

  constructor(config: Partial<EvalConfig> = {}) {
    this.config = {
      toolsDir: config.toolsDir || path.join(__dirname, "..", "tools"),
      timeout: config.timeout || 30000,
    };
  }

  /**
   * Execute TypeScript code
   */
  async execute(input: EvalTsInput): Promise<EvalTsOutput> {
    this.logs = [];

    try {
      // Transform imports and write to temp file
      const moduleFile = await this.prepareModule(input.code);

      try {
        // Execute using Bun's native TypeScript support
        const result = await this.executeModule(moduleFile);

        return {
          result,
          logs: this.logs,
        };
      } finally {
        // Clean up temp file
        await this.cleanup(moduleFile);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      return {
        result: null,
        logs: this.logs,
        error: errorMessage,
      };
    }
  }

  /**
   * Prepare the module by transforming imports and writing to temp file
   */
  private async prepareModule(tsCode: string): Promise<string> {
    const tempDir = path.join(__dirname, "..", "..", ".eval-temp");
    await fs.mkdir(tempDir, { recursive: true });

    const moduleFile = path.join(tempDir, `eval-${Date.now()}.ts`);

    // Transform @tools/* imports to absolute paths
    const transformedCode = this.transformImports(tsCode);
    await fs.writeFile(moduleFile, transformedCode);

    return moduleFile;
  }

  /**
   * Transform @tools/* imports to actual paths with cache busting
   */
  private transformImports(code: string): string {
    const timestamp = Date.now();
    // Replace @tools/... with absolute paths to the tools directory
    // Bun handles TypeScript natively, so we can use .ts extension directly
    // Add cache-busting query param to avoid stale module cache
    return code.replace(
      /from\s+["']@tools\/([^"']+)["']/g,
      (_match, toolPath) => {
        // Remove any existing extension and add .ts
        const basePath = toolPath.replace(/\.(js|ts)$/, "");
        const fullPath = path.join(this.config.toolsDir, `${basePath}.ts`);
        // Use file:// URL with cache-busting query param
        const fileUrl = `file://${fullPath}?t=${timestamp}`;
        return `from "${fileUrl}"`;
      },
    );
  }

  /**
   * Execute the module using Bun's native dynamic import
   */
  private async executeModule(moduleFile: string): Promise<unknown> {
    // Create a custom console that captures logs
    const capturedLogs = this.logs;
    const customConsole = {
      log: (...args: unknown[]) => {
        capturedLogs.push(args.map(String).join(" "));
      },
      error: (...args: unknown[]) => {
        capturedLogs.push(`[ERROR] ${args.map(String).join(" ")}`);
      },
      warn: (...args: unknown[]) => {
        capturedLogs.push(`[WARN] ${args.map(String).join(" ")}`);
      },
      info: (...args: unknown[]) => {
        capturedLogs.push(`[INFO] ${args.map(String).join(" ")}`);
      },
    };

    // Replace console temporarily
    const originalConsole = globalThis.console;
    // @ts-expect-error - Replacing console for log capture
    globalThis.console = customConsole;

    try {
      // Use Bun's native dynamic import (handles TypeScript natively)
      // Adding timestamp to bust module cache
      const moduleUrl = `${url.pathToFileURL(moduleFile).href}?t=${Date.now()}`;
      const module = await import(moduleUrl);

      // Execute the default export if it's a function
      if (typeof module.default === "function") {
        return await module.default();
      } else if (module.default !== undefined) {
        return module.default;
      } else if (typeof module.main === "function") {
        return await module.main();
      }

      return null;
    } finally {
      // Restore console
      globalThis.console = originalConsole;
    }
  }

  /**
   * Clean up temporary files
   */
  private async cleanup(moduleFile: string): Promise<void> {
    try {
      await fs.unlink(moduleFile);
    } catch {
      // Ignore cleanup errors
    }
  }

  /**
   * Update the tools directory path
   */
  setToolsDir(toolsDir: string): void {
    this.config.toolsDir = toolsDir;
  }
}

// Singleton instance
export const tsEvalRuntime = new TsEvalRuntime();
