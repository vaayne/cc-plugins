package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"mcp-hub-go/internal/client"
	"mcp-hub-go/internal/config"
	"mcp-hub-go/internal/js"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// BuiltinToolRegistry manages built-in tools
type BuiltinToolRegistry struct {
	logger *zap.Logger
	tools  map[string]config.BuiltinTool
	mu     sync.RWMutex
}

// NewBuiltinToolRegistry creates a new registry
func NewBuiltinToolRegistry(logger *zap.Logger) *BuiltinToolRegistry {
	return &BuiltinToolRegistry{
		logger: logger,
		tools:  make(map[string]config.BuiltinTool),
	}
}

// RegisterTool adds a tool to the registry
func (r *BuiltinToolRegistry) RegisterTool(tool config.BuiltinTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logger.Info("Registering built-in tool", zap.String("name", tool.Name))
	r.tools[tool.Name] = tool
}

// GetTool retrieves a tool from the registry
func (r *BuiltinToolRegistry) GetTool(name string) (config.BuiltinTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, exists := r.tools[name]
	return tool, exists
}

// GetAllTools returns all registered tools
func (r *BuiltinToolRegistry) GetAllTools() map[string]config.BuiltinTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Return a copy to prevent external modification
	toolsCopy := make(map[string]config.BuiltinTool, len(r.tools))
	for k, v := range r.tools {
		toolsCopy[k] = v
	}
	return toolsCopy
}

// ToolSearchResult represents a single search result
type ToolSearchResult struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Server      string                 `json:"server"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// SearchToolsResponse represents the response from the search tool
type SearchToolsResponse struct {
	Tools []ToolSearchResult `json:"tools"`
}

// HandleSearchTool handles the search tool call
func HandleSearchTool(ctx context.Context, registry *BuiltinToolRegistry, manager *client.Manager, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse search arguments: %w", err)
	}

	// Validate query length
	const maxQueryLength = 1000
	if len(args.Query) > maxQueryLength {
		return nil, fmt.Errorf("query too long (max %d characters)", maxQueryLength)
	}

	if args.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	queryLower := strings.ToLower(args.Query)
	var results []ToolSearchResult
	const maxResults = 100 // Limit results to prevent DoS

	// Search remote tools only (built-in tools are not included in search results)
	allRemoteTools := manager.GetAllTools()
	for namespacedName, tool := range allRemoteTools {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if len(results) >= maxResults {
			break
		}
		if matchesTool(tool.Name, tool.Description, queryLower) {
			// Extract server ID from namespaced name
			parts := strings.SplitN(namespacedName, ".", 2)
			serverID := "unknown"
			if len(parts) == 2 {
				serverID = parts[0]
			}

			// Convert InputSchema to map if possible
			var inputSchema map[string]interface{}
			if tool.InputSchema != nil {
				if schema, ok := tool.InputSchema.(map[string]interface{}); ok {
					inputSchema = schema
				}
			}

			results = append(results, ToolSearchResult{
				Name:        namespacedName,
				Description: tool.Description,
				Server:      serverID,
				InputSchema: inputSchema,
			})
		}
	}

	// Create response
	response := SearchToolsResponse{
		Tools: results,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search results: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
	}, nil
}

// matchesTool checks if a tool matches the search query
func matchesTool(name, description, queryLower string) bool {
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)
	return strings.Contains(nameLower, queryLower) || strings.Contains(descLower, queryLower)
}

// ExecuteToolResponse represents the response from the execute tool
type ExecuteToolResponse struct {
	Result interface{}   `json:"result"`
	Logs   []js.LogEntry `json:"logs"`
}

// HandleExecuteTool implements the execute built-in tool
func HandleExecuteTool(ctx context.Context, logger *zap.Logger, manager *client.Manager, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Unmarshal arguments
	var args struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Code == "" {
		return nil, fmt.Errorf("code parameter is required")
	}

	// Validate code length
	const maxCodeLength = 100 * 1024 // 100KB
	if len(args.Code) > maxCodeLength {
		return nil, fmt.Errorf("code exceeds maximum length of %d bytes", maxCodeLength)
	}

	// Create JS runtime
	runtime := js.NewRuntime(logger, manager, nil)

	// Execute code
	result, logs, err := runtime.Execute(ctx, args.Code)
	if err != nil {
		// Check if it's a RuntimeError for structured error response
		if runtimeErr, ok := err.(*js.RuntimeError); ok {
			// Return error as part of response for better UX
			response := ExecuteToolResponse{
				Result: map[string]interface{}{
					"error": map[string]string{
						"type":    string(runtimeErr.Type),
						"message": runtimeErr.Message,
					},
				},
				Logs: logs,
			}

			jsonBytes, marshalErr := json.Marshal(response)
			if marshalErr != nil {
				return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(jsonBytes),
					},
				},
				IsError: true,
			}, nil
		}

		// Generic error
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Create response
	response := ExecuteToolResponse{
		Result: result,
		Logs:   logs,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal execute result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
	}, nil
}

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
