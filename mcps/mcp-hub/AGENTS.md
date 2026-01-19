# MCP Hub Agent Instructions

## Naming Convention

Following the GitHub/`gh` pattern:
- **Full name**: MCP Hub (for documentation and display)
- **CLI command**: `mh` (short, easy to type)
- **Go module**: `mcphub`

## Build/Test Commands

- `mise run build` — Build the binary to `bin/mh`
- `mise run test` — Run all tests with coverage
- `mise run run` — Build and run with example config
- `mise run format` — Format code (go fmt, go mod tidy)

## Architecture

- `main.go` — CLI entry point (cobra commands)
- `internal/cli/` — CLI subcommands (serve, list, inspect, invoke, exec, update)
- `internal/server/` — MCP hub server implementation
- `internal/client/` — Remote MCP client manager
- `internal/config/` — Configuration loading and validation
- `internal/tools/` — Built-in tools (list, invoke, inspect, exec)
- `internal/js/` — JavaScript runtime (Goja) for exec tool
- `internal/transport/` — Transport factory (stdio, http, sse)
- `internal/logging/` — Structured logging (zap)

## Code Style

- Go 1.23+, standard gofmt
- Internal packages in `internal/`
- Comprehensive test coverage
- Structured JSON logging
