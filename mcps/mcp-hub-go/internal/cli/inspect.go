package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// InspectCmd is the inspect subcommand that shows details of a specific tool
var InspectCmd = &cobra.Command{
	Use:   "inspect <tool-name>",
	Short: "Inspect a tool from a remote MCP service",
	Long: `Show detailed information about a specific tool from a remote MCP service.

Requires --server (-s) flag to specify the remote MCP service URL.
Takes tool name as a required positional argument.

Examples:
  # Inspect a tool
  hub -s http://localhost:3000 inspect my-tool

  # Inspect a tool with JSON output
  hub -s http://localhost:3000 inspect my-tool --json

  # Inspect a tool using SSE transport
  hub -s http://localhost:3000 -t sse inspect my-tool`,
	Args: cobra.ExactArgs(1),
	RunE: runInspect,
}

func runInspect(cmd *cobra.Command, args []string) error {
	// Check if --server is provided
	server, _ := cmd.Flags().GetString("server")
	if server == "" {
		return fmt.Errorf("--server is required for inspect command")
	}

	toolName := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx := context.Background()

	// Create remote client
	client, err := createRemoteClient(ctx, cmd)
	if err != nil {
		return err
	}
	defer client.Close()

	// Get tool
	tool, err := client.GetTool(ctx, toolName)
	if err != nil {
		return err // Error message from RemoteClient is already user-friendly
	}

	// Output
	if jsonOutput {
		// JSON output: full tool object
		output, err := json.MarshalIndent(tool, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Text output: pretty-print tool schema
		fmt.Printf("Name: %s\n", tool.Name)
		fmt.Printf("Description: %s\n", tool.Description)

		if tool.InputSchema != nil {
			fmt.Println("\nInput Schema:")
			schemaJSON, err := json.MarshalIndent(tool.InputSchema, "  ", "  ")
			if err != nil {
				fmt.Printf("  (error formatting schema: %v)\n", err)
			} else {
				fmt.Printf("  %s\n", string(schemaJSON))
			}
		} else {
			fmt.Println("\nInput Schema: (none)")
		}
	}

	return nil
}
