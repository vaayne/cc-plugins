"""CLI entrypoint for MCP-FS Server."""

import logging
import signal
import sys
from typing import Any

import click
import uvicorn

from mcp_fs.mcp_server import backend_manager, load_config_from_file, mcp

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Signal handling for graceful shutdown
shutdown_requested = False


def signal_handler(signum: int, frame: Any) -> None:
    """Handle shutdown signals gracefully."""
    global shutdown_requested
    signal_name = signal.Signals(signum).name
    logger.info(f"\n{signal_name} received, shutting down gracefully...")
    shutdown_requested = True
    sys.exit(0)


def setup_signal_handlers() -> None:
    """Setup signal handlers for graceful shutdown."""
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)


@click.command()
@click.argument("url", required=False, default=None)
@click.option(
    "--config",
    "-c",
    type=click.Path(exists=True),
    help="JSON configuration file with backend definitions",
)
@click.option(
    "--transport",
    "-t",
    type=click.Choice(["stdio", "http"]),
    default="stdio",
    help="Transport mechanism (stdio or http)",
)
@click.option(
    "--port",
    "-p",
    type=int,
    default=8000,
    help="Port for HTTP transport",
)
@click.option(
    "--host",
    type=str,
    default="localhost",
    help="Host for HTTP transport",
)
def main(
    url: str | None,
    config: str | None,
    transport: str,
    port: int,
    host: str,
) -> int:
    """MCP-FS Server - Multi-backend filesystem server for MCP.

    \b
    Examples:
      mcp-fs memory://                           # Single backend mode
      mcp-fs config.json                         # Multi-backend from config
      mcp-fs --config config.json --transport http --port 8080
      mcp-fs fs:///tmp --transport stdio
    """
    # Setup signal handlers for graceful shutdown
    setup_signal_handlers()

    logger.info("Starting MCP-FS Server")

    try:
        # Handle configuration and backend setup
        if config:
            # Explicit config file specified
            try:
                load_config_from_file(config)
                logger.info(f"Loaded configuration from {config}")
            except Exception as e:
                logger.error(f"Failed to load configuration: {e}")
                raise click.ClickException(f"Failed to load configuration: {e}") from e
        elif url:
            # Check if the URL argument is actually a config file or a URL
            if url.endswith(".json"):
                # It's a config file
                try:
                    load_config_from_file(url)
                    logger.info(f"Loaded configuration from {url}")
                except Exception as e:
                    logger.error(f"Failed to load configuration: {e}")
                    raise click.ClickException(
                        f"Failed to load configuration: {e}"
                    ) from e
            else:
                # It's a URL for single backend mode
                logger.info(f"Single backend mode with URL: {url}")
                backend_manager.register_backend(
                    name="default",
                    url=url,
                    description="Legacy single backend",
                    set_as_default=True,
                )

        # Start the server
        if transport == "stdio":
            logger.info("Starting server with stdio transport")
            logger.info("Press Ctrl+C to stop the server")
            mcp.run(transport="stdio")
        elif transport == "http":
            logger.info(f"Starting server with HTTP transport on {host}:{port}")
            logger.info("Press Ctrl+C to stop the server")
            app = mcp.streamable_http_app()
            uvicorn.run(app, host=host, port=port)

    except KeyboardInterrupt:
        logger.info("\nServer stopped by user")
        return 0
    except Exception as e:
        logger.error(f"Server error: {e}")
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
