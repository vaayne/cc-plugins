package config

// Config represents the MCP hub configuration
type Config struct {
	Version       string                 `json:"version" mapstructure:"version"`
	MCPServers    map[string]MCPServer   `json:"mcpServers" mapstructure:"mcpServers"`
	BuiltinTools  map[string]BuiltinTool `json:"builtinTools" mapstructure:"builtinTools"`
}

// MCPServer represents a remote MCP server configuration
type MCPServer struct {
	Command string            `json:"command" mapstructure:"command"`
	Args    []string          `json:"args,omitempty" mapstructure:"args"`
	Env     map[string]string `json:"env,omitempty" mapstructure:"env"`
	Enable  bool              `json:"enable" mapstructure:"enable"`
}

// BuiltinTool represents a built-in tool configuration
type BuiltinTool struct {
	Name        string                 `json:"name" mapstructure:"name"`
	Description string                 `json:"description" mapstructure:"description"`
	Script      string                 `json:"script" mapstructure:"script"`
	InputSchema map[string]interface{} `json:"inputSchema" mapstructure:"inputSchema"`
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// TODO: Implement config loading using viper
	return &Config{}, nil
}
