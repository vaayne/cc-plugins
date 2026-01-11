package cli

import (
	"context"
	"strings"

	"mcp-hub-go/internal/logging"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// createRemoteClient creates a RemoteClient from command flags
func createRemoteClient(ctx context.Context, cmd *cobra.Command) (*RemoteClient, error) {
	server, _ := cmd.Flags().GetString("server")
	transport, _ := cmd.Flags().GetString("transport")
	timeout, _ := cmd.Flags().GetInt("timeout")
	headers, _ := cmd.Flags().GetStringArray("header")
	verbose, _ := cmd.Flags().GetBool("verbose")
	logFile, _ := cmd.Flags().GetString("log-file")

	// Default to http for remote commands
	if transport == "" {
		transport = "http"
	}

	// Initialize logging
	logLevel := zapcore.InfoLevel
	if verbose {
		logLevel = zapcore.DebugLevel
	}

	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	if _, err := logging.InitLogger(logConfig); err != nil {
		return nil, err
	}

	opts := RemoteClientOpts{
		ServerURL: server,
		Transport: transport,
		Headers:   parseHeaders(headers),
		Timeout:   timeout,
		Logger:    logging.Logger,
	}

	return NewRemoteClient(ctx, opts)
}

// parseHeaders parses headers from []string in format "Key: Value" into map[string]string.
// Malformed headers (without ":") are silently skipped.
// Header values can contain environment variables that are expanded (e.g., $TOKEN or ${TOKEN}).
func parseHeaders(headers []string) map[string]string {
	result := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
		// Malformed headers without ":" are silently skipped
	}
	return result
}

// getLogger returns a configured logger based on command flags
func getLogger(cmd *cobra.Command) *zap.Logger {
	verbose, _ := cmd.Flags().GetBool("verbose")
	logFile, _ := cmd.Flags().GetString("log-file")

	logLevel := zapcore.InfoLevel
	if verbose {
		logLevel = zapcore.DebugLevel
	}

	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	if _, err := logging.InitLogger(logConfig); err != nil {
		return zap.NewNop() // Safe fallback
	}

	return logging.Logger
}
