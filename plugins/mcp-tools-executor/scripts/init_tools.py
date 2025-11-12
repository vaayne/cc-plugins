#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "fastmcp",
#     "click",
#     "jinja2",
# ]
# ///

import asyncio
import json
import os
import shlex
import stat
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Sequence

from fastmcp import Client
from fastmcp.mcp_config import MCPConfig
from jinja2 import Template

BASE_DIR = Path(__file__).parent
DEFAULT_CONFIG_PATH = BASE_DIR / "mcp.json"
DEFAULT_OUTPUT_DIR = BASE_DIR.parent / "tools"
FORMATTER_CMD: str | None = "uvx ruff check --fix"
RUN_FORMATTER = True


@dataclass
class ServerInfo:
    name: str
    title: str | None
    version: str | None


@dataclass
class ToolSummary:
    server_name: str
    tool_name: str
    description: str
    filename: str
    path: Path


TOOL_SCRIPT_TEMPLATE = Template(
    '''#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "fastmcp",
#     "click"
# ]
# ///

import asyncio
import json

import click
from fastmcp import Client
from fastmcp.mcp_config import MCPConfig

SERVER_NAME = "{{ server_name }}"
SERVER_CONFIG = json.loads("""{{ server_config_json }}""")


def get_client() -> Client:
    return Client(MCPConfig(mcpServers={SERVER_NAME: SERVER_CONFIG}))


@click.command()
{% for decorator in decorators %}
{{ decorator }}
{% endfor %}
def invoke({{ param_signature }}):
    """{{ tool_description }}"""

    async def _execute():
        kwargs = {
            {% for param in param_infos %}"{{ param.original_name }}": {{ param.param_name }},
            {% endfor %}
        }
        kwargs = {k: v for k, v in kwargs.items() if v is not None}

        client = get_client()
        async with client:
            return await client.call_tool("{{ tool_name }}", kwargs)

    result = asyncio.run(_execute())
    click.echo(result)


if __name__ == "__main__":
    invoke()
''',
    trim_blocks=True,
    lstrip_blocks=True,
)


class ToolScriptGenerator:
    def __init__(self, template: Template = TOOL_SCRIPT_TEMPLATE):
        self.template = template

    def generate_script(
        self,
        *,
        server_name: str,
        server_config: Dict[str, Any],
        tool_name: str,
        tool_description: str,
        decorators: Sequence[str],
        param_infos: Sequence[Dict[str, Any]],
    ) -> str:
        param_signature = ", ".join(param["param_name"] for param in param_infos)
        server_config_json = json.dumps(server_config, indent=4)

        return self.template.render(
            server_name=server_name,
            server_config_json=server_config_json,
            tool_name=tool_name,
            tool_description=tool_description,
            decorators=decorators,
            param_infos=param_infos,
            param_signature=param_signature,
        )

    def save_script(self, *, output_path: Path, content: str) -> Path:
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(content)
        return output_path


