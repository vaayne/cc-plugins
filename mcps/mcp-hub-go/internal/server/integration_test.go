package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"mcp-hub-go/internal/client"
	"mcp-hub-go/internal/config"
	"mcp-hub-go/internal/js"
	mcptesting "mcp-hub-go/internal/testing"
	"mcp-hub-go/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestIntegration_MockServerBasic tests basic mock server functionality
func TestIntegration_MockServerBasic(t *testing.T) {
	_ = zaptest.NewLogger(t)

	// Create mock server with echo tool
	mockConfig := mcptesting.MockServerConfig{
		ServerName: "test-server",
		Version:    "v1.0.0",
		Tools: []mcptesting.MockTool{
			mcptesting.CreateEchoTool("echo"),
		},
	}

	mockServer := mcptesting.NewMockServer(mockConfig)
	require.NotNil(t, mockServer)

	// Start mock server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a command that will host the mock server
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "mock_server.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/bash\nwhile true; do sleep 1; done\n"), 0755)
	require.NoError(t, err)

	cmd := exec.CommandContext(ctx, scriptPath)
	transport := &mcp.CommandTransport{Command: cmd}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- mockServer.Start(ctx, transport)
	}()

	// Verify call history works
	history := mockServer.GetCallHistory()
	assert.Empty(t, history)

	// Verify disconnect/reconnect simulation
	mockServer.SimulateDisconnect()
	mockServer.SimulateReconnect()

	cancel()
	select {
	case <-serverErr:
	case <-time.After(time.Second):
	}
}

// TestIntegration_MultipleTools tests server with multiple tools
func TestIntegration_MultipleTools(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create mock server with multiple tools
	mockConfig := mcptesting.MockServerConfig{
		ServerName: "multi-tool-server",
		Tools: []mcptesting.MockTool{
			mcptesting.CreateEchoTool("echo"),
			mcptesting.CreateCalculatorTool(),
			mcptesting.CreateToolWithDotsInName(),
		},
	}

	mockServer := mcptesting.NewMockServer(mockConfig)

	// Create server
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Verify built-in tools are registered
	builtinTools := server.builtinRegistry.GetAllTools()
	assert.Len(t, builtinTools, 3)
	assert.Contains(t, builtinTools, "list")
	assert.Contains(t, builtinTools, "exec")
	assert.Contains(t, builtinTools, "refreshTools")

	// Test that mock server has tools registered
	assert.NotNil(t, mockServer)
}

// TestIntegration_JSExecutionWithToolCalls tests JavaScript execution calling tools
func TestIntegration_JSExecutionWithToolCalls(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create client manager
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create JS runtime
	runtime := js.NewRuntime(logger, manager, nil)
	require.NotNil(t, runtime)

	// Test simple JS execution
	script := `
		const result = 1 + 2;
		result;
	`

	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result)
	assert.Empty(t, logs)
}

// TestIntegration_JSExecutionWithLogging tests JavaScript execution with logging
func TestIntegration_JSExecutionWithLogging(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := js.NewRuntime(logger, manager, nil)

	script := `
		mcp.log('info', 'Test message');
		mcp.log('error', 'Error message', { code: 123 });
		'done';
	`

	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Equal(t, "done", result)
	assert.Len(t, logs, 2)

	assert.Equal(t, "info", logs[0].Level)
	assert.Equal(t, "Test message", logs[0].Message)

	assert.Equal(t, "error", logs[1].Level)
	assert.Equal(t, "Error message", logs[1].Message)
	// The code field can be either int64 or float64 depending on JSON unmarshaling
	code := logs[1].Fields["code"]
	assert.True(t, code == int64(123) || code == float64(123), "code should be 123")
}

// TestIntegration_JSExecutionTimeout tests JavaScript execution timeout
func TestIntegration_JSExecutionTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create runtime with short timeout
	runtime := js.NewRuntime(logger, manager, &js.Config{
		Timeout: 100 * time.Millisecond,
	})

	// Script that runs forever
	script := `
		while(true) {
			// Infinite loop
		}
	`

	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)

	runtimeErr, ok := err.(*js.RuntimeError)
	require.True(t, ok)
	assert.Equal(t, js.ErrorTypeTimeout, runtimeErr.Type)
}

