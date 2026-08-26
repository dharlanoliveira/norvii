# Quickstart: MCP Research Tools and Skills

## Start the Docker service

Create `infra/.env`, replace the required password markers, then build and start the
MCP profile with its backing stores:

```bash
docker compose --env-file infra/.env -f infra/compose.yaml --profile mcp up --build --wait
```

The MCP endpoint is `http://127.0.0.1:8091/mcp`. Docker exposes it only on the local
host; other Norvii services reach it through the Compose network.

## Development transport

For a local MCP client that launches a subprocess instead of the container service:

```bash
uv --directory apps/agent run norvii-mcp --transport stdio
```

Do not write ordinary output to stdout when using stdio; MCP owns that stream.

## Verify

1. Discover the eight tools and three prompts.
2. Run `list_corpora`, choose a corpus ID, then run `list_documents`.
3. Search the corpus and confirm evidence references have the returned `snapshot_id`.
4. Retrieve one legal unit with `get_article`.
5. Run a graph tool against a corpus without a ready graph release and verify the
   safe `unavailable` outcome.
6. Stop the profile with `docker compose --env-file infra/.env -f infra/compose.yaml --profile mcp down`.

## Verification Record

- 2026-08-25: Agent format checking, linting, type checking, package build, and 38
  selected pytest tests passed.
- 2026-08-25: Contract and repository-language validation plus `git diff --check`
  passed. The MCP Docker image built, the profile Compose configuration validated,
  and an SDK client outside the started container completed initialization and
  discovered all eight tools and three workflow prompts through
  `http://127.0.0.1:18091/mcp`. An SDK stdio client also discovered all eight tools.
