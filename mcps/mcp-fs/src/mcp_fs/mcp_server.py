import json
import logging
from typing import Any

import opendal
from mcp.server.fastmcp import FastMCP

from mcp_fs.backend_manager import BackendManager

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastMCP server
mcp = FastMCP("MCP-FS Server")

# Global backend manager
backend_manager = BackendManager()


@mcp.tool()
def register_backend(
    name: str,
    url: str,
    description: str = "",
    readonly: bool = False,
    timeout: int = 30,
    retry_attempts: int = 3,
    set_as_default: bool = False,
    validate_connection: bool = True,
) -> str:
    """
    Register a new backend with the given configuration.

    Args:
        name: Unique identifier for the backend (alphanumeric, hyphens, underscores only)
        url: Backend connection URL (e.g., "fs:///", "s3://bucket", "memory:///")
        description: Human-readable description of the backend
        readonly: Whether this backend should be treated as read-only
        timeout: Connection timeout in seconds
        retry_attempts: Number of retry attempts for failed operations
        set_as_default: Whether to set this as the default backend
        validate_connection: Whether to validate the connection during registration

    Returns:
        Success or failure message
    """
    try:
        success = backend_manager.register_backend(
            name=name,
            url=url,
            description=description,
            readonly=readonly,
            timeout=timeout,
            retry_attempts=retry_attempts,
            set_as_default=set_as_default,
            validate_connection=validate_connection,
        )

        if success:
            return f"Successfully registered backend '{name}'"
        else:
            return f"Failed to register backend '{name}' (connection validation failed)"

    except ValueError as e:
        return f"Configuration error: {e}"
    except ConnectionError as e:
        return f"Connection error: {e}"
    except Exception as e:
        logger.error(f"Unexpected error registering backend '{name}': {e}")
        return f"Unexpected error: {e}"


@mcp.tool()
def list_backends() -> list[dict[str, Any]]:
    """
    List all registered backends with their configurations and status.

    Returns:
        List of backend information including name, URL, description,
        readonly status, default status, and health status
    """
    return backend_manager.list_backends()


@mcp.tool()
def set_default_backend(name: str) -> str:
    """
    Set the default backend for operations when no backend is specified.

    Args:
        name: Name of the backend to set as default

    Returns:
        Success message
    """
    try:
        backend_manager.set_default_backend(name)
        return f"Successfully set '{name}' as the default backend"
    except ValueError as e:
        return f"Error: {e}"


@mcp.tool()
def remove_backend(name: str, force: bool = False) -> str:
    """
    Remove a backend from the manager.

    Args:
        name: Name of the backend to remove
        force: Force removal even if it's the default backend

    Returns:
        Success message
    """
    try:
        backend_manager.remove_backend(name, force)
        return f"Successfully removed backend '{name}'"
    except ValueError as e:
        return f"Error: {e}"


@mcp.tool()
def check_backend_health(backend: str | None = None) -> dict[str, bool]:
    """
    Check the health status of backends by attempting basic operations.

    Args:
        backend: Specific backend to check (checks all if not specified)

    Returns:
        Dictionary mapping backend names to their health status (True/False)
    """
    return backend_manager.check_backend_health(backend)


@mcp.tool()
def get_backend_stats() -> dict[str, Any]:
    """
    Get statistics and information about the backend manager.

    Returns:
        Dictionary with total backends, default backend, healthy backends count, etc.
    """
    return backend_manager.get_backend_stats()


@mcp.tool()
def list_files(path: str = "/", backend: str | None = None) -> list[dict[str, Any]]:
    """
    List files and directories in the specified path.

    Args:
        path: The directory path to list (default: "/")
        backend: Backend name to use (uses default backend if not specified)

    Returns:
        List of entries with their metadata
    """
    try:
        logger.info(f"Listing files in path: {path} (backend: {backend or 'default'})")
        client = backend_manager.get_backend(backend)

        entries = []
        for entry in client.list(path):
            entries.append(
                {
                    "path": entry.path,
                    "backend": backend or backend_manager.default_backend,
                }
            )

        logger.info(f"Found {len(entries)} entries in path: {path}")
        return entries

    except ValueError as e:
        logger.error(f"Backend error: {e}")
        raise ValueError(f"Backend error: {e}") from e
    except Exception as e:
        logger.error(f"Failed to list files in {path}: {e}")
        raise Exception(f"Failed to list files: {e}") from e


