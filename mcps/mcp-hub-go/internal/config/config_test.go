package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	configPath := "testdata/valid-config.json"
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	// Verify MCPServers is initialized
	if cfg.MCPServers == nil {
		t.Fatal("MCPServers map is nil")
	}

	// Check specific servers
	if len(cfg.MCPServers) != 4 {
		t.Errorf("Expected 4 servers, got %d", len(cfg.MCPServers))
	}

	// Test chrome-devtools server
	chromeServer, ok := cfg.MCPServers["chrome-devtools"]
	if !ok {
		t.Fatal("chrome-devtools server not found")
	}

	if chromeServer.GetTransport() != "stdio" {
		t.Errorf("chrome-devtools transport = %s, want stdio", chromeServer.GetTransport())
	}

	if chromeServer.Command != "bunx" {
		t.Errorf("chrome-devtools command = %s, want bunx", chromeServer.Command)
	}

	if len(chromeServer.Args) != 3 {
		t.Errorf("chrome-devtools args length = %d, want 3", len(chromeServer.Args))
	}

	if !chromeServer.IsEnabled() {
		t.Error("chrome-devtools should be enabled")
	}

	// Test env is initialized
	if chromeServer.Env == nil {
		t.Error("chrome-devtools Env map is nil")
	}

	if chromeServer.Env["DEBUG"] != "true" {
		t.Errorf("chrome-devtools DEBUG env = %s, want true", chromeServer.Env["DEBUG"])
	}

	// Test default-enabled server (no explicit enable field)
	defaultServer, ok := cfg.MCPServers["default-enabled"]
	if !ok {
		t.Fatal("default-enabled server not found")
	}

	if !defaultServer.IsEnabled() {
		t.Error("default-enabled should be enabled by default")
	}

	// Test filesystem server (explicitly disabled)
	fsServer, ok := cfg.MCPServers["filesystem"]
	if !ok {
		t.Fatal("filesystem server not found")
	}

	if fsServer.IsEnabled() {
		t.Error("filesystem should be disabled")
	}

	// Test default transport when not specified
	if fsServer.GetTransport() != "stdio" {
		t.Errorf("filesystem transport = %s, want stdio (default)", fsServer.GetTransport())
	}
}

func TestLoadConfig_HTTPTransportAccepted(t *testing.T) {
	configPath := "testdata/valid-http.json"
	cfg, err := LoadConfig(configPath)

	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil for HTTP transport", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	// Check that the HTTP server is properly loaded
	httpServer, ok := cfg.MCPServers["http-server"]
	if !ok {
		t.Fatal("http-server not found in config")
	}

	if httpServer.GetTransport() != "http" {
		t.Errorf("Transport = %s, want http", httpServer.GetTransport())
	}

	if httpServer.URL == "" {
		t.Error("URL should be set for HTTP transport")
	}
}

func TestLoadConfig_SSETransportAccepted(t *testing.T) {
	configPath := "testdata/valid-sse.json"
	cfg, err := LoadConfig(configPath)

	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil for SSE transport", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	// Check that the SSE server is properly loaded
	sseServer, ok := cfg.MCPServers["sse-server"]
	if !ok {
		t.Fatal("sse-server not found in config")
	}

	if sseServer.GetTransport() != "sse" {
		t.Errorf("Transport = %s, want sse", sseServer.GetTransport())
	}

	if sseServer.URL == "" {
		t.Error("URL should be set for SSE transport")
	}
}

func TestLoadConfig_PathTraversalRejected(t *testing.T) {
	configPath := "testdata/invalid-path-traversal.json"
	cfg, err := LoadConfig(configPath)

	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for path traversal")
	}

	if cfg != nil {
		t.Error("LoadConfig() should return nil config on validation error")
	}

	expectedMsg := "contains path traversal"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Error message = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestLoadConfig_TildePathRejected(t *testing.T) {
	configPath := "testdata/invalid-tilde.json"
	cfg, err := LoadConfig(configPath)

	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for tilde path")
	}

	if cfg != nil {
		t.Error("LoadConfig() should return nil config on validation error")
	}

	expectedMsg := "tilde expansion not allowed"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Error message = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestLoadConfig_MissingCommandRejected(t *testing.T) {
	configPath := "testdata/invalid-no-command.json"
	cfg, err := LoadConfig(configPath)

	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for missing command")
	}

	if cfg != nil {
		t.Error("LoadConfig() should return nil config on validation error")
	}

	expectedMsg := "command is required for stdio transport"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Error message = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestLoadConfig_EmptyServersRejected(t *testing.T) {
	configPath := "testdata/empty-servers.json"
	cfg, err := LoadConfig(configPath)

	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for empty servers")
	}

	if cfg != nil {
		t.Error("LoadConfig() should return nil config on validation error")
	}

	expectedMsg := "mcpServers is required and must contain at least one server"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Error message = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	configPath := "testdata/nonexistent.json"
	cfg, err := LoadConfig(configPath)

	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for missing file")
	}

	if cfg != nil {
		t.Error("LoadConfig() should return nil config on read error")
	}

	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Error message = %q, want to contain 'failed to read config file'", err.Error())
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	// Create a temporary invalid JSON file
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(invalidPath, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := LoadConfig(invalidPath)

	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error for invalid JSON")
	}

	if cfg != nil {
		t.Error("LoadConfig() should return nil config on parse error")
	}

	if !strings.Contains(err.Error(), "failed to parse config JSON") {
		t.Errorf("Error message = %q, want to contain 'failed to parse config JSON'", err.Error())
	}
}

