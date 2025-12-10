package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mcp-hub-go/internal/config"
	"mcp-hub-go/internal/logging"
	"mcp-hub-go/internal/server"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	configPath string
	verbose    bool
	logFile    string
	transport  string
	port       int
	host       string
)

var rootCmd = &cobra.Command{
	Use:   "mcp-hub-go",
	Short: "MCP Hub - Go implementation of Model Context Protocol hub server",
	Long: `MCP Hub aggregates multiple MCP servers and built-in tools,
providing a unified interface for tool execution and management.

Transport Types:
  stdio  - Standard input/output (default, for CLI integration)
  http   - HTTP server with StreamableHTTP protocol
  sse    - HTTP server with Server-Sent Events protocol

Examples:
  # Run with stdio transport (default)
  mcp-hub-go -c config.json

  # Run with HTTP transport on port 8080
  mcp-hub-go -c config.json -t http -p 8080

  # Run with SSE transport on custom host and port
  mcp-hub-go -c config.json -t sse --host 0.0.0.0 -p 3000`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to configuration file (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	rootCmd.Flags().StringVar(&logFile, "log-file", "./mcp-hub.log", "path to log file (set to empty string to disable file logging)")
	rootCmd.Flags().StringVarP(&transport, "transport", "t", "stdio", "transport type: stdio, http, or sse")
	rootCmd.Flags().IntVarP(&port, "port", "p", 3000, "port for HTTP/SSE transport")
	rootCmd.Flags().StringVar(&host, "host", "localhost", "host for HTTP/SSE transport")
	rootCmd.MarkFlagRequired("config")
}

func run(cmd *cobra.Command, args []string) error {
	// Validate transport type
	if transport != "stdio" && transport != "http" && transport != "sse" {
		return fmt.Errorf("invalid transport type: %s (must be stdio, http, or sse)", transport)
	}

	// Determine log level based on verbose flag
	logLevel := zapcore.InfoLevel
	if verbose {
		logLevel = zapcore.DebugLevel
	}

	// Initialize logging with new config
	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	result, err := logging.InitLogger(logConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		if err := logging.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	logger := logging.Logger

	// Log initialization status
	if result.FileLoggingEnabled {
		logger.Info("File logging enabled", zap.String("log_file", logFile))
	} else if result.FileLoggingError != nil {
		logger.Warn("File logging disabled due to error",
			zap.String("log_file", logFile),
			zap.Error(result.FileLoggingError),
		)
	}

	// Validate config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Error("Configuration file does not exist", zap.String("path", configPath))
		return fmt.Errorf("config file not found: %s", configPath)
	}

	logger.Info("Starting MCP Hub",
		zap.String("config", configPath),
		zap.String("transport", transport),
	)

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("Failed to load configuration", zap.Error(err))
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create server
	srv := server.NewServer(cfg, logger)

	// Create transport config
	transportCfg := server.TransportConfig{
		Type: transport,
		Host: host,
		Port: port,
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(ctx, transportCfg); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
		cancel()
	case err := <-errChan:
		logger.Error("Server error", zap.Error(err))
		return err
	}

	// Graceful shutdown
	if err := srv.Stop(); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
		return err
	}

	logger.Info("MCP Hub stopped gracefully")
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
