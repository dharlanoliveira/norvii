"""Run the Norvii MCP server over local stdio or Streamable HTTP."""

from __future__ import annotations

import argparse
import logging

from norvii_agent.config import AgentConfig
from norvii_agent.mcp.research import MCPResearchRepository
from norvii_agent.mcp.server import build_server


def main() -> None:
    """Start the selected MCP transport without writing ordinary output to stdout."""
    logging.basicConfig(level=logging.INFO)
    parser = argparse.ArgumentParser()
    parser.add_argument("--transport", choices=("stdio", "streamable-http"), default="stdio")
    arguments = parser.parse_args()
    configuration = AgentConfig.from_environment()
    server = build_server(MCPResearchRepository(configuration))
    if arguments.transport == "stdio":
        server.run()
        return
    server.run(
        transport="streamable-http", host=configuration.mcp_host, port=configuration.mcp_port
    )


if __name__ == "__main__":
    main()