@mcp.tool()
def read_file(path: str, backend: str | None = None) -> str:
    """
    Read file contents from the specified backend.

    Args:
        path: File path to read
        backend: Backend name to use (uses default backend if not specified)

    Returns:
        File contents as string
    """
    try:
        logger.info(f"Reading file: {path} (backend: {backend or 'default'})")
        client = backend_manager.get_backend(backend)
        data = client.read(path)
        content = data.decode("utf-8")

        logger.info(f"Successfully read file: {path} ({len(content)} characters)")
        return content

    except ValueError as e:
        logger.error(f"Backend error: {e}")
        raise ValueError(f"Backend error: {e}") from e
    except Exception as e:
        logger.error(f"Failed to read file {path}: {e}")
        raise Exception(f"Failed to read file: {e}") from e


@mcp.tool()
def write_file(path: str, content: str, backend: str | None = None) -> str:
    """
    Write content to a file on the specified backend.

    Args:
        path: File path to write to
        content: Content to write to the file
        backend: Backend name to use (uses default backend if not specified)

    Returns:
        Success message
    """
    try:
        logger.info(
            f"Writing file: {path} (backend: {backend or 'default'}, {len(content)} characters)"
        )
        client = backend_manager.get_backend(backend)

        # Check if backend is readonly
        backend_name = backend or backend_manager.default_backend
        if backend_name:
            config = backend_manager.get_backend_config(backend_name)
            if config.readonly:
                raise ValueError(f"Backend '{backend_name}' is configured as read-only")

        client.write(path, content.encode("utf-8"))
        logger.info(f"Successfully wrote file: {path}")
        return f"Successfully wrote to {path}"

    except ValueError as e:
        logger.error(f"Backend error: {e}")
        raise ValueError(f"Backend error: {e}") from e
    except Exception as e:
        logger.error(f"Failed to write file {path}: {e}")
        raise Exception(f"Failed to write file: {e}") from e


@mcp.tool()
def copy_file(
    src: str,
    dst: str,
    src_backend: str | None = None,
    dst_backend: str | None = None,
) -> str:
    """
    Copy a file between backends or within the same backend.

    Args:
        src: Source file path
        dst: Destination file path
        src_backend: Source backend name (uses default if not specified)
        dst_backend: Destination backend name (uses default if not specified)

    Returns:
        Success message
    """
    try:
        src_backend_name = src_backend or backend_manager.default_backend
        dst_backend_name = dst_backend or backend_manager.default_backend

        logger.info(
            f"Copying {src} to {dst} (backends: {src_backend_name} -> {dst_backend_name})"
        )

        # Check if destination backend is readonly
        if dst_backend_name:
            dst_config = backend_manager.get_backend_config(dst_backend_name)
            if dst_config.readonly:
                raise ValueError(
                    f"Destination backend '{dst_backend_name}' is read-only"
                )

        # If same backend, use native copy operation
        if src_backend_name == dst_backend_name:
            client = backend_manager.get_backend(src_backend_name)
            client.copy(src, dst)
        else:
            # Cross-backend copy: read from source, write to destination
            src_client = backend_manager.get_backend(src_backend)
            dst_client = backend_manager.get_backend(dst_backend)

            # Read from source
            data = src_client.read(src)
            # Write to destination
            dst_client.write(dst, data)

        logger.info(f"Successfully copied {src} to {dst}")
        return f"Successfully copied {src} to {dst}"

    except ValueError as e:
        logger.error(f"Backend error: {e}")
        raise ValueError(f"Backend error: {e}") from e
    except Exception as e:
        logger.error(f"Failed to copy {src} to {dst}: {e}")
        raise Exception(f"Failed to copy file: {e}") from e


