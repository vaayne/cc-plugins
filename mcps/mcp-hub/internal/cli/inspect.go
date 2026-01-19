package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// InspectCmd is the inspect subcommand that shows details of a specific tool
var InspectCmd = &cobra.Command{
	Use:   "inspect <tool-name>",
	Short: "Inspect a tool from an MCP service",
	Long: `Show detailed information about a specific tool from an MCP service.

Provide --url (-u) for a remote MCP service, --config (-c) to load local
stdio/http/sse servers from config, or --stdio to spawn a subprocess.
Takes tool name as a required positional argument.

Examples:
  # Inspect a tool
  mh -u http://localhost:3000 inspect my-tool

  # Inspect a tool with JSON output
  mh -u http://localhost:3000 inspect my-tool --json

  # Inspect a tool using SSE transport
  mh -u http://localhost:3000 -t sse inspect my-tool

  # Inspect a tool from config (stdio/http/sse)
  mh -c config.json inspect github__search_repos

  # Inspect a tool from a stdio MCP server
  mh --stdio inspect echo -- npx @modelcontextprotocol/server-everything`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Filter out args after "--" (used for stdio command)
		filteredArgs := filterArgsBeforeDash(args)
		if len(filteredArgs) != 1 {
			return fmt.Errorf("accepts 1 arg(s), received %d", len(filteredArgs))
		}
		return nil
	},
	RunE: runInspect,
}

func init() {
	InspectCmd.Flags().StringP("config", "c", "", "path to configuration file")
}

func runInspect(cmd *cobra.Command, args []string) error {
	url, _ := cmd.Flags().GetString("url")
	configPath, _ := cmd.Flags().GetString("config")
	stdio, _ := cmd.Flags().GetBool("stdio")

	// Count how many modes are specified
	modeCount := 0
	if url != "" {
		modeCount++
	}
	if configPath != "" {
		modeCount++
	}
	if stdio {
		modeCount++
	}

	if modeCount == 0 {
		return fmt.Errorf("--url, --config, or --stdio is required for inspect command")
	}
	if modeCount > 1 {
		return fmt.Errorf("--url, --config, and --stdio are mutually exclusive")
	}

	// Filter args to get only those before "--"
	filteredArgs := filterArgsBeforeDash(args)
	toolName := filteredArgs[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx := context.Background()

	var tool *mcp.Tool
	var mapper *ToolNameMapper

	if configPath != "" {
		client, err := createConfigClient(ctx, cmd)
		if err != nil {
			return err
		}
		defer client.Close()

		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper, err = NewToolNameMapperWithCollisionCheck(tools)
		if err != nil {
			return err
		}
		originalName := mapper.ToOriginal(toolName)
		if err := ensureNamespacedToolName(originalName); err != nil {
			return err
		}

		tool, err = client.GetTool(ctx, originalName)
		if err != nil {
			return err
		}
	} else if stdio {
		client, err := createStdioClientFromCmd(ctx, cmd)
		if err != nil {
			return err
		}
		defer client.Close()

		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper = NewToolNameMapper(tools)
		originalName := mapper.ToOriginal(toolName)

		tool, err = client.GetTool(ctx, originalName)
		if err != nil {
			return err
		}
	} else {
		client, err := createRemoteClient(ctx, cmd)
		if err != nil {
			return err
		}
		defer client.Close()

		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper = NewToolNameMapper(tools)
		originalName := mapper.ToOriginal(toolName)

		tool, err = client.GetTool(ctx, originalName)
		if err != nil {
			return err // Error message from RemoteClient is already user-friendly
		}
	}

	// Get JS name for display
	jsName := mapper.ToJSName(tool.Name)

	// Output
	if jsonOutput {
		// JSON output: full tool object with JS name
		type toolOutput struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema any    `json:"inputSchema,omitempty"`
		}
		output, err := json.MarshalIndent(toolOutput{
			Name:        jsName,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Text output: pretty-print tool schema
		fmt.Printf("Name: %s\n", jsName)
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
