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
and Python production drivers, install locked module dependencies, and start the Go
API facade, Python LangGraph agent, Python worker, and React application with:

```bash
make bootstrap
```

The command is idempotent for long-running local processes. It retains output in the
repository-root `.log/` directory:

| File | Owner |
| --- | --- |
| `bootstrap.log` | Bootstrap orchestration and persistence startup |
| `web.log` | Web dependency installation and React/Vite development server |
| `api.log` | Go migrations, verification, and HTTP API process |
| `agent.log` | Python LangGraph agent and provider process |
| `ingestion.log` | Python verification and ingestion worker process |
| `postgres.log` | PostgreSQL container |
| `neo4j.log` | Neo4j container |

The command waits for authenticated database health, the agent and API `/healthz`
responses, the web server, and both stable initial sources to reach `ready` or safe `failed`
before reporting readiness. A source becomes `ready` only after the worker stages its immutable
snapshot, builds its Neo4j graph release, and activates that graph-ready snapshot. The
initial-ingestion wait is bounded by
`NORVII_INITIAL_INGESTION_TIMEOUT_SECONDS` (1,800 seconds in the example configuration).
The semantic extraction budget is 240 seconds per document and its ingestion lease is
1,800 seconds, leaving room for the separately bounded embedding stage. Semantic extraction
uses `reasoning_effort=none` so its bounded completion budget is reserved for validated JSON.
Repeating bootstrap reapplies idempotent
migrations and reuses verified process identities instead of creating duplicates.
The worker claims pending work in PostgreSQL and turns URL or PDF sources into
immutable document revisions.

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

Chat and embedding providers are optional. Set `NORVII_CHAT_BASE_URL` together with
the corresponding local credential only when model-backed chat is needed. Set the
`NORVII_EMBEDDING_*` values before publishing sources that need vector retrieval; a
blank embedding key reuses the chat key. Keep provider credentials only in the
ignored `infra/.env` file. An empty chat URL fails chat requests closed while leaving
the catalog and document reader available. Automated checks use deterministic fakes
and never require provider credentials.

Validate the rendered service list:

```bash
make persistence-config
```

The output must contain only `postgres` and `neo4j`.

## MCP container profile

Feature 010 adds an MCP service. The managed `make bootstrap` and `make local-start`
workflows start it after persistence migration and verification. The persistence-only
commands do not start it; use the explicit command below when needed:

```bash
docker compose --env-file infra/.env -f infra/compose.yaml --profile mcp up --build --wait
```

The service offers Streamable HTTP at `http://127.0.0.1:8091/mcp`. Compose publishes
the port only to the local host while the service remains available to the Norvii
Docker network. Do not publish it remotely without adding authentication, TLS
termination, and Origin validation.

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

Initialize pgvector plus the corpus-ingestion schema and inspect the tern migration
ledger:

```bash
make persistence-migrate
make persistence-migration-status
```

Initialization is idempotent. Migration `001_enable_vector.sql` enables pgvector;
`002_corpus_ingestion.sql` creates corpora, sources, origins, work leases, attempts,
immutable revisions, documents, and addressable units. It also inserts exactly one
English GDPR corpus and one Portuguese LGPD corpus with official URL sources. Migration
`011_normative_assertions.sql` replaces the legacy direct semantic-relationship storage with
canonical `normative_assertions` and graph-release memberships for legal units and assertions.
Each published assertion has atomic entity endpoints plus exact establishing and evidence legal
units.

The ingestion worker stages and activates immutable active releases automatically after its
embeddings and semantic graph artifacts are ready. The historical command remains available only
for recovery of an existing canonical database:

```bash
make persistence-initialize-snapshots
```

The command is idempotent. Normal ingestion does not require it: a reingestion activates its
candidate only after the matching graph release is ready.

`make bootstrap` relies on this automatic ingestion lifecycle. A successful bootstrap therefore
leaves Vector, Graph, and Hybrid retrieval available for both curated corpora. The standalone
command below remains useful when rebuilding a specific historical snapshot.

To rebuild the derived GraphRAG projection for a historical snapshot, use the corpus and snapshot
identifiers shown in the snapshot history:

```bash
python infra/scripts/run-with-environment.py infra/.env \
  uv run --directory apps/ingestion norvii-build-graph-release \
  --corpus-id <corpus-id> --snapshot-id <snapshot-id>
```

