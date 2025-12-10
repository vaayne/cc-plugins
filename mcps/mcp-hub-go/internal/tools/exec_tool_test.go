package tools

import (
	"context"
	"encoding/json"
	"testing"

	"mcp-hub-go/internal/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

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

	assert.Equal(t, float64(2), response.Result) // JSON numbers decode as float64
	assert.Empty(t, response.Logs)
}

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

	assert.Equal(t, float64(42), response.Result)
	assert.Len(t, response.Logs, 1)
	assert.Equal(t, "info", response.Logs[0].Level)
	assert.Equal(t, "test message", response.Logs[0].Message)
}

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

func TestHandleExecuteTool_CodeTooLarge(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

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