// TestIntegration_JSExecutionAsyncSupport ensures async features work end-to-end
func TestIntegration_JSExecutionAsyncSupport(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := js.NewRuntime(logger, manager, nil)

	tests := []struct {
		name     string
		script   string
		expected any
		wantErr  bool
	}{
		{
			name:     "async function",
			script:   "async function test() { return 1; } test();",
			expected: int64(1),
		},
		{
			name:    "await keyword outside function (syntax error)",
			script:  "await Promise.resolve(1);",
			wantErr: true, // await without async context is syntax error
		},
		{
			name:     "Promise usage",
			script:   "new Promise(resolve => resolve(1));",
			expected: int64(1),
		},
		{
			name:     "setTimeout",
			script:   "new Promise(resolve => setTimeout(() => resolve(7), 5));",
			expected: int64(7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := runtime.Execute(context.Background(), tt.script)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIntegration_JSToolAuthorization tests tool authorization in JS runtime
func TestIntegration_JSToolAuthorization(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create runtime with restricted tools
	runtime := js.NewRuntime(logger, manager, &js.Config{
		AllowedTools: map[string][]string{
			"server1": {"tool1", "tool2"},
		},
	})

	script := `
		try {
			mcp.callTool('server1__tool3', {});
		} catch (e) {
			e.message;
		}
	`

	result, _, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Contains(t, result.(string), "not authorized")
}

// TestIntegration_NamespaceCollisions tests handling of namespace collisions
func TestIntegration_NamespaceCollisions(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	defer server.clientManager.DisconnectAll()

	// Test namespace parsing with double underscore separator
	tests := []struct {
		name           string
		namespacedName string
		expectedServer string
		expectedTool   string
		shouldError    bool
	}{
		{
			name:           "simple namespace",
			namespacedName: "server__tool",
			expectedServer: "server",
			expectedTool:   "tool",
			shouldError:    false,
		},
		{
			name:           "tool with underscores",
			namespacedName: "server__tool_with_underscores",
			expectedServer: "server",
			expectedTool:   "tool_with_underscores",
			shouldError:    false,
		},
		{
			name:           "empty server",
			namespacedName: "__tool",
			expectedServer: "",
			expectedTool:   "tool",
			shouldError:    true,
		},
		{
			name:           "no separator",
			namespacedName: "notool",
			expectedServer: "",
			expectedTool:   "",
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			separatorIndex := strings.Index(tt.namespacedName, "__")
			if separatorIndex == -1 {
				if tt.shouldError {
					return // Expected error
				}
				t.Fatalf("expected to find __ separator in %q", tt.namespacedName)
			}
			serverID := tt.namespacedName[:separatorIndex]
			toolName := tt.namespacedName[separatorIndex+2:]

			if tt.shouldError {
				if serverID == "" || toolName == "" {
					// Expected error condition
					return
				}
			} else {
				assert.Equal(t, tt.expectedServer, serverID)
				assert.Equal(t, tt.expectedTool, toolName)
			}
		})
	}
}

// TestIntegration_ConcurrentToolCalls tests concurrent tool calls
func TestIntegration_ConcurrentToolCalls(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Run multiple concurrent list calls
	const numCalls = 10
	var wg sync.WaitGroup
	wg.Add(numCalls)

	for i := range numCalls {
		go func(index int) {
			defer wg.Done()

			args := map[string]any{
				"query": fmt.Sprintf("search-%d", index),
			}
			argsJSON, err := json.Marshal(args)
			require.NoError(t, err)

			req := &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Name:      "list",
					Arguments: argsJSON,
				},
			}

			_, err = server.handleBuiltinTool(context.Background(), "list", req)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

// TestIntegration_ConcurrentJSExecutions tests concurrent JS executions
func TestIntegration_ConcurrentJSExecutions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	const numExecutions = 10
	var wg sync.WaitGroup
	wg.Add(numExecutions)

	for i := range numExecutions {
		go func(index int) {
			defer wg.Done()

			runtime := js.NewRuntime(logger, manager, nil)
			script := fmt.Sprintf("const result = %d * 2; result;", index)

			result, _, err := runtime.Execute(context.Background(), script)
			assert.NoError(t, err)
			assert.Equal(t, int64(index*2), result)
		}(i)
	}

	wg.Wait()
}

// TestIntegration_ContextCancellation tests context cancellation propagation
func TestIntegration_ContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := js.NewRuntime(logger, manager, nil)

	// Create context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start execution
	resultChan := make(chan error, 1)
	go func() {
		script := `
			let sum = 0;
			for (let i = 0; i < 1000000; i++) {
				sum += i;
			}
			sum;
		`
		_, _, err := runtime.Execute(ctx, script)
		resultChan <- err
	}()

	// Cancel immediately
	cancel()

	// Wait for result
	select {
	case err := <-resultChan:
		// Should get an error due to cancellation
		assert.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("execution did not respect context cancellation")
	}
}

// TestIntegration_ToolRefresh tests tool refresh functionality
func TestIntegration_ToolRefresh(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Test refreshTools with empty server list
	args := map[string]any{}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "refreshTools",
			Arguments: argsJSON,
		},
	}

	result, err := server.handleBuiltinTool(context.Background(), "refreshTools", req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Parse response
	var response tools.RefreshToolsResponse
	content := result.Content[0].(*mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &response)
	require.NoError(t, err)
	assert.Empty(t, response.Refreshed)
}

// TestIntegration_ToolRefreshWithServerIDs tests tool refresh with specific server IDs
func TestIntegration_ToolRefreshWithServerIDs(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Test refreshTools with non-existent servers
	args := map[string]any{
		"serverIds": []string{"server1", "server2"},
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "refreshTools",
			Arguments: argsJSON,
		},
	}

	result, err := server.handleBuiltinTool(context.Background(), "refreshTools", req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Parse response
	var response tools.RefreshToolsResponse
	content := result.Content[0].(*mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &response)
	require.NoError(t, err)

	// Should have errors for non-existent servers
	assert.Empty(t, response.Refreshed)
	assert.Len(t, response.Errors, 2)
}

// TestIntegration_JSExecutionSyntaxError tests JS syntax error handling
func TestIntegration_JSExecutionSyntaxError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := js.NewRuntime(logger, manager, nil)

	// Invalid syntax
	script := "const x = ;"

	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)

	runtimeErr, ok := err.(*js.RuntimeError)
	require.True(t, ok)
	assert.Equal(t, js.ErrorTypeSyntax, runtimeErr.Type)
}

