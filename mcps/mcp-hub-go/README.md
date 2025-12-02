# MCP Hub Go

A Go implementation of the Model Context Protocol (MCP) hub server that aggregates multiple MCP servers and built-in tools, providing a unified interface for tool execution and management.

## Overview

MCP Hub Go is a secure, production-ready hub that:

- **Aggregates multiple MCP servers** into a single unified interface
- **Provides built-in tools** for search, JavaScript execution, and tool management
- **Namespaces tools** to avoid conflicts (e.g., `server1.tool`, `server2.tool`)
- **Enforces security** through sync-only JavaScript execution, input validation, and sandboxed runtime
- **Handles reconnection** automatically with exponential backoff
- **Logs structured JSON** to stdout and optionally to file

## Features

### Transport Support

The hub now supports multiple transport types for connecting to remote MCP servers:

#### Stdio Transport

Traditional command-based transport for local MCP servers:

```json
{
  "mcpServers": {
    "filesystem": {
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"],
      "env": {
        "CUSTOM_VAR": "value"
      }
    }
  }
}
```

#### HTTP Transport

Streamable HTTP transport for remote MCP servers:

```json
{
  "mcpServers": {
    "api-server": {
      "transport": "http",
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}",
        "X-Custom-Header": "value"
      },
      "timeout": 30,
      "tlsSkipVerify": false
    }
  }
}
```

Features:

- Custom headers with environment variable expansion
- Configurable timeout (seconds)
- TLS verification control (use `tlsSkipVerify: true` only for development)
- Automatic retry with exponential backoff

#### SSE Transport

Server-Sent Events transport for real-time streaming:

```json
{
  "mcpServers": {
    "streaming-server": {
      "transport": "sse",
      "url": "http://localhost:8080/sse",
      "headers": {
        "X-Client-ID": "mcp-hub"
      },
      "timeout": 60
    }
  }
}
```

Features:

- Real-time server-to-client streaming
- Automatic reconnection on connection loss
- POST requests for client-to-server communication
- Same header and TLS configuration as HTTP transport

### Tool Namespacing

All tools from remote servers are automatically namespaced with the server ID to prevent naming conflicts:

- Server `github` with tool `search_repos` → `github.search_repos`
- Server `filesystem` with tool `read_file` → `filesystem.read_file`

Built-in tools (`search`, `execute`, `refreshTools`) are not namespaced and available directly.

### Built-in Tools

1. **`search`** - Search across all available tools from connected servers
   - Searches tool names and descriptions
   - Returns results with server attribution
   - Limited to 100 results for performance

2. **`execute`** - Execute JavaScript code using the Goja runtime
   - Sync-only enforcement (no async/await, Promises, setTimeout)
   - Access to `mcp.callTool()` for calling remote tools
   - Access to `mcp.log()` for logging
   - 15-second execution timeout (configurable)
   - Sandboxed with blocked dangerous globals

3. **`refreshTools`** - Refresh tool lists from connected servers
   - Can refresh all servers or specific ones
   - Useful after server configuration changes

### Security Features

- **Sync-only JavaScript execution**: Blocks async constructs (async/await, Promise, setTimeout)
- **Sandboxed runtime**: Dangerous globals blocked (eval, Function constructor, Reflect, Proxy)
- **Input validation**: Command paths, arguments, and environment variables validated
- **Shell protection**: Shell metacharacters and interpreters blocked
- **Environment isolation**: Clean environment with no inheritance of potentially malicious variables
- **Resource limits**: Script size (100KB), execution timeout (15s), memory limits
- **Secure logging**: Log sanitization to prevent log injection and information leakage

### Timeout Configuration

- **Proxied tool calls**: 60 seconds (hardcoded in MVP)
- **JavaScript execution**: 15 seconds default (configurable via runtime config)
- **Connection timeout**: 60 seconds for initial connection and tool discovery

### Logging

Structured JSON logging with configurable levels:

