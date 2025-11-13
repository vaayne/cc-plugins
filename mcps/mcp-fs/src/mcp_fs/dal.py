import logging
from collections.abc import Iterable
from urllib.parse import parse_qs, urlparse

import opendal


class DAL:
    def __init__(self, schema: str, options: dict | None = None):
        if options is None:
            options = {}
        self.schema = schema
        self.options = options
        self._op: opendal.Operator | None = (
            None  # Lazy-initialized opendal.Operator instance
        )
        self.logger = logging.getLogger(__name__)

    @property
    def op(self) -> opendal.Operator:
        """
        Lazily initialize and return the opendal.Operator instance for WebDAV.

        Returns:
            opendal.Operator: The operator instance.
        """
        if self._op is None:
            self.logger.info(f"Initializing OpenDAL connection to {self.schema}")
            self._op = opendal.Operator(scheme=self.schema, **self.options)
            self.logger.debug("OpenDAL operator initialized successfully")
        return self._op

    @classmethod
    def from_url(cls, url: str) -> "DAL":
        """
        Create an OpenDAL instance from a URL.

        Args:
            url (str): The URL to create the instance from.

        Returns:
            OpenDAL: The created instance.
        """
        u = urlparse(url)
        parsed_options = parse_qs(u.query)
        options = {key: val[0] for key, val in parsed_options.items() if val}

        schema = u.scheme

        # Handle different backend types with their specific requirements
        if schema == "fs":
            # For local filesystem, we don't need the netloc
            return cls(schema, options=options)
        elif schema in ("webdav", "http", "https"):
            # WebDAV, HTTP, and HTTPS backends need an 'endpoint' parameter
            # Convert webdav:// to http:// for the endpoint
            if schema == "webdav":
                endpoint_scheme = "http"
            else:
                endpoint_scheme = schema

            # Construct the endpoint URL
            if u.netloc:
                endpoint = f"{endpoint_scheme}://{u.netloc}"
                if u.path:
                    endpoint += u.path
                options["endpoint"] = endpoint

            return cls(schema, options=options)
        else:
            # For other schemes (s3, memory, ftp, etc.), use existing behavior
            return cls(schema, options=options)

    def list(self, path: str) -> Iterable[opendal.Entry]:
        """
        List entries in the given directory path.

        Args:
            path (str): The directory path to list.

        Returns:
            Iterable[opendal.Entry]: An iterable of entries in the directory.
        """
        self.logger.debug(f"Listing directory: {path}")
        try:
            result = self.op.list(path)
            self.logger.info(f"Successfully listed directory: {path}")
            return result
        except Exception as e:
            self.logger.error(f"Failed to list directory {path}: {e}")
            raise

    def open(self, path: str, mode: str) -> opendal.File:
        """
        Open a file at the given path with the specified mode.

        Args:
            path (str): The file path to open.
            mode (str): The mode to open the file with (e.g., 'rb', 'wb').

        Returns:
            opendal.File: The opened file object.
        """
        self.logger.debug(f"Opening file: {path} with mode: {mode}")
        try:
            result = self.op.open(path, mode)
            self.logger.info(f"Successfully opened file: {path}")
            return result
        except Exception as e:
            self.logger.error(f"Failed to open file {path}: {e}")
            raise

    def read(self, path: str) -> bytes:
        """
        Read the contents of a file at the given path.

        Args:
            path (str): The file path to read.

        Returns:
            bytes: The contents of the file.
        """
        self.logger.debug(f"Reading file: {path}")
        try:
            result = self.op.read(path)
            self.logger.info(f"Successfully read file: {path} ({len(result)} bytes)")
            return result
        except Exception as e:
            self.logger.error(f"Failed to read file {path}: {e}")
            raise

    def stat(self, path: str) -> opendal.Metadata:
        """
        Get metadata/statistics for a file or directory at the given path.

        Args:
            path (str): The path to stat.

        Returns:
            opendal.Metadata: The metadata of the file or directory.
        """
        self.logger.debug(f"Getting stats for: {path}")
        try:
            result = self.op.stat(path)
            self.logger.info(f"Successfully got stats for: {path}")
            return result
        except Exception as e:
            self.logger.error(f"Failed to get stats for {path}: {e}")
            raise

    def copy(self, src: str, dst: str) -> None:
        """
        Copy a file or directory from src to dst.

        Args:
            src (str): The source path.
            dst (str): The destination path.
        """
        self.logger.debug(f"Copying from {src} to {dst}")
        try:
            self.op.copy(src, dst)
            self.logger.info(f"Successfully copied {src} to {dst}")
        except Exception as e:
            self.logger.error(f"Failed to copy {src} to {dst}: {e}")
            raise

    def write(self, path: str, data: bytes) -> None:
        """
        Write data from src to dst.

        Args:
            path (str): The destination path.
            data (bytes): The data to write.
        """
        self.logger.debug(f"Writing to file: {path} ({len(data)} bytes)")
        try:
            self.op.write(path, data)
            self.logger.info(f"Successfully wrote to file: {path}")
        except Exception as e:
            self.logger.error(f"Failed to write to file {path}: {e}")
            raise

    def rename(self, src: str, dst: str) -> None:
        """
        Rename or move a file or directory from src to dst.

        Args:
            src (str): The source path.
            dst (str): The destination path.
        """
        self.logger.debug(f"Renaming from {src} to {dst}")
        try:
            self.op.rename(src, dst)
            self.logger.info(f"Successfully renamed {src} to {dst}")
        except Exception as e:
            self.logger.error(f"Failed to rename {src} to {dst}: {e}")
            raise

    def create_dir(self, path: str) -> None:
        """
        Create a directory at the specified path.

        Args:
            path (str): The directory path to create.
        """
        self.logger.debug(f"Creating directory: {path}")
        try:
            self.op.create_dir(path)
            self.logger.info(f"Successfully created directory: {path}")
        except Exception as e:
            self.logger.error(f"Failed to create directory {path}: {e}")
            raise

    def exists(self, path: str) -> bool:
        """
        Check if a file or directory exists at the specified path.

        Args:
            path (str): The path to check.

        Returns:
            bool: True if the path exists, False otherwise.
        """
        self.logger.debug(f"Checking if path exists: {path}")
        try:
            result = self.op.exists(path)
            self.logger.info(f"Path exists: {path}")
            return result
        except Exception as e:
            self.logger.error(f"Failed to check if path exists {path}: {e}")
            raise

    def delete(self, path: str) -> None:
        """
        Delete a file or directory at the specified path.

        Args:
            path (str): The path to delete.

        Returns:
            bool: True if the path was deleted, False otherwise.
        """
        self.logger.debug(f"Deleting path: {path}")
        try:
            self.op.delete(path)
            self.logger.info(f"Path deleted: {path}")
        except Exception as e:
            self.logger.error(f"Failed to delete path {path}: {e}")
            raise
