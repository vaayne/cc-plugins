---
name: grep-code-search
description: Search real-world code examples from millions of GitHub repositories using literal patterns. Use when users need to find implementation examples, API usage patterns, or production code snippets. Triggers on "search code for", "find code examples", "how do developers use", "show me real examples of", "grep code".
---

# Grep Code Search

MCP service at `https://mcp.grep.app` (http) with 1 tool.

## Requirements

- `hub` CLI must be installed. If not available, install with:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/vaayne/cc-plugins/main/mcps/hub/scripts/install.sh | sh
  ```

## Usage

List tools: `hub -s https://mcp.grep.app -t http list`
Get tool details: `hub -s https://mcp.grep.app -t http inspect searchGitHub`
Invoke tool: `hub -s https://mcp.grep.app -t http invoke searchGitHub '{"query": "useState("}'`

## Notes

- Run `inspect` before invoking unfamiliar tools to get full parameter schema
- Timeout: 30s default, use `--timeout <seconds>` to adjust
- **IMPORTANT**: This searches for literal code patterns (like grep), not keywords
- Use regex with `useRegexp=true` and `(?s)` prefix for multi-line patterns

## Tools

### searchGitHub

Find real-world code examples from over a million public GitHub repositories.

**Parameters:**

| Parameter         | Type     | Required | Default | Description                                             |
| ----------------- | -------- | -------- | ------- | ------------------------------------------------------- |
| `query`           | string   | Yes      | -       | Literal code pattern to search (e.g., `useState(`)      |
| `language`        | string[] | No       | -       | Filter by language (e.g., `["TypeScript", "TSX"]`)      |
| `repo`            | string   | No       | -       | Filter by repository (e.g., `facebook/react`)           |
| `path`            | string   | No       | -       | Filter by file path (e.g., `src/components/Button.tsx`) |
| `useRegexp`       | boolean  | No       | false   | Interpret query as regular expression                   |
| `matchCase`       | boolean  | No       | false   | Case sensitive search                                   |
| `matchWholeWords` | boolean  | No       | false   | Match whole words only                                  |

**Examples:**

```bash
# Find React useState patterns
hub -s https://mcp.grep.app -t http invoke searchGitHub '{"query": "useState(", "language": ["TypeScript", "TSX"]}'

# Find authentication patterns in Next.js
hub -s https://mcp.grep.app -t http invoke searchGitHub '{"query": "getServerSession", "language": ["TypeScript"]}'

# Find CORS handling in Flask (case-sensitive)
hub -s https://mcp.grep.app -t http invoke searchGitHub '{"query": "CORS(", "matchCase": true, "language": ["Python"]}'

# Find error boundary patterns with regex (multi-line)
hub -s https://mcp.grep.app -t http invoke searchGitHub '{"query": "(?s)useEffect\\(\\(\\) => {.*removeEventListener", "useRegexp": true}'

# Find examples in a specific repository
hub -s https://mcp.grep.app -t http invoke searchGitHub '{"query": "createContext", "repo": "vercel/ai"}'
```

**Best Practices:**

- ✅ Good queries: `useState(`, `import React from`, `async function`, `export default`
- ❌ Bad queries: `react tutorial`, `best practices`, `how to use`
- Use regex with `(?s)` prefix to match across multiple lines
- Filter by language for more relevant results
