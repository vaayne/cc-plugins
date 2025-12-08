package tools

import (
	"context"
	"encoding/json"
	"testing"

	"mcp-hub-go/internal/client"
	"mcp-hub-go/internal/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestNewBuiltinToolRegistry verifies registry initialization
func TestNewBuiltinToolRegistry(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.logger)
	assert.NotNil(t, registry.tools)
	assert.Empty(t, registry.tools)
}

// TestRegisterTool verifies tool registration
func TestRegisterTool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

	tool := config.BuiltinTool{
		Name:        "test-tool",
		Description: "Test tool description",
		Script:      "console.log('test')",
	}

	registry.RegisterTool(tool)

	retrievedTool, exists := registry.GetTool("test-tool")
	assert.True(t, exists)
	assert.Equal(t, "test-tool", retrievedTool.Name)
	assert.Equal(t, "Test tool description", retrievedTool.Description)
}

// TestGetTool_NotFound verifies behavior when tool doesn't exist
func TestGetTool_NotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

	_, exists := registry.GetTool("nonexistent")
	assert.False(t, exists)
}

// TestGetAllTools verifies retrieving all tools
func TestGetAllTools(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

	// Register multiple tools
	tools := []config.BuiltinTool{
		{Name: "tool1", Description: "First tool"},
		{Name: "tool2", Description: "Second tool"},
		{Name: "tool3", Description: "Third tool"},
	}

	for _, tool := range tools {
		registry.RegisterTool(tool)
	}

	allTools := registry.GetAllTools()
	assert.Len(t, allTools, 3)
	assert.Contains(t, allTools, "tool1")
	assert.Contains(t, allTools, "tool2")
	assert.Contains(t, allTools, "tool3")

	// Verify it's a copy
	allTools["tool4"] = config.BuiltinTool{Name: "tool4"}
	allTools2 := registry.GetAllTools()
	assert.Len(t, allTools2, 3)
	assert.NotContains(t, allTools2, "tool4")
}

// TestHandleSearchTool_NoResults verifies empty search results
func TestHandleSearchTool_NoResults(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]interface{}{
		"query": "nonexistent",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "search",
			Arguments: argsJSON,
		},
	}

	result, err := HandleSearchTool(context.Background(), registry, manager, req)
	require.NoError(t, err)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	var response SearchToolsResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	require.NoError(t, err)

	assert.Empty(t, response.Tools)
}

// TestHandleSearchTool_MissingQuery verifies error on missing query
func TestHandleSearchTool_MissingQuery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "search",
			Arguments: json.RawMessage(`{}`),
		},
	}

	_, err := HandleSearchTool(context.Background(), registry, manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query is required")
}

// TestHandleSearchTool_InvalidJSON verifies error on invalid JSON
func TestHandleSearchTool_InvalidJSON(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "search",
			Arguments: json.RawMessage(`{invalid json}`),
		},
	}

	_, err := HandleSearchTool(context.Background(), registry, manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse search arguments")
}

// TestHandleExecuteTool_Success verifies execute tool with valid JavaScript
func TestHandleExecuteTool_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]interface{}{
		"code": "1 + 1",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "execute",
			Arguments: argsJSON,
		},
	}

	result, err := HandleExecuteTool(context.Background(), logger, manager, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	var response ExecuteToolResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	require.NoError(t, err)

	// JSON unmarshaling produces float64 for numbers
	assert.Equal(t, float64(2), response.Result)
	assert.Empty(t, response.Logs)
}

// TestHandleExecuteTool_WithLogs verifies execute tool logging
func TestHandleExecuteTool_WithLogs(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]interface{}{
		"code": "mcp.log('info', 'test message'); 42",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "execute",
			Arguments: argsJSON,
		},
	}

	result, err := HandleExecuteTool(context.Background(), logger, manager, req)
	require.NoError(t, err)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	var response ExecuteToolResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	require.NoError(t, err)

	// JSON unmarshaling produces float64 for numbers
	assert.Equal(t, float64(42), response.Result)
	assert.Len(t, response.Logs, 1)
	assert.Equal(t, "info", response.Logs[0].Level)
	assert.Equal(t, "test message", response.Logs[0].Message)
}

