package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"hub/internal/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestHandleInvokeTool_EmptyName tests error when name is empty
func TestHandleInvokeTool_EmptyName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "invoke",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInvokeTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

// TestHandleInvokeTool_NoNamespace tests error when name lacks namespace
func TestHandleInvokeTool_NoNamespace(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{
		"name": "toolwithoutnamespace",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "invoke",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInvokeTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be namespaced")
}

// TestHandleInvokeTool_EmptyServerID tests error when server ID is empty
func TestHandleInvokeTool_EmptyServerID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{
		"name": "__toolname",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "invoke",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInvokeTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server ID cannot be empty")
}

// TestHandleInvokeTool_EmptyToolName tests error when tool name is empty
func TestHandleInvokeTool_EmptyToolName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{
		"name": "server__",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "invoke",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInvokeTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool name cannot be empty")
}

// TestHandleInvokeTool_ServerNotFound tests error when server doesn't exist
func TestHandleInvokeTool_ServerNotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{
		"name": "nonexistent__tool",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "invoke",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInvokeTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not found")
}

// TestHandleInvokeTool_NameTooLong tests error when name exceeds max length
func TestHandleInvokeTool_NameTooLong(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create name longer than 500 characters
	longName := "server__" + strings.Repeat("a", 500)
	args := map[string]any{
		"name": longName,
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "invoke",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInvokeTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name too long")
}

// TestHandleInvokeTool_InvalidJSON tests error when arguments are invalid JSON
func TestHandleInvokeTool_InvalidJSON(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "invoke",
			Arguments: []byte("invalid json"),
		},
	}

	_, err := HandleInvokeTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestHandleInvokeTool_ContextCancellation tests context cancellation handling
func TestHandleInvokeTool_ContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{
		"name": "server__tool",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "invoke",
			Arguments: argsJSON,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = HandleInvokeTool(ctx, manager, req)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
