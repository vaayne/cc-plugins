package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MockTool represents a configurable mock tool
type MockTool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error)
}

// MockServerConfig holds configuration for a mock server
type MockServerConfig struct {
	ServerName    string
	Version       string
	Tools         []MockTool
	SimulateDelay time.Duration
	FailOnCall    bool
}

// MockServer implements a mock MCP server for testing
type MockServer struct {
	config       MockServerConfig
	server       *mcp.Server
	callHistory  []CallRecord
	mu           sync.Mutex
	disconnected bool
}

// CallRecord tracks a tool call
type CallRecord struct {
	ToolName  string
	Arguments map[string]any
	Timestamp time.Time
}

// NewMockServer creates a new mock MCP server
func NewMockServer(config MockServerConfig) *MockServer {
	if config.ServerName == "" {
		config.ServerName = "mock-server"
	}
	if config.Version == "" {
		config.Version = "v1.0.0"
	}

	ms := &MockServer{
		config:      config,
		callHistory: make([]CallRecord, 0),
	}

	// Create MCP server
	ms.server = mcp.NewServer(&mcp.Implementation{
		Name:    config.ServerName,
		Version: config.Version,
	}, nil)

	// Register tools
	for _, tool := range config.Tools {
		ms.registerTool(tool)
	}

	return ms
}

// registerTool registers a mock tool with the server
func (ms *MockServer) registerTool(tool MockTool) {
	mcpTool := &mcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
		InputSchema: tool.InputSchema,
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Track call history
		var args map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
			}
		}

		ms.mu.Lock()
		ms.callHistory = append(ms.callHistory, CallRecord{
			ToolName:  tool.Name,
			Arguments: args,
			Timestamp: time.Now(),
		})
		disconnected := ms.disconnected
		ms.mu.Unlock()

		// Simulate disconnection
		if disconnected {
			return nil, fmt.Errorf("server disconnected")
		}

		// Simulate failure
		if ms.config.FailOnCall {
			return nil, fmt.Errorf("simulated tool failure")
		}

		// Simulate delay
		if ms.config.SimulateDelay > 0 {
			select {
			case <-time.After(ms.config.SimulateDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Call handler if provided
		if tool.Handler != nil {
			return tool.Handler(ctx, args)
		}

		// Default response
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Mock response from %s", tool.Name),
				},
			},
		}, nil
	}

	ms.server.AddTool(mcpTool, handler)
}

// Start starts the mock server with the given transport
func (ms *MockServer) Start(ctx context.Context, transport mcp.Transport) error {
	return ms.server.Run(ctx, transport)
}

// GetCallHistory returns the call history
func (ms *MockServer) GetCallHistory() []CallRecord {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	history := make([]CallRecord, len(ms.callHistory))
	copy(history, ms.callHistory)
	return history
}

// ClearCallHistory clears the call history
func (ms *MockServer) ClearCallHistory() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.callHistory = make([]CallRecord, 0)
}

// SimulateDisconnect simulates server disconnection
func (ms *MockServer) SimulateDisconnect() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.disconnected = true
}

// SimulateReconnect simulates server reconnection
func (ms *MockServer) SimulateReconnect() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.disconnected = false
}

// GetCallCount returns the number of calls to a specific tool
func (ms *MockServer) GetCallCount(toolName string) int {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	count := 0
	for _, call := range ms.callHistory {
		if call.ToolName == toolName {
			count++
		}
	}
	return count
}

// CreateEchoTool creates a simple echo tool that returns its input
func CreateEchoTool(name string) MockTool {
	return MockTool{
		Name:        name,
		Description: fmt.Sprintf("Echo tool named %s", name),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "Message to echo",
				},
			},
			"required": []string{"message"},
		},
		Handler: func(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
			message, ok := args["message"].(string)
			if !ok {
				return nil, fmt.Errorf("message must be a string")
			}

			response := map[string]any{
				"echoed": message,
			}
			jsonBytes, err := json.Marshal(response)
			if err != nil {
				return nil, err
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(jsonBytes),
					},
				},
			}, nil
		},
	}
}

// CreateCalculatorTool creates a tool that performs simple calculations
func CreateCalculatorTool() MockTool {
	return MockTool{
		Name:        "calculate",
		Description: "Perform simple calculations",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{
					"type":        "string",
					"description": "Operation to perform (add, subtract, multiply, divide)",
					"enum":        []string{"add", "subtract", "multiply", "divide"},
				},
				"a": map[string]any{
					"type":        "number",
					"description": "First operand",
				},
				"b": map[string]any{
					"type":        "number",
					"description": "Second operand",
				},
			},
			"required": []string{"operation", "a", "b"},
		},
		Handler: func(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
			operation, ok := args["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation must be a string")
			}

			a, ok := args["a"].(float64)
			if !ok {
				return nil, fmt.Errorf("a must be a number")
			}

			b, ok := args["b"].(float64)
			if !ok {
				return nil, fmt.Errorf("b must be a number")
			}

			var result float64
			switch operation {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				result = a / b
			default:
				return nil, fmt.Errorf("unknown operation: %s", operation)
			}

			response := map[string]any{
				"result": result,
			}
			jsonBytes, err := json.Marshal(response)
			if err != nil {
				return nil, err
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(jsonBytes),
					},
				},
			}, nil
		},
	}
}

// CreateDelayTool creates a tool that responds after a delay
func CreateDelayTool(delay time.Duration) MockTool {
	return MockTool{
		Name:        "delayed-tool",
		Description: "Tool that responds after a delay",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"value": map[string]any{
					"type":        "string",
					"description": "Value to return",
				},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			value := "delayed"
			if v, ok := args["value"].(string); ok {
				value = v
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: value,
					},
				},
			}, nil
		},
	}
}

// CreateToolWithDotsInName creates a tool with dots in its name
func CreateToolWithDotsInName() MockTool {
	return MockTool{
		Name:        "tool.with.dots.in.name",
		Description: "Tool with dots in name for namespace collision testing",
		InputSchema: map[string]any{
			"type": "object",
		},
		Handler: func(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "tool with dots response",
					},
				},
			}, nil
		},
	}
}
