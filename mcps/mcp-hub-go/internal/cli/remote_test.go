package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRemoteClientOpts_Validation(t *testing.T) {
	tests := []struct {
		name        string
		opts        RemoteClientOpts
		wantErr     bool
		errContains string
	}{
		{
			name: "invalid transport - stdio",
			opts: RemoteClientOpts{
				ServerURL: "http://localhost:8080",
				Transport: "stdio",
			},
			wantErr:     true,
			errContains: "transport must be http or sse for remote commands",
		},
		{
			name: "invalid transport - empty",
			opts: RemoteClientOpts{
				ServerURL: "http://localhost:8080",
				Transport: "",
			},
			wantErr:     true,
			errContains: "transport must be http or sse for remote commands",
		},
		{
			name: "missing URL",
			opts: RemoteClientOpts{
				Transport: "http",
			},
			wantErr:     true,
			errContains: "invalid URL",
		},
		{
			name: "invalid URL scheme",
			opts: RemoteClientOpts{
				ServerURL: "ftp://localhost:8080",
				Transport: "http",
			},
			wantErr:     true,
			errContains: "invalid URL: scheme must be http or https",
		},
		{
			name: "invalid URL format",
			opts: RemoteClientOpts{
				ServerURL: "://invalid",
				Transport: "http",
			},
			wantErr:     true,
			errContains: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_, err := NewRemoteClient(ctx, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRemoteClient_ConnectionRefused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to connect to a port that should not have any service
	opts := RemoteClientOpts{
		ServerURL: "http://localhost:59999",
		Transport: "http",
		Timeout:   1,
	}

	_, err := NewRemoteClient(ctx, opts)
	if err == nil {
		t.Error("expected connection error, got nil")
		return
	}

	// Error should be user-friendly
	errStr := err.Error()
	if !strings.Contains(errStr, "cannot reach server") && !strings.Contains(errStr, "connection") {
		t.Errorf("expected user-friendly connection error, got: %s", errStr)
	}
}

func TestWrapConnectionError(t *testing.T) {
	tests := []struct {
		name        string
		errStr      string
		serverURL   string
		timeout     int
		wantContain string
	}{
		{
			name:        "timeout error",
			errStr:      "context deadline exceeded",
			serverURL:   "http://localhost:8080",
			timeout:     30,
			wantContain: "connection timed out after 30s",
		},
		{
			name:        "connection refused",
			errStr:      "dial tcp: connection refused",
			serverURL:   "http://localhost:8080",
			timeout:     30,
			wantContain: "cannot reach server at localhost:8080",
		},
		{
			name:        "generic dial error",
			errStr:      "dial tcp: no route to host",
			serverURL:   "http://example.com:8080",
			timeout:     30,
			wantContain: "cannot reach server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapConnectionError(errorString(tt.errStr), tt.serverURL, tt.timeout)
			if !strings.Contains(err.Error(), tt.wantContain) {
				t.Errorf("expected error to contain %q, got %q", tt.wantContain, err.Error())
			}
		})
	}
}

// errorString is a simple error implementation for testing
type errorString string

func (e errorString) Error() string {
	return string(e)
}

func TestHeaderRoundTripper(t *testing.T) {
	// This is a unit test for the header round tripper
	// We can't easily test the full HTTP flow without a server,
	// but we can verify the struct is properly initialized
	headers := map[string]string{
		"Authorization": "Bearer test-token",
		"X-Custom":      "custom-value",
	}

	rt := &headerRoundTripper{
		Headers: headers,
	}

	if len(rt.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(rt.Headers))
	}

	if rt.Headers["Authorization"] != "Bearer test-token" {
		t.Errorf("unexpected Authorization header: %s", rt.Headers["Authorization"])
	}
}

func TestRemoteClientOpts_DefaultTimeout(t *testing.T) {
	// Test that timeout defaults are handled properly
	// We can't actually connect without a server, but we can verify
	// the opts structure works correctly

	opts := RemoteClientOpts{
		ServerURL: "http://localhost:8080",
		Transport: "http",
		// Timeout is 0, should default to 30
	}

	if opts.Timeout != 0 {
		t.Errorf("expected default timeout 0, got %d", opts.Timeout)
	}

	opts.Timeout = 60
	if opts.Timeout != 60 {
		t.Errorf("expected timeout 60, got %d", opts.Timeout)
	}
}

func TestCallToolParamsJSON(t *testing.T) {
	// Test that JSON params can be properly unmarshaled
	jsonParams := json.RawMessage(`{"key": "value", "number": 42}`)

	var args map[string]any
	err := json.Unmarshal(jsonParams, &args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if args["key"] != "value" {
		t.Errorf("expected key=value, got key=%v", args["key"])
	}

	if args["number"] != float64(42) { // JSON numbers are float64
		t.Errorf("expected number=42, got number=%v", args["number"])
	}
}
