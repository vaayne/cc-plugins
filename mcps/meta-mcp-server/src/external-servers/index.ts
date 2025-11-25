/**
 * ExternalServers Module
 *
 * Manages connections to multiple external MCP servers and provides
 * unified APIs for listing tools and calling them.
 */

import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import type { ServerConfig, ExternalToolMeta } from "../types.js";

export class ExternalServersManager {
  private configs: Map<string, ServerConfig> = new Map();
  private clients: Map<string, Client> = new Map();
  private toolCache: Map<string, Tool[]> = new Map();

  /**
   * Configure the manager with server definitions
   */
  configure(servers: ServerConfig[]): void {
    this.configs.clear();
    for (const server of servers) {
      this.validateConfig(server);
      this.configs.set(server.id, server);
    }
  }

  /**
   * Validate server configuration
   */
  private validateConfig(config: ServerConfig): void {
    if (config.transport === "http" && !config.endpoint) {
      throw new Error(`Server '${config.id}' with http transport requires an endpoint`);
    }
    if (config.transport === "stdio" && !config.command) {
      throw new Error(`Server '${config.id}' with stdio transport requires a command`);
    }
  }

  /**
   * Get or create a client for the specified server
   */
  private async getClient(serverId: string): Promise<Client> {
    const existing = this.clients.get(serverId);
    if (existing) {
      return existing;
    }

    const config = this.configs.get(serverId);
    if (!config) {
      throw new Error(`Server '${serverId}' not configured`);
    }

    const client = new Client({
      name: `meta-mcp-client-${serverId}`,
      version: "1.0.0",
    });

    const transport = this.createTransport(config);
    await client.connect(transport);
    this.clients.set(serverId, client);
    return client;
  }

  /**
   * Create transport based on server configuration
   */
  private createTransport(config: ServerConfig) {
    if (config.transport === "http") {
      return new StreamableHTTPClientTransport(
        new URL(config.endpoint!)
      );
    } else {
      return new StdioClientTransport({
        command: config.command!,
        args: config.args || [],
        env: config.env,
      });
    }
  }

  /**
   * Get list of configured server IDs
   */
  getServerIds(): string[] {
    return Array.from(this.configs.keys());
  }

  /**
   * Get server configuration by ID
   */
  getServerConfig(serverId: string): ServerConfig | undefined {
    return this.configs.get(serverId);
  }

  /**
   * List tools from a specific external server
   */
  async listTools(serverId: string, refresh = false): Promise<ExternalToolMeta[]> {
    if (!refresh && this.toolCache.has(serverId)) {
      return this.toolsToMeta(serverId, this.toolCache.get(serverId)!);
    }

    const client = await this.getClient(serverId);
    const result = await client.listTools();
    const tools = result.tools || [];
    this.toolCache.set(serverId, tools);

    return this.toolsToMeta(serverId, tools);
  }

  /**
   * List tools from all configured servers
   */
  async listAllTools(refresh = false): Promise<ExternalToolMeta[]> {
    const allTools: ExternalToolMeta[] = [];
    const serverIds = this.getServerIds();

    for (const serverId of serverIds) {
      try {
        const tools = await this.listTools(serverId, refresh);
        allTools.push(...tools);
      } catch (error) {
        console.error(`Failed to list tools from server '${serverId}':`, error);
      }
    }

    return allTools;
  }

  /**
   * Convert SDK Tool objects to ExternalToolMeta
   */
  private toolsToMeta(serverId: string, tools: Tool[]): ExternalToolMeta[] {
    return tools.map((tool) => ({
      serverId,
      toolName: tool.name,
      description: tool.description,
      inputSchema: tool.inputSchema as Record<string, unknown> | undefined,
    }));
  }

  /**
   * Call a tool on an external server
   */
  async callTool(
    serverId: string,
    toolName: string,
    args: Record<string, unknown>
  ): Promise<unknown> {
    const client = await this.getClient(serverId);
    const result = await client.callTool({
      name: toolName,
      arguments: args,
    });

    // Extract result from MCP response
    if (result.content && Array.isArray(result.content)) {
      const textContent = result.content.find((c) => c.type === "text");
      if (textContent && "text" in textContent) {
        try {
          return JSON.parse(textContent.text);
        } catch {
          return textContent.text;
        }
      }
    }

    return result;
  }

  /**
   * Close all client connections
   */
  async closeAll(): Promise<void> {
    for (const [serverId, client] of this.clients) {
      try {
        await client.close();
      } catch (error) {
        console.error(`Failed to close client for server '${serverId}':`, error);
      }
    }
    this.clients.clear();
    this.toolCache.clear();
  }

  /**
   * Refresh tool cache for all servers
   */
  async refreshAllTools(): Promise<ExternalToolMeta[]> {
    this.toolCache.clear();
    return this.listAllTools(true);
  }
}

// Singleton instance
export const externalServersManager = new ExternalServersManager();