class SchemaToClickMapper:
    def __init__(self):
        self.type_mapping = {
            "string": "click.STRING",
            "integer": "click.INT",
            "number": "click.FLOAT",
            "boolean": "click.BOOL",
        }

    def map_schema_to_click(
        self, schema: Dict[str, Any] | None
    ) -> tuple[list[str], list[Dict[str, Any]]]:
        if not schema:
            return [], []

        properties = schema.get("properties", {})
        required = schema.get("required", [])

        decorators: list[str] = []
        param_infos: list[Dict[str, Any]] = []

        def sort_key(prop_name: str):
            is_req = prop_name in required
            is_primary = prop_name in {"query", "search", "text", "content"}
            return (not is_req, not is_primary, prop_name)

        for prop_name in sorted(properties, key=sort_key):
            decorator, info = self.generate_click_decorator(
                prop_name, properties[prop_name], required
            )
            decorators.append(decorator)
            param_infos.append(info)

        return decorators, param_infos

    def generate_click_decorator(
        self, property_name: str, schema: Dict[str, Any], required: List[str]
    ) -> tuple[str, Dict[str, Any]]:
        param_name = self.generate_parameter_name(property_name)
        schema_type = schema.get("type", "string")
        default_value = schema.get("default")
        help_text = self.build_help_text(property_name, schema)
        is_required = property_name in required
        is_argument = self.should_be_argument(property_name, schema, is_required)

        if is_argument:
            decorator = self.build_argument_decorator(
                param_name, schema_type, is_required, default_value
            )
        elif schema_type == "boolean":
            decorator = self.build_boolean_decorator(
                param_name, default_value, help_text
            )
        else:
            decorator = self.build_option_decorator(
                param_name, schema, default_value, help_text
            )

        info = {
            "param_name": param_name,
            "original_name": property_name,
            "is_argument": is_argument,
            "is_required": is_required,
        }
        return decorator, info

    def build_help_text(self, property_name: str, schema: Dict[str, Any]) -> str:
        description = schema.get("description", f"{property_name} parameter")
        if "enum" in schema:
            enum_values = ", ".join(str(v) for v in schema["enum"])
            description += f" (choices: {enum_values})"
        if "minimum" in schema:
            description += f" (min: {schema['minimum']})"
        if "maximum" in schema:
            description += f" (max: {schema['maximum']})"
        return description

    def should_be_argument(
        self, property_name: str, schema: Dict[str, Any], is_required: bool
    ) -> bool:
        if property_name in {"query", "search", "text", "content"}:
            return True
        schema_type = schema.get("type", "string")
        if schema_type == "string" and is_required and "default" not in schema:
            return True
        return False

    def generate_parameter_name(self, property_name: str) -> str:
        snake = []
        for char in property_name:
            if char.isupper():
                snake.append("_")
                snake.append(char.lower())
            else:
                snake.append(char)
        normalized = "".join(snake).strip("_")
        return normalized.replace("-", "_")

    def get_click_type(self, schema: Dict[str, Any]) -> str:
        schema_type = schema.get("type", "string")
        if schema_type == "array":
            item_schema = schema.get("items", {})
            if "enum" in item_schema:
                enum_literal = self.python_literal(item_schema["enum"])
                return f"click.Choice({enum_literal})"
            return self.type_mapping.get(
                item_schema.get("type", "string"), "click.STRING"
            )
        if "enum" in schema:
            enum_literal = self.python_literal(schema["enum"])
            return f"click.Choice({enum_literal})"
        return self.type_mapping.get(schema_type, "click.STRING")

    def build_argument_decorator(
        self,
        param_name: str,
        schema_type: str,
        is_required: bool,
        default_value: Any,
    ) -> str:
        decorator = f"@click.argument('{param_name}'"
        if not is_required:
            decorator += ", required=False"
        if default_value is not None:
            decorator += f", default={self.python_literal(default_value)}"
        decorator += ")"
        return decorator

    def build_boolean_decorator(
        self, param_name: str, default_value: Any, help_text: str
    ) -> str:
        default_literal = self.python_literal(bool(default_value))
        return (
            f"@click.option('--{param_name}/--no-{param_name}', "
            f'default={default_literal}, help="""{help_text}""")'
        )

    def build_option_decorator(
        self,
        param_name: str,
        schema: Dict[str, Any],
        default_value: Any,
        help_text: str,
    ) -> str:
        option_name = f"--{param_name}"
        click_type = self.get_click_type(schema)
        parts = [f"@click.option('{option_name}'"]
        if schema.get("type") == "array":
            parts.append(", multiple=True")
        if click_type != "click.STRING":
            parts.append(f", type={click_type}")
        if default_value is not None:
            parts.append(f", default={self.python_literal(default_value)}")
        parts.append(f', help="""{help_text}""")')
        return "".join(parts)

    @staticmethod
    def python_literal(value: Any) -> str:
        return repr(value)


def load_config(cfg_path: Path) -> Dict[str, Any]:
    if not cfg_path.exists():
        raise SystemExit(f"Config file not found: {cfg_path}")
    with cfg_path.open("r") as handle:
        return json.load(handle)


def build_client(server_name: str, cfg: Dict[str, Any]) -> Client:
    return Client(MCPConfig(mcpServers={server_name: cfg}))


async def fetch_server_inventory(
    server_name: str, cfg: Dict[str, Any]
) -> tuple[ServerInfo, list[Any], Dict[str, Any]]:
    client = build_client(server_name, cfg)
    async with client:
        info = client.initialize_result.serverInfo
        server_info = ServerInfo(
            name=server_name,
            title=getattr(info, "title", None),
            version=getattr(info, "version", None),
        )
        tools = await client.list_tools()
    return server_info, tools, cfg


