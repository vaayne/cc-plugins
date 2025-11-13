#!/usr/bin/env python3
"""
Quick Start Example for Multi-Backend MCP-FS

This example demonstrates how to:
1. Start the multi-backend server programmatically
2. Register backends at runtime
3. Perform basic file operations across multiple backends
4. Handle errors and check backend health

Run this example:
    python examples/quick_start.py
"""

import json
import logging
from pathlib import Path

# Import the multi-backend components
from mcp_fs.mcp_server import backend_manager

# Configure logging to see what's happening
logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")
logger = logging.getLogger(__name__)


def demonstrate_basic_usage():
    """Demonstrate basic multi-backend operations."""

    print("=== Multi-Backend MCP-FS Quick Start ===\n")

    # 1. Register some backends
    print("1. Registering backends...")

    # Register local filesystem as default
    result = backend_manager.register_backend(
        name="local",
        url="fs:///",
        description="Local filesystem",
        set_as_default=True,
        validate_connection=False,  # Skip validation for demo
    )
    print(f"   Local backend: {result}")

    # Register in-memory storage for testing
    result = backend_manager.register_backend(
        name="memory",
        url="memory:///",
        description="In-memory storage for testing",
        validate_connection=False,
    )
    print(f"   Memory backend: {result}")

    # 2. List all backends
    print("\n2. Listing all registered backends:")
    backends = backend_manager.list_backends()
    for backend in backends:
        status = "✓" if backend["health_status"] else "✗"
        default = " (default)" if backend["is_default"] else ""
        print(f"   {status} {backend['name']}: {backend['description']}{default}")

    # 3. Check backend health
    print("\n3. Checking backend health:")
    health = backend_manager.check_backend_health()
    for name, is_healthy in health.items():
        status = "✓ Healthy" if is_healthy else "✗ Unhealthy"
        print(f"   {name}: {status}")

    # 4. Get backend statistics
    print("\n4. Backend statistics:")
    stats = backend_manager.get_backend_stats()
    print(f"   Total backends: {stats['total_backends']}")
    print(f"   Default backend: {stats['default_backend']}")
    print(f"   Healthy backends: {stats['healthy_backends']}")
    print(f"   Supported schemes: {', '.join(stats['backend_schemes'])}")

    return True


def demonstrate_file_operations():
    """Demonstrate file operations across multiple backends."""

    print("\n=== File Operations Demo ===\n")

    try:
        # Get backend instances for direct operations
        backend_manager.get_backend("local")
        memory_backend = backend_manager.get_backend("memory")

        # 1. Create some test content in memory backend
        print("1. Creating test content in memory backend...")
        test_content = "Hello from Multi-Backend MCP!\nThis is a test file."
        memory_backend.write("/test.txt", test_content.encode("utf-8"))
        print("   ✓ Created /test.txt in memory backend")

        # 2. Read the content back
        print("\n2. Reading content from memory backend...")
        read_content = memory_backend.read("/test.txt").decode("utf-8")
        print(f"   Content: {repr(read_content[:50])}...")

        # 3. List files in memory backend
        print("\n3. Listing files in memory backend:")
        for entry in memory_backend.list("/"):
            print(f"   - {entry.path}")

        # 4. Create a directory structure
        print("\n4. Creating directory structure...")
        memory_backend.create_dir("/documents/")
        memory_backend.create_dir("/backups/")
        memory_backend.write(
            "/documents/readme.txt", b"Welcome to the documents folder!"
        )
        memory_backend.write(
            "/backups/config.json", b'{"version": "1.0", "name": "backup"}'
        )

        print("   ✓ Created directories and files")

        # 5. List the directory structure
        print("\n5. Directory structure in memory backend:")
        for entry in memory_backend.list("/"):
            print(f"   {entry.path}")

        return True

    except Exception as e:
        print(f"   Error during file operations: {e}")
        return False


