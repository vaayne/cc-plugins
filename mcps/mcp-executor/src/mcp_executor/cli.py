from __future__ import annotations

import asyncio
import datetime
import json
from collections.abc import Callable
from typing import Any, TypeVar

import click
from fastmcp.server.server import Transport

from .runtime import executor, mcp

F = TypeVar("F", bound=Callable[..., Any])


def _config_option(command: F) -> F:
    """Attach the shared --config option and hydrate the executor."""

    def _callback(
        ctx: click.Context, param: click.Parameter, value: str | None
    ) -> str | None:
        if value is None:
            return None
        try:
            executor.configure_from_path(value)
        except (FileNotFoundError, ValueError) as exc:
            raise click.BadParameter(str(exc), ctx=ctx, param=param) from exc
        ctx.ensure_object(dict)
        ctx.obj["config_path"] = value
        return value

    return click.option(
        "-c",
        "--config",
        "config_path",
        type=click.Path(exists=True, dir_okay=False, path_type=str),
        required=True,
        help="Path to JSON file that defines the MCP client configuration.",
        callback=_callback,
        expose_value=False,
    )(command)


def _echo_cli_output(result: Any) -> None:
    """Pretty-print tool responses for CLI users."""

    try:
        rendered = json.dumps(result, indent=2, default=_json_default)
    except (TypeError, OverflowError):
        click.echo(result)
        return
    click.echo(rendered)


def _json_default(value: Any) -> str:
    if isinstance(value, (datetime.date, datetime.datetime)):
        return value.isoformat()
    return repr(value)


def _coerce_arg_value(raw_value: str) -> Any:
    """Best-effort parsing for --arg KEY=VALUE pairs."""

    candidate = raw_value.strip()
    if candidate == "":
        return ""
    try:
        return json.loads(candidate)
    except json.JSONDecodeError:
        return candidate


def _merge_cli_arguments(
    arguments_json: str | None,
    arg_pairs: tuple[str, ...],
) -> dict[str, Any] | None:
    merged: dict[str, Any] = {}

    if arguments_json:
        try:
            parsed = json.loads(arguments_json)
        except json.JSONDecodeError as exc:
            raise click.BadParameter(
                f"--arguments must contain valid JSON: {exc.msg}",
                param_hint="--arguments",
            ) from exc
        if not isinstance(parsed, dict):
            raise click.BadParameter(
                "--arguments must be a JSON object with string keys.",
                param_hint="--arguments",
            )
        merged.update(parsed)

    for pair in arg_pairs:
        if "=" not in pair:
            raise click.BadParameter(
                "--arg expects KEY=VALUE assignments.",
                param_hint="--arg",
            )
        key, value = pair.split("=", 1)
        key = key.strip()
        if not key:
            raise click.BadParameter(
                "--arg key cannot be empty.",
                param_hint="--arg",
            )
        merged[key] = _coerce_arg_value(value)

    return merged or None


@click.group()
def cli() -> None:
    """Entry point exposing the ``serve``, ``list`` and ``call`` subcommands."""


@cli.command()
@_config_option
@click.option(
    "--transport",
    type=click.Choice(Transport.__args__),
    default="http",
    help='Transport protocol to use ("stdio", "sse", "http" or "streamable-http").',
)
@click.option(
    "--log-level",
    type=str,
    default=None,
    metavar="LEVEL",
    help="Override the FastMCP server log level (e.g. info, debug).",
)
@click.option(
    "--host",
    type=str,
    default="localhost",
    help="Host address to bind to.",
)
@click.option(
    "--port",
    type=int,
    default=23456,
    help="Port to bind to.",
)
@click.option(
    "--path",
    type=str,
    default=None,
    help="Path for the endpoint (defaults to /mcp for http transport or /sse for sse).",
)
@click.option(
    "--json-response",
    default=None,
    help="Enable or disable JSON responses (defaults to FastMCP's config).",
)
@click.option(
    "--stateless-http",
    default=None,
    help="Toggle stateless HTTP mode when using HTTP transports.",
)
def serve(
    transport: Transport,
    log_level: str | None,
    host: str,
    port: int,
    path: str | None,
    json_response: bool | None,
    stateless_http: bool | None,
) -> None:
    """Launch the FastMCP server that proxies tools from the configured client."""

    kwargs: dict[str, Any] = {"transport": transport}
    if log_level is not None:
        kwargs["log_level"] = log_level
    if host is not None:
        kwargs["host"] = host
    if port is not None:
        kwargs["port"] = port
    if path is not None:
        kwargs["path"] = path
    if json_response is not None:
        kwargs["json_response"] = json_response
    if stateless_http is not None:
        kwargs["stateless_http"] = stateless_http

    mcp.run(**kwargs)


@cli.command("list")
@_config_option
def list_cmd() -> None:
    """Render tool signatures without starting the FastMCP server."""

    output = asyncio.run(executor.render_tool_index(refresh=False))
    click.echo(output)


@cli.command("call")
@_config_option
@click.argument("name", metavar="TOOL_NAME")
@click.option(
    "--arguments",
    "arguments_json",
    type=str,
    default=None,
    help="Raw JSON object containing tool arguments.",
)
@click.option(
    "--arg",
    "arg_pairs",
    multiple=True,
    help="Repeatable KEY=VALUE flags merged into the arguments JSON.",
)
@click.option(
    "--timeout",
    type=float,
    default=None,
    help="Optional timeout in seconds before cancelling the tool call.",
)
def call_cmd(
    name: str,
    arguments_json: str | None,
    arg_pairs: tuple[str, ...],
    timeout: float | None,
) -> None:
    """Call a tool directly from the command line."""

    arguments = _merge_cli_arguments(arguments_json, arg_pairs)
    result = asyncio.run(
        executor.call_tool(
            name,
            arguments=arguments,
            timeout=timeout,
        )
    )
    _echo_cli_output(result)


if __name__ == "__main__":
    cli()
