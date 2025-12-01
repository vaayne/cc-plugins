package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"mcp-hub-go/internal/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// clientInfo holds information about a connected client
type clientInfo struct {
	serverID      string
	session       *mcp.ClientSession
	tools         map[string]*mcp.Tool // tool name -> tool schema
	mu            sync.RWMutex
	reconnecting  bool
	lastConnected time.Time
	backoff       time.Duration
	cancelFunc    context.CancelFunc
}

// Manager manages connections to remote MCP servers
type Manager struct {
	logger   *zap.Logger
	clients  map[string]*clientInfo // serverID -> client info
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	timeout  time.Duration
}

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	backoffFactor  = 2.0
	defaultTimeout = 60 * time.Second
)

// NewManager creates a new client manager
func NewManager(logger *zap.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		logger:  logger,
		clients: make(map[string]*clientInfo),
		ctx:     ctx,
		cancel:  cancel,
		timeout: defaultTimeout,
	}
}

// ConnectToServer connects to a remote MCP server
func (m *Manager) ConnectToServer(serverID string, serverCfg config.MCPServer) error {
	m.logger.Info("Connecting to remote MCP server", zap.String("serverID", serverID))

	// Validate transport
	transport := serverCfg.GetTransport()
	if transport != "stdio" {
		return fmt.Errorf("unsupported transport: %s (only stdio is supported)", transport)
	}

	// Check if already connected
	m.mu.RLock()
	if existing, ok := m.clients[serverID]; ok {
		m.mu.RUnlock()
		existing.mu.RLock()
		isConnected := existing.session != nil
		existing.mu.RUnlock()
		if isConnected {
			m.logger.Info("Already connected to server", zap.String("serverID", serverID))
			return nil
		}
	} else {
		m.mu.RUnlock()
	}

	// Create client info
	clientCtx, clientCancel := context.WithCancel(m.ctx)
	info := &clientInfo{
		serverID:      serverID,
		tools:         make(map[string]*mcp.Tool),
		backoff:       initialBackoff,
		lastConnected: time.Now(),
		cancelFunc:    clientCancel,
	}

	// Attempt connection
	if err := m.connectClient(clientCtx, info, serverCfg); err != nil {
		clientCancel()
		return fmt.Errorf("failed to connect to server %s: %w", serverID, err)
	}

	// Store client info
	m.mu.Lock()
	m.clients[serverID] = info
	m.mu.Unlock()

	// Start reconnection goroutine
	go m.maintainConnection(clientCtx, serverID, serverCfg, info)

	return nil
}

// connectClient establishes a connection to a remote MCP server
func (m *Manager) connectClient(ctx context.Context, info *clientInfo, serverCfg config.MCPServer) error {
	// Create command with context
	cmd := exec.CommandContext(ctx, serverCfg.Command, serverCfg.Args...)
	
	// SECURITY: Always set a clean environment to prevent injection attacks
	// Never inherit parent environment which could contain malicious variables
	cleanEnv := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
	}
	
	// Append user-provided environment variables
	for k, v := range serverCfg.Env {
		cleanEnv = append(cleanEnv, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = cleanEnv

	// Create transport
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	// Create client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-hub-go",
		Version: "v1.0.0",
	}, nil)

	// Connect with timeout
	connectCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Discover tools
	toolsCtx, toolsCancel := context.WithTimeout(ctx, m.timeout)
	defer toolsCancel()

	toolsResult, err := session.ListTools(toolsCtx, nil)
	if err != nil {
		// Clean up session with timeout on error
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		
		// Try to close with timeout
		done := make(chan struct{})
		go func() {
			session.Close()
			close(done)
		}()
		
		select {
		case <-done:
			// Closed successfully
		case <-closeCtx.Done():
			// Timeout - log and continue
			m.logger.Warn("Timeout closing session after ListTools error",
				zap.String("serverID", info.serverID),
				zap.Error(err))
		}
		
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Store session and tools
	info.mu.Lock()
	info.session = session
	info.tools = make(map[string]*mcp.Tool)
	for _, tool := range toolsResult.Tools {
		info.tools[tool.Name] = tool
	}
	info.lastConnected = time.Now()
	info.reconnecting = false
	info.mu.Unlock()

	m.logger.Info("Connected to server",
		zap.String("serverID", info.serverID),
		zap.Int("toolCount", len(toolsResult.Tools)),
	)

	return nil
}

