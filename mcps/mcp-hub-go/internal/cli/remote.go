package cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// RemoteClientOpts contains options for creating a RemoteClient
type RemoteClientOpts struct {
	ServerURL string
	Transport string // "http" or "sse"
	Headers   map[string]string
	Timeout   int // seconds
	Logger    *zap.Logger
}

// RemoteClient connects to remote MCP services
type RemoteClient struct {
	session *mcp.ClientSession
	logger  *zap.Logger
}

// NewRemoteClient creates a new RemoteClient and connects to the remote MCP service
func NewRemoteClient(ctx context.Context, opts RemoteClientOpts) (*RemoteClient, error) {
	// Validate transport type
	transport := strings.ToLower(opts.Transport)
	if transport != "http" && transport != "sse" {
		return nil, fmt.Errorf("transport must be http or sse for remote commands")
	}

	// Validate URL
	if opts.ServerURL == "" {
		return nil, fmt.Errorf("invalid URL: server URL is required")
	}

	parsedURL, err := url.Parse(opts.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %s", err.Error())
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL: scheme must be http or https")
	}

	// Set default timeout
	timeout := 30
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	// Use a no-op logger if none provided
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create HTTP client with custom configuration
	httpClient := createHTTPClient(opts.Headers, timeout)

	// Create MCP transport based on type
	var mcpTransport mcp.Transport
	switch transport {
	case "http":
		mcpTransport = &mcp.StreamableClientTransport{
			Endpoint:   opts.ServerURL,
			HTTPClient: httpClient,
			MaxRetries: 3,
		}
	case "sse":
		mcpTransport = &mcp.SSEClientTransport{
			Endpoint:   opts.ServerURL,
			HTTPClient: httpClient,
		}
	}

	logger.Debug("Created MCP transport",
		zap.String("type", transport),
		zap.String("url", opts.ServerURL))

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-hub-cli",
		Version: "v1.0.0",
	}, nil)

	// Connect with timeout
	connectCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	session, err := client.Connect(connectCtx, mcpTransport, nil)
	if err != nil {
		return nil, wrapConnectionError(err, opts.ServerURL, timeout)
	}

	// Verify server responded with valid initialization
	if session.InitializeResult() == nil {
		session.Close()
		return nil, fmt.Errorf("server at %s did not complete MCP handshake", opts.ServerURL)
	}

	logger.Debug("Connected to remote MCP server", zap.String("url", opts.ServerURL))

	return &RemoteClient{
		session: session,
		logger:  logger,
	}, nil
}

// ListTools returns all available tools from the remote MCP server
func (c *RemoteClient) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	result, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	return result.Tools, nil
}

// GetTool returns a specific tool by name
func (c *RemoteClient) GetTool(ctx context.Context, name string) (*mcp.Tool, error) {
	tools, err := c.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	for _, tool := range tools {
		if tool.Name == name {
			return tool, nil
		}
	}

	return nil, fmt.Errorf("tool '%s' not found", name)
}

// CallTool invokes a tool on the remote MCP server
func (c *RemoteClient) CallTool(ctx context.Context, name string, params json.RawMessage) (*mcp.CallToolResult, error) {
	// Parse the raw JSON into a map for the Arguments field
	var args map[string]any
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid tool arguments: %w", err)
		}
	}

	callParams := &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	result, err := c.session.CallTool(ctx, callParams)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool '%s': %w", name, err)
	}

	return result, nil
}

// Close closes the connection to the remote MCP server
func (c *RemoteClient) Close() error {
	if c.session != nil {
		err := c.session.Close()
		c.session = nil // Clear reference for GC
		return err
	}
	return nil
}

// createHTTPClient creates an HTTP client with custom headers and timeout
func createHTTPClient(headers map[string]string, timeout int) *http.Client {
	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Expand environment variables in headers once at construction time
	expandedHeaders := make(map[string]string, len(headers))
	for k, v := range headers {
		expandedHeaders[k] = os.ExpandEnv(v)
	}

	// Create transport with custom headers
	transport := &headerRoundTripper{
		Base: &http.Transport{
			TLSClientConfig: tlsConfig,
			MaxIdleConns:    10,
			IdleConnTimeout: 90 * time.Second,
		},
		Headers: expandedHeaders,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}
}

// headerRoundTripper is an http.RoundTripper that adds custom headers to requests
type headerRoundTripper struct {
	Base    http.RoundTripper
	Headers map[string]string
}

// RoundTrip implements http.RoundTripper
func (t *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(req.Context())

	// Add custom headers (already expanded at construction time)
	for k, v := range t.Headers {
		req2.Header.Set(k, v)
	}

	return t.Base.RoundTrip(req2)
}

// wrapConnectionError wraps connection errors with user-friendly messages
func wrapConnectionError(err error, serverURL string, timeout int) error {
	errStr := err.Error()

	// Check for context deadline exceeded (timeout)
	if strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "timeout") {
		return fmt.Errorf("connection timed out after %ds (use --timeout to increase)", timeout)
	}

	// Check for connection refused
	if strings.Contains(errStr, "connection refused") {
		// Extract host:port from URL
		parsedURL, parseErr := url.Parse(serverURL)
		if parseErr == nil {
			host := parsedURL.Host
			return fmt.Errorf("cannot reach server at %s—is it running?", host)
		}
		return fmt.Errorf("cannot reach server at %s—is it running?", serverURL)
	}

	// Check for network errors
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			return fmt.Errorf("connection timed out after %ds (use --timeout to increase)", timeout)
		}
	}

	// Check for dial errors (no route to host, etc.)
	if strings.Contains(errStr, "dial") {
		parsedURL, parseErr := url.Parse(serverURL)
		if parseErr == nil {
			host := parsedURL.Host
			return fmt.Errorf("cannot reach server at %s—is it running?", host)
		}
	}

	// Default: return the original error with context
	return fmt.Errorf("failed to connect to %s: %w", serverURL, err)
}
