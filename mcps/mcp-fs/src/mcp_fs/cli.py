"""CLI entrypoint for MCP-FS Server."""

import json
import logging
import signal
import sys
from pathlib import Path
from typing import Any

import click
from fastmcp.server.server import Transport

from mcp_fs.mcp_server import backend_manager, load_config_from_file, mcp

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def signal_handler(signum: int, frame: Any) -> None:
    """Handle shutdown signals gracefully."""
    signal_name = signal.Signals(signum).name
    logger.info(f"\n{signal_name} received, shutting down gracefully...")
    raise KeyboardInterrupt()


def setup_signal_handlers() -> None:
    """Setup signal handlers for graceful shutdown."""
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)


def _load_and_validate_config(config_path: str) -> None:
    """Load and validate configuration file.

    Args:
        config_path: Path to the JSON configuration file

    Raises:
        FileNotFoundError: If the config file doesn't exist
        ValueError: If the config file contains invalid JSON or wrong type
    """
    path = Path(config_path).expanduser()
    if not path.is_file():
        raise FileNotFoundError(f"Config file '{path}' does not exist")

    try:
        data = json.loads(path.read_text())
    except json.JSONDecodeError as exc:
        raise ValueError(f"Invalid JSON in config file '{path}': {exc.msg}") from exc

    if not isinstance(data, dict):
        raise ValueError(f"Config file '{path}' must contain a JSON object")

    # Load into backend manager
    load_config_from_file(str(path))


def _config_callback(
    ctx: click.Context, param: click.Parameter, value: str | None
) -> str | None:
    """Validate and load config file when --config option is provided."""
    if value is None:
        return None
    try:
        _load_and_validate_config(value)
    except FileNotFoundError as exc:
        raise click.BadParameter(str(exc), ctx=ctx, param=param) from exc
    except ValueError as exc:
        raise click.BadParameter(str(exc), ctx=ctx, param=param) from exc
    return value


@click.command()
@click.option(
    "--url",
    "-u",
    type=str,
    default="memory://",
    help="Backend URL for single-backend mode, default 'memory://' (e.g., memory://, fs:///tmp, s3://bucket)",
)
@click.option(
    "--config",
    "-c",
    type=click.Path(exists=True, dir_okay=False),
    callback=_config_callback,
    help="JSON configuration file with backend definitions",
)
@click.option(
    "--transport",
    "-t",
    type=click.Choice(Transport.__args__),
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
      mcp-fs                                     # Default memory:// backend
      mcp-fs -u fs:///tmp                        # Local filesystem backend
      mcp-fs -c config.json                      # Multi-backend from config
      mcp-fs -c config.json -t http -p 8080      # Multi-backend with HTTP
    """
    # Setup signal handlers for graceful shutdown
    setup_signal_handlers()

    logger.info("Starting MCP-FS Server")

    try:
        # Handle configuration and backend setup
        if config:
            # Config validation already done in callback
            logger.info(f"Loaded configuration from {config}")
        elif url:
            # Single backend mode with URL
            logger.info(f"Single backend mode with URL: {url}")
            backend_manager.register_backend(
                name="default",
                url=url,
                description="Default single backend",
                set_as_default=True,
            )
        else:
            raise click.ClickException("Either --url or --config option is required")

        # Build kwargs for mcp.run()
        kwargs: dict[str, Any] = {"transport": transport}
        if transport != "stdio":
            if host is not None:
                kwargs["host"] = host
            if port is not None:
                kwargs["port"] = port

        mcp.run(**kwargs)

    except KeyboardInterrupt:
        logger.info("\nServer stopped by user")
        return 0
    except click.ClickException:
        raise  # Re-raise Click exceptions as-is
    except Exception as e:
        logger.error(f"Server error: {e}", exc_info=True)
        raise click.ClickException(f"Server error: {e}") from e

    return 0


if __name__ == "__main__":
    sys.exit(main())