// maintainConnection monitors the connection and handles reconnection
func (m *Manager) maintainConnection(ctx context.Context, serverID string, serverCfg config.MCPServer, info *clientInfo) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Wait for connection to fail
		info.mu.RLock()
		session := info.session
		info.mu.RUnlock()

		if session != nil {
			// Block until connection fails
			err := session.Wait()
			
			info.mu.Lock()
			if err != nil && ctx.Err() == nil {
				m.logger.Warn("Server connection lost",
					zap.String("serverID", serverID),
					zap.Error(err),
				)
				info.reconnecting = true
			}
			info.session = nil
			info.mu.Unlock()
		}

		// Check if context is done
		if ctx.Err() != nil {
			return
		}

		// Reconnect with backoff
		info.mu.Lock()
		if !info.reconnecting {
			info.mu.Unlock()
			return
		}
		backoff := info.backoff
		info.mu.Unlock()

		m.logger.Info("Attempting to reconnect",
			zap.String("serverID", serverID),
			zap.Duration("backoff", backoff),
		)

		// Wait before reconnecting
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}

		// Attempt reconnection
		if err := m.connectClient(ctx, info, serverCfg); err != nil {
			m.logger.Error("Failed to reconnect",
				zap.String("serverID", serverID),
				zap.Error(err),
			)

			// Increase backoff
			info.mu.Lock()
			info.backoff = time.Duration(float64(info.backoff) * backoffFactor)
			if info.backoff > maxBackoff {
				info.backoff = maxBackoff
			}
			info.mu.Unlock()
		} else {
			// Reset backoff on successful connection
			info.mu.Lock()
			info.backoff = initialBackoff
			info.reconnecting = false
			info.mu.Unlock()
		}
	}
}

// DisconnectAll disconnects from all remote servers
func (m *Manager) DisconnectAll() error {
	m.logger.Info("Disconnecting from all remote servers")
	
	// Cancel context to stop all goroutines
	m.cancel()
	
	// Collect all clients first to avoid holding lock during disconnect
	m.mu.Lock()
	clients := make([]*clientInfo, 0, len(m.clients))
	for _, info := range m.clients {
		clients = append(clients, info)
	}
	// Clear the registry
	m.clients = make(map[string]*clientInfo)
	m.mu.Unlock()
	
	// Disconnect each client with timeout
	var errs []error
	for _, info := range clients {
		// Cancel the client's context
		info.cancelFunc()
		
		info.mu.Lock()
		session := info.session
		info.session = nil
		info.mu.Unlock()
		
		if session != nil {
			// Close session with timeout
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			
			done := make(chan error, 1)
			go func() {
				done <- session.Close()
			}()
			
			select {
			case err := <-done:
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to disconnect from %s: %w", info.serverID, err))
				}
			case <-closeCtx.Done():
				errs = append(errs, fmt.Errorf("timeout disconnecting from %s", info.serverID))
			}
			
			closeCancel()
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors during disconnect: %v", errs)
	}
	
	return nil
}

// GetClient returns the client session for a server
func (m *Manager) GetClient(serverID string) (*mcp.ClientSession, error) {
	m.mu.RLock()
	info, ok := m.clients[serverID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	info.mu.RLock()
	session := info.session
	info.mu.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("server not connected: %s", serverID)
	}

	return session, nil
}

// ListClients returns the IDs of all connected clients
func (m *Manager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]string, 0, len(m.clients))
	for serverID := range m.clients {
		clients = append(clients, serverID)
	}
	return clients
}

// GetTools returns all tools from a specific server
func (m *Manager) GetTools(serverID string) (map[string]*mcp.Tool, error) {
	m.mu.RLock()
	info, ok := m.clients[serverID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	info.mu.RLock()
	defer info.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	tools := make(map[string]*mcp.Tool, len(info.tools))
	for name, tool := range info.tools {
		tools[name] = tool
	}

	return tools, nil
}

// GetAllTools returns all tools from all servers with namespace prefix
func (m *Manager) GetAllTools() map[string]*mcp.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allTools := make(map[string]*mcp.Tool)
	
	for serverID, info := range m.clients {
		info.mu.RLock()
		for toolName, tool := range info.tools {
			namespacedName := fmt.Sprintf("%s.%s", serverID, toolName)
			allTools[namespacedName] = tool
		}
		info.mu.RUnlock()
	}

	return allTools
}

// DetectNameCollisions returns tools with duplicate names across servers
func (m *Manager) DetectNameCollisions() map[string][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Map tool name to list of server IDs that provide it
	toolToServers := make(map[string][]string)

	for serverID, info := range m.clients {
		info.mu.RLock()
		for toolName := range info.tools {
			toolToServers[toolName] = append(toolToServers[toolName], serverID)
		}
		info.mu.RUnlock()
	}

	// Filter to only collisions (tools provided by multiple servers)
	collisions := make(map[string][]string)
	for toolName, serverIDs := range toolToServers {
		if len(serverIDs) > 1 {
			collisions[toolName] = serverIDs
		}
	}

	return collisions
}

// RefreshTools reloads tool manifests from a specific server
func (m *Manager) RefreshTools(serverID string) error {
	m.mu.RLock()
	info, ok := m.clients[serverID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("server not found: %s", serverID)
	}

	info.mu.RLock()
	session := info.session
	info.mu.RUnlock()

	if session == nil {
		return fmt.Errorf("server not connected: %s", serverID)
	}

	// List tools with timeout
	ctx, cancel := context.WithTimeout(m.ctx, m.timeout)
	defer cancel()

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Update tools
	info.mu.Lock()
	info.tools = make(map[string]*mcp.Tool)
	for _, tool := range toolsResult.Tools {
		info.tools[tool.Name] = tool
	}
	info.mu.Unlock()

	m.logger.Info("Refreshed tools",
		zap.String("serverID", serverID),
		zap.Int("toolCount", len(toolsResult.Tools)),
	)

	return nil
}
