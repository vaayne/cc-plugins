"""Core runtime helpers for the mcp-executor CLI and FastMCP server."""

from __future__ import annotations

import asyncio
import datetime
import json
import re
from pathlib import Path
from typing import Any

from fastmcp import Client, FastMCP
from jinja2 import Template
from mcp.types import Tool

__all__ = ["ToolExecutor", "executor", "mcp", "list_tools", "call_tool"]

mcp = FastMCP("MCPExecutor", version="v0.0.1")

LIST_TOOLS_TMPL = Template(
    """
{% for tool in tools %}
{{ tool.definition }}

{% endfor %}

Guide:
1. Call `list_tools` (this output) to discover tool names and signatures. Set
   `refresh=True` if upstream servers have changed.
2. Use the generated definitions above to understand each argument.
3. Call `call_tool` with keyword arguments that match the signature. You can
   use either the raw tool name or the generated identifier when invoking.
"""
)

_MISSING = object()


class ToolExecutor:
    """Stateful helper that owns the FastMCP client and tool metadata cache."""

    def __init__(self) -> None:
        self._client: Client | None = None
        self._tool_aliases: dict[str, str] = {}
        self._list_task: asyncio.Task[list[Tool]] | None = None

    def configure_from_path(self, config_path: str) -> None:
        """Load JSON config from disk and hydrate the MCP client."""

        config = self._load_config(config_path)
        self._client = Client(config)
        self._reset_state()

    async def render_tool_index(self, refresh: bool = False) -> str:
        """Return a templated summary of every discovered tool signature."""

        tools = await self._get_tools(refresh=refresh)
        self._reset_tool_aliases()

        rendered_tools = [
            {
                "name": tool.name,
                "definition": self._tool_definition(tool),
            }
            for tool in tools
        ]
        return LIST_TOOLS_TMPL.render(tools=rendered_tools)

    async def call_tool(
        self,
        name: str,
        arguments: dict[str, Any] | None = None,
        timeout: datetime.timedelta | float | int | None = None,
        raise_on_error: bool = True,
    ) -> Any:
        """Invoke a tool by name, handling sanitized aliases when present."""

        await self._ensure_tool_aliases()
        resolved_name = self._tool_aliases.get(name, name)

        client = self._require_client()
        async with client:
            return await client.call_tool(
                resolved_name,
                arguments=arguments,
                timeout=timeout,
                raise_on_error=raise_on_error,
            )

    async def list_tools(
        self,
        refresh: bool = False,
    ) -> list[Tool]:
        """Return raw tool metadata, optionally bypassing the cache."""

        return await self._get_tools(refresh=refresh)

    def _load_config(self, config_path: str) -> dict[str, Any]:
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

    def _require_client(self) -> Client:
        if self._client is None:
            raise RuntimeError(
                "Client not configured. Run with `--config /path/to/config.json`."
            )
        return self._client

    async def _get_tools(self, refresh: bool = False) -> list[Tool]:
        if refresh or self._list_task is None:
            self._list_task = asyncio.create_task(self._fetch_tools())
        try:
            return await self._list_task
        except Exception:
            self._list_task = None
            raise

    async def _fetch_tools(self) -> list[Tool]:
        client = self._require_client()
        async with client:
            return await client.list_tools()

    def _reset_state(self) -> None:
        self._tool_aliases.clear()
        self._list_task = None

    def _reset_tool_aliases(self) -> None:
        self._tool_aliases.clear()

    def _register_tool_alias(self, raw_name: str, identifier: str) -> str:
        if self._tool_aliases.get(identifier) == raw_name:
            return identifier

        if identifier not in self._tool_aliases:
            self._tool_aliases[identifier] = raw_name
            return identifier

        base = identifier
        suffix = 2
        while True:
            candidate = f"{base}_{suffix}"
            existing = self._tool_aliases.get(candidate)
            if existing is None or existing == raw_name:
                self._tool_aliases[candidate] = raw_name
                return candidate
            suffix += 1

    async def _ensure_tool_aliases(self) -> None:
        if self._tool_aliases:
            return

        tools = await self._get_tools()
        self._reset_tool_aliases()
        for tool in tools:
            self._tool_name_to_identifier(tool.name)

    def _tool_definition(self, tool: Tool) -> str:
        schema = tool.inputSchema if isinstance(tool.inputSchema, dict) else None
        func_name = self._tool_name_to_identifier(tool.name)
        signature, doc_entries = self._schema_parameters(schema)
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

    def _schema_parameters(self, schema: dict[str, Any] | None) -> tuple[str, list[str]]:
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
                annotation, nullable_from_type = self._annotation_from_schema(prop_schema)
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

    def _annotation_from_schema(self, prop_schema: dict[str, Any]) -> tuple[str, bool]:
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
                self._annotation_from_schema(items)
                if isinstance(items, dict)
                else ("Any", False)
            )
            annotation = f"list[{inner}]"
        else:
            annotation = (
                type_map.get(schema_type, "Any")
                if isinstance(schema_type, str)
                else "Any"
            )
            if schema_type is None and prop_schema.get("enum"):
                annotation = "str"

        return annotation, nullable

    def _tool_name_to_identifier(self, name: str) -> str:
        identifier = re.sub(r"[^0-9a-zA-Z_]", "_", name).strip("_")
        if not identifier:
            identifier = "tool"
        if identifier[0].isdigit():
            identifier = f"tool_{identifier}"
        return self._register_tool_alias(name, identifier)


executor = ToolExecutor()


@mcp.tool
async def list_tools(refresh: bool = False) -> str:
    """Discover tools across servers and cache metadata for later use."""

    return await executor.render_tool_index(refresh=refresh)


@mcp.tool
async def call_tool(
    name: str,
    arguments: dict[str, Any] | None = None,
    timeout: datetime.timedelta | float | int | None = None,
) -> Any:
    """Invoke an MCP tool, optionally supplying arguments and a timeout."""

    return await executor.call_tool(
        name,
        arguments=arguments,
        timeout=timeout,
    )