def demonstrate_cross_backend_copy():
    """Demonstrate copying files between different backends."""

    print("\n=== Cross-Backend Copy Demo ===\n")

    try:
        memory_backend = backend_manager.get_backend("memory")

        # For this demo, we'll simulate cross-backend operations
        # In a real scenario, you'd have different actual backends

        print("1. Creating source file in memory backend...")
        source_content = "This file will be copied between backends"
        memory_backend.write("/source/data.txt", source_content.encode("utf-8"))
        memory_backend.create_dir("/destination/")

        # Simulate cross-backend copy by reading from source and writing to destination
        print("\n2. Simulating cross-backend copy...")
        data = memory_backend.read("/source/data.txt")
        memory_backend.write("/destination/data_copy.txt", data)

        print("   ✓ File copied successfully")

        # Verify the copy
        print("\n3. Verifying copied file...")
        copied_content = memory_backend.read("/destination/data_copy.txt").decode(
            "utf-8"
        )
        print(f"   Original: {repr(source_content)}")
        print(f"   Copied:   {repr(copied_content)}")
        print(f"   Match: {'✓' if source_content == copied_content else '✗'}")

        return True

    except Exception as e:
        print(f"   Error during cross-backend copy: {e}")
        return False


def demonstrate_error_handling():
    """Demonstrate error handling and edge cases."""

    print("\n=== Error Handling Demo ===\n")

    # 1. Try to access non-existent backend
    print("1. Testing non-existent backend access...")
    try:
        backend_manager.get_backend("non-existent")
        print("   ✗ Should have failed!")
    except ValueError as e:
        print(f"   ✓ Correctly caught error: {e}")

    # 2. Try to register backend with invalid name
    print("\n2. Testing invalid backend name...")
    try:
        backend_manager.register_backend(
            name="invalid name with spaces", url="memory:///", validate_connection=False
        )
        print("   ✗ Should have failed!")
    except ValueError as e:
        print(f"   ✓ Correctly caught error: {e}")

    # 3. Try to access non-existent file
    print("\n3. Testing non-existent file access...")
    try:
        memory_backend = backend_manager.get_backend("memory")
        memory_backend.read("/non-existent-file.txt")
        print("   ✗ Should have failed!")
    except Exception as e:
        print(f"   ✓ Correctly caught error: {type(e).__name__}")

    print("\n   ✓ Error handling working correctly")


def create_sample_config():
    """Create a sample configuration file."""

    config = {
        "backends": [
            {
                "name": "local-demo",
                "url": "fs:///",
                "description": "Local filesystem for demo",
                "default": True,
                "readonly": False,
                "timeout": 30,
                "retry_attempts": 3,
                "validate_connection": False,
            },
            {
                "name": "memory-demo",
                "url": "memory:///",
                "description": "In-memory storage for demo",
                "readonly": False,
                "timeout": 10,
                "retry_attempts": 1,
                "validate_connection": False,
            },
        ]
    }

    config_path = Path("examples/demo_config.json")
    config_path.parent.mkdir(exist_ok=True)

    with open(config_path, "w") as f:
        json.dump(config, f, indent=2)

    print(f"\n📄 Sample configuration saved to: {config_path}")
    print("   You can use this with: mcp-fs examples/demo_config.json")


def main():
    """Run the complete demonstration."""

    try:
        # Run all demonstrations
        success = True
        success &= demonstrate_basic_usage()
        success &= demonstrate_file_operations()
        success &= demonstrate_cross_backend_copy()
        demonstrate_error_handling()

        if success:
            print("\n🎉 All demonstrations completed successfully!")
        else:
            print("\n⚠️  Some demonstrations encountered issues.")

        # Create sample configuration
        create_sample_config()

        print("\n📚 Next steps:")
        print("   1. Review the README.md for complete documentation")
        print(
            "   2. Try running the server: mcp-fs examples/demo_config.json"
        )
        print("   3. Experiment with different backend configurations")
        print("   4. Integrate with your MCP client application")

    except Exception as e:
        logger.error(f"Demo failed with error: {e}")
        print(f"\n❌ Demo failed: {e}")


if __name__ == "__main__":
    main()
