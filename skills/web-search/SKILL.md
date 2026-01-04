---
name: web-search
description: Search the web using Exa AI for real-time information retrieval. Use when users ask to search online, find current information, look up recent events, or need data beyond knowledge cutoff. Triggers on "search the web", "find online", "look up", "what's the latest on", "search for".
---

# Web Search

## Overview

Search the web using Exa AI's MCP endpoint for real-time information retrieval. Returns raw JSON results with source URLs optimized for LLM consumption.

## When to Use

- User asks to search the web or find online information
- Need current/recent information beyond knowledge cutoff
- Looking up documentation, news, or real-time data
- Researching topics that require up-to-date sources

## Workflow

1. Identify the search query from user request
2. Determine appropriate search parameters:
   - `--type auto` (default) for balanced results
   - `--type fast` for quick lookups
   - `--type deep` for comprehensive research
3. Run the search script
4. Present results with source URLs to user

## Usage

```bash
# Basic search
uv run --script scripts/web_search.py --query "your search query"

# With options
uv run --script scripts/web_search.py \
  --query "search query" \
  --num-results 8 \
  --type auto \
  --livecrawl fallback \
  --context-max-chars 10000 \
  --timeout 25
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--query` | (required) | Search query string |
| `--num-results` | 8 | Number of results to return |
| `--type` | auto | Search type: `auto`, `fast`, `deep` |
| `--livecrawl` | fallback | Live crawl mode: `fallback`, `preferred` |
| `--context-max-chars` | 10000 | Max characters for context |
| `--timeout` | 25 | Request timeout in seconds |

## Output Contract

| Scenario | stdout | stderr | exit code |
|----------|--------|--------|-----------|
| Success | Raw JSON from Exa | (empty) | 0 |
| No results | `{"results": []}` | Warning message | 0 |
| Error | (empty) | Error message | 1 |

Success output contains:
- Page titles and URLs
- Content snippets optimized for LLM context
- Source attribution

## Prerequisites

- Uses Exa AI's free MCP endpoint (no API key required)
- Requires `uv` for running PEP 723 scripts

## Examples

### Quick lookup
```bash
uv run --script scripts/web_search.py \
  --query "Python 3.12 new features" \
  --type fast \
  --num-results 3
```

### Deep research
```bash
uv run --script scripts/web_search.py \
  --query "LLM agent architectures 2024" \
  --type deep \
  --num-results 10 \
  --livecrawl preferred
```
