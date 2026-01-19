package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"hub/internal/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed invoke_description.md
var InvokeDescription string

// HandleInvokeTool handles the invoke tool call
func HandleInvokeTool(ctx context.Context, manager *client.Manager, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Name   string         `json:"name"`
		Params map[string]any `json:"params,omitempty"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse invoke arguments: %w", err)
	}

	// Validate name
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Validate name length
	const maxNameLength = 500
	if len(args.Name) > maxNameLength {
		return nil, fmt.Errorf("name too long (max %d characters)", maxNameLength)
	}

	// Enforce namespaced format (serverID__toolName)
	separatorIndex := strings.Index(args.Name, "__")
	if separatorIndex == -1 {
		return nil, fmt.Errorf("tool name must be namespaced (serverID__toolName)")
	}

	serverID := args.Name[:separatorIndex]
	toolName := args.Name[separatorIndex+2:]

	if serverID == "" {
		return nil, fmt.Errorf("server ID cannot be empty")
	}
	if toolName == "" {
		return nil, fmt.Errorf("tool name cannot be empty")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get the client session
	session, err := manager.GetClient(serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	// Build call params
	callParams := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args.Params,
	}

	// Call the tool
	result, err := session.CallTool(ctx, callParams)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return result, nil
}
