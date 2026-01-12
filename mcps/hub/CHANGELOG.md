# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Renamed project from `mcp-hub-go` to `hub`
- Version injection via ldflags (`--version` flag and `version` subcommand)
- Goreleaser configuration for multi-platform releases
- GitHub Actions workflow for automated releases
- Install script with checksum verification

### Changed
- Go module name changed from `mcp-hub-go` to `hub`
- Binary renamed from `mcp-hub-go` to `hub`

## [1.1.0] - 2025-XX-XX

### Added
- HTTP transport support for remote MCP servers
- SSE (Server-Sent Events) transport support
- Custom header configuration with environment variable expansion
- TLS skip verify option for development environments

## [1.0.0] - 2025-XX-XX

### Added
- Initial release
- Stdio transport for local MCP servers
- Tool namespacing to prevent naming conflicts
- Built-in tools: `search`, `execute`, `refreshTools`
- JavaScript execution with Goja runtime (sync-only)
- Structured JSON logging
- Security features: input validation, sandbox, resource limits
