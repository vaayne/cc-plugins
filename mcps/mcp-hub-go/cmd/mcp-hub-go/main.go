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
)

var (
	configPath string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "mcp-hub-go",
	Short: "MCP Hub - Go implementation of Model Context Protocol hub server",
	Long: `MCP Hub aggregates multiple MCP servers and built-in tools,
providing a unified interface for tool execution and management.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to configuration file (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	rootCmd.MarkFlagRequired("config")
}

func run(cmd *cobra.Command, args []string) error {
	// Initialize logging
	if err := logging.InitLogger(verbose); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logging.Sync()

	logger := logging.Logger

	// Validate config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Error("Configuration file does not exist", zap.String("path", configPath))
		return fmt.Errorf("config file not found: %s", configPath)
	}

	logger.Info("Starting MCP Hub", zap.String("config", configPath))

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("Failed to load configuration", zap.Error(err))
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create server
	srv := server.NewServer(cfg, logger)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(ctx); err != nil {
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
