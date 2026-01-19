package tools

import (
	"testing"

	"mcphub/internal/config"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// Registry creation and CRUD
func TestNewBuiltinToolRegistry(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.logger)
	assert.NotNil(t, registry.tools)
	assert.Empty(t, registry.tools)
}

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

func TestGetTool_NotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

	_, exists := registry.GetTool("nonexistent")
	assert.False(t, exists)
}

func TestGetAllTools(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

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

	// Verify copy semantics
	allTools["tool4"] = config.BuiltinTool{Name: "tool4"}
	allTools2 := registry.GetAllTools()
	assert.Len(t, allTools2, 3)
	assert.NotContains(t, allTools2, "tool4")
}

// Concurrency safety
func TestBuiltinToolRegistry_ThreadSafety(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewBuiltinToolRegistry(logger)

	done := make(chan bool)

	// Concurrent writes
	for i := range 10 {
		go func(id int) {
			tool := config.BuiltinTool{
				Name:        string(rune('a' + id)),
				Description: "Test tool",
			}
			registry.RegisterTool(tool)
			done <- true
		}(i)
	}
	for range 10 {
		<-done
	}

	// Concurrent reads
	for range 100 {
		go func() {
			_ = registry.GetAllTools()
			_, _ = registry.GetTool("a")
			done <- true
		}()
	}
	for range 100 {
		<-done
	}
}
