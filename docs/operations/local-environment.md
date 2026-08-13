# Local Environment

Norvii will use Docker Compose for backing services required by local development and
evaluation. The application modules remain runnable with their native Go, Python, and
Node.js toolchains unless a later deployment feature decides otherwise.

## Compose ownership

The canonical file will be `infra/compose.yaml`. The local foundation feature creates it according to [ADR 0005](../decisions/0005-postgresql-and-neo4j-persistence.md). The default environment starts PostgreSQL with pgvector and one standalone Neo4j Community instance. Exact image versions remain feature-owned and must be pinned before the Compose file is committed.

Compose is responsible for infrastructure such as:

- canonical relational, document, binary, and vector persistence in PostgreSQL with pgvector;
- rebuildable graph projection in standalone Neo4j Community;
- queues or object storage only when a feature requires them;
- optional local model services when explicitly selected.

## Service rules

Every service added to Compose MUST include:

- an image pinned to a deliberate version, never `latest`;
- a stable service name and documented purpose;
- a health check used by dependent services;
- named volumes only for data that must survive restart;
- ports exposed only when a local process or operator needs them;
- resource and data-size expectations appropriate for the POC;
- environment variables documented in `.env.example` without secrets;
- a clean initialization or migration path;
- a verification command in the owning feature's `quickstart.md`.

## Profiles

Use Compose profiles only when they express real optional capabilities, such as local observability or local model services. PostgreSQL and Neo4j are both part of the default POC persistence environment. The default profile MUST not introduce clustered database modes or unrelated services.

## Persistence implementation gate

[ADR 0005](../decisions/0005-postgresql-and-neo4j-persistence.md) resolves the database topology. The local foundation feature must still research and record deliberate versions, resource limits, credentials, ports, migration tooling, driver versions, and health checks. `infra/compose.yaml`, the initial PostgreSQL migration, environment template, and feature quickstart are delivered together.

PostgreSQL is the canonical service. Neo4j contains only a projection that can be rebuilt from a published PostgreSQL artifact release. Removing the Neo4j volume must not remove source documents or canonical extraction artifacts.

## Expected operator commands

The foundation feature should provide one documented command for each operation:

- start PostgreSQL with pgvector and standalone Neo4j Community;
- inspect health;
- run migrations or initialization;
- stop services without deleting data;
- intentionally reset only named POC data with an explicit warning;
- run all module verification checks.

Exact commands remain feature-owned until the toolchain and backing services exist.
