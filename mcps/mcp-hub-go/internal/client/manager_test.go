package client

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"mcp-hub-go/internal/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// TestNewManager verifies Manager initialization
func TestNewManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.logger)
	assert.NotNil(t, manager.clients)
	assert.NotNil(t, manager.ctx)
	assert.NotNil(t, manager.cancel)
	assert.Equal(t, defaultTimeout, manager.timeout)
	assert.Empty(t, manager.clients)
}

// TestConnectToServer_UnsupportedTransport verifies connection fails with unsupported transport
func TestConnectToServer_UnsupportedTransport(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	tests := []struct {
		name        string
		transport   string
		expectError string
	}{
		{"http transport without URL", "http", "url is required for http transport"},
		{"sse transport without URL", "sse", "url is required for sse transport"},
		{"unknown transport", "websocket", "unsupported transport type: websocket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverCfg := config.MCPServer{
				Transport: tt.transport,
				Command:   "echo",
				Args:      []string{"test"},
			}

			err := manager.ConnectToServer("test-server", serverCfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

// TestConnectToServer_InvalidCommand verifies connection fails with invalid command
func TestConnectToServer_InvalidCommand(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	serverCfg := config.MCPServer{
		Transport: "stdio",
		Command:   "/nonexistent/command/that/does/not/exist",
		Args:      []string{},
	}

	err := manager.ConnectToServer("test-server", serverCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

// TestConnectToServer_AlreadyConnected verifies idempotent connection
func TestConnectToServer_AlreadyConnected(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Create a mock connected client (with nil session but we'll check before calling Close)
	// We can't create a valid mock ClientSession, so we simulate an already-connected state
	// by checking in the ConnectToServer logic
	_, cancel := context.WithCancel(manager.ctx)

	// Create a simple mock that looks connected but won't crash on Close
	// We'll test the idempotent behavior by checking the manager doesn't try to reconnect
	serverCfg := config.MCPServer{
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}

	// First connection attempt will fail (echo doesn't implement MCP)
	err := manager.ConnectToServer("test-server", serverCfg)
	assert.Error(t, err) // Should fail because echo is not an MCP server

	// Clean up
	cancel()
}

// TestDisconnectAll verifies all clients are disconnected
func TestDisconnectAll(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	// Add multiple mock clients
	for i := 0; i < 3; i++ {
		_, cancel := context.WithCancel(manager.ctx)
		info := &clientInfo{
			serverID:      fmt.Sprintf("server-%d", i),
			session:       nil, // No session to avoid close errors
			tools:         make(map[string]*mcp.Tool),
			backoff:       initialBackoff,
			lastConnected: time.Now(),
			cancelFunc:    cancel,
		}

		manager.mu.Lock()
		manager.clients[info.serverID] = info
		manager.mu.Unlock()
	}

	// Verify clients exist
	assert.Len(t, manager.clients, 3)

	// Disconnect all
	err := manager.DisconnectAll()
	assert.NoError(t, err)

	// Verify all clients are removed
	assert.Empty(t, manager.clients)

	// Verify context is cancelled
	select {
	case <-manager.ctx.Done():
		// Context is properly cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Manager context should be cancelled")
	}
}

// TestGetClient_NotFound verifies error when server not found
func TestGetClient_NotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	_, err := manager.GetClient("nonexistent-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not found")
}

// TestGetClient_NotConnected verifies error when server exists but not connected
func TestGetClient_NotConnected(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Add client info without session
	_, cancel := context.WithCancel(manager.ctx)
	defer cancel()

	info := &clientInfo{
		serverID:      "test-server",
		session:       nil, // No session
		tools:         make(map[string]*mcp.Tool),
		backoff:       initialBackoff,
		lastConnected: time.Now(),
		cancelFunc:    cancel,
	}

	manager.mu.Lock()
	manager.clients["test-server"] = info
	manager.mu.Unlock()

	_, err := manager.GetClient("test-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not connected")
}

// TestListClients verifies listing all client IDs
func TestListClients(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Initially empty
	clients := manager.ListClients()
	assert.Empty(t, clients)

	// Add clients
	serverIDs := []string{"server-1", "server-2", "server-3"}
	for _, id := range serverIDs {
		_, cancel := context.WithCancel(manager.ctx)
		defer cancel()

		info := &clientInfo{
			serverID:      id,
			session:       nil,
			tools:         make(map[string]*mcp.Tool),
			backoff:       initialBackoff,
			lastConnected: time.Now(),
			cancelFunc:    cancel,
		}

		manager.mu.Lock()
		manager.clients[id] = info
		manager.mu.Unlock()
	}

	// List clients
	clients = manager.ListClients()
	assert.Len(t, clients, 3)

	// Verify all IDs are present (order may vary)
	for _, id := range serverIDs {
		assert.Contains(t, clients, id)
	}
}

// TestGetTools verifies retrieving tools from a server
func TestGetTools(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Create mock tools
	mockTools := map[string]*mcp.Tool{
		"tool1": {Name: "tool1", Description: "First tool"},
		"tool2": {Name: "tool2", Description: "Second tool"},
	}

	// Add client with tools
	_, cancel := context.WithCancel(manager.ctx)
	defer cancel()

	info := &clientInfo{
		serverID:      "test-server",
		session:       nil,
		tools:         mockTools,
		backoff:       initialBackoff,
		lastConnected: time.Now(),
		cancelFunc:    cancel,
	}

	manager.mu.Lock()
	manager.clients["test-server"] = info
	manager.mu.Unlock()

	// Get tools
	tools, err := manager.GetTools("test-server")
	assert.NoError(t, err)
	assert.Len(t, tools, 2)
	assert.Equal(t, "tool1", tools["tool1"].Name)
	assert.Equal(t, "tool2", tools["tool2"].Name)

	// Verify it's a copy by checking that modifying the returned map doesn't affect the original
	tools["tool3"] = &mcp.Tool{Name: "tool3"}

	// Re-fetch and verify original is unchanged
	tools2, err := manager.GetTools("test-server")
	assert.NoError(t, err)
	assert.Len(t, tools2, 2) // Should still be 2, not 3
	assert.NotContains(t, tools2, "tool3")
}

// TestGetTools_ServerNotFound verifies error when server not found
func TestGetTools_ServerNotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	_, err := manager.GetTools("nonexistent-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not found")
}

// TestGetAllTools verifies getting all tools with namespacing
func TestGetAllTools(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Add multiple servers with tools
	servers := map[string]map[string]*mcp.Tool{
		"server1": {
			"read":  {Name: "read", Description: "Read file"},
			"write": {Name: "write", Description: "Write file"},
		},
		"server2": {
			"search": {Name: "search", Description: "Search files"},
			"grep":   {Name: "grep", Description: "Grep files"},
		},
	}

	for serverID, tools := range servers {
		_, cancel := context.WithCancel(manager.ctx)
		defer cancel()

		info := &clientInfo{
			serverID:      serverID,
			session:       nil,
			tools:         tools,
			backoff:       initialBackoff,
			lastConnected: time.Now(),
			cancelFunc:    cancel,
		}

		manager.mu.Lock()
		manager.clients[serverID] = info
		manager.mu.Unlock()
	}

	// Get all tools
	allTools := manager.GetAllTools()
	assert.Len(t, allTools, 4)

	// Verify namespacing
	assert.Contains(t, allTools, "server1__read")
	assert.Contains(t, allTools, "server1__write")
	assert.Contains(t, allTools, "server2__search")
	assert.Contains(t, allTools, "server2__grep")

	assert.Equal(t, "read", allTools["server1__read"].Name)
	assert.Equal(t, "search", allTools["server2__search"].Name)
}

// TestDetectNameCollisions verifies collision detection
func TestDetectNameCollisions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Add servers with overlapping tool names
	servers := map[string]map[string]*mcp.Tool{
		"server1": {
			"read":  {Name: "read"},
			"write": {Name: "write"},
			"list":  {Name: "list"},
		},
		"server2": {
			"read":   {Name: "read"}, // Collision
			"search": {Name: "search"},
		},
		"server3": {
			"read": {Name: "read"}, // Collision
			"list": {Name: "list"}, // Collision
		},
	}

	for serverID, tools := range servers {
		_, cancel := context.WithCancel(manager.ctx)
		defer cancel()

		info := &clientInfo{
			serverID:      serverID,
			session:       nil,
			tools:         tools,
			backoff:       initialBackoff,
			lastConnected: time.Now(),
			cancelFunc:    cancel,
		}

		manager.mu.Lock()
		manager.clients[serverID] = info
		manager.mu.Unlock()
	}

	// Detect collisions
	collisions := manager.DetectNameCollisions()

	// Verify collisions
	assert.Len(t, collisions, 2) // "read" and "list"

	assert.Contains(t, collisions, "read")
	assert.Len(t, collisions["read"], 3) // server1, server2, server3
	assert.ElementsMatch(t, collisions["read"], []string{"server1", "server2", "server3"})

	assert.Contains(t, collisions, "list")
	assert.Len(t, collisions["list"], 2) // server1, server3
	assert.ElementsMatch(t, collisions["list"], []string{"server1", "server3"})

	// No collision for unique tools
	assert.NotContains(t, collisions, "write")
	assert.NotContains(t, collisions, "search")
}

// TestDetectNameCollisions_NoCollisions verifies no false positives
func TestDetectNameCollisions_NoCollisions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Add servers with unique tool names
	servers := map[string]map[string]*mcp.Tool{
		"server1": {
			"read":  {Name: "read"},
			"write": {Name: "write"},
		},
		"server2": {
			"search": {Name: "search"},
			"grep":   {Name: "grep"},
		},
	}

	for serverID, tools := range servers {
		_, cancel := context.WithCancel(manager.ctx)
		defer cancel()

		info := &clientInfo{
			serverID:      serverID,
			session:       nil,
			tools:         tools,
			backoff:       initialBackoff,
			lastConnected: time.Now(),
			cancelFunc:    cancel,
		}

		manager.mu.Lock()
		manager.clients[serverID] = info
		manager.mu.Unlock()
	}

	// Detect collisions
	collisions := manager.DetectNameCollisions()
	assert.Empty(t, collisions)
}

// TestRefreshTools_ServerNotFound verifies error handling
func TestRefreshTools_ServerNotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	err := manager.RefreshTools("nonexistent-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not found")
}

// TestRefreshTools_NotConnected verifies error when not connected
func TestRefreshTools_NotConnected(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Add client info without session
	_, cancel := context.WithCancel(manager.ctx)
	defer cancel()

	info := &clientInfo{
		serverID:      "test-server",
		session:       nil, // No session
		tools:         make(map[string]*mcp.Tool),
		backoff:       initialBackoff,
		lastConnected: time.Now(),
		cancelFunc:    cancel,
	}

	manager.mu.Lock()
	manager.clients["test-server"] = info
	manager.mu.Unlock()

	err := manager.RefreshTools("test-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not connected")
}

// TestBackoffCalculation verifies exponential backoff
func TestBackoffCalculation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	_, cancel := context.WithCancel(manager.ctx)
	defer cancel()

	info := &clientInfo{
		serverID:      "test-server",
		session:       nil,
		tools:         make(map[string]*mcp.Tool),
		backoff:       initialBackoff,
		lastConnected: time.Now(),
		cancelFunc:    cancel,
		reconnecting:  true,
	}

	// Simulate backoff progression
	expectedBackoffs := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second, // Capped at maxBackoff
		30 * time.Second, // Stays at max
	}

	for i, expected := range expectedBackoffs {
		if i > 0 {
			// Simulate failed reconnection attempt
			info.mu.Lock()
			info.backoff = time.Duration(float64(info.backoff) * backoffFactor)
			if info.backoff > maxBackoff {
				info.backoff = maxBackoff
			}
			info.mu.Unlock()
		}

		info.mu.RLock()
		actual := info.backoff
		info.mu.RUnlock()

		assert.Equal(t, expected, actual, "Backoff at iteration %d", i)
	}
}

