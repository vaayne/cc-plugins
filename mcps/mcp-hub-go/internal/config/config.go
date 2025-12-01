package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the MCP hub configuration
type Config struct {
	Version       string                 `json:"version,omitempty"`
	MCPServers    map[string]MCPServer   `json:"mcpServers"`
	BuiltinTools  map[string]BuiltinTool `json:"builtinTools,omitempty"`
}

// MCPServer represents a remote MCP server configuration
type MCPServer struct {
	Transport string            `json:"transport,omitempty"` // defaults to "stdio"
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
	Enable    *bool             `json:"enable,omitempty"` // pointer to distinguish between false and unset
	Required  bool              `json:"required,omitempty"`
}

// IsEnabled returns true if the server should be enabled (default true if not specified)
func (s *MCPServer) IsEnabled() bool {
	if s.Enable == nil {
		return true
	}
	return *s.Enable
}

// GetTransport returns the transport type, defaulting to "stdio" if not specified
func (s *MCPServer) GetTransport() string {
	if s.Transport == "" {
		return "stdio"
	}
	return s.Transport
}

// BuiltinTool represents a built-in tool configuration
type BuiltinTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Script      string                 `json:"script"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(configPath string) (*Config, error) {
	// Read file contents
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Initialize maps if nil to prevent panics
	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]MCPServer)
	}
	if cfg.BuiltinTools == nil {
		cfg.BuiltinTools = make(map[string]BuiltinTool)
	}

	// Initialize nested maps for each server
	for name, server := range cfg.MCPServers {
		if server.Env == nil {
			server.Env = make(map[string]string)
		}
		cfg.MCPServers[name] = server
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	if c.MCPServers == nil || len(c.MCPServers) == 0 {
		return fmt.Errorf("mcpServers is required and must contain at least one server")
	}

	for name, server := range c.MCPServers {
		if err := validateServer(name, server); err != nil {
			return err
		}
	}

	return nil
}

// validateServer validates a single MCP server configuration
func validateServer(name string, server MCPServer) error {
	// Validate server name
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("server %q: name exceeds maximum length of 255", name)
	}
	// Reject dangerous characters in names
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("server %q: name contains invalid characters", name)
	}

	// Check transport (default to stdio if empty)
	transport := strings.ToLower(server.GetTransport())
	switch transport {
	case "stdio":
		// Valid - this is the only supported transport
	case "http", "sse":
		return fmt.Errorf("server %q: unsupported transport: %s (only stdio is supported)", name, server.GetTransport())
	default:
		return fmt.Errorf("server %q: invalid transport: %s (must be stdio)", name, server.GetTransport())
	}

	// For stdio transport, command is required
	if server.Command == "" {
		return fmt.Errorf("server %q: command is required for stdio transport", name)
	}

	// Validate command path
	if err := validateCommandPath(server.Command); err != nil {
		return fmt.Errorf("server %q: %w", name, err)
	}

	// Validate args
	const maxArgs = 100
	const maxArgLength = 4096
	
	if len(server.Args) > maxArgs {
		return fmt.Errorf("server %q: too many args (max %d)", name, maxArgs)
	}
	
	for i, arg := range server.Args {
		if len(arg) > maxArgLength {
			return fmt.Errorf("server %q: arg[%d] exceeds maximum length of %d", name, i, maxArgLength)
		}
		
		// Check for path traversal
		if strings.Contains(arg, "..") {
			return fmt.Errorf("server %q: arg[%d] contains path traversal sequence: %s", name, i, arg)
		}
		
		// Check for shell metacharacters
		if err := validateNoShellMetachars(arg); err != nil {
			return fmt.Errorf("server %q: arg[%d] %w", name, i, err)
		}
	}

	// Validate environment variables
	if err := validateEnvironment(name, server.Env); err != nil {
		return err
	}

	// If stdio transport, URL should not be set
	if server.URL != "" {
		return fmt.Errorf("server %q: url should not be set for stdio transport", name)
	}

	return nil
}

// validateCommandPath validates a command path for security issues
func validateCommandPath(command string) error {
	const maxCommandLength = 1024
	
	if len(command) > maxCommandLength {
		return fmt.Errorf("command exceeds maximum length of %d", maxCommandLength)
	}
	
	// Check for path traversal
	if strings.Contains(command, "..") {
		return fmt.Errorf("invalid command path (contains path traversal): %s", command)
	}

	// Check for tilde expansion (security risk)
	if strings.HasPrefix(command, "~") {
		return fmt.Errorf("invalid command path (tilde expansion not allowed): %s", command)
	}

	// Check for null bytes
	if strings.Contains(command, "\x00") {
		return fmt.Errorf("invalid command path (contains null byte): %s", command)
	}

	// Check for shell metacharacters
	if err := validateNoShellMetachars(command); err != nil {
		return fmt.Errorf("invalid command path: %w", err)
	}

	// Block shell interpreters
	bannedCommands := []string{"sh", "bash", "zsh", "ksh", "csh", "tcsh", "fish", "dash", "ash"}
	commandBase := filepath.Base(command)
	for _, banned := range bannedCommands {
		if commandBase == banned {
			return fmt.Errorf("shell interpreters are not allowed: %s", commandBase)
		}
	}

	return nil
}

// validateNoShellMetachars checks for dangerous shell metacharacters
func validateNoShellMetachars(s string) error {
	dangerousChars := []string{";", "|", "&", "$", "`", ">", "<", "\n", "\r", "$(", "${"}
	for _, char := range dangerousChars {
		if strings.Contains(s, char) {
			return fmt.Errorf("contains dangerous character %q", char)
		}
	}
	return nil
}

// validateEnvironment validates environment variables for security
func validateEnvironment(serverName string, env map[string]string) error {
	// Dangerous environment variables that can be used for code injection
	dangerousEnvVars := []string{
		"LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES",
		"DYLD_LIBRARY_PATH", "PATH", "PYTHONPATH", "NODE_PATH",
		"PERL5LIB", "RUBY_LIB", "CLASSPATH",
	}
	
	for key, value := range env {
		keyUpper := strings.ToUpper(key)
		for _, dangerous := range dangerousEnvVars {
			if keyUpper == dangerous {
				return fmt.Errorf("server %q: dangerous environment variable not allowed: %s", serverName, key)
			}
		}
		
		// Validate env values for shell metacharacters
		if err := validateNoShellMetachars(value); err != nil {
			return fmt.Errorf("server %q: env var %q value %w", serverName, key, err)
		}
		
		// Check for null bytes
		if strings.Contains(value, "\x00") {
			return fmt.Errorf("server %q: env var %q contains null byte", serverName, key)
		}
	}
	
	return nil
}
