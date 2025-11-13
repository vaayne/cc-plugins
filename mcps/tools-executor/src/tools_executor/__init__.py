"""Runtime exports for the tools_executor package."""
from __future__ import annotations

from importlib import metadata as importlib_metadata

from .cli import cli
from .runtime import ToolExecutor, call_tool, executor, list_tools, mcp

try:  # pragma: no cover - fallback when package metadata is missing
    __version__ = importlib_metadata.version("tools-executor")
except importlib_metadata.PackageNotFoundError:  # pragma: no cover
    __version__ = "0.0.0"

__all__ = [
    "__version__",
    "ToolExecutor",
    "call_tool",
    "executor",
    "list_tools",
    "mcp",
    "cli",
]
