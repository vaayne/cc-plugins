"""Tools Executor

A minimal FastMCP server that proxies every tool exposed by configured MCP
backends. It lets operators inspect the generated Python-like signatures and
then call a tool directly, keeping the workflow simple: discover, inspect,
invoke.

CLI usage follows the same pattern:

    ./main.py serve -c path/to/mcp.json --transport http
    ./main.py list -c path/to/mcp.json
    ./main.py call -c path/to/mcp.json TOOL_NAME --arg query=hello \
        --arguments '{"limit": 5}'

``serve`` launches the FastMCP server with the configured transport, while
``list`` and ``call`` offer quick inspection and invocation helpers without
needing to spin up the server. Use ``-c/--config`` to point at the MCP JSON
definition for every subcommand.
"""

import asyncio
import datetime
import json
import re
from functools import lru_cache
from pathlib import Path
from typing import Any, Callable, Mapping, TypeVar

import click
from fastmcp import Client, FastMCP
from fastmcp.server.server import Transport
from jinja2 import Template
from mcp.types import Tool

mcp = FastMCP("ToolsExecutor", version="v0.0.1")

LIST_TOOLS_TMPL = Template("""
{% for tool in tools %}
{{ tool.definition }}

{% endfor %}

Guide:
1. Call `list_tools` (this output) to discover tool names and signatures. Set
   `refresh=True` if upstream servers have changed.
2. Use the generated definitions above to understand each argument.
3. Call `call_tool` with keyword arguments that match the signature. You can
   use either the raw tool name or the generated identifier when invoking.
""")

_CLIENT: Client | None = None
_TOOL_ALIASES: dict[str, str] = {}
F = TypeVar("F", bound=Callable[..., Any])


def configure_client_from_path(config_path: str) -> None:
    """Load JSON config from disk and hydrate the MCP client."""
    config = _load_config(config_path)
    _set_client(config)


def _set_client(config: dict[str, Any]) -> None:
    global _CLIENT
    _CLIENT = Client(config)


def _require_client() -> Client:
    if _CLIENT is None:
        raise RuntimeError(
            "Client not configured. Run with `--config /path/to/config.json`."
        )
    return _CLIENT


def _load_config(config_path: str) -> dict[str, Any]:
    path = Path(config_path).expanduser()
    if not path.is_file():
        raise FileNotFoundError(f"Config file '{path}' does not exist.")
    try:
        data = json.loads(path.read_text())
    except json.JSONDecodeError as exc:
        raise ValueError(f"Invalid JSON in config file '{path}': {exc.msg}") from exc
    if not isinstance(data, dict):
        raise ValueError(f"Config file '{path}' must contain a JSON object.")
    return data


def _reset_tool_aliases() -> None:
    """Drop cached sanitized tool names."""

    _TOOL_ALIASES.clear()


def _register_tool_alias(raw_name: str, identifier: str) -> str:
    """Ensure aliases stay unique while mapping back to the raw tool name."""

    if _TOOL_ALIASES.get(identifier) == raw_name:
        return identifier

    if identifier not in _TOOL_ALIASES:
        _TOOL_ALIASES[identifier] = raw_name
        return identifier

    base = identifier
    suffix = 2
    while True:
        candidate = f"{base}_{suffix}"
        existing = _TOOL_ALIASES.get(candidate)
        if existing is None or existing == raw_name:
            _TOOL_ALIASES[candidate] = raw_name
            return candidate
        suffix += 1


@lru_cache(maxsize=1)
def _list_tools_task_cache() -> asyncio.Task[list[Tool]]:
    """Cache the in-flight or completed task that fetches tool metadata."""

    loop = asyncio.get_running_loop()
    return loop.create_task(_fetch_tools())


def _reset_tools_cache() -> None:
    """Clear cached tool metadata so the next call refetches it."""

    _list_tools_task_cache.cache_clear()


async def _fetch_tools() -> list[Tool]:
    client = _require_client()
    async with client:
        return await client.list_tools()


_MISSING = object()


def _tool_definition(tool: Tool) -> str:
    schema = tool.inputSchema if isinstance(tool.inputSchema, dict) else None
    func_name = _tool_name_to_identifier(tool.name)
    signature, doc_entries = _schema_parameters(schema)
    def_line = f"def {func_name}{signature}:"

    description = getattr(tool, "description", "").strip() or "No description provided."
    doc_lines = [description]
    if doc_entries:
        doc_lines.append("")
        doc_lines.append("Args:")
        doc_lines.extend(f"    {entry}" for entry in doc_entries)

    lines = [def_line, '    """']
    for line in doc_lines:
        lines.append(f"    {line}")
    lines.append('    """')
    lines.append("    pass")
    return "\n".join(lines)


def _schema_parameters(schema: dict[str, Any] | None) -> tuple[str, list[str]]:
    if not isinstance(schema, dict):
        return "()", []

    properties = schema.get("properties")
    if not isinstance(properties, dict) or not properties:
        return "()", []

    required_field = schema.get("required")
    required = set(required_field) if isinstance(required_field, list) else set()

    params: list[str] = []
    doc_entries: list[str] = []

    for param_name, prop_schema in properties.items():
        if not isinstance(prop_schema, dict):
            annotation = "Any"
            nullable_from_type = False
            description = "No description provided."
        else:
            annotation, nullable_from_type = _annotation_from_schema(prop_schema)
            description = (
                prop_schema.get("description", "No description provided.").strip()
                or "No description provided."
            )

        is_required = param_name in required
        default_value = (
            prop_schema.get("default", _MISSING)
            if isinstance(prop_schema, dict)
            else _MISSING
        )
        optional = nullable_from_type or not is_required or default_value is None

        if optional and not annotation.endswith(" | None"):
            annotation = f"{annotation} | None"

        param_str = f"{param_name}: {annotation}"
        if default_value is not _MISSING:
            param_str += f" = {repr(default_value)}"
        elif not is_required:
            param_str += " = None"

        params.append(param_str)

        meta_parts: list[str] = []
        if default_value is not _MISSING:
            meta_parts.append(f"default={repr(default_value)}")
        meta_parts.append("required" if is_required else "optional")
        meta_text = f" [{', '.join(meta_parts)}]" if meta_parts else ""
        doc_entries.append(f"{param_name} ({annotation}): {description}{meta_text}")

    if params:
        joined = ", ".join(params)
        return f"(*, {joined})", doc_entries
    return "()", doc_entries