async def collect_server_data(
    servers: Dict[str, Dict[str, Any]],
) -> list[tuple[ServerInfo, list[Any], Dict[str, Any]]]:
    tasks = [fetch_server_inventory(name, cfg) for name, cfg in servers.items()]
    results: list[tuple[ServerInfo, list[Any], Dict[str, Any]]] = []
    for item in await asyncio.gather(*tasks, return_exceptions=True):
        if isinstance(item, Exception):
            print(f"Error while collecting tools: {item}")
            continue
        results.append(item)
    return results


def build_tool_filename(server_name: str, tool_name: str) -> str:
    return f"{server_name}__{tool_name}.py"


def ensure_executable(path: Path) -> None:
    if os.name == "nt":
        return
    mode = path.stat().st_mode
    path.chmod(mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)


def generate_tool_script(
    *,
    server_info: ServerInfo,
    server_config: Dict[str, Any],
    tool: Any,
    output_dir: Path,
    mapper: SchemaToClickMapper,
    generator: ToolScriptGenerator,
) -> Path:
    decorators, param_infos = mapper.map_schema_to_click(tool.inputSchema or {})
    script_content = generator.generate_script(
        server_name=server_info.name,
        server_config=server_config,
        tool_name=tool.name,
        tool_description=tool.description or f"Tool: {tool.name}",
        decorators=decorators,
        param_infos=param_infos,
    )

    filename = build_tool_filename(server_info.name, tool.name)
    output_path = output_dir / filename
    generator.save_script(output_path=output_path, content=script_content)
    ensure_executable(output_path)
    return output_path


def create_tools_index(tools: Sequence[ToolSummary], output_dir: Path) -> Path:
    lines = ["# Avaliable MCP Tools", ""]
    lines.append("This directory contains all MCP tools you can use in current env.\n")

    grouped: Dict[str, list[ToolSummary]] = {}
    for tool in tools:
        grouped.setdefault(tool.server_name, []).append(tool)

    for server_name in sorted(grouped):
        lines.append(f"## {server_name}")
        lines.append("")
        for tool in sorted(grouped[server_name], key=lambda t: t.tool_name):
            lines.append(
                f"- **{tool.filename}** — {tool.tool_name}: {tool.description}"
            )
        lines.append("")

    lines.append("## Usage")
    lines.append("")
    lines.append("Run any script directly, passing arguments as needed:")
    lines.append("")
    lines.append("```bash")
    lines.append("./tools/<server>__<tool>.py --help")
    lines.append("./tools/<server>__<tool>.py --option value")
    lines.append("```")

    index_path = output_dir / "tools.md"
    index_path.write_text("\n".join(lines))
    return index_path


def format_generated_scripts(paths: Sequence[Path], formatter_cmd: str | None) -> None:
    if not formatter_cmd or not paths:
        return
    cmd = shlex.split(formatter_cmd)
    if not cmd:
        return
    try:
        subprocess.run(cmd + [str(p) for p in paths], check=False, timeout=60)
    except (FileNotFoundError, subprocess.SubprocessError) as exc:
        print(f"Formatter failed: {exc}")


async def main() -> None:
    config = load_config(DEFAULT_CONFIG_PATH)
    tools_dir = DEFAULT_OUTPUT_DIR.resolve()

    servers = config.get("mcpServers", {})
    if not servers:
        raise SystemExit("No MCP servers defined in config.")

    print(f"Generating tool scripts into {tools_dir}...")
    server_data = await collect_server_data(servers)

    mapper = SchemaToClickMapper()
    generator = ToolScriptGenerator()
    summaries: list[ToolSummary] = []
    generated_paths: list[Path] = []

    for server_info, tools, server_cfg in server_data:
        for tool in tools:
            path = generate_tool_script(
                server_info=server_info,
                server_config=server_cfg,
                tool=tool,
                output_dir=tools_dir,
                mapper=mapper,
                generator=generator,
            )
            generated_paths.append(path)
            summaries.append(
                ToolSummary(
                    server_name=server_info.name,
                    tool_name=tool.name,
                    description=tool.description or f"Tool: {tool.name}",
                    filename=path.name,
                    path=path,
                )
            )

    if RUN_FORMATTER:
        format_generated_scripts(generated_paths, FORMATTER_CMD)

    index_path = create_tools_index(summaries, tools_dir)

    print(f"Generated {len(summaries)} tool scripts.")
    print(f"Index written to {index_path}")


if __name__ == "__main__":
    asyncio.run(main())
