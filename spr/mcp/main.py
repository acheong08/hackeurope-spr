import os
import requests
from typing import Dict, Any, List
from mcp.server.fastmcp import FastMCP

API_BASE = os.getenv("TRACE_API_BASE", "http://localhost:8001")

mcp = FastMCP("tracee-analysis")


# =====================================================
# TOOL: Health Check
# =====================================================

@mcp.tool()
def health() -> Dict[str, Any]:
    """Check if the Tracee API is healthy."""
    r = requests.get(f"{API_BASE}/health")
    r.raise_for_status()
    return r.json()


# =====================================================
# TOOL: List Collections (via stats discovery)
# =====================================================

@mcp.tool()
def list_collections() -> List[str]:
    """
    List available collections.
    (Calls Mongo indirectly via FastAPI if you add an endpoint for it.)
    """
    # If you don’t have a list endpoint yet,
    # you should add one in FastAPI:
    # GET /collections
    r = requests.get(f"{API_BASE}/collections")
    r.raise_for_status()
    return r.json()


# =====================================================
# TOOL: Get Behavioral Stats
# =====================================================

@mcp.tool()
def get_stats(collection_name: str) -> Dict[str, Any]:
    """Get aggregated behavioral stats."""
    r = requests.get(f"{API_BASE}/stats/{collection_name}")
    r.raise_for_status()
    return r.json()


# =====================================================
# TOOL: DNS Activity
# =====================================================

@mcp.tool()
def get_dns_activity(
    collection_name: str,
    dns: str,
    limit: int = 50,
    offset: int = 0
) -> Dict[str, Any]:
    """Get DNS drill-down data."""
    r = requests.get(
        f"{API_BASE}/specific/{collection_name}",
        params={
            "dns": dns,
            "limit": limit,
            "offset": offset
        }
    )
    r.raise_for_status()
    return r.json()


# =====================================================
# TOOL: Command Activity
# =====================================================

@mcp.tool()
def get_command_activity(
    collection_name: str,
    command: str,
    limit: int = 50,
    offset: int = 0
) -> Dict[str, Any]:
    """Get execve drill-down data."""
    r = requests.get(
        f"{API_BASE}/specific/{collection_name}",
        params={
            "command": command,
            "limit": limit,
            "offset": offset
        }
    )
    r.raise_for_status()
    return r.json()


# =====================================================
# TOOL: File Activity
# =====================================================

@mcp.tool()
def get_file_activity(
    collection_name: str,
    file: str,
    limit: int = 50,
    offset: int = 0
) -> Dict[str, Any]:
    """Get file access drill-down data."""
    r = requests.get(
        f"{API_BASE}/specific/{collection_name}",
        params={
            "file": file,
            "limit": limit,
            "offset": offset
        }
    )
    r.raise_for_status()
    return r.json()


if __name__ == "__main__":
    mcp.run()
