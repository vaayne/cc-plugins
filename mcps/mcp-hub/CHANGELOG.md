# Changelog

All notable changes to Hub will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **BREAKING**: Project renamed from `mcp-hub-go` to `hub`
- Go module path changed from `mcp-hub-go` to `hub`
- Binary renamed from `mcp-hub-go` to `hub`
- **BREAKING**: Rename CLI flag `--server/-s` to `--url/-u` for remote commands

### Added

- `list`, `inspect`, and `invoke` support `--config/-c` for stdio/http/sse servers

- Version information embedded at build time via ldflags
- `hub version` subcommand showing version, commit, and build date
- `--version` flag for quick version check
- Goreleaser configuration for automated multi-platform releases
  - Targets: linux/darwin/windows × amd64/arm64
  - SHA256 checksums for all artifacts
- GitHub Actions workflow for release automation (`hub/v*` tags)
- Install script (`scripts/install.sh`) with:
  - Automatic OS/architecture detection
  - SHA256 checksum verification
  - XDG Base Directory support (`~/.local/bin` fallback)
  - curl/wget fallback for downloads

## [1.2.0] - 2026-01-12

### Added

- CLI subcommands for interacting with remote MCP services (#14):
  - `hub serve` - Start the MCP hub server (was default behavior)
  - `hub list` - List tools from a remote MCP service
  - `hub inspect` - Inspect tool details and input schema
  - `hub invoke` - Invoke tools with JSON params (arg or stdin)
- Global CLI flags: `--server`, `--transport`, `--timeout`, `--header`, `--json`, `--verbose`
- Remote client with HTTP/SSE transport support
- Tool name mapping to camelCase JS method names
- Proxy environment variable support (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`)

### Changed

- Modernized codebase with Go 1.18+ features (`maps.Copy`, `slices.Contains`, `any` type)

## [1.1.0] - 2025-12-15

### Added

- Dynamic "Available Tools" section in `list` tool description (#12)
- JavaScript function stubs output from `list` tool for easier scripting (#11)
- Strict server name validation (must start with letter, alphanumeric + underscore only)

### Changed

- Tool namespace separator changed from `.` to `__` (double underscore) for valid JS identifiers
  - Example: `github__createIssue` instead of `github.createIssue`

### Security

- Bump `golang.org/x/net` from 0.27.0 to 0.38.0 (#10)

## [1.0.0] - 2025-12-10

Initial release of MCP Hub Go.

### Added

- **Multi-server aggregation**: Connect multiple MCP servers through a single hub
- **Transport support**:
  - Stdio for local MCP servers
  - HTTP with custom headers, timeout, and TLS configuration
  - SSE (Server-Sent Events) for real-time streaming
- **Tool namespacing**: Automatic `server__tool` naming to prevent conflicts
- **Built-in tools**:
  - `list` - Search and list available tools across all connected servers
  - `exec` - Execute JavaScript code with access to MCP tools
  - `refreshTools` - Refresh tool lists from connected servers
- **JavaScript runtime** (Goja):
  - Synchronous execution only (async/await blocked)
  - `mcp.callTool("server__tool", params)` for invoking remote tools
  - `console.log/info/warn/error/debug` for logging
  - 15-second execution timeout (configurable)
- **Security features**:
  - Input validation for commands, arguments, and environment variables
  - Shell metacharacter and interpreter blocking
  - Clean environment isolation (no parent env inheritance)
  - AST-based async detection to prevent bypass
  - Dangerous globals blocked: `eval`, `Function`, `Reflect`, `Proxy`, `WebAssembly`
  - Frozen prototype chains to prevent pollution
  - Script size limits (100KB)
  - Log sanitization to prevent injection
- **Structured logging**: JSON format with configurable levels and file output
- **Automatic reconnection**: Exponential backoff (max 30s) for failed server connections
- **Comprehensive configuration validation** on startup

### Security

- Thread-safe connection management with mutex protection
- SSRF protection for HTTP/SSE transports
- TLS verification enabled by default
- Owner-only file permissions (0600) for log files

[Unreleased]: https://github.com/vaayne/cc-plugins/compare/hub/v1.2.0...HEAD
[1.2.0]: https://github.com/vaayne/cc-plugins/compare/hub/v1.1.0...hub/v1.2.0
[1.1.0]: https://github.com/vaayne/cc-plugins/compare/hub/v1.0.0...hub/v1.1.0
[1.0.0]: https://github.com/vaayne/cc-plugins/releases/tag/hub/v1.0.0
