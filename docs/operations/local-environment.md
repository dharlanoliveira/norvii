# Local Environment

Norvii will use Docker Compose for backing services required by local development and
evaluation. The application modules remain runnable with their native Go, Python, and
Node.js toolchains unless a later deployment feature decides otherwise.

## Compose ownership

The canonical file will be `infra/compose.yaml`. It is created by the first feature
that selects a backing service. Until database research is complete, the repository
MUST not contain placeholder images that imply an architectural decision.

Compose is responsible for infrastructure such as:

- relational or document persistence;
- vector indexing when not provided by the primary database;
- graph storage when a dedicated graph database is justified;
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

Use Compose profiles only when they express real optional capabilities, such as a
dedicated graph database or local observability stack. The default profile MUST be
the smallest environment required by the current MVP.

## Database decision gate

The feature that first requires persistence MUST compare at least these concerns:

- binary PDF storage at POC scale;
- structured corpus and source metadata;
- vector search needs and language support;
- graph traversal needs and whether they justify a separate database;
- migrations, backups, health checks, and local resource use;
- Go and Python driver maturity;
- operational complexity versus demonstration value.

The accepted choice is recorded as an ADR. `infra/compose.yaml`, migrations, and the
feature quickstart are delivered together after that decision.

## Expected operator commands

The foundation feature should provide one documented command for each operation:

- start the minimum backing services;
- inspect health;
- run migrations or initialization;
- stop services without deleting data;
- intentionally reset only named POC data with an explicit warning;
- run all module verification checks.

Exact commands remain feature-owned until the toolchain and backing services exist.
