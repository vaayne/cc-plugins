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

//go:embed inspect_description.md
var InspectDescription string

// InspectToolResponse represents the response from the inspect tool
type InspectToolResponse struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Server      string         `json:"server"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// HandleInspectTool handles the inspect tool call
func HandleInspectTool(ctx context.Context, manager *client.Manager, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse inspect arguments: %w", err)
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
	if !strings.Contains(args.Name, "__") {
		return nil, fmt.Errorf("tool name must be namespaced (serverID__toolName)")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Look up the tool
	allTools := manager.GetAllTools()
	tool, ok := allTools[args.Name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", args.Name)
	}

	// Extract server ID from namespaced name
	serverID, _, _ := strings.Cut(args.Name, "__")

	// Convert InputSchema to map if possible
	var inputSchema map[string]any
	if tool.InputSchema != nil {
		if schema, ok := tool.InputSchema.(map[string]any); ok {
			inputSchema = schema
		}
	}

	// Build response
	response := InspectToolResponse{
		Name:        args.Name,
		Description: tool.Description,
		Server:      serverID,
		InputSchema: inputSchema,
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inspect result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
	}, nil
}
