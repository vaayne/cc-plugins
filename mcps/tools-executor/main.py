#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "fastmcp",
#     "click",
#     "jinja2"
# ]
# ///

from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parent
SRC = ROOT / "src"
if SRC.exists():
    sys.path.insert(0, str(SRC))

from tools_executor.cli import main  # noqa: E402

if __name__ == "__main__":
    main()
