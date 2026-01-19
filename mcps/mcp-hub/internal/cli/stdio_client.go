package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"mcphub/internal/logging"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// StdioClientOpts contains options for creating a StdioClient
type StdioClientOpts struct {
	Command []string // Command and arguments to spawn
	Timeout int      // seconds
	Logger  *zap.Logger
}

// StdioClient connects to an MCP server via stdio (spawning a subprocess)
type StdioClient struct {
	session *mcp.ClientSession
	logger  *zap.Logger
}

// NewStdioClient creates a new StdioClient and connects to the spawned MCP server
func NewStdioClient(ctx context.Context, opts StdioClientOpts) (*StdioClient, error) {
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("command is required for stdio transport")
	}

	// Set default timeout
	timeout := 30
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	// Use a no-op logger if none provided
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create the command
	cmd := exec.Command(opts.Command[0], opts.Command[1:]...)

	// Create MCP transport
	mcpTransport := &mcp.CommandTransport{
		Command: cmd,
	}

	logger.Debug("Created stdio transport",
		zap.String("command", opts.Command[0]),
		zap.Strings("args", opts.Command[1:]))

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mh-cli",
		Version: "v1.0.0",
	}, nil)

	// Connect with timeout
	connectCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	session, err := client.Connect(connectCtx, mcpTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to stdio server: %w", err)
	}

	// Verify server responded with valid initialization
	if session.InitializeResult() == nil {
		session.Close()
		return nil, fmt.Errorf("stdio server did not complete MCP handshake")
	}

	logger.Debug("Connected to stdio MCP server",
		zap.String("command", opts.Command[0]))

	return &StdioClient{
		session: session,
		logger:  logger,
	}, nil
}

// ListTools returns all available tools from the stdio MCP server
func (c *StdioClient) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	result, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	return result.Tools, nil
}

// GetTool returns a specific tool by name
func (c *StdioClient) GetTool(ctx context.Context, name string) (*mcp.Tool, error) {
	tools, err := c.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	for _, tool := range tools {
		if tool.Name == name {
			return tool, nil
		}
	}

	return nil, fmt.Errorf("tool '%s' not found", name)
}

// CallTool invokes a tool on the stdio MCP server
func (c *StdioClient) CallTool(ctx context.Context, name string, params json.RawMessage) (*mcp.CallToolResult, error) {
	// Parse the raw JSON into a map for the Arguments field
	var args map[string]any
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid tool arguments: %w", err)
		}
	}

	callParams := &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	result, err := c.session.CallTool(ctx, callParams)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool '%s': %w", name, err)
	}

	return result, nil
}

// Close closes the connection to the stdio MCP server
func (c *StdioClient) Close() error {
	if c.session != nil {
		err := c.session.Close()
		c.session = nil // Clear reference for GC
		return err
	}
	return nil
}

// createStdioClient creates a StdioClient from command flags and args
func createStdioClient(ctx context.Context, command []string, timeout int, verbose bool, logFile string) (*StdioClient, error) {
	// Initialize logging
	logLevel := zapcore.InfoLevel
	if verbose {
		logLevel = zapcore.DebugLevel
	}

	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	if _, err := logging.InitLogger(logConfig); err != nil {
		return nil, err
	}

	opts := StdioClientOpts{
		Command: command,
		Timeout: timeout,
		Logger:  logging.Logger,
	}

	return NewStdioClient(ctx, opts)
}
