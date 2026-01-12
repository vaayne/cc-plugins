---
name: mcp-skill-gen
description: Generate standalone skills from MCP servers. Use when users want to create a reusable skill for an MCP service. Triggers on "create skill for MCP", "generate MCP skill", "make skill from MCP server".
---

# MCP Skill Generator

Generate reusable skills from any MCP server using `hub` CLI.

## Prerequisites

- `hub` CLI must be installed. If not available, install with:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/vaayne/cc-plugins/main/mcps/hub/scripts/install.sh | sh
  ```

## Workflow

### 1. Gather Input

| Parameter | Required | Default             | Example                        |
| --------- | -------- | ------------------- | ------------------------------ |
| URL       | Yes      | -                   | `https://mcp.exa.ai`           |
| Transport | No       | `http`              | `http` or `sse`                |
| Name      | No       | from URL            | `exa-search`                   |
| Output    | No       | `./<name>/SKILL.md` | `./skills/exa-search/SKILL.md` |

### 2. Discover Tools

```bash
hub -s <url> -t <transport> list
```

### 3. Generate Skill

Read `references/skill-template.md`, fill placeholders: `{skill-name}`, `{description}`, `{Title}`, `{url}`, `{transport}`, `{tool-count}`, `{tools-list}`

### 4. Write Output

Save to output path, create directories if needed.

## Naming Guidelines

**Name**: Focus on capability, not source. Pattern: `{source}-{capability}` (kebab-case, 2-3 words)

| URL                               | Tools                           | Good          | Bad           |
| --------------------------------- | ------------------------------- | ------------- | ------------- |
| `https://mcp.exa.ai`              | webSearchExa, getCodeContextExa | `exa-search`  | `exa-mcp`     |
| `https://api.example.com/weather` | getWeather, getForecast         | `weather-api` | `weather-mcp` |

**Description**: `{Action + capability}. Use when {conditions}. Triggers on "{phrase1}", "{phrase2}".`

- Start with action verb (Search, Fetch, Get, Create, Analyze)
- Include 3-5 trigger phrases, mention service name, keep under 200 chars

## Error Handling

| Error              | Action                                          |
| ------------------ | ----------------------------------------------- |
| Connection timeout | Verify URL, check network, increase `--timeout` |
| No tools returned  | Server may require auth or have no tools        |
| Transport mismatch | Try `http` first, fall back to `sse`            |
