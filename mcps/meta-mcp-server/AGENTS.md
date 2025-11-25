# Repository Guidelines

## Project Structure & Module Organization

- Core package lives in `src/`, with `index.ts` as the main entry point and orchestrator.
- `src/external-servers/` manages connections to external MCP servers.
- `src/generator/` handles TypeScript wrapper code generation from tool definitions.
- `src/search/` provides BM25 and regex search capabilities over generated code.
- `src/runtime/` handles TypeScript code execution in a sandboxed environment.
- `src/mcp-server.ts` registers the MCP tools (search_tools, eval_ts, refresh_tools).
- `src/types.ts` contains all TypeScript type definitions and Zod schemas.
- Generated tool wrappers go into `src/tools/<serverId>/` at runtime.

## Build, Test, and Development Commands

- `bun install` — install all dependencies.
- `bun run build` — compile TypeScript to `dist/` using tsc.
- `bun run dev -- -c config.json` — run in development mode with auto-reload.
- `bun start -- -c config.json` — run the built server with a config file.
- `bun run clean` — remove the `dist/` directory.

## Coding Style & Naming Conventions

- TypeScript strict mode enabled; avoid `any` type.
- Use camelCase for functions/variables, PascalCase for types/classes.
- MCP tool names use snake_case with `meta_` prefix (e.g., `meta_search_tools`).
- Input schemas use Zod for validation with `.describe()` for documentation.
- Use `as const` for literal types in MCP responses.

## Testing Guidelines

- Test the server with sample configurations before deployment.
- Use MCP Inspector for interactive testing: `npx @modelcontextprotocol/inspector`.
- Verify TypeScript compilation with `bun run build`.

## Commit & Pull Request Guidelines

- Use emoji-prefixed Conventional Commits (e.g., `✨ feat: add new search mode`, `🐛 fix: handle empty tool list`).
- Reference related issues in the body and summarize validation steps.
- Keep diffs focused; open follow-up PRs for unrelated refactors.

## Configuration

- Configuration file is JSON with `servers`, `toolsOutputDir`, and `searchIndexPath`.
- Support both HTTP and stdio transports for external MCP servers.
- Environment variables should be used for sensitive data (API keys, etc.).