```bash
# Enable debug logging
mcp-hub-go -c config.json -v

# Specify custom log file
mcp-hub-go -c config.json --log-file=/var/log/mcp-hub.log

# Disable file logging (stdout only)
mcp-hub-go -c config.json --log-file=""
```

Log format:

```json
{
  "level": "info",
  "timestamp": "2025-12-01T12:00:00Z",
  "caller": "server/server.go:39",
  "message": "Starting MCP hub server",
  "config": "/path/to/config.json"
}
```

## Installation

### Option 1: Download Binary

Download the latest release from the releases page and place it in your PATH.

### Option 2: Build from Source

Requirements:

- Go 1.23.0 or later

```bash
# Clone the repository
git clone <repository-url>
cd mcp-hub-go

# Install dependencies
go mod download

# Build
go build -o mcp-hub-go cmd/mcp-hub-go/main.go

# Or install to $GOPATH/bin
go install ./cmd/mcp-hub-go
```

### Option 3: Go Get

```bash
go install github.com/your-org/mcp-hub-go/cmd/mcp-hub-go@latest
```

## Configuration

Create a JSON configuration file (`config.json`):

```json
{
  "version": "1.0",
  "mcpServers": {
    "filesystem": {
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
      "env": {
        "NODE_ENV": "production"
      },
      "enable": true,
      "required": false
    },
    "github": {
      "transport": "stdio",
      "command": "node",
      "args": ["dist/index.js"],
      "env": {
        "GITHUB_TOKEN": "ghp_xxx"
      },
      "enable": true,
      "required": true
    }
  },
  "builtinTools": {}
}
```

### Configuration Fields

#### Server Configuration

- **`transport`** (string, optional): Transport type. Must be `"stdio"`. Defaults to `"stdio"` if omitted.
- **`command`** (string, required): Path to the executable. Must not contain shell metacharacters or path traversal.
- **`args`** (array, optional): Command arguments. Limited to 100 args, each max 4KB.
- **`env`** (object, optional): Environment variables. Dangerous variables (PATH, LD_PRELOAD, etc.) are blocked.
- **`enable`** (boolean, optional): Enable/disable server. Defaults to `true`.
- **`required`** (boolean, optional): If true, hub startup fails if server connection fails. Defaults to `false`.

#### Built-in Tools

Currently, built-in tools are registered programmatically and do not need configuration. The `builtinTools` field is reserved for future custom tool definitions.

See [config.example.json](config.example.json) for a complete example.

## Usage

### Start the Hub

```bash
mcp-hub-go -c config.json
```

With verbose logging:

```bash
mcp-hub-go -c config.json -v
```

With custom log file:

```bash
mcp-hub-go -c config.json --log-file=/var/log/mcp-hub.log
```

### Calling Tools

Once running, the hub exposes all tools via the MCP protocol on stdio. Use an MCP client to call tools:

```javascript
// Built-in tool
await client.callTool("search", { query: "file" });

// Namespaced remote tool
await client.callTool("filesystem.read_file", { path: "/tmp/test.txt" });
```

### JavaScript Execution

The `execute` built-in tool allows running JavaScript code that can call remote tools:

```javascript
// Example: Search and then read a file
const code = `
  const searchResult = mcp.callTool("filesystem", "list_directory", { path: "/tmp" });
  mcp.log("info", "Search complete", { count: searchResult.length });
  return searchResult;
`;

await client.callTool("execute", { code });
```

See [docs/js-authoring.md](docs/js-authoring.md) for JavaScript authoring guide.

## Development

### Project Structure

