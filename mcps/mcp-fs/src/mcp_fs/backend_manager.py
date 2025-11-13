import logging
import re
from typing import Any
from urllib.parse import urlparse

from mcp_fs.dal import DAL

logger = logging.getLogger(__name__)


class BackendConfig:
    """Configuration for a backend."""

    def __init__(
        self,
        name: str,
        url: str,
        description: str = "",
        readonly: bool = False,
        timeout: int = 30,
        retry_attempts: int = 3,
    ):
        self.name = name
        self.url = url
        self.description = description
        self.readonly = readonly
        self.timeout = timeout
        self.retry_attempts = retry_attempts
        self._validate()

    def _validate(self) -> None:
        """Validate backend configuration."""
        if not self.name or not isinstance(self.name, str):
            raise ValueError("Backend name must be a non-empty string")

        if not re.match(r"^[a-zA-Z0-9_-]+$", self.name):
            raise ValueError(
                "Backend name can only contain letters, numbers, hyphens, and underscores"
            )

        if not self.url or not isinstance(self.url, str):
            raise ValueError("Backend URL must be a non-empty string")

        # Validate URL format
        parsed = urlparse(self.url)
        if not parsed.scheme:
            raise ValueError(f"Invalid URL format: {self.url}")

        # Validate supported schemes
        supported_schemes = {"fs", "s3", "webdav", "memory", "http", "https", "ftp"}
        if parsed.scheme not in supported_schemes:
            logger.warning(
                f"Unsupported scheme '{parsed.scheme}' for backend '{self.name}'"
            )


class BackendManager:
    """Manages multiple DAL backend instances with advanced features."""

    def __init__(self) -> None:
        self.backends: dict[str, DAL] = {}
        self.configs: dict[str, BackendConfig] = {}
        self.default_backend: str | None = None
        self._health_status: dict[str, bool] = {}

    def register_backend(
        self,
        name: str,
        url: str,
        description: str = "",
        readonly: bool = False,
        timeout: int = 30,
        retry_attempts: int = 3,
        set_as_default: bool = False,
        validate_connection: bool = True,
    ) -> bool:
        """
        Register a new backend with comprehensive configuration.

        Args:
            name: Unique identifier for the backend
            url: Backend connection URL
            description: Human-readable description
            readonly: Whether this backend is read-only
            timeout: Connection timeout in seconds
            retry_attempts: Number of retry attempts for failed operations
            set_as_default: Whether to set as default backend
            validate_connection: Whether to validate connection during registration

        Returns:
            bool: True if registration successful

        Raises:
            ValueError: If configuration is invalid
            ConnectionError: If connection validation fails
        """
        if name in self.backends:
            logger.warning(f"Backend '{name}' already exists, replacing")

        config = BackendConfig(
            name, url, description, readonly, timeout, retry_attempts
        )

        try:
            logger.info(f"Registering backend '{name}' with URL: {url}")
            dal_instance = DAL.from_url(url)

            if validate_connection:
                self._validate_connection(dal_instance, name)

            self.configs[name] = config
            self.backends[name] = dal_instance
            self._health_status[name] = True

            if set_as_default or self.default_backend is None:
                self.default_backend = name
                logger.info(f"Set '{name}' as default backend")

            logger.info(f"Successfully registered backend '{name}'")
            return True

        except Exception as e:
            logger.error(f"Failed to register backend '{name}': {e}")
            self._health_status[name] = False
            if validate_connection:
                raise ConnectionError(f"Backend '{name}' connection failed: {e}") from e
            return False

    def _validate_connection(self, dal_instance: DAL, name: str) -> None:
        """Validate connection to a backend."""
        try:
            # Try a simple operation to validate connection
            dal_instance.list("/")
            logger.info(f"Connection validation successful for backend '{name}'")
        except Exception as e:
            logger.error(f"Connection validation failed for backend '{name}': {e}")
            raise

    def get_backend(self, name: str | None = None) -> DAL:
        """
        Get backend by name, or default if no name provided.

        Args:
            name: Backend name (uses default if None)

        Returns:
            DAL: The backend instance

        Raises:
            ValueError: If backend not found or no default set
        """
        if name is None:
            if self.default_backend is None:
                raise ValueError("No default backend set and no backend name provided")
            name = self.default_backend

        if name not in self.backends:
            available = ", ".join(self.backends.keys())
            raise ValueError(
                f"Backend '{name}' not found. Available backends: {available}"
            )

        return self.backends[name]

    def get_backend_config(self, name: str) -> BackendConfig:
        """Get backend configuration."""
        if name not in self.configs:
            raise ValueError(f"Backend config '{name}' not found")
        return self.configs[name]

    def list_backends(self) -> list[dict[str, Any]]:
        """
        List all registered backends with their configurations.

        Returns:
            List of backend information dictionaries
        """
        result = []
        for name in self.backends.keys():
            config = self.configs.get(name)
            result.append(
                {
                    "name": name,
                    "url": config.url if config else "unknown",
                    "description": config.description if config else "",
                    "readonly": config.readonly if config else False,
                    "is_default": name == self.default_backend,
                    "health_status": self._health_status.get(name, False),
                }
            )
        return result

    def set_default_backend(self, name: str) -> None:
        """Set the default backend."""
        if name not in self.backends:
            raise ValueError(f"Backend '{name}' not found")

        old_default = self.default_backend
        self.default_backend = name
        logger.info(f"Changed default backend from '{old_default}' to '{name}'")

    def remove_backend(self, name: str, force: bool = False) -> None:
        """
        Remove a backend.

        Args:
            name: Backend name to remove
            force: Remove even if it's the default backend

        Raises:
            ValueError: If trying to remove default backend without force
        """
        if name not in self.backends:
            raise ValueError(f"Backend '{name}' not found")

        if name == self.default_backend and not force:
            raise ValueError(
                f"Cannot remove default backend '{name}' without force=True"
            )

        del self.backends[name]
        if name in self.configs:
            del self.configs[name]
        if name in self._health_status:
            del self._health_status[name]

        if self.default_backend == name:
            self.default_backend = next(iter(self.backends.keys()), None)
            if self.default_backend:
                logger.info(f"Set new default backend to '{self.default_backend}'")

        logger.info(f"Removed backend '{name}'")

    def check_backend_health(self, name: str | None = None) -> dict[str, bool]:
        """
        Check health status of backends.

        Args:
            name: Specific backend to check (checks all if None)

        Returns:
            Dictionary mapping backend names to health status
        """
        backends_to_check = [name] if name else list(self.backends.keys())
        results = {}

        for backend_name in backends_to_check:
            if backend_name not in self.backends:
                results[backend_name] = False
                continue

            try:
                backend = self.backends[backend_name]
                # Simple health check - try to list root directory
                backend.list("/")
                results[backend_name] = True
                self._health_status[backend_name] = True

            except Exception as e:
                logger.warning(f"Health check failed for backend '{backend_name}': {e}")
                results[backend_name] = False
                self._health_status[backend_name] = False

        return results

    def get_backend_stats(self) -> dict[str, Any]:
        """Get statistics about the backend manager."""
        return {
            "total_backends": len(self.backends),
            "default_backend": self.default_backend,
            "healthy_backends": sum(
                1 for status in self._health_status.values() if status
            ),
            "readonly_backends": sum(
                1 for config in self.configs.values() if config.readonly
            ),
            "backend_schemes": list(
                {urlparse(config.url).scheme for config in self.configs.values()}
            ),
        }
