# Research: Local Persistence Foundation

## Container versions and reproducibility

**Decision**: Use `pgvector/pgvector:0.8.6-pg18-trixie` at OCI index digest
`sha256:1963bc48febf543433baa1ce3edcc6cc08154de722e22495f86681cc9a849026`
and `neo4j:2026.06.0-community-trixie` at OCI index digest
`sha256:42fd5b9ead4dd4211f6f91bd831c358e4e2117367d04633fbf88682ca4792b30`.
Both inspected indexes publish Linux amd64 and arm64 manifests.

**Rationale**: The [pgvector project](https://github.com/pgvector/pgvector) lists the
exact `0.8.6-pg18-trixie` image variant, and the
[Neo4j Docker documentation](https://neo4j.com/docs/operations-manual/current/docker/operations/)
documents the 2026.06.0 line. Keeping the readable immutable release tag beside the
verified multi-architecture digest provides reviewable intent and protects clean
checkouts from a moved tag.

**Alternatives considered**:

- Floating `pg18`, `community`, or `latest` tags were rejected because they cannot
  reproduce a reviewed environment.
- PostgreSQL 17 was viable but rejected because the current exact pgvector image
  supports PostgreSQL 18 and no Norvii compatibility constraint requires the older
  major version.
- Neo4j Enterprise was rejected because clustering, licensed features, and enterprise
  operations are outside the POC.

## Service readiness and authentication

**Decision**: PostgreSQL health executes a credentialed `SELECT 1` with `psql`, while
Neo4j health executes `RETURN 1` with `cypher-shell` over Bolt. Both checks use
container environment values, redact command output, retry for at most two minutes,
and report service health through Compose.

**Rationale**: A port probe cannot prove credentials or query readiness. Neo4j's
[Cypher Shell documentation](https://neo4j.com/docs/operations-manual/current/cypher-shell/)
defines authenticated scripted queries over Bolt, while the
[Neo4j Docker authentication guide](https://neo4j.com/docs/operations-manual/current/docker/introduction/)
defines `NEO4J_AUTH` and notes that an existing data volume retains its prior
credentials. An authenticated operation therefore detects both startup failures and
stale-volume credential mismatches.

**Alternatives considered**:

- `pg_isready` and raw TCP probes were rejected because they do not prove a successful
  authenticated query.
- HTTP-only Neo4j readiness was rejected because production drivers use Bolt.
- Disabling authentication was rejected because it would bypass an important part of
  the runtime contract.

## Local credential contract

**Decision**: Use discrete, `NORVII_`-prefixed environment variables for host, port,
database, user, password, and a shared five-second verification timeout. Commit only
`infra/.env.example`; ignore `infra/.env`; require actual passwords without usable
defaults.

**Rationale**: Discrete fields avoid placing credential-bearing connection strings in
logs, exception text, process listings, or documentation. One feature-local contract
keeps Go, Python, Compose, and CI aligned without making either language module the
implicit source of truth.

**Alternatives considered**:

- Full database URLs were rejected because driver errors commonly echo URLs and
  require error-prone escaping for special characters.
- Hard-coded local passwords were rejected because versioned examples must not
  contain usable secrets.
- Docker secrets were deferred: they are appropriate for deployment, but add
  ceremony without improving the explicitly contributor-local POC boundary.

## PostgreSQL migration ownership and tool

**Decision**: The Go module owns migrations and embeds them in a small `migrate`
composition root using tern 2.4.2 with pgx. Migration `001` enables the `vector`
extension. The command reports the current migration version and is safe to repeat.

**Rationale**: The accepted module model already assigns migrations to `apps/api/`.
[tern](https://github.com/jackc/tern) is PostgreSQL-focused, uses pgx, supports an
inspectable schema-version ledger, and can be embedded so contributors do not need a
separate globally installed CLI. This avoids maintaining a custom migration engine.

**Alternatives considered**:

- A hand-written migration ledger was rejected as undifferentiated infrastructure
  code with subtle locking and partial-failure behavior.
- Running `go run ...@latest` was rejected because it downloads an unpinned tool at
  execution time.
- Python-owned migrations were rejected because the repository's accepted Go module
  boundary already owns online canonical storage migrations.

## Go runtime and drivers

**Decision**: Use Go 1.26.0 as the module language level and the current Go 1.26.5
patch toolchain in CI, with pgx 5.10.0 and Neo4j Go Driver 6.2.0. Adapters accept
typed configuration, construct vendor clients internally, enforce a bounded context,
and expose only a narrow verification operation.

**Rationale**: The [Go release history](https://go.dev/doc/devel/release) identifies
1.26.5 as the current supported patch release. pgx is the PostgreSQL-native driver
already used by tern, minimizing duplicate database stacks. The official Neo4j
driver keeps Bolt semantics inside an infrastructure adapter. Context deadlines make
the ten-second feature criterion enforceable.

**Alternatives considered**:

- `database/sql` plus a second PostgreSQL driver was rejected because tern and direct
  connectivity can share pgx without leaking it into future domain packages.
- Shelling out to database CLIs was rejected because it would not prove production
  driver compatibility.
- A shared Go/Python wrapper was rejected because it would violate independent module
  ownership.

## Python runtime, packaging, and drivers

**Decision**: Use Python 3.13 with a locked uv project, psycopg 3.3.4, Neo4j Python
Driver 6.2.0, pytest 9.1.1, Ruff 0.16.3, and mypy 2.3.0. A cohesive verifier class
coordinates two small adapter classes; typed immutable configuration objects validate
environment input before clients are created.

**Rationale**: Both official drivers support the selected Python runtime. A `src/`
package, strict type checking, and class-based stateful adapters match the project's
ingestion guidance while keeping vendor objects at the publication boundary. uv
provides a reproducible lock and fast CI installation.

**Alternatives considered**:

- Async drivers were rejected because a sequential startup verifier has no concurrent
  workload and would add lifecycle complexity.
- SQLAlchemy was rejected because this feature needs driver connectivity, not an ORM
  or product persistence model.
- Module-level mutable clients were rejected because they complicate deterministic
  tests and resource cleanup.

## Lifecycle orchestration and reset safety

**Decision**: Root Make targets delegate to Compose and module commands for start,
health, migration, verification, stop, and reset. Reset requires an explicit
confirmation token, validates the rendered Compose project name and the two exact
named volumes, rejects symlinks or arbitrary filesystem paths by never accepting a
path, and removes no container resource outside the `norvii` Compose project.

**Rationale**: A stable contributor command satisfies the one-command requirement
without hiding module output. Compose labels and exact volume identities provide a
machine-checkable ownership boundary. Normal shutdown never passes `--volumes`.

**Alternatives considered**:

- An unguarded `docker compose down --volumes` target was rejected because a typo or
  changed project scope could delete an unexpected volume.
- Host bind mounts were rejected because path handling and platform permissions make
  safe reset harder.
- A general cleanup script accepting arbitrary names or paths was rejected because the
  feature owns exactly two data targets.

## Continuous integration journey

**Decision**: Extend module CI with language-specific setup and add one isolated
service-backed job that validates Compose, starts both stores, waits for authenticated
health, applies migrations twice, runs Go and Python checks, verifies restart
persistence and graph-volume isolation, then performs cleanup. Sonar analysis depends
on this job.

**Rationale**: Unit tests cannot prove image, migration, credential, and driver
compatibility together. One bounded journey provides the required cross-layer
evidence while the module matrix preserves independently attributable language
quality failures.

**Alternatives considered**:

- Mock-only CI was rejected because it cannot satisfy the service-backed acceptance
  scenarios.
- Separate full environments per runtime were rejected because they duplicate image
  startup cost without testing a different boundary.
- Leaving persistence out of Sonar dependencies was rejected because the project
  requires any affected build failure to fail the workflow.
