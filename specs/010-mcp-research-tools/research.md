# Research: MCP Research Tools and Skills

## Decisions

- Use the official Python MCP SDK 2.x. It owns protocol negotiation, lifecycle, and
  Streamable HTTP compatibility instead of a hand-written JSON-RPC implementation.
- Run Streamable HTTP in the Docker service and retain stdio for local development.
- Keep tool handlers in the agent module and use only bounded, read-only queries.
- Resolve the active published snapshot per corpus-scoped invocation. Graph queries
  return `unavailable` when no matching ready graph release exists.
- Expose workflows as MCP prompts. They guide evidence-grounded tool composition but
  do not make provider calls or generate legal advice.

## Deferred

Remote MCP publication, authentication delegation, TLS termination, and non-local
host access are outside the trusted-user POC. Any future remote deployment must add
authentication and Origin validation before publishing the endpoint.
