# Quickstart: Local Persistence Foundation

This guide is the executable acceptance journey for Feature 003. Commands are run
from the repository root.

## Prerequisites

- Docker Engine with Docker Compose support
- Go 1.26.5 or a compatible Go 1.26 patch release
- Python 3.13
- uv
- GNU Make and Bash on the reference Linux environment

The container images support Linux amd64 and arm64. Allow up to 2 GiB of memory for
the two stores. First image download time is excluded from readiness targets.

## Configure the local environment

```bash
cp infra/.env.example infra/.env
```

Replace both password markers in `infra/.env` with local values. Do not commit this
file. Variable semantics are defined by the
[local persistence environment contract](contracts/local-persistence-environment.md).

Validate configuration without starting services:

```bash
make persistence-config
```

Expected outcome: Compose renders exactly the `postgres` and `neo4j` services without
printing credentials.

## Start and initialize

```bash
make persistence
```

Expected outcome:

- PostgreSQL and Neo4j become healthy within two minutes after images are present;
- migration version `1` is current;
- PostgreSQL reports the `vector` extension installed;
- the Go API and Python ingestion module verify both stores through their production
  drivers.

Inspect health and migration state independently:

```bash
make persistence-health
make persistence-migration-status
```

If readiness fails, inspect bounded service output:

```bash
docker compose --env-file infra/.env -f infra/compose.yaml ps
docker compose --env-file infra/.env -f infra/compose.yaml logs --tail 100 postgres
docker compose --env-file infra/.env -f infra/compose.yaml logs --tail 100 neo4j
```

Review logs before sharing them because third-party service logs are outside Norvii's
structured logging boundary.

## Verify production runtimes independently

```bash
make persistence-verify
```

Expected outcome: the Go API and Python ingestion module each authenticate to
PostgreSQL and Neo4j, execute bounded read-only checks, and return success within ten
seconds without creating product data.

The module checks also remain independently runnable:

```bash
make -C apps/api verify-persistence
make -C apps/ingestion verify-persistence
```

## Verify repeatability

Run initialization again:

```bash
make persistence-migrate
make persistence-migration-status
```

Expected outcome: no duplicate migration is applied and the current version remains
`1`.

Run the maintained service-backed foundation journey:

```bash
make persistence-integration
```

Expected outcome: authenticated health, repeated migration, driver checks, normal
restart persistence, and graph-volume isolation all pass. Disposable marker data is
removed by the journey.

## Stop without deleting data

```bash
make persistence-stop
```

Restart with `make persistence-up`; migration and marker state created outside the
disposable test journey remain intact.

## Intentionally reset local data

Reset permanently removes only Norvii's PostgreSQL and Neo4j named data volumes. The
data cannot be recovered through this project.

```bash
make persistence-reset CONFIRM=reset-norvii-data
```

The command refuses to proceed without the exact confirmation token or when Compose
ownership and exact volume targets cannot be established. It accepts no filesystem
path or caller-supplied volume name.

Recreate the clean state:

```bash
make persistence
```

## Run affected quality checks

```bash
make -C apps/api ci
make -C apps/ingestion ci
python .github/scripts/validate_repository_language.py
```

Continuous integration additionally runs the service-backed foundation journey and
makes SonarQube Cloud plus failure notification depend on it.

## Recorded implementation evidence

Validation completed on 2026-08-13 on the reference Linux amd64 environment with
Docker Engine 29.1.5, Docker Compose 5.0.1, Go 1.26, and uv 0.11.16:

- a cached-image default startup reached authenticated health in 7.28 seconds;
- aggregate Go and Python runtime verification completed in 0.48 seconds;
- migration version 1 remained current after repeated initialization and normal
  restart;
- the isolated integration journey passed restart retention, graph-volume removal,
  canonical-marker isolation, invalid-credential and unavailable-store failures, and
  clean-volume reproduction;
- the guarded default reset removed only the two validated `norvii` data volumes,
  after which startup, migration, and both runtime verifiers passed from clean state;
- Neo4j logs use bounded tmpfs storage, so the default environment creates no
  anonymous log volume.

Timing values describe this reference run rather than a guarantee for image download,
host scheduling, or other machines. The success criteria remain the normative limits.
