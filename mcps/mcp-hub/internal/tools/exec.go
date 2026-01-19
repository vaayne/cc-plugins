package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"hub/internal/client"
	"hub/internal/js"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

//go:embed exec_description.md
var ExecDescription string

// ExecResult represents the result from executing JavaScript code
type ExecResult struct {
	Result any           `json:"result"`
	Logs   []js.LogEntry `json:"logs"`
	Error  *ExecError    `json:"error,omitempty"`
}

// ExecError represents a structured execution error
type ExecError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ExecuteCode executes JavaScript code using the provided ToolCaller.
// This is the shared implementation used by both CLI and MCP tool handler.
func ExecuteCode(ctx context.Context, logger *zap.Logger, caller js.ToolCaller, code string) (*ExecResult, error) {
	// Validate code
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}

	// Validate code length
	const maxCodeLength = 100 * 1024 // 100KB
	if len(code) > maxCodeLength {
		return nil, fmt.Errorf("code exceeds maximum length of %d bytes", maxCodeLength)
	}

	// Create JS runtime
	runtime := js.NewRuntime(logger, caller, nil)

	// Execute code
	result, logs, err := runtime.Execute(ctx, code)

	execResult := &ExecResult{
		Result: result,
		Logs:   logs,
	}

	if err != nil {
		if runtimeErr, ok := err.(*js.RuntimeError); ok {
			execResult.Error = &ExecError{
				Type:    string(runtimeErr.Type),
				Message: runtimeErr.Message,
			}
		} else {
			execResult.Error = &ExecError{
				Type:    "error",
				Message: err.Error(),
			}
		}
	}

	return execResult, nil
}

// HandleExecuteTool implements the execute built-in tool (MCP server handler)
func HandleExecuteTool(ctx context.Context, logger *zap.Logger, manager *client.Manager, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Unmarshal arguments
	var args struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Create caller from manager
	caller := js.NewManagerCaller(manager)

	// Execute using shared implementation
	execResult, err := ExecuteCode(ctx, logger, caller, args.Code)
	if err != nil {
		return nil, err
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(execResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal execute result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
		IsError: execResult.Error != nil,
	}, nil
}
