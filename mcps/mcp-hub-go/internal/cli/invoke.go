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

// InvokeCmd is the invoke subcommand that invokes a tool on a remote MCP service
var InvokeCmd = &cobra.Command{
	Use:   "invoke <tool-name> [params-json | -]",
	Short: "Invoke a tool on a remote MCP service",
	Long: `Invoke a tool on a remote MCP service with optional JSON parameters.

Requires --server (-s) flag to specify the remote MCP service URL.
Takes tool name as a required positional argument.
Parameters can be provided as:
  - A JSON string argument
  - "-" to read JSON from stdin
  - Omitted for tools with no required parameters

Examples:
  # Invoke a tool with no parameters
  hub -s http://localhost:3000 invoke my-tool

  # Invoke a tool with JSON parameters
  hub -s http://localhost:3000 invoke my-tool '{"key": "value"}'

  # Invoke a tool with parameters from stdin
  echo '{"key": "value"}' | hub -s http://localhost:3000 invoke my-tool -

  # Invoke a tool with JSON output
  hub -s http://localhost:3000 invoke my-tool '{"key": "value"}' --json`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runInvoke,
}

func runInvoke(cmd *cobra.Command, args []string) error {
	// Check if --server is provided
	server, _ := cmd.Flags().GetString("server")
	if server == "" {
		return fmt.Errorf("--server is required for invoke command")
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

	// Create remote client
	client, err := createRemoteClient(ctx, cmd)
	if err != nil {
		return err
	}
	defer client.Close()

	// Call tool
	result, err := client.CallTool(ctx, toolName, params)
	if err != nil {
		return err
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
