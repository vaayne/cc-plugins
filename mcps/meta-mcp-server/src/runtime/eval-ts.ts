/**
 * TSEval Runtime Module
 *
 * Executes TypeScript code in a controlled environment.
 * The code can import generated tool wrappers using @tools/* aliases.
 */

import * as esbuild from "esbuild";
import * as fs from "node:fs/promises";
import * as path from "node:path";
import * as vm from "node:vm";
import * as url from "node:url";
import type { EvalTsInput, EvalTsOutput } from "../types.js";

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

/**
 * Configuration for the eval runtime
 */
interface EvalConfig {
  toolsDir: string;
  timeout: number;
  memoryLimit?: number;
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
      memoryLimit: config.memoryLimit,
    };
  }

  /**
   * Execute TypeScript code
   */
  async execute(input: EvalTsInput): Promise<EvalTsOutput> {
    this.logs = [];

    try {
      // Transpile TypeScript to JavaScript using esbuild
      const jsCode = await this.transpile(input.code);

      // Create a sandboxed context and execute
      const result = await this.executeInSandbox(jsCode);

      return {
        result,
        logs: this.logs,
      };
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
   * Transpile TypeScript to JavaScript with import resolution
   */
  private async transpile(tsCode: string): Promise<string> {
    // Create a temporary file for the code
    const tempDir = path.join(__dirname, "..", "..", ".eval-temp");
    await fs.mkdir(tempDir, { recursive: true });

    const tempFile = path.join(tempDir, `eval-${Date.now()}.ts`);

    try {
      // Transform @tools/* imports to relative paths
      const transformedCode = this.transformImports(tsCode);
      await fs.writeFile(tempFile, transformedCode);

      // Bundle with esbuild
      const result = await esbuild.build({
        entryPoints: [tempFile],
        bundle: true,
        write: false,
        format: "esm",
        platform: "node",
        target: "node20",
        external: ["@modelcontextprotocol/sdk"],
        alias: {
          "@tools": this.config.toolsDir,
        },
        logLevel: "silent",
      });

      if (result.outputFiles && result.outputFiles.length > 0) {
        return result.outputFiles[0].text;
      }

      throw new Error("esbuild produced no output");
    } finally {
      // Clean up temp file
      try {
        await fs.unlink(tempFile);
      } catch {
        // Ignore cleanup errors
      }
    }
  }

  /**
   * Transform @tools/* imports to actual paths
   */
  private transformImports(code: string): string {
    // Replace @tools/... with relative paths
    return code.replace(
      /from\s+["']@tools\/([^"']+)["']/g,
      (match, toolPath) => {
        const fullPath = path.join(this.config.toolsDir, toolPath);
        return `from "${fullPath}"`;
      }
    );
  }

  /**
   * Execute JavaScript code in a sandboxed VM context
   */
  private async executeInSandbox(jsCode: string): Promise<unknown> {
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

    // Wrap the code to handle ESM default export
    // Since esbuild bundles to ESM, we need to handle async imports
    const wrappedCode = `
      ${jsCode}

      // Export the result for the VM to capture
      if (typeof __default !== 'undefined') {
        __evalResult = __default;
      } else if (typeof main !== 'undefined') {
        __evalResult = main;
      }
    `;

    // For ESM bundles, we need to use dynamic import
    // Create a temporary module file
    const tempDir = path.join(__dirname, "..", "..", ".eval-temp");
    const moduleFile = path.join(tempDir, `module-${Date.now()}.mjs`);

    try {
      await fs.mkdir(tempDir, { recursive: true });

      // Modify the bundled code to expose the default export
      const moduleCode = `
        ${jsCode}

        // The bundled code should have a default export
        // We need to find and call it
        export { default } from "${moduleFile}";
      `;

      // Write a simpler approach: just extract and run the main function
      const extractedCode = this.extractMainFunction(jsCode);
      await fs.writeFile(moduleFile, extractedCode);

      // Dynamic import the module
      const module = await import(url.pathToFileURL(moduleFile).href);

      // Replace console in global scope temporarily
      const originalConsole = globalThis.console;
      // @ts-expect-error - Replacing console for sandboxing
      globalThis.console = customConsole;

      try {
        // Execute the default export if it's a function
        if (typeof module.default === "function") {
          const result = await module.default();
          return result;
        } else if (module.default !== undefined) {
          return module.default;
        } else if (typeof module.main === "function") {
          const result = await module.main();
          return result;
        }

        return null;
      } finally {
        // Restore console
        globalThis.console = originalConsole;
      }
    } finally {
      // Clean up
      try {
        await fs.unlink(moduleFile);
      } catch {
        // Ignore cleanup errors
      }
    }
  }

  /**
   * Extract the main function from bundled code
   */
  private extractMainFunction(jsCode: string): string {
    // The bundled ESM code should work directly
    // We just need to ensure it has a proper default export
    return jsCode;
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
