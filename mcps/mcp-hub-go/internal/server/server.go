package server

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"mcp-hub-go/internal/client"
	"mcp-hub-go/internal/config"
	"mcp-hub-go/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

//go:embed exec_tool.md
var execToolDescription string

// Server represents the MCP hub server
type Server struct {
	config          *config.Config
	logger          *zap.Logger
	mcpServer       *mcp.Server
	clientManager   *client.Manager
	builtinRegistry *tools.BuiltinToolRegistry
	toolCallTimeout time.Duration
}

// NewServer creates a new MCP hub server
func NewServer(cfg *config.Config, logger *zap.Logger) *Server {
	return &Server{
		config:          cfg,
		logger:          logger,
		toolCallTimeout: 60 * time.Second,
	}
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting MCP hub server")

	// Initialize client manager
	s.clientManager = client.NewManager(s.logger)

	// Initialize builtin tool registry
	s.builtinRegistry = tools.NewBuiltinToolRegistry(s.logger)

	// Register built-in tools
	s.registerBuiltinTools()

	// Connect to remote servers
	if err := s.connectToRemoteServers(); err != nil {
		return fmt.Errorf("failed to connect to remote servers: %w", err)
	}

	// Create MCP server
	s.mcpServer = mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-hub-go",
		Version: "v1.0.0",
	}, nil)

	// Register all tools with the MCP server
	if err := s.registerAllTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	// Start the server with stdio transport
	transport := &mcp.StdioTransport{}
	if err := s.mcpServer.Run(ctx, transport); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	s.logger.Info("Stopping MCP hub server")

	// Disconnect from all remote servers
	if s.clientManager != nil {
		if err := s.clientManager.DisconnectAll(); err != nil {
			s.logger.Error("Error disconnecting from remote servers", zap.Error(err))
			return err
		}
	}

	return nil
}

// registerBuiltinTools registers all built-in tools
func (s *Server) registerBuiltinTools() {
	// Register search tool
	s.builtinRegistry.RegisterTool(config.BuiltinTool{
		Name: "search",
		Description: `Search for available tools by keyword across all connected MCP servers.

This tool discovers available MCP tools from connected servers. Use it to find tools before calling them with exec.

## Workflow

1. Search for tools using keywords
2. Review the results to find the right tool(s)
3. Use exec to execute JavaScript code that calls the discovered tools via mcp.callTool()

## Output Format

Each result includes:
- name: Full namespaced tool name (serverID.toolName)
- description: Tool description
- server: The server ID hosting this tool

## Examples

Search for user-related tools:
  query: "user"

Search for file operations:
  query: "file"

Search by server name:
  query: "github"

## Tips

- Start broad, then narrow down: search "file" before "file upload"
- Note the server ID from results - you'll need it for mcp.callTool()`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search keywords to find tools (case-insensitive substring match)",
					"minLength":   1,
					"maxLength":   1000,
				},
			},
			"required": []string{"query"},
		},
	})

	// Register exec tool
	s.builtinRegistry.RegisterTool(config.BuiltinTool{
		Name:        "exec",
		Description: execToolDescription,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code": map[string]interface{}{
					"type":        "string",
					"minLength":   1,
					"description": "JavaScript code to execute. ES5.1 + ES6 features. Use mcp.callTool() for MCP tools.",
				},
			},
			"required": []string{"code"},
		},
	})

	// Register refreshTools tool
	s.builtinRegistry.RegisterTool(config.BuiltinTool{
		Name: "refreshTools",
		Description: `Refresh tool lists from connected MCP servers.

Use this when tools on a remote server have been updated and you need to fetch the latest tool definitions.

## When to Use

- After a remote MCP server has added new tools
- When tool definitions may have changed
- To verify current tool availability

## Parameters

- serverIds (optional): Array of specific server IDs to refresh. If not provided, refreshes all connected servers.

## Output

- refreshed: List of server IDs that were successfully refreshed
- errors: Map of server IDs to error messages for failed refreshes (if any)`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"serverIds": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "Optional list of server IDs to refresh. If not provided, refreshes all connected servers.",
				},
			},
		},
	})
}

// connectToRemoteServers connects to all configured remote MCP servers
func (s *Server) connectToRemoteServers() error {
	var errors []error

	for serverID, serverCfg := range s.config.MCPServers {
		// Skip disabled servers
		if !serverCfg.IsEnabled() {
			s.logger.Info("Skipping disabled server", zap.String("serverID", serverID))
			continue
		}

		s.logger.Info("Connecting to server", zap.String("serverID", serverID))
		if err := s.clientManager.ConnectToServer(serverID, serverCfg); err != nil {
			s.logger.Error("Failed to connect to server",
				zap.String("serverID", serverID),
				zap.Error(err),
			)

			// If server is required, return error immediately
			if serverCfg.Required {
				return fmt.Errorf("required server %s failed to connect: %w", serverID, err)
			}

			errors = append(errors, fmt.Errorf("server %s: %w", serverID, err))
		}
	}

	if len(errors) > 0 {
		s.logger.Warn("Some optional servers failed to connect", zap.Int("count", len(errors)))
	}

	return nil
}

// registerAllTools registers all tools (built-in only) with the MCP server
func (s *Server) registerAllTools() error {
	// Register built-in tools
	for toolName, builtinTool := range s.builtinRegistry.GetAllTools() {
		if err := s.registerBuiltinToolHandler(toolName, builtinTool); err != nil {
			return fmt.Errorf("failed to register built-in tool %s: %w", toolName, err)
		}
	}

	s.logger.Info("Registered built-in tools",
		zap.Int("count", len(s.builtinRegistry.GetAllTools())),
	)

	return nil
}

// registerBuiltinToolHandler registers a handler for a built-in tool
func (s *Server) registerBuiltinToolHandler(toolName string, builtinTool config.BuiltinTool) error {
	// Create MCP tool schema
	mcpTool := &mcp.Tool{
		Name:        toolName,
		Description: builtinTool.Description,
		InputSchema: builtinTool.InputSchema,
	}

	// Register the tool with a handler that calls the appropriate built-in function
	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return s.handleBuiltinTool(ctx, toolName, req)
	}

	// Use Server.AddTool to register the tool
	s.mcpServer.AddTool(mcpTool, handler)

	s.logger.Debug("Registered built-in tool", zap.String("name", toolName))
	return nil
}

// handleBuiltinTool handles calls to built-in tools
func (s *Server) handleBuiltinTool(ctx context.Context, toolName string, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Handling built-in tool call", zap.String("tool", toolName))

	// Apply timeout to prevent DoS attacks
	callCtx, cancel := context.WithTimeout(ctx, s.toolCallTimeout)
	defer cancel()

	switch toolName {
	case "search":
		return tools.HandleSearchTool(callCtx, s.builtinRegistry, s.clientManager, req)
	case "exec":
		return tools.HandleExecuteTool(callCtx, s.logger, s.clientManager, req)
	case "refreshTools":
		return tools.HandleRefreshToolsTool(callCtx, s.clientManager, req)
	default:
		return nil, fmt.Errorf("unknown built-in tool: %s", toolName)
	}
}
