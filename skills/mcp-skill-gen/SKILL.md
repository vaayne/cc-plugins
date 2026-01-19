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
| URL       | No       | -                   | `https://mcp.exa.ai`           |
| Config    | No       | -                   | `./mcp.json`                   |
| Transport | No       | `http`              | `http` or `sse`                |
| Name      | No       | from URL or config  | `exa-search`                   |
| Output    | No       | `./<name>/SKILL.md` | `./skills/exa-search/SKILL.md` |

Notes:

- `--url/-u` and `--config/-c` are mutually exclusive
- Config mode uses the tool names returned by `hub list`, with JS-style name mapping and collision checks

### 2. Discover Tools

URL mode:

```bash
hub -u <url> -t <transport> list
```

Stdio mode:

```bash
hub -t stdio list -- cmd args ...
```

Config mode:

```bash
hub -c <config> list
```

### 3. Generate Skill

Read `references/skill-template.md`, fill placeholders: `{skill-name}`, `{description}`, `{Title}`, `{url}`, `{transport}`, `{tool-count}`, `{tools-list}`, `{usage-block}`, `{notes-block}`.

Usage blocks:

URL mode:

```
List tools: `hub -u {url} -t {transport} list`
Get tool details: `hub -u {url} -t {transport} inspect <tool-name>`
Invoke tool: `hub -u {url} -t {transport} invoke <tool-name> '{"param": "value"}'`
```

Config mode (use local config file name):

```
List tools: `hub -c {config-file} list`
Get tool details: `hub -c {config-file} inspect <tool-name>`
Invoke tool: `hub -c {config-file} invoke <tool-name> '{"param": "value"}'`
```

Notes blocks:

URL mode:

```
- Run `inspect` before invoking unfamiliar tools to get full parameter schema
- Timeout: 30s default, use `--timeout <seconds>` to adjust
```

Config mode:

```
- Run `inspect` before invoking unfamiliar tools to get full parameter schema
- Timeout: 30s default, use `--timeout <seconds>` to adjust
```

### 4. Write Output

Save to output path, create directories if needed.

If input is a config file, copy it into the generated skill folder as `config.json` and set `{config-file}` in the usage block to `./config.json`, relative to the generated `SKILL.md` location.

## Naming Guidelines

**Name**: Focus on capability, not source. Pattern: `{source}-{capability}` (kebab-case, 2-3 words)

| URL                               | Tools                           | Good              | Bad       |
| --------------------------------- | ------------------------------- | ----------------- | --------- |
| `https://mcp.exa.ai`              | webSearchExa, getCodeContextExa | `mcp-exa-search`  | `exa`     |
| `https://api.example.com/weather` | getWeather, getForecast         | `mcp-weather-api` | `weather` |

**Description**: `{Action + capability}. Use when {conditions}. Triggers on "{phrase1}", "{phrase2}".`

- Start with action verb (Search, Fetch, Get, Create, Analyze)
- Include 3-5 trigger phrases, mention service name, keep under 200 chars

## Error Handling

| Error              | Action                                          |
| ------------------ | ----------------------------------------------- |
| Connection timeout | Verify URL, check network, increase `--timeout` |
| No tools returned  | Server may require auth or have no tools        |
| Transport mismatch | Try `http` first, fall back to `sse`            |