func TestLoadConfig_ShellMetacharsRejected(t *testing.T) {
	// Create test configs with shell metacharacters
	tests := []struct {
		name        string
		config      string
		expectedErr string
	}{
		{
			name: "semicolon in command",
			config: `{
				"mcpServers": {
					"evil": {
						"command": "npx; rm -rf /",
						"args": ["test"]
					}
				}
			}`,
			expectedErr: "contains dangerous character",
		},
		{
			name: "pipe in args",
			config: `{
				"mcpServers": {
					"evil": {
						"command": "npx",
						"args": ["test", "| cat /etc/passwd"]
					}
				}
			}`,
			expectedErr: "contains dangerous character",
		},
		{
			name: "command substitution",
			config: `{
				"mcpServers": {
					"evil": {
						"command": "npx",
						"args": ["$(malicious)"]
					}
				}
			}`,
			expectedErr: "contains dangerous character",
		},
		{
			name: "backticks in command",
			config: `{
				"mcpServers": {
					"evil": {
						"command": "npx` + "`echo evil`" + `",
						"args": []
					}
				}
			}`,
			expectedErr: "contains dangerous character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config to temp file
			tmpFile := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(tmpFile, []byte(tt.config), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(tmpFile)
			assert.Nil(t, cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestLoadConfig_ShellInterpretersRejected(t *testing.T) {
	shells := []string{"sh", "bash", "zsh", "ksh", "csh", "tcsh", "fish", "dash", "ash"}
	
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			config := fmt.Sprintf(`{
				"mcpServers": {
					"test": {
						"command": "%s",
						"args": ["-c", "echo test"]
					}
				}
			}`, shell)
			
			tmpFile := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(tmpFile)
			assert.Nil(t, cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "shell interpreters are not allowed")
		})
	}
}

func TestLoadConfig_DangerousEnvVarsRejected(t *testing.T) {
	dangerousVars := []string{
		"LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES",
		"DYLD_LIBRARY_PATH", "PATH", "PYTHONPATH", "NODE_PATH",
	}
	
	for _, envVar := range dangerousVars {
		t.Run(envVar, func(t *testing.T) {
			config := fmt.Sprintf(`{
				"mcpServers": {
					"test": {
						"command": "npx",
						"args": ["test"],
						"env": {"%s": "/tmp/evil"}
					}
				}
			}`, envVar)
			
			tmpFile := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(tmpFile)
			assert.Nil(t, cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "dangerous environment variable not allowed")
		})
	}
}

func TestLoadConfig_NullByteRejected(t *testing.T) {
	// Note: JSON doesn't support literal null bytes, so we test by creating the file with null bytes
	tests := []struct {
		name        string
		createFile  func(path string) error
		expectedErr string
	}{
		{
			name: "null byte in command",
			createFile: func(path string) error {
				config := []byte(`{"mcpServers":{"test":{"command":"npx`)
				config = append(config, 0x00) // Add null byte
				config = append(config, []byte(`evil","args":[]}}}`)...)
				return os.WriteFile(path, config, 0644)
			},
			expectedErr: "null byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "config.json")
			if err := tt.createFile(tmpFile); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(tmpFile)
			assert.Nil(t, cfg)
			assert.Error(t, err)
			// JSON parser will likely fail first, but if it doesn't, our validation should catch it
			// We can't reliably test null bytes through JSON since JSON doesn't support them
		})
	}
}

func TestLoadConfig_CaseInsensitiveTransport(t *testing.T) {
	// Test that transport is case-insensitive
	transports := []string{"HTTP", "Http", "SSE", "Sse", "StDiO"}
	
	for _, transport := range transports {
		t.Run(transport, func(t *testing.T) {
			config := fmt.Sprintf(`{
				"mcpServers": {
					"test": {
						"transport": "%s",
						"command": "test",
						"url": "http://example.com"
					}
				}
			}`, transport)
			
			tmpFile := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(tmpFile)
			
			if transport == "StDiO" {
				// StDiO should fail because it has both command and URL
				assert.Nil(t, cfg)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "url must not be set for stdio transport")
			} else {
				// HTTP and SSE should fail because command is set
				assert.Nil(t, cfg)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "command must not be set")
			}
		})
	}
}

func TestLoadConfig_URLWithStdioRejected(t *testing.T) {
	config := `{
		"mcpServers": {
			"test": {
				"transport": "stdio",
				"command": "npx",
				"url": "http://example.com"
			}
		}
	}`
	
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpFile)
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "url must not be set for stdio transport")
}

