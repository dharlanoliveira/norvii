# Data Model: Local Persistence Foundation

This feature defines operational state only. It intentionally creates no corpus,
source, document, embedding, semantic artifact, or graph domain record.

## Local Persistence Environment

Represents one contributor-owned Compose project.

| Field | Type | Rules |
| --- | --- | --- |
| `project_name` | string | Fixed to `norvii`; used to validate resource ownership. |
| `configuration` | Environment Configuration | Must be complete before startup; secrets remain outside version control. |
| `canonical_store` | Canonical Store | Exactly one PostgreSQL service. |
| `graph_store` | Graph Projection Store | Exactly one standalone Neo4j service. |
| `state` | enum | `stopped`, `starting`, `degraded`, `healthy`, or `failed`. |

### State transitions

```text
stopped -> starting -> healthy
                    -> degraded
                    -> failed
healthy -> degraded -> healthy
healthy -> stopped
degraded -> stopped
failed -> stopped
```

`healthy` requires authenticated operations against both stores. `degraded` means one
store is healthy while the other is not. Normal stop preserves both named volumes.

## Environment Configuration

Represents validated runtime inputs shared by Compose, Go, and Python through the
feature contract.

| Field | Type | Rules |
| --- | --- | --- |
| PostgreSQL host | hostname | Required for native clients; never included in a logged credential URL. |
| PostgreSQL port | integer | Required, range 1-65535. |
| PostgreSQL database | identifier | Required, nonblank. |
| PostgreSQL user | identifier | Required, nonblank. |
| PostgreSQL password | secret string | Required for runtime; never logged or committed with a usable value. |
| Neo4j URI | URI | Required; local POC uses `neo4j://` and Bolt. |
| Neo4j user | identifier | Required; initial local user is `neo4j`. |
| Neo4j password | secret string | Required for runtime; never logged or committed with a usable value. |
| Neo4j database | identifier | Required; local Community database is `neo4j`. |
| verification timeout | integer seconds | Required, range 1-10; default documented value is 5. |

## Canonical Store

Represents PostgreSQL with vector capability.

| Field | Type | Rules |
| --- | --- | --- |
| `service_name` | string | Fixed to `postgres`. |
| `image` | OCI reference | Exact release tag and multi-architecture digest. |
| `health` | enum | `starting`, `healthy`, or `unhealthy`; based on authenticated `SELECT 1`. |
| `data_volume` | Named Persistence Area | Fixed project-owned PostgreSQL volume. |
| `migration_version` | integer | `0` before initialization; monotonically increases through tern. |
| `vector_enabled` | boolean | True only when the `vector` extension is installed. |

The canonical store is authoritative. Removing graph state can never mutate this
store or its volume.

## Graph Projection Store

Represents the standalone Neo4j Community projection boundary.

| Field | Type | Rules |
| --- | --- | --- |
| `service_name` | string | Fixed to `neo4j`. |
| `image` | OCI reference | Exact Community release tag and multi-architecture digest. |
| `health` | enum | `starting`, `healthy`, or `unhealthy`; based on authenticated `RETURN 1`. |
| `data_volume` | Named Persistence Area | Fixed project-owned Neo4j volume, distinct from PostgreSQL. |
| `projection_state` | enum | `empty` for this feature; future features may add published versions. |

No graph node, relationship, constraint, or index is created by this feature.

## Migration

Represents one ordered PostgreSQL initialization change.

| Field | Type | Rules |
| --- | --- | --- |
| `version` | positive integer | Unique and strictly ordered. |
| `name` | string | English, stable, and descriptive. |
| `direction` | enum | `up` or `down`, although local reset is the supported destructive recovery path. |
| `status` | enum | `pending`, `applying`, `applied`, or `failed`. |
| `applied_at` | timestamp | Maintained by the migration ledger when applied. |

Initial migration `001_enable_vector` installs the `vector` extension. Reapplying an
already applied migration does not duplicate state.

## Connectivity Check

Represents a bounded, module-owned verification run.

| Field | Type | Rules |
| --- | --- | --- |
| `module` | enum | `api` or `ingestion`. |
| `store_results` | two Store Check Results | Exactly one PostgreSQL result and one Neo4j result. |
| `started_at` | monotonic time | Internal diagnostic only. |
| `timeout_seconds` | integer | Inherited from validated configuration, maximum 10. |
| `outcome` | enum | `succeeded` or `failed`. |

Each store check authenticates and executes a constant read-only query. It creates no
product or projection record. Success output may name stores and duration; failure
output may name the service and safe error category but never credentials, connection
strings, or stored content.

## Named Persistence Area

| Field | Type | Rules |
| --- | --- | --- |
| `logical_name` | string | Exactly PostgreSQL data or Neo4j data. |
| `compose_volume_name` | string | Exact fixed name owned by project `norvii`. |
| `store` | enum | `canonical` or `graph`; never both. |
| `retained_on_stop` | boolean | Always true. |
| `eligible_for_reset` | boolean | True only after exact ownership validation and explicit confirmation. |

The two volumes have no containment or filesystem-path relationship. A reset accepts
no user-supplied volume name or path.
