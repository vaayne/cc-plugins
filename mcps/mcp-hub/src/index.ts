#!/usr/bin/env bun
/**
 * Meta MCP Server
 *
 * A Meta MCP Server that aggregates multiple external MCP servers and exposes
 * their tools through a unified search and eval interface.
 *
 * Features:
 * - Connects to multiple external MCP servers (HTTP or stdio)
 * - Auto-generates TypeScript wrappers for all discovered tools
 * - Provides BM25 and Regex search over generated tool code
 * - Executes TypeScript code that can import and combine tools
 *
 * Tools exposed:
 * - meta_search_tools: Search for available tools by keyword or pattern
 * - meta_eval_ts: Execute TypeScript code using the generated tool wrappers
 */

import * as fs from "node:fs/promises";
import * as path from "node:path";
import * as url from "node:url";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import express from "express";

import { externalServersManager } from "./external-servers/index.js";
import { ToolGenerator } from "./generator/index.js";
import { mcpServer } from "./mcp-server.js";
import { tsEvalRuntime } from "./runtime/eval-ts.js";
import { codeSearchIndex } from "./search/index.js";
import {
	type McpServerEntry,
	type MetaServerConfig,
	MetaServerConfigSchema,
	type ServerConfig,
} from "./types.js";

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

// ============================================================================
// Orchestrator Class
// ============================================================================

class MetaMcpOrchestrator {
	private config: MetaServerConfig | null = null;
	private servers: ServerConfig[] = [];
	private generator: ToolGenerator | null = null;
	private initialized = false;

	/**
	 * Convert mcpServers config to internal ServerConfig array
	 */
	private parseServers(
		mcpServers: Record<string, McpServerEntry>,
	): ServerConfig[] {
		return Object.entries(mcpServers).map(([id, entry]) => {
			// Determine transport type
			let transport: "http" | "sse" | "stdio";
			if (entry.transport) {
				transport = entry.transport;
			} else if (entry.url) {
				transport = "http";
			} else if (entry.command) {
				transport = "stdio";
			} else {
				throw new Error(`Server '${id}' must have either 'url' or 'command'`);
			}

			return {
				id,
				transport,
				url: entry.url,
				command: entry.command,
				args: entry.args,
				env: entry.env,
				required: entry.required,
			};
		});
	}

	/**
	 * Load configuration from file
	 */
	async loadConfig(configPath: string): Promise<void> {
		const configContent = await fs.readFile(configPath, "utf-8");
		const configData = JSON.parse(configContent);
		this.config = MetaServerConfigSchema.parse(configData);

		// Parse servers from mcpServers object
		this.servers = this.parseServers(this.config.mcpServers);

		console.error(`Loaded config with ${this.servers.length} server(s)`);
	}

	/**
	 * Initialize all components
	 */
	async initialize(): Promise<void> {
		if (!this.config) {
			throw new Error("Configuration not loaded. Call loadConfig() first.");
		}

		console.error("Initializing Meta MCP Server...");

		// Default tools output directory
		const toolsOutputDir = path.join(__dirname, "tools");

		// Configure external servers manager
		externalServersManager.configure(this.servers);

		// Eagerly connect to all configured MCPs
		console.error("Connecting to external MCP servers...");
		const connectionResult = await externalServersManager.connectAll();

		// Check for required server failures
		const requiredFailures = connectionResult.failed.filter((f) => f.required);
		if (requiredFailures.length > 0) {
			const failedIds = requiredFailures.map((f) => f.id).join(", ");
			throw new Error(`Required MCP server(s) failed to connect: ${failedIds}`);
		}

		// Create generator
		this.generator = new ToolGenerator(toolsOutputDir);

		// Configure eval runtime with tools directory
		tsEvalRuntime.setToolsDir(toolsOutputDir);

		// Initial tool refresh (only for successfully connected servers)
		await this.refreshTools();

		this.initialized = true;
		console.error("Meta MCP Server initialized successfully");
	}

	/**
	 * Refresh tools from all external servers
	 */
	async refreshTools(): Promise<{ toolCount: number; servers: string[] }> {
		if (!this.config || !this.generator) {
			throw new Error("Not initialized");
		}

		console.error("Refreshing tools from external servers...");

		// Fetch tools from all servers
		const allTools = await externalServersManager.refreshAllTools();
		console.error(
			`Found ${allTools.length} tool(s) from ${externalServersManager.getServerIds().length} server(s)`,
		);

		// Generate TypeScript wrappers
		const generatedTools = await this.generator.generateAll(allTools);
		console.error(`Generated ${generatedTools.length} TypeScript wrapper(s)`);

		// Index generated tools for search
		codeSearchIndex.indexAll(generatedTools);
		console.error(`Indexed ${codeSearchIndex.size} tool(s) for search`);

		return {
			toolCount: generatedTools.length,
			servers: externalServersManager.getServerIds(),
		};
	}

