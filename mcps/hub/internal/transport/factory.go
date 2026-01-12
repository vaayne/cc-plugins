package transport

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	"hub/internal/config"
)

// Factory creates appropriate transport based on server configuration
type Factory interface {
	CreateTransport(cfg config.MCPServer) (mcp.Transport, error)
}

// DefaultFactory implements Factory with support for stdio, http, and sse
type DefaultFactory struct {
	logger     *zap.Logger
	httpClient *http.Client // Optional custom HTTP client
}

// NewDefaultFactory creates a new DefaultFactory
func NewDefaultFactory(logger *zap.Logger) *DefaultFactory {
	return &DefaultFactory{
		logger: logger,
	}
}

// CreateTransport creates the appropriate transport based on the server configuration
func (f *DefaultFactory) CreateTransport(cfg config.MCPServer) (mcp.Transport, error) {
	transport := strings.ToLower(cfg.GetTransport())

	switch transport {
	case "stdio":
		return f.createStdioTransport(cfg)
	case "http":
		return f.createHTTPTransport(cfg)
	case "sse":
		return f.createSSETransport(cfg)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transport)
	}
}

// createStdioTransport creates a CommandTransport for stdio communication
func (f *DefaultFactory) createStdioTransport(cfg config.MCPServer) (mcp.Transport, error) {
	// Create command
	cmd := exec.Command(cfg.Command, cfg.Args...)

	// Set up environment
	cleanEnv := os.Environ()
	for k, v := range cfg.Env {
		// Sanitize environment variable name
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		cleanEnv = append(cleanEnv, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = cleanEnv

	// Create transport
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	f.logger.Debug("Created stdio transport",
		zap.String("command", cfg.Command),
		zap.Strings("args", cfg.Args))

	return transport, nil
}

// createHTTPTransport creates a StreamableClientTransport for HTTP communication
func (f *DefaultFactory) createHTTPTransport(cfg config.MCPServer) (mcp.Transport, error) {
	// Validate URL is provided
	if cfg.URL == "" {
		return nil, fmt.Errorf("url is required for http transport")
	}

	// Parse URL to ensure it's valid
	_, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Create HTTP client with custom configuration
	httpClient := f.getHTTPClient(cfg)

	// Create transport
	transport := &mcp.StreamableClientTransport{
		Endpoint:   cfg.URL,
		HTTPClient: httpClient,
		MaxRetries: 3, // Default retry count, can be made configurable
	}

	f.logger.Debug("Created HTTP transport",
		zap.String("url", cfg.URL),
		zap.Bool("tlsSkipVerify", cfg.TLSSkipVerify != nil && *cfg.TLSSkipVerify))

	return transport, nil
}

// createSSETransport creates an SSEClientTransport for Server-Sent Events communication
func (f *DefaultFactory) createSSETransport(cfg config.MCPServer) (mcp.Transport, error) {
	// Validate URL is provided
	if cfg.URL == "" {
		return nil, fmt.Errorf("url is required for sse transport")
	}

	// Parse URL to ensure it's valid
	_, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Create HTTP client with custom configuration
	httpClient := f.getHTTPClient(cfg)

	// Create transport
	transport := &mcp.SSEClientTransport{
		Endpoint:   cfg.URL,
		HTTPClient: httpClient,
	}

	f.logger.Debug("Created SSE transport",
		zap.String("url", cfg.URL),
		zap.Bool("tlsSkipVerify", cfg.TLSSkipVerify != nil && *cfg.TLSSkipVerify))

	return transport, nil
}

// getHTTPClient creates an HTTP client with the appropriate configuration
func (f *DefaultFactory) getHTTPClient(cfg config.MCPServer) *http.Client {
	// Use provided HTTP client if available
	if f.httpClient != nil {
		return f.httpClient
	}

	// Create new HTTP client with custom configuration
	timeout := 30 * time.Second // Default timeout
	if cfg.Timeout != nil && *cfg.Timeout > 0 {
		timeout = time.Duration(*cfg.Timeout) * time.Second
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Handle TLS verification skip (with warning)
	if cfg.TLSSkipVerify != nil && *cfg.TLSSkipVerify {
		f.logger.Warn("TLS verification disabled - this is insecure and should only be used for development",
			zap.String("url", cfg.URL))
		tlsConfig.InsecureSkipVerify = true
	}

	// Create transport with custom headers
	transport := &headerTransport{
		Base: &http.Transport{
			TLSClientConfig: tlsConfig,
			MaxIdleConns:    10,
			IdleConnTimeout: 90 * time.Second,
		},
		Headers: cfg.Headers,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// headerTransport is an http.RoundTripper that adds custom headers to requests
type headerTransport struct {
	Base    http.RoundTripper
	Headers map[string]string
}

// RoundTrip implements http.RoundTripper
func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(req.Context())

	// Add custom headers
	for k, v := range t.Headers {
		// Expand environment variables in header values
		expandedValue := os.ExpandEnv(v)
		req2.Header.Set(k, expandedValue)
	}

	// Use the base transport to make the actual request
	return t.Base.RoundTrip(req2)
}
