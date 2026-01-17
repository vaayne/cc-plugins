package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// InvokeCmd is the invoke subcommand that invokes a tool on an MCP service
var InvokeCmd = &cobra.Command{
	Use:   "invoke <tool-name> [params-json | -]",
	Short: "Invoke a tool on an MCP service",
	Long: `Invoke a tool on an MCP service with optional JSON parameters.

Provide --url (-u) for a remote MCP service, or --config (-c) to load local
stdio/http/sse servers from config.
Takes tool name as a required positional argument.
Parameters can be provided as:
  - A JSON string argument
  - "-" to read JSON from stdin
  - Omitted for tools with no required parameters

Examples:
  # Invoke a tool with no parameters
  hub -u http://localhost:3000 invoke my-tool

  # Invoke a tool with JSON parameters
  hub -u http://localhost:3000 invoke my-tool '{"key": "value"}'

  # Invoke a tool with parameters from stdin
  echo '{"key": "value"}' | hub -u http://localhost:3000 invoke my-tool -

  # Invoke a tool with JSON output
  hub -u http://localhost:3000 invoke my-tool '{"key": "value"}' --json

  # Invoke a tool from config (stdio/http/sse)
  hub -c config.json invoke github__search_repos '{"query": "mcp"}'`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runInvoke,
}

func init() {
	InvokeCmd.Flags().StringP("config", "c", "", "path to configuration file")
}

func runInvoke(cmd *cobra.Command, args []string) error {
	url, _ := cmd.Flags().GetString("url")
	configPath, _ := cmd.Flags().GetString("config")
	if url == "" && configPath == "" {
		return fmt.Errorf("--url or --config is required for invoke command")
	}
	if url != "" && configPath != "" {
		return fmt.Errorf("--url and --config are mutually exclusive")
	}

	toolName := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Parse parameters
	var params json.RawMessage
	if len(args) > 1 {
		paramsArg := args[1]
		if paramsArg == "-" {
			// Check if stdin is a TTY (would hang waiting for input)
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				return fmt.Errorf("stdin is a terminal; pipe JSON or use argument instead")
			}
			// Read from stdin
			reader := bufio.NewReader(os.Stdin)
			input, err := io.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			// Validate JSON
			var js json.RawMessage
			if err := json.Unmarshal(input, &js); err != nil {
				return fmt.Errorf("invalid JSON from stdin: %v", err)
			}
			params = js
		} else {
			// Validate and use JSON string argument
			var js json.RawMessage
			if err := json.Unmarshal([]byte(paramsArg), &js); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}
			params = js
		}
	}

	ctx := context.Background()

	var result *mcp.CallToolResult

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
		mapper, err := NewToolNameMapperWithCollisionCheck(tools)
		if err != nil {
			return err
		}
		originalName := mapper.ToOriginal(toolName)
		if err := ensureNamespacedToolName(originalName); err != nil {
			return err
		}

		result, err = client.CallTool(ctx, originalName, params)
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
		mapper := NewToolNameMapper(tools)
		originalName := mapper.ToOriginal(toolName)

		result, err = client.CallTool(ctx, originalName, params)
		if err != nil {
			return err
		}
	}

	// Output
	if jsonOutput {
		// JSON output: full CallToolResult
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Text output: pretty-print result content
		printCallToolResult(result)
	}

	return nil
}

// printCallToolResult pretty-prints a CallToolResult
func printCallToolResult(result *mcp.CallToolResult) {
	if result.IsError {
		fmt.Println("Error:")
	}

	for _, content := range result.Content {
		switch c := content.(type) {
		case *mcp.TextContent:
			fmt.Println(c.Text)
		case *mcp.ImageContent:
			fmt.Printf("[Image: %s, %d bytes]\n", c.MIMEType, len(c.Data))
		case *mcp.EmbeddedResource:
			printEmbeddedResource(c)
		default:
			// Fallback: try to marshal as JSON
			if data, err := json.MarshalIndent(content, "", "  "); err == nil {
				fmt.Println(string(data))
			} else {
				fmt.Printf("%v\n", content)
			}
		}
	}
}

// printEmbeddedResource prints an embedded resource
func printEmbeddedResource(r *mcp.EmbeddedResource) {
	if r.Resource != nil {
		uri := r.Resource.URI
		if r.Resource.Text != "" {
			fmt.Printf("[Resource: %s]\n", uri)
			fmt.Println(r.Resource.Text)
		} else if len(r.Resource.Blob) > 0 {
			fmt.Printf("[Resource: %s, blob %d bytes]\n", uri, len(r.Resource.Blob))
		} else {
			fmt.Printf("[Resource: %s]\n", uri)
		}
	} else {
		fmt.Println("[Resource: empty]")
	}
}