	/**
	 * Run the server with stdio transport
	 */
	async runStdio(): Promise<void> {
		if (!this.initialized) {
			throw new Error("Not initialized. Call initialize() first.");
		}

		const transport = new StdioServerTransport();
		await mcpServer.connect(transport);
		console.error("Meta MCP Server running via stdio");
	}

	/**
	 * Run the server with HTTP transport
	 */
	async runHttp(port: number, host = "localhost"): Promise<void> {
		if (!this.initialized) {
			throw new Error("Not initialized. Call initialize() first.");
		}

		const app = express();
		app.use(express.json());

		app.post("/mcp", async (req, res) => {
			const transport = new StreamableHTTPServerTransport({
				sessionIdGenerator: undefined,
				enableJsonResponse: true,
			});
			res.on("close", () => transport.close());
			await mcpServer.connect(transport);
			await transport.handleRequest(req, res, req.body);
		});

		// Health check endpoint
		app.get("/health", (_req, res) => {
			res.json({
				status: "ok",
				toolCount: codeSearchIndex.size,
				servers: externalServersManager.getServerIds(),
			});
		});

		app.listen(port, host, () => {
			console.error(`Meta MCP Server running on http://${host}:${port}/mcp`);
		});
	}

	/**
	 * Cleanup resources
	 */
	async shutdown(): Promise<void> {
		await externalServersManager.closeAll();
		console.error("Meta MCP Server shut down");
	}
}

// ============================================================================
// CLI Entry Point
// ============================================================================

async function main(): Promise<void> {
	const args = process.argv.slice(2);

	// Parse arguments
	let configPath: string | undefined;
	let transport: "stdio" | "http" = "stdio";
	let port = 3000;
	let host = "localhost";

	for (let i = 0; i < args.length; i++) {
		const arg = args[i];
		if (arg === "-c" || arg === "--config") {
			configPath = args[++i];
		} else if (arg === "-t" || arg === "--transport") {
			transport = args[++i] as "stdio" | "http";
		} else if (arg === "-p" || arg === "--port") {
			port = parseInt(args[++i], 10);
		} else if (arg === "-h" || arg === "--host") {
			host = args[++i];
		} else if (arg === "--help") {
			console.log(`
Meta MCP Server - Aggregate and search across multiple MCP servers

Usage: meta-mcp-server [options]

Options:
  -c, --config <path>     Path to configuration JSON file (required)
  -t, --transport <type>  Transport type: 'stdio' or 'http' (default: stdio)
  -p, --port <number>     Port for HTTP transport (default: 3000)
  -h, --host <string>     Host for HTTP transport (default: localhost)
  --help                  Show this help message

Configuration File Format:
{
  "mcpServers": {
    "serverA": {
      "url": "http://localhost:4001/mcp"
    },
    "serverB": {
      "transport": "sse",
      "url": "http://localhost:64342/sse"
    },
    "serverC": {
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "some-mcp-server"]
    }
  }
}
`);
			process.exit(0);
		} else if (!configPath && !arg.startsWith("-")) {
			// First positional argument is config path
			configPath = arg;
		}
	}

	if (!configPath) {
		console.error("Error: Configuration file path is required");
		console.error("Usage: meta-mcp-server -c <config.json>");
		process.exit(1);
	}

	// Create and initialize orchestrator
	const orchestrator = new MetaMcpOrchestrator();

	try {
		await orchestrator.loadConfig(configPath);
		await orchestrator.initialize();

		// Handle shutdown signals
		const shutdown = async () => {
			console.error("\nShutting down...");
			await orchestrator.shutdown();
			process.exit(0);
		};

		process.on("SIGINT", shutdown);
		process.on("SIGTERM", shutdown);

		// Start server
		if (transport === "http") {
			await orchestrator.runHttp(port, host);
		} else {
			await orchestrator.runStdio();
		}
	} catch (error) {
		console.error("Failed to start Meta MCP Server:", error);
		process.exit(1);
	}
}

// Run if executed directly
main().catch((error) => {
	console.error("Fatal error:", error);
	process.exit(1);
});

export { MetaMcpOrchestrator };
