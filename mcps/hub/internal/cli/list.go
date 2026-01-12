package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// ListCmd is the list subcommand that lists tools from a remote MCP service
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tools from a remote MCP service",
	Long: `List all available tools from a remote MCP service.

Requires --server (-s) flag to specify the remote MCP service URL.

Examples:
  # List tools from a remote server
  hub -s http://localhost:3000 list

  # List tools with JSON output
  hub -s http://localhost:3000 list --json

  # List tools using SSE transport
  hub -s http://localhost:3000 -t sse list`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	// Check if --server is provided
	server, _ := cmd.Flags().GetString("server")
	if server == "" {
		return fmt.Errorf("--server is required for list command")
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx := context.Background()

	// Create remote client
	client, err := createRemoteClient(ctx, cmd)
	if err != nil {
		return err
	}
	defer client.Close()

	// List tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		return err
	}

	// Create name mapper for consistent JS method names
	mapper := NewToolNameMapper(tools)

	// Sort tools by JS name for consistent output
	sort.Slice(tools, func(i, j int) bool {
		return mapper.ToJSName(tools[i].Name) < mapper.ToJSName(tools[j].Name)
	})

	// Output
	if jsonOutput {
		// JSON output: array of tool objects with name and description
		type toolInfo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}

		toolList := make([]toolInfo, 0, len(tools))
		for _, tool := range tools {
			toolList = append(toolList, toolInfo{
				Name:        mapper.ToJSName(tool.Name),
				Description: tool.Description,
			})
		}

		output, err := json.MarshalIndent(toolList, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Text output: similar to renderAvailableToolsLines style
		if len(tools) == 0 {
			fmt.Println("No tools available")
			return nil
		}

		for _, tool := range tools {
			jsName := mapper.ToJSName(tool.Name)
			desc := tool.Description
			if strings.TrimSpace(desc) == "" {
				desc = jsName
			}
			fmt.Printf("- %s: %s\n", jsName, truncateDescription(desc, 50))
		}
	}

	return nil
}

// truncateDescription truncates a description to a maximum number of words
func truncateDescription(s string, maxWords int) string {
	if maxWords <= 0 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:maxWords], " ") + "…"
}
