#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "click",
#     "httpx",
# ]
# ///
"""
Fetch and extract clean content from URLs using Jina Reader API.

Returns raw JSON with title, content, and metadata to stdout.
"""

import sys
from urllib.parse import quote, urlparse

import click
import httpx

API_BASE_URL = "https://r.jina.ai"
DEFAULT_TIMEOUT = 30


def validate_url(url: str) -> bool:
    """Validate URL format."""
    try:
        result = urlparse(url)
        return all([result.scheme in ("http", "https"), result.netloc])
    except Exception:
        return False


def fetch_content(url: str, timeout: int) -> str:
    """Fetch and extract content from URL using Jina Reader API."""
    # Encode the URL for the Jina API path
    encoded_url = quote(url, safe="")
    api_url = f"{API_BASE_URL}/{encoded_url}"

    headers = {
        "Accept": "application/json",
    }

    with httpx.Client(timeout=timeout) as client:
        response = client.get(api_url, headers=headers)
        response.raise_for_status()
        return response.text


@click.command()
@click.option(
    "--url",
    "-u",
    required=True,
    help="URL to fetch and extract content from",
)
@click.option(
    "--timeout",
    "-t",
    default=DEFAULT_TIMEOUT,
    type=int,
    help=f"Request timeout in seconds (default: {DEFAULT_TIMEOUT})",
)
def main(url: str, timeout: int) -> None:
    """Fetch and extract clean content from URLs using Jina Reader API."""
    # Validate URL
    if not validate_url(url):
        click.echo(f"Error: Invalid URL format: {url}", err=True)
        sys.exit(1)

    # Fetch content
    try:
        result = fetch_content(url=url, timeout=timeout)
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