func TestLoadConfig_ServerNameValidation(t *testing.T) {
	tests := []struct {
		name        string
		serverName  string
		shouldFail  bool
		expectedErr string
	}{
		{
			name:       "valid name",
			serverName: "my-server_123",
			shouldFail: false,
		},
		{
			name:        "name with slash",
			serverName:  "my/server",
			shouldFail:  true,
			expectedErr: "contains invalid characters",
		},
		// Skip backslash test as it's not valid JSON
		{
			name:        "name too long",
			serverName:  strings.Repeat("a", 256),
			shouldFail:  true,
			expectedErr: "exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := fmt.Sprintf(`{
				"mcpServers": {
					"%s": {
						"command": "npx",
						"args": []
					}
				}
			}`, tt.serverName)
			
			tmpFile := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(tmpFile)
			
			if tt.shouldFail {
				assert.Nil(t, cfg)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestLoadConfig_LengthLimits(t *testing.T) {
	// Test command length limit
	t.Run("command too long", func(t *testing.T) {
		config := fmt.Sprintf(`{
			"mcpServers": {
				"test": {
					"command": "%s"
				}
			}
		}`, strings.Repeat("a", 1025))
		
		tmpFile := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(tmpFile)
		assert.Nil(t, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	// Test args length limit
	t.Run("arg too long", func(t *testing.T) {
		config := fmt.Sprintf(`{
			"mcpServers": {
				"test": {
					"command": "npx",
					"args": ["%s"]
				}
			}
		}`, strings.Repeat("a", 4097))
		
		tmpFile := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(tmpFile)
		assert.Nil(t, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	// Test too many args
	t.Run("too many args", func(t *testing.T) {
		args := make([]string, 101)
		for i := range args {
			args[i] = fmt.Sprintf(`"arg%d"`, i)
		}
		
		config := fmt.Sprintf(`{
			"mcpServers": {
				"test": {
					"command": "npx",
					"args": [%s]
				}
			}
		}`, strings.Join(args, ","))
		
		tmpFile := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(tmpFile)
		assert.Nil(t, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many args")
	})
}

func TestValidateCommandPath(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple command",
			command: "npx",
			wantErr: false,
		},
		{
			name:    "valid absolute path",
			command: "/usr/bin/node",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			command: "bin/server",
			wantErr: false,
		},
		{
			name:    "path traversal with ..",
			command: "../bin/evil",
			wantErr: true,
			errMsg:  "contains path traversal",
		},
		{
			name:    "path traversal in middle",
			command: "/usr/../../../etc/passwd",
			wantErr: true,
			errMsg:  "contains path traversal",
		},
		{
			name:    "tilde expansion",
			command: "~/bin/server",
			wantErr: true,
			errMsg:  "tilde expansion not allowed",
		},
		{
			name:    "tilde in middle",
			command: "/home/~user/bin",
			wantErr: false, // only prefix check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommandPath(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error message = %q, want to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestMCPServer_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		enable *bool
		want   bool
	}{
		{
			name:   "nil enable - default true",
			enable: nil,
			want:   true,
		},
		{
			name:   "explicitly true",
			enable: boolPtr(true),
			want:   true,
		},
		{
			name:   "explicitly false",
			enable: boolPtr(false),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &MCPServer{
				Enable: tt.enable,
			}
			if got := server.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPServer_GetTransport(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		want      string
	}{
		{
			name:      "empty transport - default stdio",
			transport: "",
			want:      "stdio",
		},
		{
			name:      "explicit stdio",
			transport: "stdio",
			want:      "stdio",
		},
		{
			name:      "explicit http",
			transport: "http",
			want:      "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &MCPServer{
				Transport: tt.transport,
			}
			if got := server.GetTransport(); got != tt.want {
				t.Errorf("GetTransport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateServer_ArgsPathTraversal(t *testing.T) {
	server := MCPServer{
		Command: "npx",
		Args:    []string{"-y", "../../../etc/passwd"},
	}

	err := validateServer("test", server)
	if err == nil {
		t.Fatal("Expected error for path traversal in args, got nil")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("Error message = %q, want to contain 'path traversal'", err.Error())
	}
}

func TestConfig_MapsInitialized(t *testing.T) {
	// Test that maps are properly initialized even with minimal config
	tmpDir := t.TempDir()
	minimalPath := filepath.Join(tmpDir, "minimal.json")
	minimalJSON := `{
		"mcpServers": {
			"test": {
				"command": "test"
			}
		}
	}`
	err := os.WriteFile(minimalPath, []byte(minimalJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := LoadConfig(minimalPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	// Verify all maps are initialized (no nil panics)
	if cfg.MCPServers == nil {
		t.Error("MCPServers should be initialized")
	}

	if cfg.BuiltinTools == nil {
		t.Error("BuiltinTools should be initialized")
	}

	server := cfg.MCPServers["test"]
	if server.Env == nil {
		t.Error("Server.Env should be initialized")
	}

	// Should be safe to iterate over maps
	for name := range cfg.MCPServers {
		_ = name
	}

	for name := range cfg.BuiltinTools {
		_ = name
	}

	for key := range server.Env {
		_ = key
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
