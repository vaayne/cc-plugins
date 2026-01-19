package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"hub/internal/js"
	"hub/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// ExecCmd is the exec subcommand that executes JavaScript code against MCP tools
var ExecCmd = &cobra.Command{
	Use:   "exec <code | ->",
	Short: "Execute JavaScript code to orchestrate multiple MCP tool calls",
	Long: `Execute JavaScript code that can call multiple MCP tools with logic.

Use this when you need to:
- Chain multiple tool calls
- Process data between calls
- Use conditionals or loops
- Aggregate results

For single tool calls, use 'invoke' instead.

The 'mcp.callTool(name, params)' function is available to call tools.
Tool names use the format 'serverID__toolName' (e.g., 'github__searchRepos').
For --url or --stdio mode, use 'default__toolName' as the server prefix.

Examples:
  # Chain tool calls (config mode)
  hub -c config.json exec 'const user = mcp.callTool("db__getUser", {id: 1}); mcp.callTool("email__send", {to: user.email})'

  # Read code from stdin
  cat script.js | hub -c config.json exec -

  # With remote server
  hub -u http://localhost:3000 exec 'const a = mcp.callTool("default__add", {x: 1, y: 2}); mcp.callTool("default__multiply", {x: a, y: 3})'

  # With stdio server
  hub --stdio exec 'mcp.callTool("default__echo", {message: "hello"})' -- npx @modelcontextprotocol/server-everything

  # JSON output
  hub -c config.json exec --json 'mcp.callTool("github__listRepos", {})'`,
	Args: func(cmd *cobra.Command, args []string) error {
		filteredArgs := filterArgsBeforeDash(args)
		if len(filteredArgs) != 1 {
			return fmt.Errorf("accepts 1 arg (code or -), received %d", len(filteredArgs))
		}
		return nil
	},
	RunE: runExec,
}

func init() {
	ExecCmd.Flags().StringP("config", "c", "", "path to configuration file")
}

// cliToolCaller adapts CLI clients to js.ToolCaller interface
type cliToolCaller struct {
	callFn func(ctx context.Context, name string, params json.RawMessage) (*mcp.CallToolResult, error)
	// defaultServer is used for --url and --stdio modes where there's only one server
	defaultServer string
	// mapper converts JS names back to original tool names
	mapper *ToolNameMapper
}

func (c *cliToolCaller) CallTool(ctx context.Context, serverID, toolName string, params map[string]any) (*mcp.CallToolResult, error) {
	// Build the full tool name
	var fullName string
	if c.defaultServer != "" {
		// Single server mode: ignore serverID, use toolName directly
		fullName = toolName
	} else {
		// Multi-server mode: use serverID__toolName format (already namespaced in ConfigClient)
		fullName = serverID + "__" + toolName
	}

	// Convert params to JSON
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = data
	}

	// Use mapper to convert back to original name if available
	if c.mapper != nil {
		fullName = c.mapper.ToOriginal(fullName)
	}

	return c.callFn(ctx, fullName, paramsJSON)
}

func runExec(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("--url, --config, or --stdio is required for exec command")
	}
	if modeCount > 1 {
		return fmt.Errorf("--url, --config, and --stdio are mutually exclusive")
	}

	// Get code from args or stdin
	filteredArgs := filterArgsBeforeDash(args)
	codeArg := filteredArgs[0]

	var code string
	if codeArg == "-" {
		// Check if stdin is a TTY
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("stdin is a terminal; pipe code or use argument instead")
		}
		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		input, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		code = string(input)
	} else {
		code = codeArg
	}

	if code == "" {
		return fmt.Errorf("code is required")
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	ctx := context.Background()

	var caller js.ToolCaller
	var cleanup func() error

	if configPath != "" {
		client, err := createConfigClient(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close

		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper, err := NewToolNameMapperWithCollisionCheck(tools)
		if err != nil {
			return err
		}

		caller = &cliToolCaller{
			callFn: client.CallTool,
			mapper: mapper,
		}
	} else if stdio {
		client, err := createStdioClientFromCmd(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close

		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper := NewToolNameMapper(tools)

		caller = &cliToolCaller{
			callFn:        client.CallTool,
			defaultServer: "default",
			mapper:        mapper,
		}
	} else {
		client, err := createRemoteClient(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close

		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper := NewToolNameMapper(tools)

		caller = &cliToolCaller{
			callFn:        client.CallTool,
			defaultServer: "default",
			mapper:        mapper,
		}
	}
	defer cleanup()

	// Execute using shared implementation
	logger := getLogger(cmd)
	execResult, err := tools.ExecuteCode(ctx, logger, caller, code)
	if err != nil {
		return err
	}

	// Output
	if jsonOutput {
		data, marshalErr := json.MarshalIndent(execResult, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
		}
		fmt.Println(string(data))
	} else {
		// Print logs first
		for _, log := range execResult.Logs {
			fmt.Printf("[%s] %s\n", log.Level, log.Message)
		}

		if execResult.Error != nil {
			return fmt.Errorf("execution failed: %s: %s", execResult.Error.Type, execResult.Error.Message)
		}

		// Print result
		if execResult.Result != nil {
			switch v := execResult.Result.(type) {
			case string:
				fmt.Println(v)
			default:
				data, marshalErr := json.MarshalIndent(execResult.Result, "", "  ")
				if marshalErr != nil {
					fmt.Printf("%v\n", execResult.Result)
				} else {
					fmt.Println(string(data))
				}
			}
		}
	}

	return nil
}