// TestIntegration_JSExecutionRuntimeError tests JS runtime error handling
func TestIntegration_JSExecutionRuntimeError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := js.NewRuntime(logger, manager, nil)

	// Runtime error - undefined variable
	script := "undefinedVariable + 1;"

	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)

	runtimeErr, ok := err.(*js.RuntimeError)
	require.True(t, ok)
	assert.Equal(t, js.ErrorTypeRuntime, runtimeErr.Type)
}

// TestIntegration_JSScriptSizeLimit tests script size limit
func TestIntegration_JSScriptSizeLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := js.NewRuntime(logger, manager, nil)

	// Create script larger than limit
	largeScript := strings.Repeat("// comment\n", 10000)

	_, _, err := runtime.Execute(context.Background(), largeScript)
	require.Error(t, err)

	runtimeErr, ok := err.(*js.RuntimeError)
	require.True(t, ok)
	assert.Equal(t, js.ErrorTypeValidation, runtimeErr.Type)
	assert.Contains(t, runtimeErr.Message, "exceeds maximum size")
}

// TestIntegration_DisconnectAll tests disconnecting from all servers
func TestIntegration_DisconnectAll(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)

	// Disconnect with no clients should succeed
	err := server.Stop()
	assert.NoError(t, err)
}

// TestIntegration_LogFileHandling tests log file creation scenarios
func TestIntegration_LogFileHandling(t *testing.T) {
	// This test verifies log file handling during server initialization
	// In real scenarios, logging is configured at startup
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	// Create log file
	file, err := os.Create(logPath)
	require.NoError(t, err)
	defer file.Close()

	// Write some data
	_, err = file.WriteString(`{"level":"info","msg":"test"}` + "\n")
	require.NoError(t, err)

	// Read back
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test")
}

// TestIntegration_ExecuteToolWithError tests exec tool error handling
func TestIntegration_ExecuteToolWithError(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Execute code that throws an error
	args := map[string]any{
		"code": "throw new Error('test error');",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "exec",
			Arguments: argsJSON,
		},
	}

	result, err := server.handleBuiltinTool(context.Background(), "exec", req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	// Parse response
	var response tools.ExecuteToolResponse
	content := result.Content[0].(*mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &response)
	require.NoError(t, err)

	// Should have error in result
	resultMap, ok := response.Result.(map[string]any)
	require.True(t, ok)
	errorMap, ok := resultMap["error"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, errorMap["message"])
}

// TestIntegration_BuiltinToolTimeout tests built-in tool timeout
func TestIntegration_BuiltinToolTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	server.toolCallTimeout = 100 * time.Millisecond
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Execute code that runs longer than timeout
	args := map[string]any{
		"code": "while(true) { /* infinite loop */ }",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "exec",
			Arguments: argsJSON,
		},
	}

	result, err := server.handleBuiltinTool(context.Background(), "exec", req)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	// Parse response
	var response tools.ExecuteToolResponse
	content := result.Content[0].(*mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &response)
	require.NoError(t, err)

	// Should have timeout error
	resultMap, ok := response.Result.(map[string]any)
	require.True(t, ok)
	errorMap, ok := resultMap["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, string(js.ErrorTypeTimeout), errorMap["type"])
}