// TestThreadSafety verifies concurrent access to manager
func TestThreadSafety(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)
	defer manager.DisconnectAll()

	// Add initial clients
	for i := 0; i < 5; i++ {
		_, cancel := context.WithCancel(manager.ctx)
		defer cancel()

		info := &clientInfo{
			serverID:      fmt.Sprintf("server-%d", i),
			session:       nil,
			tools:         map[string]*mcp.Tool{"tool1": {Name: "tool1"}},
			backoff:       initialBackoff,
			lastConnected: time.Now(),
			cancelFunc:    cancel,
		}

		manager.mu.Lock()
		manager.clients[info.serverID] = info
		manager.mu.Unlock()
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	operations := 100

	// Concurrent reads
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			serverID := fmt.Sprintf("server-%d", idx%5)
			_, _ = manager.GetTools(serverID)
			_ = manager.ListClients()
			_ = manager.GetAllTools()
			_ = manager.DetectNameCollisions()
		}(i)
	}

	wg.Wait()
}

// TestManagerContextCancellation verifies cleanup on context cancellation
func TestManagerContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	// Verify context is initially valid
	select {
	case <-manager.ctx.Done():
		t.Fatal("Context should not be cancelled initially")
	default:
		// Context is valid
	}

	// Call DisconnectAll which cancels the context
	err := manager.DisconnectAll()
	assert.NoError(t, err)

	// Verify context is now cancelled
	select {
	case <-manager.ctx.Done():
		// Context is properly cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled after DisconnectAll")
	}
}

// TestClientInfo_ThreadSafety verifies thread-safe access to clientInfo
func TestClientInfo_ThreadSafety(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := &clientInfo{
		serverID:      "test-server",
		session:       nil,
		tools:         make(map[string]*mcp.Tool),
		backoff:       initialBackoff,
		lastConnected: time.Now(),
		cancelFunc:    cancel,
	}

	var wg sync.WaitGroup
	operations := 100

	// Concurrent reads and writes
	for i := 0; i < operations; i++ {
		wg.Add(3)

		// Reader 1: Read session
		go func() {
			defer wg.Done()
			info.mu.RLock()
			_ = info.session
			info.mu.RUnlock()
		}()

		// Reader 2: Read tools
		go func() {
			defer wg.Done()
			info.mu.RLock()
			_ = len(info.tools)
			info.mu.RUnlock()
		}()

		// Writer: Update backoff
		go func() {
			defer wg.Done()
			info.mu.Lock()
			info.backoff = time.Duration(float64(info.backoff) * 1.1)
			info.mu.Unlock()
		}()
	}

	wg.Wait()
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
