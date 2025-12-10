package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"mcp-hub-go/internal/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed list_description.md
var ListDescription string

// ListToolResult represents a tool in the list
type ListToolResult struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Server      string                 `json:"server"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// ListToolsResponse represents the response from the list tool
type ListToolsResponse struct {
	Tools []ListToolResult `json:"tools"`
	Total int              `json:"total"`
}

// HandleListTool handles the list tool call
func HandleListTool(ctx context.Context, manager *client.Manager, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Server string `json:"server"` // optional: filter by server name
		Query  string `json:"query"`  // optional: comma-separated keywords; tool matches if ANY keyword appears in name or description
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse list arguments: %w", err)
	}

	// Validate query length
	const maxQueryLength = 1000
	if len(args.Query) > maxQueryLength {
		return nil, fmt.Errorf("query too long (max %d characters)", maxQueryLength)
	}

	var results []ListToolResult
	const maxResults = 100 // Limit results to prevent DoS
	totalMatches := 0

	// Get all remote tools
	allRemoteTools := manager.GetAllTools()
	for namespacedName, tool := range allRemoteTools {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Extract server ID from namespaced name
		parts := strings.SplitN(namespacedName, ".", 2)
		serverID := "unknown"
		if len(parts) == 2 {
			serverID = parts[0]
		}

		// Filter by server if specified
		if args.Server != "" && !strings.EqualFold(serverID, args.Server) {
			continue
		}

		// Filter by query keywords if specified
		if !matchesKeywords(tool.Name, tool.Description, args.Query) {
			continue
		}

		totalMatches++

		if len(results) >= maxResults {
			continue
		}

		// Convert InputSchema to map if possible
		var inputSchema map[string]interface{}
		if tool.InputSchema != nil {
			if schema, ok := tool.InputSchema.(map[string]interface{}); ok {
				inputSchema = schema
			}
		}

		results = append(results, ListToolResult{
			Name:        namespacedName,
			Description: tool.Description,
			Server:      serverID,
			InputSchema: inputSchema,
		})
	}

	// Create response
	response := ListToolsResponse{
		Tools: results,
		Total: totalMatches,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list results: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
	}, nil
}

// matchesKeywords checks if tool matches any of the comma-separated keywords
func matchesKeywords(name, description, query string) bool {
	if query == "" {
		return true // no filter, match all
	}

	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)

	// Split by comma and match if ANY keyword appears in name or description
	keywords := strings.Split(query, ",")
	foundKeyword := false
	for _, raw := range keywords {
		kw := strings.TrimSpace(strings.ToLower(raw))
		if kw == "" {
			continue
		}
		foundKeyword = true
		if strings.Contains(nameLower, kw) || strings.Contains(descLower, kw) {
			return true
		}
	}

	// If no non-empty keywords were provided, treat as no filter
	if !foundKeyword {
		return true
	}

	return false
}
