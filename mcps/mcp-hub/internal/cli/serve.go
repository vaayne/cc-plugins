package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mcphub/internal/config"
	"mcphub/internal/logging"
	"mcphub/internal/server"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ServeCmd is the serve subcommand that starts the MCP hub server
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP hub server",
	Long: `Start the MCP hub server with the specified transport.

Transport Types:
  stdio  - Standard input/output (default, for CLI integration)
  http   - HTTP server with StreamableHTTP protocol
  sse    - HTTP server with Server-Sent Events protocol

Examples:
  # Run with stdio transport (default)
  mh serve -c config.json

  # Run with HTTP transport on port 8080
  mh serve -c config.json -t http -p 8080

  # Run with SSE transport on custom host and port
  mh serve -c config.json -t sse --host 0.0.0.0 -p 3000

  # Run with verbose logging
  mh serve -c config.json -v`,
	RunE: runServe,
}

func init() {
	// Note: "config" flag is defined as a persistent flag on root command
	// to support both "mh -c config.json" and "mh serve -c config.json"
	ServeCmd.Flags().IntP("port", "p", 3000, "port for HTTP/SSE transport")
	ServeCmd.Flags().String("host", "localhost", "host for HTTP/SSE transport")
}

// RunServeFromRoot allows running serve command when invoked via root command with -c flag
func RunServeFromRoot(cmd *cobra.Command, args []string) error {
	return runServeWithCmd(cmd, args)
}

func runServe(cmd *cobra.Command, args []string) error {
	return runServeWithCmd(cmd, args)
}

func runServeWithCmd(cmd *cobra.Command, args []string) error {
	// Get config flag - works for both persistent (from root) and local flags
	configPath, _ := cmd.Flags().GetString("config")

	// Get local flags (only on serve command)
	port, _ := cmd.Flags().GetInt("port")
	host, _ := cmd.Flags().GetString("host")
	// Use defaults if not set (when called from root command)
	if port == 0 {
		port = 3000
	}
	if host == "" {
		host = "localhost"
	}

	// Get persistent flags from parent (root command)
	transport, _ := cmd.Flags().GetString("transport")
	verbose, _ := cmd.Flags().GetBool("verbose")
	logFile, _ := cmd.Flags().GetString("log-file")

	// For serve command, default to stdio if transport wasn't explicitly set
	// (parent default is empty string for subcommand-specific defaults)
	if transport == "" {
		transport = "stdio"
	}

	// Validate transport type for serve command (stdio/http/sse)
	if transport != "stdio" && transport != "http" && transport != "sse" {
		return fmt.Errorf("invalid transport type for serve: %s (must be stdio, http, or sse)", transport)
	}

	// Validate port range
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %d", port)
	}

	// Determine log level based on verbose flag
	logLevel := zapcore.InfoLevel
	if verbose {
		logLevel = zapcore.DebugLevel
	}

	// Initialize logging
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
