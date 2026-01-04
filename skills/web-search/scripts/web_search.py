#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "click",
#     "httpx",
# ]
# ///
"""
Web search using Exa AI's MCP endpoint.

Returns raw JSON results with source URLs to stdout.
"""

import json
import sys
from typing import Literal

import click
import httpx

API_BASE_URL = "https://mcp.exa.ai"
API_ENDPOINT = "/mcp"
DEFAULT_NUM_RESULTS = 8
DEFAULT_TIMEOUT = 25
DEFAULT_CONTEXT_MAX_CHARS = 10000


def build_request(
    query: str,
    num_results: int,
    livecrawl: Literal["fallback", "preferred"],
    search_type: Literal["auto", "fast", "deep"],
    context_max_chars: int | None,
) -> dict:
    """Build JSON-RPC 2.0 request for Exa MCP endpoint."""
    arguments = {
        "query": query,
        "numResults": num_results,
        "livecrawl": livecrawl,
        "type": search_type,
    }
    if context_max_chars is not None:
        arguments["contextMaxCharacters"] = context_max_chars

    return {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "web_search_exa",
            "arguments": arguments,
        },
    }


def parse_sse_response(response_text: str) -> str | None:
    """
    Parse SSE response to extract JSON data.

    Exa returns SSE format with data: prefix.
    Returns the text content from the first valid data line.
    """
    last_parse_error: str | None = None

    for line in response_text.split("\n"):
        if line.startswith("data: "):
            try:
                data = json.loads(line[6:])  # Skip "data: " prefix
                # Safely access nested structure
                result = data.get("result", {})
                content = result.get("content", [])
                if content and isinstance(content, list) and len(content) > 0:
                    first_item = content[0]
                    if isinstance(first_item, dict) and "text" in first_item:
                        return first_item["text"]
            except json.JSONDecodeError as e:
                last_parse_error = f"JSON parse error: {e} in line: {line[:100]}"
                continue

    # Log last parse error to stderr if we failed to find valid data
    if last_parse_error:
        click.echo(f"Warning: {last_parse_error}", err=True)

    return None


def search(
    query: str,
    num_results: int,
    livecrawl: Literal["fallback", "preferred"],
    search_type: Literal["auto", "fast", "deep"],
    context_max_chars: int | None,
    timeout: int,
) -> str:
    """Execute web search and return results."""
    request_body = build_request(
        query=query,
        num_results=num_results,
        livecrawl=livecrawl,
        search_type=search_type,
        context_max_chars=context_max_chars,
    )

    headers = {
        "accept": "application/json, text/event-stream",
        "content-type": "application/json",
    }

    with httpx.Client(timeout=timeout) as client:
        response = client.post(
            f"{API_BASE_URL}{API_ENDPOINT}",
            headers=headers,
            json=request_body,
        )
        response.raise_for_status()

        result = parse_sse_response(response.text)
        if result is None:
            return json.dumps({"results": []})
        return result


@click.command()
@click.option(
    "--query",
    "-q",
    required=True,
    help="Search query string",
)
@click.option(
    "--num-results",
    "-n",
    default=DEFAULT_NUM_RESULTS,
    type=int,
    help=f"Number of results to return (default: {DEFAULT_NUM_RESULTS})",
)
@click.option(
    "--livecrawl",
    "-l",
    default="fallback",
    type=click.Choice(["fallback", "preferred"]),
    help="Live crawl mode: fallback (default) or preferred",
)
@click.option(
    "--type",
    "search_type",
    "-t",
    default="auto",
    type=click.Choice(["auto", "fast", "deep"]),
    help="Search type: auto (default), fast, or deep",
)
@click.option(
    "--context-max-chars",
    "-c",
    default=DEFAULT_CONTEXT_MAX_CHARS,
    type=int,
    help=f"Max characters for context (default: {DEFAULT_CONTEXT_MAX_CHARS})",
)
@click.option(
    "--timeout",
    default=DEFAULT_TIMEOUT,
    type=int,
    help=f"Request timeout in seconds (default: {DEFAULT_TIMEOUT})",
)
@click.option(
    "--dry-run",
    is_flag=True,
    help="Preview request without sending",
)
def main(
    query: str,
    num_results: int,
    livecrawl: Literal["fallback", "preferred"],
    search_type: Literal["auto", "fast", "deep"],
    context_max_chars: int,
    timeout: int,
    dry_run: bool,
) -> None:
    """Search the web using Exa AI's MCP endpoint."""
    # Validate query
    if not query.strip():
        click.echo("Error: Query cannot be empty", err=True)
        sys.exit(1)

    # Build request
    request_body = build_request(
        query=query,
        num_results=num_results,
        livecrawl=livecrawl,
        search_type=search_type,
        context_max_chars=context_max_chars,
    )

    # Dry run - just show the request
    if dry_run:
        click.echo("Dry run - request that would be sent:", err=True)
        click.echo(json.dumps(request_body, indent=2))
        return

    # Execute search
    try:
        result = search(
            query=query,
            num_results=num_results,
            livecrawl=livecrawl,
            search_type=search_type,
            context_max_chars=context_max_chars,
            timeout=timeout,
        )

        # Check for empty results
        try:
            parsed = json.loads(result)
            if isinstance(parsed, dict) and not parsed.get("results"):
                click.echo("Warning: No results found", err=True)
        except json.JSONDecodeError:
            pass  # Not JSON, just output as-is

        click.echo(result)

    except httpx.TimeoutException:
        click.echo(f"Error: Request timed out after {timeout} seconds", err=True)
        sys.exit(1)
    except httpx.HTTPStatusError as e:
        click.echo(
            f"Error: HTTP {e.response.status_code} - {e.response.text}", err=True
        )
        sys.exit(1)
    except httpx.RequestError as e:
        click.echo(f"Error: Network error - {e}", err=True)
        sys.exit(1)
    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
