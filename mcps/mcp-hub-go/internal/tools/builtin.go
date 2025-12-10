package tools

import (
	"sync"

	"mcp-hub-go/internal/config"

	"go.uber.org/zap"
)

// BuiltinToolRegistry manages built-in tools
type BuiltinToolRegistry struct {
	logger *zap.Logger
	tools  map[string]config.BuiltinTool
	mu     sync.RWMutex
}

// NewBuiltinToolRegistry creates a new registry
func NewBuiltinToolRegistry(logger *zap.Logger) *BuiltinToolRegistry {
	return &BuiltinToolRegistry{
		logger: logger,
		tools:  make(map[string]config.BuiltinTool),
	}
}

// RegisterTool adds a tool to the registry
func (r *BuiltinToolRegistry) RegisterTool(tool config.BuiltinTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logger.Info("Registering built-in tool", zap.String("name", tool.Name))
	r.tools[tool.Name] = tool
}

// GetTool retrieves a tool from the registry
func (r *BuiltinToolRegistry) GetTool(name string) (config.BuiltinTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, exists := r.tools[name]
	return tool, exists
}

// GetAllTools returns all registered tools
func (r *BuiltinToolRegistry) GetAllTools() map[string]config.BuiltinTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Return a copy to prevent external modification
	toolsCopy := make(map[string]config.BuiltinTool, len(r.tools))
	for k, v := range r.tools {
		toolsCopy[k] = v
	}
	return toolsCopy
}
