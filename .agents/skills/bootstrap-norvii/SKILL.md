---
name: bootstrap-norvii
description: Start and verify the complete Norvii local development environment with persistent component logs. Use when the user asks to bootstrap, start, run, or diagnose the full Norvii application locally, including React, PostgreSQL, Neo4j, migrations, and production-driver verification.
---

# Bootstrap Norvii

Start Norvii through its repository-owned orchestration and report a concise readiness result.

## Workflow

1. Resolve the Norvii repository root with `git rev-parse --show-toplevel` and confirm
   that it contains `Makefile`, `infra/compose.yaml`, and `apps/web/package.json`.
2. Check for `infra/.env`. If it is absent or still contains password markers, stop
   and ask the user to copy `infra/.env.example` and replace both markers. Never
   generate, display, or infer passwords.
3. Select the repository-owned command:
   - run `make bootstrap` or `make local-start` to start the environment;
   - run `make local-status` to diagnose lifecycle or health state;
   - run `make local-stop` only when the user asks to stop the environment; stored
     PostgreSQL and Neo4j data is preserved.
   Do not start Vite or Compose directly because that bypasses managed logging and
   process identity checks.
4. On success, report the web URL and the absolute `.log/` path. State that the Go
   API and Python ingestion module currently run migration and verification commands
   during bootstrap; they do not yet expose long-lived server or worker processes.
5. On failure, inspect only the relevant bounded tails from:
   - `.log/bootstrap.log` for persistence orchestration and startup failures;
   - `.log/web.log` for dependency installation, React, and Vite;
   - `.log/api.log` for migrations and Go persistence verification;
   - `.log/ingestion.log` for Python persistence verification;
   - `.log/postgres.log` for PostgreSQL;
   - `.log/neo4j.log` for Neo4j.
6. Explain the failing component and next corrective action. Do not paste entire logs
   or expose credentials, connection strings, document content, or corpus data.

Repeated execution is expected and must reuse healthy managed processes rather than
starting duplicates.