def _annotation_from_schema(prop_schema: dict[str, Any]) -> tuple[str, bool]:
    schema_type = prop_schema.get("type")
    nullable = False

    if isinstance(schema_type, list):
        nullable = "null" in schema_type
        schema_type = next((t for t in schema_type if t != "null"), None)

    type_map = {
        "string": "str",
        "integer": "int",
        "number": "float",
        "boolean": "bool",
        "array": "list",
        "object": "dict",
    }

    if schema_type == "array":
        items = prop_schema.get("items")
        inner, _ = (
            _annotation_from_schema(items)
            if isinstance(items, dict)
            else ("Any", False)
        )
        annotation = f"list[{inner}]"
    else:
        annotation = (
            type_map.get(schema_type, "Any") if isinstance(schema_type, str) else "Any"
        )
        if schema_type is None and prop_schema.get("enum"):
            annotation = "str"

    return annotation, nullable


def _tool_name_to_identifier(name: str) -> str:
    identifier = re.sub(r"[^0-9a-zA-Z_]", "_", name).strip("_")
    if not identifier:
        identifier = "tool"
    if identifier[0].isdigit():
        identifier = f"tool_{identifier}"
    return _register_tool_alias(name, identifier)


async def _list_tools_impl(refresh: bool = False) -> str:
    """Shared implementation for listing tools (server + CLI)."""

    if refresh:
        _reset_tools_cache()

    try:
        tools = await _list_tools_task_cache()
    except Exception:
        _reset_tools_cache()
        raise

    _reset_tool_aliases()

    rendered_tools = [
        {
            "name": tool.name,
            "definition": _tool_definition(tool),
        }
        for tool in tools
    ]
    tools_desc = LIST_TOOLS_TMPL.render(tools=rendered_tools)
    print(tools_desc)
    return tools_desc


@mcp.tool
async def list_tools(refresh: bool = False) -> str:
    """Discover tools across servers and cache metadata for later use."""

    return await _list_tools_impl(refresh=refresh)


async def _ensure_tool_aliases() -> None:
    """Build tool aliases if they haven't been built yet."""
    if not _TOOL_ALIASES:
        try:
            tools = await _list_tools_task_cache()
        except Exception:
            _reset_tools_cache()
            raise
        _reset_tool_aliases()
        for tool in tools:
            _tool_name_to_identifier(tool.name)


async def _call_tool_impl(
    name: str,
    arguments: dict[str, Any] | None = None,
    timeout: datetime.timedelta | float | int | None = None,
    raise_on_error: bool = True,
) -> Any:
    """Shared implementation for invoking tools from both MCP and CLI."""

    # Ensure tool aliases are built before resolving
    await _ensure_tool_aliases()
    resolved_name = _TOOL_ALIASES.get(name, name)

    client = _require_client()
    async with client:
        return await client.call_tool(
            resolved_name,
            arguments=arguments,
            timeout=timeout,
            raise_on_error=raise_on_error,
        )


@mcp.tool
async def call_tool(
    name: str,
    arguments: dict[str, Any] | None = None,
    timeout: datetime.timedelta | float | int | None = None,
) -> Any:
    """Invoke an MCP tool, optionally supplying arguments and a timeout."""

    return await _call_tool_impl(
        name,
        arguments=arguments,
        timeout=timeout,
    )


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
    arguments_json: str | None, arg_pairs: tuple[str, ...]
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


def _config_option(command: F) -> F:
    """Attach the shared --config option and eagerly hydrate the client."""

    def _callback(
        ctx: click.Context, param: click.Parameter, value: str | None
    ) -> str | None:
        if value is None:
            return None
        try:
            configure_client_from_path(value)
        except (FileNotFoundError, ValueError) as exc:
            raise click.BadParameter(
                str(exc),
                ctx=ctx,
                param=param,
            ) from exc
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


@click.group()
def main() -> None:
    """Entry point exposing the ``serve``, ``list`` and ``call`` subcommands."""


@main.command()
@_config_option
@click.option(
    "--transport",
    type=click.Choice(Transport.__args__),
    default="http",
    help="""Transport protocol to use ("stdio", "sse", "http" or "streamable-http"), default to http""",
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
    help="""Host address to bind to, default to localhost""",
)
@click.option(
    "--port",
    type=int,
    default=23456,
    help="""Port to bind to, default to 23456""",
)
@click.option(
    "--path",
    type=str,
    default=None,
    help="""Path for the endpoint (defaults to /mcp for http transport or /sse for sse transport)""",
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

    kwargs: Mapping[str, Any] = {"transport": transport}
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


@main.command("list")
@_config_option
def list_cmd() -> None:
    """Render tool signatures without starting the FastMCP server."""

    asyncio.run(_list_tools_impl(refresh=False))


@main.command("call")
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
        _call_tool_impl(
            name,
            arguments=arguments,
            timeout=timeout,
        )
    )
    _echo_cli_output(result)


if __name__ == "__main__":
    main()