The command reads canonical semantic extraction records from PostgreSQL and rebuilds one immutable
Neo4j release. It never changes the active snapshot. Normal ingestion automatically creates and
activates its graph-ready release; this command is a recovery and reproducibility operation.

Verify Go and Python through their production database drivers:

```bash
make persistence-verify
```

Each runtime authenticates and executes bounded read-only SQL and Cypher operations
within the configured timeout. The migration step, not verification, owns schema and
seed changes.

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

## Product inspection and troubleshooting

Open `http://127.0.0.1:5173` after bootstrap. The API health endpoint is
`http://127.0.0.1:8080/healthz` with default configuration. Use the catalog UI to
create or edit corpora, add HTTPS or PDF sources, observe lifecycle status, retry failures, and
browse ready documents. A successful ingestion activates a graph-ready snapshot, so graph and
hybrid retrieval become available without an extra operator command.

Inspect canonical records with read-only commands:

```bash
docker compose --env-file infra/.env -f infra/compose.yaml exec postgres \
  psql --username norvii --dbname norvii
```

Useful tables include `corpora`, `sources`, `url_origins`, `pdf_origins`,
`ingestion_work`, `ingestion_attempts`, `source_revisions`, `documents`,
`document_units`, `corpus_snapshots`, `corpus_snapshot_documents`,
`corpus_snapshot_releases`, `semantic_extraction_runs`, `semantic_entities`,
`normative_assertions`, `graph_releases`, `graph_release_legal_units`, and
`graph_release_assertions`. Neo4j contains the rebuildable assertion projection after successful
ingestion; inspect it at `http://127.0.0.1:7474` or with `cypher-shell` inside the container.

For a failed component, inspect its dedicated `.log/<component>.log` file first.
API errors include a request identifier but exclude credentials and document bodies.
Worker failures expose a bounded category and retain prior ready revisions. A failed
or expired attempt can be retried from the workspace; expired leases are recovered
by the worker.

The API and agent health endpoints verify that their local processes are ready to
serve; they do not call an external chat or embedding provider and therefore do not
validate provider credentials. For a provider configuration or availability failure,
run `make local-status` and review only a bounded tail of `.log/agent.log`. Agent
terminal diagnostics contain bounded outcome, evidence-count, and duration fields;
they do not log prompts, evidence text, provider payloads, or credentials. Review
logs before sharing them and never paste `infra/.env` into tickets or chat. Graph
planning logs its validated response (`use_graph`, `decision_reason`, bounded assertion
predicates, entity labels, and any scope locator). When it selects graph retrieval, it also logs
the fixed Cypher template, parameter values, and the number of returned evidence locations. It
never logs the research question, evidence text, provider payload, or credentials.

## Isolated integration journey

Run the complete service-backed acceptance journey with:

```bash
make persistence-integration
```

The journey uses project `norvii-integration`, alternate local ports, temporary
credentials, and separately named integration volumes. It proves authenticated
health, repeatable migration, Go and Python connectivity, corpus isolation, URL and
PDF publication, work lease recovery, atomic publication, safe failure, normal
restart retention, graph-volume isolation, and clean-volume reproduction. A trap
removes the isolated containers and volumes; it never targets default local data.

## Intentional local reset

Feature 011 has one supported destructive local-corpus reset. The following command runs its
preflight (persistence startup, pending migrations, ingestion tests, and agent tests) and then
permanently removes exactly `norvii_postgres_data` and `norvii_neo4j_data` after validating their
Compose ownership:

```bash
make persistence-reset CONFIRM=reset-norvii-data
```

The command accepts no volume name or filesystem path. It refuses a failed preflight, missing or
unexpected volumes, a changed project identity, invalid ownership labels, and any other
confirmation. It stops managed services before deleting the two verified volumes. Removed data
cannot be recovered through Norvii, and the operation does not contact or delete external sources.

After a successful reset, rebuild the complete local environment and ingest the configured seed
sources again:

```bash
make bootstrap
```

Before asking graph questions, wait for the new source snapshots to reach `ready`. A fresh graph
result must expose an assertion ID, predicate, atomic subject and object labels, establishing and
evidence locators, and any hierarchy context. Until a new graph-ready snapshot is active, the
corpus is empty and graph retrieval must not return stale evidence.

Feature-specific executable evidence is recorded in the
[Feature 004 quickstart](../../specs/004-corpus-catalog/quickstart.md).
