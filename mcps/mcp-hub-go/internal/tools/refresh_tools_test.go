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

func TestHandleRefreshToolsTool_AllServers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	argsJSON, err := json.Marshal(map[string]any{})
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

func TestHandleRefreshToolsTool_SpecificServers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

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

	result, err := HandleRefreshToolsTool(context.Background(), manager, req)
	require.NoError(t, err)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	var response RefreshToolsResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	require.NoError(t, err)

	assert.Empty(t, response.Refreshed)
	assert.NotNil(t, response.Errors)
	assert.Contains(t, response.Errors, "server1")
	assert.Contains(t, response.Errors, "server2")
}