```
mcp-hub-go/
├── cmd/
│   └── mcp-hub-go/          # Main entry point
│       └── main.go
├── internal/
│   ├── client/              # Remote MCP client manager
│   │   ├── manager.go
│   │   └── manager_test.go
│   ├── config/              # Configuration loading and validation
│   │   ├── config.go
│   │   └── config_test.go
│   ├── js/                  # JavaScript runtime (Goja)
│   │   ├── runtime.go
│   │   └── runtime_test.go
│   ├── logging/             # Structured logging
│   │   ├── logger.go
│   │   └── logger_test.go
│   ├── server/              # MCP server implementation
│   │   ├── server.go
│   │   └── server_test.go
│   └── tools/               # Built-in tools
│       ├── builtin.go
│       └── builtin_test.go
├── docs/                    # Documentation
│   └── js-authoring.md
├── config.example.json      # Example configuration
├── go.mod
└── README.md
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/js
```

### Running Locally

```bash
# Build
go build -o mcp-hub-go cmd/mcp-hub-go/main.go

# Run with example config
./mcp-hub-go -c config.example.json -v
```

## Dependencies

- [go-sdk](https://github.com/modelcontextprotocol/go-sdk) - Official MCP Go SDK
- [goja](https://github.com/dop251/goja) - JavaScript runtime in Go
- [zap](https://github.com/uber-go/zap) - Structured logging
- [cobra](https://github.com/spf13/cobra) - CLI framework

## Security Considerations

### Command Injection Prevention

- Command paths validated for shell metacharacters, path traversal, and null bytes
- Shell interpreters (bash, sh, zsh, etc.) explicitly blocked
- Arguments validated for dangerous characters
- Environment variables sanitized (dangerous vars like LD_PRELOAD blocked)
- Clean environment used (no parent environment inheritance)

### JavaScript Sandbox

- Sync-only execution enforced (no async/await, Promise, setTimeout)
- Dangerous globals blocked (eval, Function, Reflect, Proxy, WebAssembly)
- Prototypes frozen to prevent prototype pollution
- Script size limited (100KB)
- Execution timeout enforced (15s default)
- Resource limits to prevent DoS

### Input Validation

- Configuration validated on load with comprehensive checks
- Tool parameters validated by MCP SDK schema validation
- Log messages sanitized to prevent log injection
- Error messages sanitized to prevent information leakage

### Recommendations

1. **Run with minimal privileges**: Use a dedicated user with restricted permissions
2. **Validate server binaries**: Only configure trusted MCP servers
3. **Use required: false**: For non-critical servers to prevent startup failures
4. **Monitor logs**: Enable file logging and monitor for errors
5. **Keep dependencies updated**: Regularly update Go and dependencies
6. **Restrict network access**: If servers don't need network, use firewalls/namespaces

## Troubleshooting

### Server Connection Failures

```
Error: failed to connect to server: context deadline exceeded
```

**Solutions:**

- Check that the command path is correct and executable
- Verify command arguments are valid
- Check environment variables are set correctly
- Increase timeout (future feature)
- Check server logs for startup errors

### Tool Call Timeouts

```
Error: remote tool call failed: context deadline exceeded
```

**Solutions:**

- Tool took longer than 60 seconds
- Check server performance
- Reduce workload in tool call
- Future: configurable timeout

### JavaScript Execution Blocked

```
Error: async functions are not allowed - only synchronous code is supported
```

**Solutions:**

- Remove async/await keywords
- Remove Promise usage
- Use synchronous equivalents
- See [docs/js-authoring.md](docs/js-authoring.md)

### Invalid Configuration

```
Error: config validation failed: server "myserver": command is required for stdio transport
```

**Solutions:**

- Check configuration format matches schema
- Verify required fields are present
- Check for typos in field names
- See [config.example.json](config.example.json)

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

[Add your license here]

## Support

For issues and questions:

- GitHub Issues: [repository-url]/issues
- Documentation: [docs/](docs/)

## Roadmap

### Completed Features

- [x] HTTP/SSE transport support (v1.1.0)

### Future Features

- [ ] Configurable timeouts for tool calls and JS execution
- [ ] Custom built-in tool definitions via config
- [ ] Tool authorization policies
- [ ] Metrics and monitoring endpoints
- [ ] Hot config reload
- [ ] Tool call rate limiting
- [ ] WebAssembly runtime option
