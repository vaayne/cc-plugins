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

// TestHandleInspectTool_EmptyName tests error when name is empty
func TestHandleInspectTool_EmptyName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInspectTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

// TestHandleInspectTool_NoNamespace tests error when name lacks namespace
func TestHandleInspectTool_NoNamespace(t *testing.T) {
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
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInspectTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be namespaced")
}

// TestHandleInspectTool_ToolNotFound tests error when tool doesn't exist
func TestHandleInspectTool_ToolNotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{
		"name": "server__nonexistent",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInspectTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool not found")
}

// TestHandleInspectTool_NameTooLong tests error when name exceeds max length
func TestHandleInspectTool_NameTooLong(t *testing.T) {
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
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInspectTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name too long")
}

// TestHandleInspectTool_InvalidJSON tests error when arguments are invalid JSON
func TestHandleInspectTool_InvalidJSON(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: []byte("invalid json"),
		},
	}

	_, err := HandleInspectTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestHandleInspectTool_ContextCancellation tests context cancellation handling
func TestHandleInspectTool_ContextCancellation(t *testing.T) {
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
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = HandleInspectTool(ctx, manager, req)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
