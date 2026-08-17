# Local Environment

Norvii uses Docker Compose for backing services while application modules run with
their native Go, Python, and Node.js toolchains. The canonical Compose file is
`infra/compose.yaml`.

## Persistence topology

The default `norvii` project starts exactly two services:

- PostgreSQL 18 with pgvector 0.8.6 is canonical relational, binary, document, and
  vector storage;
- standalone Neo4j Community 2026.06.0 is a rebuildable graph projection.

Both multi-architecture images are pinned by exact release tag and OCI digest.
PostgreSQL is limited to 512 MiB and Neo4j to 1.5 GiB, so contributors should make at
least 2 GiB available to Docker. PostgreSQL and Neo4j use distinct named volumes:
`norvii_postgres_data` and `norvii_neo4j_data`.

[ADR 0005](../decisions/0005-postgresql-and-neo4j-persistence.md) defines the
authority and projection boundaries. Removing Neo4j data cannot remove canonical
PostgreSQL data.

## Prerequisites

- Docker Engine with Docker Compose
- Go 1.26.5 or a compatible Go 1.26 patch release
- Python 3.13
- uv 0.11
- GNU Make and Bash on the reference Linux environment
- Node.js 24 and npm

The reference container platforms are Linux amd64 and arm64.

## Complete local bootstrap

After configuring `infra/.env`, start persistence, apply migrations, verify the Go
and Python production drivers, install deterministic web dependencies, and start the
React application with:

```bash
make bootstrap
```

The command is idempotent for long-running local processes. It retains output in the
repository-root `.log/` directory:

| File | Owner |
| --- | --- |
| `bootstrap.log` | Bootstrap orchestration and persistence startup |
| `web.log` | Web dependency installation and React/Vite development server |
| `api.log` | Go API migrations and persistence verification |
| `ingestion.log` | Python ingestion persistence verification |
| `postgres.log` | PostgreSQL container |
| `neo4j.log` | Neo4j container |

The Go API and Python ingestion module do not yet expose long-running local
processes. Their current initialization output is retained in their component logs.

Inspect lifecycle and health without starting duplicates, or stop all managed
processes while retaining database volumes:

```bash
make local-status
make local-stop
```

## Configuration

Create the ignored local configuration and replace both password markers:

```bash
cp infra/.env.example infra/.env
```

The example contains no usable credential. Single-quote values containing spaces,
dollar signs, or other special characters so Compose treats them literally. Norvii
parses the file without evaluating it as shell code and passes discrete connection
fields to drivers so password-bearing URLs do not enter errors or logs.

Validate the rendered service list:

```bash
make persistence-config
```

The output must contain only `postgres` and `neo4j`.

## Lifecycle commands

Start both stores, apply pending migrations, and verify the Go and Python production
drivers with the recommended local journey:

```bash
make persistence
```

The aggregate command stops immediately if startup, migration, or verification
fails. Use the commands below when operating or troubleshooting an individual step.

Start both stores and wait up to two minutes for authenticated health:

```bash
make persistence-up
```

Initialize pgvector and inspect the tern migration ledger:

```bash
make persistence-migrate
make persistence-migration-status
```

Initialization is idempotent. Feature 003 owns one migration,
`001_enable_vector.sql`; it creates no product table.

Verify Go and Python through their production database drivers:

```bash
make persistence-verify
```

Each runtime authenticates and executes constant read-only SQL and Cypher operations
within the configured timeout. No corpus, source, document, ingestion artifact, node,
or relationship is created.

Stop services while retaining both volumes:

```bash
make persistence-stop
```

## Health and troubleshooting

Inspect authenticated health without changing state:

```bash
make persistence-health
```

When a service is unhealthy, the command prints a bounded log command for that
service. The direct equivalents are:

```bash
docker compose --env-file infra/.env -f infra/compose.yaml ps
docker compose --env-file infra/.env -f infra/compose.yaml logs --tail 100 postgres
docker compose --env-file infra/.env -f infra/compose.yaml logs --tail 100 neo4j
```

Review third-party logs before sharing them. Norvii never prints the configured
passwords, but database-owned diagnostic output is outside the application's logging
boundary.

Common failures include an occupied port, a stale data volume initialized with older
credentials, insufficient Docker memory, and running migration or verifier commands
before health succeeds. A normal stop never fixes a stale credential because Neo4j
correctly retains authentication in its data volume; use the intentional reset only
when local data may be discarded.

## Isolated integration journey

Run the complete service-backed acceptance journey with:

```bash
make persistence-integration
```

The journey uses project `norvii-integration`, alternate local ports, temporary
credentials, and separately named integration volumes. It proves authenticated
health, repeatable migration, Go and Python connectivity, safe failure, normal
restart retention, graph-volume isolation, and clean-volume reproduction. A trap
removes the isolated containers and volumes; it never targets default local data.

## Intentional local reset

The following command permanently removes exactly `norvii_postgres_data` and
`norvii_neo4j_data` after validating their Compose ownership:

```bash
make persistence-reset CONFIRM=reset-norvii-data
```

The command accepts no volume name or filesystem path. It refuses missing or
unexpected volumes, a changed project identity, invalid ownership labels, and any
other confirmation. Removed data cannot be recovered through Norvii.

Recreate the foundation after reset:

```bash
make persistence
```

The feature-specific executable evidence remains in
[Feature 003 quickstart](../../specs/003-local-persistence-foundation/quickstart.md).
