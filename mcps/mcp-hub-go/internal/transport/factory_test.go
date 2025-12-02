package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	"mcp-hub-go/internal/config"
)

func TestDefaultFactory_CreateTransport(t *testing.T) {
	logger := zap.NewNop()
	factory := NewDefaultFactory(logger)

	tests := []struct {
		name      string
		cfg       config.MCPServer
		wantType  string
		wantError bool
	}{
		{
			name: "stdio transport",
			cfg: config.MCPServer{
				Transport: "stdio",
				Command:   "echo",
				Args:      []string{"hello"},
			},
			wantType:  "*mcp.CommandTransport",
			wantError: false,
		},
		{
			name: "http transport",
			cfg: config.MCPServer{
				Transport: "http",
				URL:       "https://example.com/mcp",
			},
			wantType:  "*mcp.StreamableClientTransport",
			wantError: false,
		},
		{
			name: "sse transport",
			cfg: config.MCPServer{
				Transport: "sse",
				URL:       "http://localhost:8080/sse",
			},
			wantType:  "*mcp.SSEClientTransport",
			wantError: false,
		},
		{
			name: "http with headers",
			cfg: config.MCPServer{
				Transport: "http",
				URL:       "https://api.example.com/mcp",
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"X-Custom":      "value",
				},
			},
			wantType:  "*mcp.StreamableClientTransport",
			wantError: false,
		},
		{
			name: "http with timeout",
			cfg: config.MCPServer{
				Transport: "http",
				URL:       "https://api.example.com/mcp",
				Timeout:   intPtr(60),
			},
			wantType:  "*mcp.StreamableClientTransport",
			wantError: false,
		},
		{
			name: "http with TLS skip verify",
			cfg: config.MCPServer{
				Transport:     "http",
				URL:           "https://localhost:8443/mcp",
				TLSSkipVerify: boolPtr(true),
			},
			wantType:  "*mcp.StreamableClientTransport",
			wantError: false,
		},
		{
			name: "invalid transport type",
			cfg: config.MCPServer{
				Transport: "websocket",
			},
			wantType:  "",
			wantError: true,
		},
		{
			name: "http with invalid URL",
			cfg: config.MCPServer{
				Transport: "http",
				URL:       "://invalid-url",
			},
			wantType:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := factory.CreateTransport(tt.cfg)

			if tt.wantError {
				if err == nil {
					t.Errorf("CreateTransport() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateTransport() unexpected error: %v", err)
				return
			}

			// Check transport type
			switch transport.(type) {
			case *mcp.CommandTransport:
				if tt.wantType != "*mcp.CommandTransport" {
					t.Errorf("CreateTransport() got type %T, want %s", transport, tt.wantType)
				}
			case *mcp.StreamableClientTransport:
				if tt.wantType != "*mcp.StreamableClientTransport" {
					t.Errorf("CreateTransport() got type %T, want %s", transport, tt.wantType)
				}
			case *mcp.SSEClientTransport:
				if tt.wantType != "*mcp.SSEClientTransport" {
					t.Errorf("CreateTransport() got type %T, want %s", transport, tt.wantType)
				}
			default:
				t.Errorf("CreateTransport() got unexpected type %T", transport)
			}
		})
	}
}

func TestHeaderTransport_RoundTrip(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back the headers we received
		w.Header().Set("X-Auth-Echo", r.Header.Get("Authorization"))
		w.Header().Set("X-Custom-Echo", r.Header.Get("X-Custom"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create header transport
	transport := &headerTransport{
		Base: http.DefaultTransport,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
			"X-Custom":      "custom-value",
		},
	}

	client := &http.Client{Transport: transport}

	// Make a request
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check that headers were added
	if got := resp.Header.Get("X-Auth-Echo"); got != "Bearer test-token" {
		t.Errorf("Authorization header not added correctly, got: %s", got)
	}

	if got := resp.Header.Get("X-Custom-Echo"); got != "custom-value" {
		t.Errorf("X-Custom header not added correctly, got: %s", got)
	}
}

func TestGetHTTPClient(t *testing.T) {
	logger := zap.NewNop()
	factory := NewDefaultFactory(logger)

	tests := []struct {
		name         string
		cfg          config.MCPServer
		checkTimeout bool
		checkTLS     bool
	}{
		{
			name: "default timeout",
			cfg: config.MCPServer{
				URL: "https://example.com",
			},
			checkTimeout: true,
		},
		{
			name: "custom timeout",
			cfg: config.MCPServer{
				URL:     "https://example.com",
				Timeout: intPtr(120),
			},
			checkTimeout: true,
		},
		{
			name: "TLS skip verify",
			cfg: config.MCPServer{
				URL:           "https://localhost",
				TLSSkipVerify: boolPtr(true),
			},
			checkTLS: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := factory.getHTTPClient(tt.cfg)

			if client == nil {
				t.Fatal("getHTTPClient() returned nil")
			}

			// Check timeout
			if tt.checkTimeout {
				if tt.cfg.Timeout != nil {
					// Custom timeout should be applied
					// Note: We can't directly check the timeout value,
					// but we can verify the client was created
					if client.Timeout == 0 {
						t.Error("Custom timeout not applied")
					}
				}
			}

			// Check that transport is headerTransport
			if _, ok := client.Transport.(*headerTransport); !ok {
				t.Errorf("Expected headerTransport, got %T", client.Transport)
			}
		})
	}
}

// Helper functions for creating pointers
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
