---
name: mcp-jetbrains-ide
description: Control JetBrains IDE programmatically for file operations, code analysis, refactoring, and terminal execution. Use when users want to interact with IntelliJ/WebStorm/PyCharm, run configurations, refactor code, or analyze files. Triggers on "open file in IDE", "run configuration", "refactor", "find files in project", "IDE terminal".
---

# JetBrains IDE Integration

MCP service at `http://127.0.0.1:64342/sse` (sse) with 21 tools.

## Requirements

- `hub` CLI must be installed. If not available, install with:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/vaayne/cc-plugins/main/mcps/hub/scripts/install.sh | sh
  ```
- JetBrains IDE must be running with MCP server enabled

## Usage

List tools: `hub -s http://127.0.0.1:64342/sse -t sse list`
Get tool details: `hub -s http://127.0.0.1:64342/sse -t sse inspect <tool-name>`
Invoke tool: `hub -s http://127.0.0.1:64342/sse -t sse invoke <tool-name> '{"param": "value"}'`

## Notes

- Run `inspect` before invoking unfamiliar tools to get full parameter schema
- Timeout: 30s default, use `--timeout <seconds>` to adjust
- The IDE must be running and accessible at the configured port

## Tools

### File Operations

- **createNewFile** - Creates a new file at the specified path within the project directory and optionally populates it with text. Creates parent directories automatically.
- **getFileTextByPath** - Retrieves text content of a file using its path relative to project root. Returns error for binary files, truncates large files.
- **getAllOpenFilePaths** - Returns active editor's and other open editors' file paths relative to the project root.
- **openFileInEditor** - Opens the specified file in the JetBrains IDE editor.
- **replaceTextInFile** - Replaces text in a file with flexible find/replace options. Efficient for targeted changes.
- **reformatFile** - Reformats a specified file applying code formatting rules.

### Search & Discovery

- **findFilesByGlob** - Searches for files matching a glob pattern (e.g. `**/*.txt`). Recursive search in all subdirectories.
- **findFilesByNameKeyword** - Searches for files whose names contain the specified keyword (case-insensitive). Uses indexes, matches names only.
- **searchInFilesByRegex** - Searches with regex pattern within all project files using IntelliJ's search engine.
- **searchInFilesByText** - Searches for text substring within all project files. Faster than command-line tools.
- **listDirectoryTree** - Provides tree representation of a directory in pseudo-graphic format like `tree` utility.

### Code Analysis

- **getFileProblems** - Analyzes file for errors and warnings using IntelliJ's inspections. Returns severity, description, and location.
- **getSymbolInfo** - Retrieves information about symbol at specified position (like Quick Documentation feature).

### Refactoring

- **renameRefactoring** - Renames a symbol (variable, function, class, etc.) with context-aware updates to ALL references throughout the project.

### Project Information

- **getProjectDependencies** - Get list of all dependencies defined in the project.
- **getProjectModules** - Get list of all modules in the project with their types.
- **getRepositories** - Retrieves the list of VCS roots in the project.

### Execution

- **executeTerminalCommand** - Executes shell command in IDE's integrated terminal. Limits output to 2000 lines.
- **executeRunConfiguration** - Run a specific run configuration and wait for it to finish. Returns exit code, output, and success status.
- **getRunConfigurations** - Returns list of run configurations including command line, working directory, and environment variables.

### Other

- **permissionPrompt** - Handle permission prompts.