@mcp.tool()
def rename_file(src: str, dst: str, backend: str | None = None) -> str:
    """
    Rename or move a file within a backend.

    Args:
        src: Source file path
        dst: Destination file path
        backend: Backend name to use (uses default backend if not specified)

    Returns:
        Success message
    """
    try:
        logger.info(
            f"Renaming file from {src} to {dst} (backend: {backend or 'default'})"
        )
        client = backend_manager.get_backend(backend)

        # Check if backend is readonly
        backend_name = backend or backend_manager.default_backend
        if backend_name:
            config = backend_manager.get_backend_config(backend_name)
            if config.readonly:
                raise ValueError(f"Backend '{backend_name}' is read-only")

        client.rename(src, dst)
        logger.info(f"Successfully renamed {src} to {dst}")
        return f"Successfully renamed {src} to {dst}"

    except ValueError as e:
        logger.error(f"Backend error: {e}")
        raise ValueError(f"Backend error: {e}") from e
    except Exception as e:
        logger.error(f"Failed to rename {src} to {dst}: {e}")
        raise Exception(f"Failed to rename file: {e}") from e


@mcp.tool()
def create_dir(path: str, backend: str | None = None) -> str:
    """
    Create a directory on the specified backend.

    Args:
        path: Directory path to create
        backend: Backend name to use (uses default backend if not specified)

    Returns:
        Success message
    """
    try:
        logger.info(f"Creating directory: {path} (backend: {backend or 'default'})")
        client = backend_manager.get_backend(backend)

        # Check if backend is readonly
        backend_name = backend or backend_manager.default_backend
        if backend_name:
            config = backend_manager.get_backend_config(backend_name)
            if config.readonly:
                raise ValueError(f"Backend '{backend_name}' is read-only")

        client.create_dir(path)
        logger.info(f"Successfully created directory: {path}")
        return f"Successfully created directory {path}"

    except ValueError as e:
        logger.error(f"Backend error: {e}")
        raise ValueError(f"Backend error: {e}") from e
    except Exception as e:
        logger.error(f"Failed to create directory {path}: {e}")
        raise Exception(f"Failed to create directory: {e}") from e


@mcp.tool()
def stat_file(path: str, backend: str | None = None) -> opendal.Metadata:
    """
    Get metadata/statistics for a file or directory.

    Args:
        path: Path to get statistics for
        backend: Backend name to use (uses default backend if not specified)

    Returns:
        File metadata object
    """
    try:
        logger.info(f"Getting metadata for: {path} (backend: {backend or 'default'})")
        client = backend_manager.get_backend(backend)
        metadata = client.stat(path)

        logger.info(f"Successfully retrieved metadata for: {path}")
        return metadata

    except ValueError as e:
        logger.error(f"Backend error: {e}")
        raise ValueError(f"Backend error: {e}") from e
    except Exception as e:
        logger.error(f"Failed to get metadata for {path}: {e}")
        raise Exception(f"Failed to get file metadata: {e}") from e


def load_config_from_file(config_path: str) -> None:
    """Load backend configuration from a JSON file."""
    try:
        with open(config_path) as f:
            config = json.load(f)

        for backend_config in config.get("backends", []):
            try:
                backend_manager.register_backend(
                    name=backend_config["name"],
                    url=backend_config["url"],
                    description=backend_config.get("description", ""),
                    readonly=backend_config.get("readonly", False),
                    timeout=backend_config.get("timeout", 30),
                    retry_attempts=backend_config.get("retry_attempts", 3),
                    set_as_default=backend_config.get("default", False),
                    validate_connection=backend_config.get("validate_connection", True),
                )
                logger.info(f"Loaded backend '{backend_config['name']}' from config")

            except Exception as e:
                logger.error(f"Failed to load backend '{backend_config['name']}': {e}")

    except FileNotFoundError:
        logger.error(f"Configuration file not found: {config_path}")
        raise
    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON in configuration file: {e}")
        raise
