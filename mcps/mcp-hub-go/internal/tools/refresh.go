package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"mcp-hub-go/internal/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed refresh_description.md
var RefreshDescription string

// RefreshToolsResponse represents the response from the refreshTools tool
type RefreshToolsResponse struct {
	Refreshed []string          `json:"refreshed"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// HandleRefreshToolsTool handles the refreshTools tool call
func HandleRefreshToolsTool(ctx context.Context, manager *client.Manager, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		ServerIDs []string `json:"serverIds,omitempty"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse refreshTools arguments: %w", err)
	}

	// Validate and deduplicate server IDs
	const maxServerIDs = 100
	if len(args.ServerIDs) > maxServerIDs {
		return nil, fmt.Errorf("too many serverIds (max %d)", maxServerIDs)
	}

	// Get server IDs to refresh
	serverIDs := args.ServerIDs
	if len(serverIDs) == 0 {
		// Refresh all if none specified
		serverIDs = manager.ListClients()
	} else {
		// Deduplicate server IDs
		seen := make(map[string]bool)
		deduped := make([]string, 0, len(serverIDs))
		for _, id := range serverIDs {
			if id == "" {
				continue // Skip empty IDs
			}
			if !seen[id] {
				seen[id] = true
				deduped = append(deduped, id)
			}
		}
		serverIDs = deduped
	}

	// Refresh tools for each server
	var refreshed []string
	errors := make(map[string]string)

	for _, serverID := range serverIDs {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if err := manager.RefreshTools(serverID); err != nil {
			// Sanitize error message to prevent information leakage
			errors[serverID] = "refresh failed"
		} else {
			refreshed = append(refreshed, serverID)
		}
	}

	// Create response
	response := RefreshToolsResponse{
		Refreshed: refreshed,
	}
	if len(errors) > 0 {
		response.Errors = errors
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
	}, nil
}

