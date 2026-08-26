# Implementation Plan: MCP Research Tools and Skills

**Branch**: `010-mcp-research-tools` | **Date**: 2026-08-25 | **Spec**: [spec.md](spec.md)

## Summary

Add a container-ready Streamable HTTP MCP service and a local stdio entry point to
`apps/agent`. The service exposes bounded, read-only, snapshot-aware research tools.

## Technical Context

**Language/Version**: Python 3.13

**Primary Dependencies**: Official MCP Python SDK 2.x, psycopg, Neo4j driver

**Storage**: Existing PostgreSQL and Neo4j data, read-only

**Testing**: pytest, Ruff, mypy, and MCP SDK interoperability tests

**Target Platform**: Docker service with Streamable HTTP; local stdio development

## Constitution Check

All principles pass: the server is a typed agent-module transport adapter; it reuses
published corpus snapshots, returns immutable evidence references, has a feature-owned
contract, performs no mutation, and uses bounded read-only queries.

## Project Structure

```text
apps/agent/src/norvii_agent/mcp/       # MCP composition and bounded tool handlers
apps/agent/tests/unit/mcp/              # tool and transport tests
apps/agent/Dockerfile                   # MCP container image
infra/compose.yaml                      # optional mcp service and local-only port
specs/010-mcp-research-tools/contracts/ # v1 MCP contract
```

## Module Impact

| Module | Change | Responsibility |
| --- | --- | --- |
| `apps/agent/` | Change | MCP server, tools, prompts, health endpoint |
| `infra/` | Change | MCP container service and configuration |
| `apps/api/`, `apps/web/`, `apps/ingestion/` | No change | Existing interfaces remain intact |

## Constraints

- Streamable HTTP binds to the container network; host publication defaults to
  `127.0.0.1` only.
- The server never mutates data, exposes raw SQL/Cypher, or logs corpus contents.
- Every corpus-scoped tool resolves one active published snapshot.
- Graph operations fail safely when a matching graph release is unavailable.
