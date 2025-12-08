package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"mcp-hub-go/internal/client"
	"mcp-hub-go/internal/js"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

//go:embed exec_description.md
var ExecDescription string

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

