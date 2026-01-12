package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"mcp-hub-go/internal/client"
	"mcp-hub-go/internal/config"
	"mcp-hub-go/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestNewServer verifies server initialization
func TestNewServer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)

	assert.NotNil(t, server)
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.logger)
	assert.Equal(t, 60*time.Second, server.toolCallTimeout)
	assert.Nil(t, server.mcpServer)
	assert.Nil(t, server.clientManager)
	assert.Nil(t, server.builtinRegistry)
}

// TestRegisterBuiltinTools verifies built-in tools are registered
func TestRegisterBuiltinTools(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)

	server.registerBuiltinTools()

	// Verify all built-in tools are registered
	allTools := server.builtinRegistry.GetAllTools()
	assert.Len(t, allTools, 3)

	// Verify list tool
	listTool, exists := server.builtinRegistry.GetTool("list")
	assert.True(t, exists)
	assert.Equal(t, "list", listTool.Name)
	assert.Contains(t, listTool.Description, "List MCP tools")
	assert.NotNil(t, listTool.InputSchema)

	// Verify exec tool
	execTool, exists := server.builtinRegistry.GetTool("exec")
	assert.True(t, exists)
	assert.Equal(t, "exec", execTool.Name)
	assert.Contains(t, execTool.Description, "Execute JavaScript code")
	assert.NotNil(t, execTool.InputSchema)

	// Verify refreshTools tool
	refreshTool, exists := server.builtinRegistry.GetTool("refreshTools")
	assert.True(t, exists)
	assert.Equal(t, "refreshTools", refreshTool.Name)
	assert.Contains(t, refreshTool.Description, "Refresh tool lists")
	assert.NotNil(t, refreshTool.InputSchema)
}

// TestConnectToRemoteServers_EmptyConfig verifies handling of empty config
func TestConnectToRemoteServers_EmptyConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	defer server.clientManager.DisconnectAll()

	err := server.connectToRemoteServers()
	assert.NoError(t, err)
}

// TestConnectToRemoteServers_DisabledServer verifies disabled servers are skipped
func TestConnectToRemoteServers_DisabledServer(t *testing.T) {
	logger := zaptest.NewLogger(t)

	disabled := false
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServer{
			"disabled-server": {
				Transport: "stdio",
				Command:   "echo",
				Args:      []string{"test"},
				Enable:    &disabled,
			},
		},
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	defer server.clientManager.DisconnectAll()

	err := server.connectToRemoteServers()
	assert.NoError(t, err)

	// Verify no clients were added
	clients := server.clientManager.ListClients()
	assert.Empty(t, clients)
}

// TestConnectToRemoteServers_RequiredServerFails verifies required server failure handling
func TestConnectToRemoteServers_RequiredServerFails(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: map[string]config.MCPServer{
			"required-server": {
				Transport: "stdio",
				Command:   "/nonexistent/command",
				Args:      []string{},
				Required:  true,
			},
		},
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	defer server.clientManager.DisconnectAll()

	err := server.connectToRemoteServers()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required server")
	assert.Contains(t, err.Error(), "required-server")
}

// TestConnectToRemoteServers_OptionalServerFails verifies optional server failure handling
func TestConnectToRemoteServers_OptionalServerFails(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &config.Config{
		MCPServers: map[string]config.MCPServer{
			"optional-server": {
				Transport: "stdio",
				Command:   "/nonexistent/command",
				Args:      []string{},
				Required:  false,
			},
		},
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	defer server.clientManager.DisconnectAll()

	// Should not return error for optional server failure
	err := server.connectToRemoteServers()
	assert.NoError(t, err)
}

// TestHandleBuiltinTool_Search verifies search tool routing
func TestHandleBuiltinTool_Search(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Create request
	args := map[string]any{
		"query": "list",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list",
			Arguments: argsJSON,
		},
	}

	result, err := server.handleBuiltinTool(context.Background(), "list", req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Content, 1)
}

// TestHandleBuiltinTool_Exec verifies exec tool routing
func TestHandleBuiltinTool_Exec(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Create request
	args := map[string]any{
		"code": "1 + 1;",
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
	assert.Len(t, result.Content, 1)
}

// TestHandleBuiltinTool_RefreshTools verifies refreshTools routing
func TestHandleBuiltinTool_RefreshTools(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	server.registerBuiltinTools()

	// Create request
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
	assert.Len(t, result.Content, 1)
}

// TestHandleBuiltinTool_UnknownTool verifies error on unknown tool
func TestHandleBuiltinTool_UnknownTool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)
	defer server.clientManager.DisconnectAll()

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "unknown",
			Arguments: []byte("{}"),
		},
	}

	_, err := server.handleBuiltinTool(context.Background(), "unknown", req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown built-in tool")
}

// TestRegisterBuiltinToolHandler verifies built-in tool registration
func TestRegisterBuiltinToolHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.mcpServer = mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "v1.0.0",
	}, nil)
	server.builtinRegistry = tools.NewBuiltinToolRegistry(logger)

	builtinTool := config.BuiltinTool{
		Name:        "test-tool",
		Description: "Test tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"param": map[string]any{
					"type": "string",
				},
			},
		},
	}

	err := server.registerBuiltinToolHandler("test-tool", builtinTool)
	assert.NoError(t, err)
}

// TestStop verifies server shutdown
func TestStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	server.clientManager = client.NewManager(logger)

	err := server.Stop()
	assert.NoError(t, err)
}

// TestStop_NoClientManager verifies graceful handling when no client manager
func TestStop_NoClientManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)
	// Don't initialize clientManager

	err := server.Stop()
	assert.NoError(t, err)
}

// TestNamespaceParsing verifies namespace parsing logic
func TestNamespaceParsing(t *testing.T) {
	tests := []struct {
		name           string
		namespacedName string
		expectedServer string
		expectedTool   string
		expectError    bool
	}{
		{
			name:           "Valid namespace",
			namespacedName: "server1.tool1",
			expectedServer: "server1",
			expectedTool:   "tool1",
			expectError:    false,
		},
		{
			name:           "Valid namespace with dots in tool name",
			namespacedName: "server1.tool.name.with.dots",
			expectedServer: "server1",
			expectedTool:   "tool.name.with.dots",
			expectError:    false,
		},
		{
			name:           "No dot",
			namespacedName: "invalidname",
			expectedServer: "",
			expectedTool:   "",
			expectError:    true,
		},
		{
			name:           "Empty server",
			namespacedName: ".tool1",
			expectedServer: "",
			expectedTool:   "tool1",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.SplitN(tt.namespacedName, ".", 2)

			if tt.expectError {
				assert.Len(t, parts, 1)
			} else {
				assert.Len(t, parts, 2)
				assert.Equal(t, tt.expectedServer, parts[0])
				assert.Equal(t, tt.expectedTool, parts[1])
			}
		})
	}
}

// TestServerTimeout verifies timeout configuration
func TestServerTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		MCPServers: make(map[string]config.MCPServer),
	}

	server := NewServer(cfg, logger)

	// Verify default timeout
	assert.Equal(t, 60*time.Second, server.toolCallTimeout)
}
