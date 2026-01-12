package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"hub/internal/client"
	"hub/internal/config"
	"hub/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// TransportConfig holds transport configuration for the server
type TransportConfig struct {
	Type string // "stdio", "http", or "sse"
	Host string // Host to bind for HTTP/SSE
	Port int    // Port to bind for HTTP/SSE
}

// Server represents the MCP hub server
type Server struct {
	config          *config.Config
	logger          *zap.Logger
	mcpServer       *mcp.Server
	clientManager   *client.Manager
	builtinRegistry *tools.BuiltinToolRegistry
	toolCallTimeout time.Duration
	httpServer      *http.Server // for graceful shutdown of HTTP/SSE
}

// NewServer creates a new MCP hub server
func NewServer(cfg *config.Config, logger *zap.Logger) *Server {
	return &Server{
		config:          cfg,
		logger:          logger,
		toolCallTimeout: 60 * time.Second,
	}
}

// Start starts the MCP server with the specified transport
func (s *Server) Start(ctx context.Context, transportCfg TransportConfig) error {
	s.logger.Info("Starting MCP hub server",
		zap.String("transport", transportCfg.Type),
	)

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
		Name:    "hub",
		Version: "v1.0.0",
	}, nil)

	// Register all tools with the MCP server
	if err := s.registerAllTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	// Start with the appropriate transport
	switch transportCfg.Type {
	case "stdio":
		return s.startStdio(ctx)
	case "http":
		return s.startHTTP(ctx, transportCfg)
	case "sse":
		return s.startSSE(ctx, transportCfg)
	default:
		return fmt.Errorf("unsupported transport type: %s", transportCfg.Type)
	}
}

// startStdio starts the server with stdio transport
func (s *Server) startStdio(ctx context.Context) error {
	s.logger.Info("Starting stdio transport")
	transport := &mcp.StdioTransport{}
	if err := s.mcpServer.Run(ctx, transport); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// startHTTP starts the server with StreamableHTTP transport
func (s *Server) startHTTP(ctx context.Context, cfg TransportConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	s.logger.Info("Starting HTTP transport", zap.String("address", addr))

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("MCP Hub server running", zap.String("url", fmt.Sprintf("http://%s/mcp", addr)))

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}
	return nil
}

// startSSE starts the server with SSE transport
func (s *Server) startSSE(ctx context.Context, cfg TransportConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	s.logger.Info("Starting SSE transport", zap.String("address", addr))

	handler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/sse", handler)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("MCP Hub server running", zap.String("url", fmt.Sprintf("http://%s/sse", addr)))

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("SSE server failed: %w", err)
	}
	return nil
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	s.logger.Info("Stopping MCP hub server")

	// Shutdown HTTP server if running
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("Error shutting down HTTP server", zap.Error(err))
		}
	}

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
	// Schema for list tool
	listSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"server": map[string]any{
				"type":        "string",
				"description": "Optional: filter tools by server name",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Optional: comma-separated keywords for fulltext search (e.g., 'file,read,write'). Tool matches if any keyword appears in name or description.",
				"maxLength":   1000,
			},
		},
	}

	// Register list tool
	s.builtinRegistry.RegisterTool(config.BuiltinTool{
		Name:        "list",
		Description: tools.ListDescription,
		InputSchema: listSchema,
	})

	// Register exec tool
	s.builtinRegistry.RegisterTool(config.BuiltinTool{
		Name:        "exec",
		Description: tools.ExecDescription,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]any{
					"type":        "string",
					"minLength":   1,
					"description": "JavaScript to execute (async/await, timers, require for node:* built-ins). Use mcp.callTool() for MCP tools.",
				},
			},
			"required": []string{"code"},
		},
	})

	// Register refreshTools tool
	s.builtinRegistry.RegisterTool(config.BuiltinTool{
		Name:        "refreshTools",
		Description: tools.RefreshDescription,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"serverIds": map[string]any{
					"type": "array",
					"items": map[string]any{
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
	description := builtinTool.Description
	if toolName == "list" {
		description = tools.RenderListDescription(description, s.clientManager.GetAllTools())
	}

	// Create MCP tool schema
	mcpTool := &mcp.Tool{
		Name:        toolName,
		Description: description,
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
	case "list":
		return tools.HandleListTool(callCtx, s.clientManager, req)
	case "exec":
		return tools.HandleExecuteTool(callCtx, s.logger, s.clientManager, req)
	case "refreshTools":
		result, err := tools.HandleRefreshToolsTool(callCtx, s.clientManager, req)
		if err != nil {
			return nil, err
		}
		// Update the list tool description to reflect refreshed tool metadata.
		s.updateListToolDescription()
		return result, nil
	default:
		return nil, fmt.Errorf("unknown built-in tool: %s", toolName)
	}
}

func (s *Server) updateListToolDescription() {
	listTool, ok := s.builtinRegistry.GetTool("list")
	if !ok || s.mcpServer == nil || s.clientManager == nil {
		return
	}

	s.mcpServer.AddTool(&mcp.Tool{
		Name:        "list",
		Description: tools.RenderListDescription(listTool.Description, s.clientManager.GetAllTools()),
		InputSchema: listTool.InputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return s.handleBuiltinTool(ctx, "list", req)
	})
}