// TestHandleExecuteTool_AsyncSuccess verifies async/await works
func TestHandleExecuteTool_AsyncSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]interface{}{
		"code": "async function test() { return 42; } test();",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "execute",
			Arguments: argsJSON,
		},
	}

	result, err := HandleExecuteTool(context.Background(), logger, manager, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	var response ExecuteToolResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(42), response.Result)
}

// TestHandleExecuteTool_MissingCode verifies error on missing code
func TestHandleExecuteTool_MissingCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]interface{}{}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "execute",
			Arguments: argsJSON,
		},
	}

	_, err = HandleExecuteTool(context.Background(), logger, manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code parameter is required")
}

// TestHandleExecuteTool_CodeTooLarge verifies code size limit
func TestHandleExecuteTool_CodeTooLarge(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create code larger than 100KB
	largeCode := make([]byte, 101*1024)
	for i := range largeCode {
		largeCode[i] = 'a'
	}

	args := map[string]interface{}{
		"code": string(largeCode),
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "execute",
			Arguments: argsJSON,
		},
	}

	_, err = HandleExecuteTool(context.Background(), logger, manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

// TestHandleRefreshToolsTool_AllServers verifies refreshing all servers
func TestHandleRefreshToolsTool_AllServers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// No servers connected, should succeed with empty list
	args := map[string]interface{}{}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "refreshTools",
			Arguments: argsJSON,
		},
	}

	result, err := HandleRefreshToolsTool(context.Background(), manager, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	var response RefreshToolsResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	require.NoError(t, err)

	assert.Empty(t, response.Refreshed)
	assert.Nil(t, response.Errors)
}

// TestHandleRefreshToolsTool_SpecificServers verifies refreshing specific servers
func TestHandleRefreshToolsTool_SpecificServers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Request refresh for non-existent servers
	args := map[string]interface{}{
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

	result, err := HandleRefreshToolsTool(context.Background(), manager, req)
	require.NoError(t, err)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	var response RefreshToolsResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	require.NoError(t, err)

	// Should have errors for non-existent servers
	assert.Empty(t, response.Refreshed)
	assert.NotNil(t, response.Errors)
	assert.Contains(t, response.Errors, "server1")
	assert.Contains(t, response.Errors, "server2")
}

// TestMatchesTool verifies the matching logic
func TestMatchesTool(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		description string
		query       string
		expected    bool
	}{
		{
			name:        "Match by name",
			toolName:    "search",
			description: "Search for things",
			query:       "search",
			expected:    true,
		},
		{
			name:        "Match by description",
			toolName:    "tool1",
			description: "Search for things",
			query:       "search",
			expected:    true,
		},
		{
			name:        "Case insensitive match",
			toolName:    "MyTool",
			description: "Description",
			query:       "mytool",
			expected:    true,
		},
		{
			name:        "Partial match",
			toolName:    "filesystem",
			description: "Work with files",
			query:       "file",
			expected:    true,
		},
		{
			name:        "No match",
			toolName:    "database",
			description: "Database operations",
			query:       "file",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesTool(tt.toolName, tt.description, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuiltinToolRegistry_ThreadSafety verifies concurrent access
func TestBuiltinToolRegistry_ThreadSafety(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			tool := config.BuiltinTool{
				Name:        string(rune('a' + id)),
				Description: "Test tool",
			}
			registry.RegisterTool(tool)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		go func() {
			_ = registry.GetAllTools()
			_, _ = registry.GetTool("a")
			done <- true
		}()
	}

	// Wait for all reads
	for i := 0; i < 100; i++ {
		<-done
	}
}
