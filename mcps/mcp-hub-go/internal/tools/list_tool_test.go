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

// List tool request handling
func TestHandleListTool_NoResults(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	args := map[string]any{
		"query": "nonexistent",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list",
			Arguments: argsJSON,
		},
	}

	result, err := HandleListTool(context.Background(), manager, req)
	require.NoError(t, err)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	// Output is now plain text with JS stubs, not JSON
	assert.Contains(t, textContent.Text, "// Total: 0 tools")
}

func TestHandleListTool_NoArgs(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list",
			Arguments: json.RawMessage(`{}`),
		},
	}

	result, err := HandleListTool(context.Background(), manager, req)
	require.NoError(t, err)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	// Output is now plain text with JS stubs, not JSON
	assert.Contains(t, textContent.Text, "// Total: 0 tools")
}

func TestHandleListTool_InvalidJSON(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list",
			Arguments: json.RawMessage(`{invalid json}`),
		},
	}

	_, err := HandleListTool(context.Background(), manager, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse list arguments")
}

// Keyword matching helpers
func TestMatchesKeywords(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		description string
		query       string
		expected    bool
	}{
		{"Match by name", "search", "Search for things", "search", true},
		{"Match by description", "tool1", "Search for things", "search", true},
		{"Case insensitive match", "MyTool", "Description", "mytool", true},
		{"Partial match", "filesystem", "Work with files", "file", true},
		{"No match", "database", "Database operations", "file", false},
		{"Empty query matches all", "anything", "Any description", "", true},
		{"Multiple keywords - first matches", "fileReader", "Reads files", "file,write,delete", true},
		{"Multiple keywords - second matches", "writer", "Writes data", "file,write,delete", true},
		{"Multiple keywords - none match", "database", "Database operations", "file,write,delete", false},
		{"Multiple keywords - all match", "fileWriter", "Write and delete files safely", "file,write,delete", true},
		{"Keywords with spaces", "fileReader", "Reads files", "file , write , delete", true},
		{"Empty keywords ignored", "fileReader", "Reads files", "file,,,write", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesKeywords(tt.toolName, tt.description, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
