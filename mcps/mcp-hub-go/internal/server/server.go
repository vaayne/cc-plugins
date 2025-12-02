package server

import (
	"context"
	"fmt"
	"time"

	"mcp-hub-go/internal/client"
	"mcp-hub-go/internal/config"
	"mcp-hub-go/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

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
		Name:        "search",
		Description: "Search across all available tools from connected MCP servers",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query string",
				},
			},
			"required": []string{"query"},
		},
	})

	// Register execute tool
	s.builtinRegistry.RegisterTool(config.BuiltinTool{
		Name:        "execute",
		Description: "Execute JavaScript code using goja runtime",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code": map[string]interface{}{
					"type":        "string",
					"description": "JavaScript code to execute",
				},
			},
			"required": []string{"code"},
		},
	})

	// Register refreshTools tool
	s.builtinRegistry.RegisterTool(config.BuiltinTool{
		Name:        "refreshTools",
		Description: "Refresh tool lists from connected MCP servers",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"serverIds": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "Optional list of server IDs to refresh (refreshes all if not provided)",
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
	case "execute":
		return tools.HandleExecuteTool(callCtx, s.logger, s.clientManager, req)
	case "refreshTools":
		return tools.HandleRefreshToolsTool(callCtx, s.clientManager, req)
	default:
		return nil, fmt.Errorf("unknown built-in tool: %s", toolName)
	}
}
