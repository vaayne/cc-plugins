package main

import (
	"fmt"
	"os"

	"hub/internal/cli"

	"github.com/spf13/cobra"
)

// Version information - injected at build time via ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:     "hub",
	Short:   "MCP Hub - Go implementation of Model Context Protocol hub",
	Version: version,
	Long: `MCP Hub aggregates multiple MCP servers and built-in tools,
providing a unified interface for tool execution and management.

Use 'hub serve' to start the hub server, or other commands to interact
with remote MCP services.`,
	PersistentPreRunE: validateGlobalFlags,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("hub %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

func init() {
	// Global flags (PersistentFlags) - shared across all subcommands
	rootCmd.PersistentFlags().StringP("server", "s", "", "remote MCP service URL")
	rootCmd.PersistentFlags().StringP("transport", "t", "", "transport type (http/sse for remote; stdio/http/sse for serve)")
	rootCmd.PersistentFlags().Int("timeout", 30, "connection timeout in seconds")
	rootCmd.PersistentFlags().StringArray("header", []string{}, "HTTP headers (repeatable, format: \"Key: Value\")")
	rootCmd.PersistentFlags().Bool("json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose logging")
	rootCmd.PersistentFlags().String("log-file", "", "log file path (empty disables file logging)")

	// Register subcommands
	rootCmd.AddCommand(cli.ServeCmd)
	rootCmd.AddCommand(cli.ListCmd)
	rootCmd.AddCommand(cli.InspectCmd)
	rootCmd.AddCommand(cli.InvokeCmd)
	rootCmd.AddCommand(versionCmd)

	// Set version template
	rootCmd.SetVersionTemplate("hub {{.Version}}\n")
}

func validateGlobalFlags(cmd *cobra.Command, args []string) error {
	// Get the server flag
	server, _ := cmd.Flags().GetString("server")
	transport, _ := cmd.Flags().GetString("transport")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Validate timeout is positive
	if timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got: %d", timeout)
	}

	// When -s/--server is provided (remote commands), transport must be http or sse
	if server != "" {
		// Default to http for remote commands when transport not specified
		if transport == "" {
			transport = "http"
		}
		if transport != "http" && transport != "sse" {
			return fmt.Errorf("invalid transport type for remote server: %s (must be http or sse)", transport)
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
