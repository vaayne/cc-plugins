# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2025-12-01

### Added

- New `enable` field in server configuration to allow disabling servers without removing them
- Console logging to indicate when servers are disabled (`⊘ Server 'id' is disabled`)
- Documentation for the `enable` field in README.md
- Examples of `enable` field usage in config.example.json

### Changed

- Server parsing logic now filters out disabled servers during initialization
- ExternalServersManager skips disabled servers in configure and connectAll methods
- Default value for `enable` field is `true` to maintain backward compatibility

## [1.0.1] - Previous Release

### Added

- Initial implementation of MCP Hub
- Tool aggregation from multiple MCP servers
- TypeScript wrapper generation for discovered tools
- BM25 and Regex search capabilities
- Code execution runtime for combining tools
- Support for HTTP, SSE, and stdio transports
