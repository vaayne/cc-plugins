#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "fastmcp",
#     "click",
#     "jinja2"
# ]
# ///

from tools_executor.cli import main

if __name__ == "__main__":
    main()
