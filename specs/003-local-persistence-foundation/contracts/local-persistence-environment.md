# Contract: Local Persistence Environment

**Contract version**: 1

**Owners**: `infra/`, `apps/api/`, and `apps/ingestion/`

This contract defines the environment inputs and observable command behavior shared
by Norvii's local persistence foundation. It is not a production deployment or
public application API contract.

## Environment variables

| Variable | Consumer | Required | Validation | Sensitive |
| --- | --- | --- | --- | --- |
| `NORVII_POSTGRES_HOST` | Go, Python | Yes | Nonblank hostname; quickstart uses `localhost`. | No |
| `NORVII_POSTGRES_PORT` | Compose, Go, Python | Yes | Integer from 1 through 65535. | No |
| `NORVII_POSTGRES_DATABASE` | Compose, Go, Python | Yes | Nonblank database identifier. | No |
| `NORVII_POSTGRES_USER` | Compose, Go, Python | Yes | Nonblank role identifier. | No |
| `NORVII_POSTGRES_PASSWORD` | Compose, Go, Python | Yes | Nonblank runtime secret. | Yes |
| `NORVII_NEO4J_HTTP_PORT` | Compose | Yes | Integer from 1 through 65535. | No |
| `NORVII_NEO4J_BOLT_PORT` | Compose | Yes | Integer from 1 through 65535. | No |
| `NORVII_NEO4J_URI` | Go, Python | Yes | `neo4j://` URI for the local Bolt endpoint. | No |
| `NORVII_NEO4J_USER` | Compose, Go, Python | Yes | Must be `neo4j` for initial local initialization. | No |
| `NORVII_NEO4J_PASSWORD` | Compose, Go, Python | Yes | Nonblank runtime secret accepted by Neo4j. | Yes |
| `NORVII_NEO4J_DATABASE` | Go, Python | Yes | Nonblank; quickstart uses `neo4j`. | No |
| `NORVII_PERSISTENCE_TIMEOUT_SECONDS` | Go, Python | Yes | Integer from 1 through 10. | No |

The versioned example MUST use conspicuous replacement markers for both passwords.
It MUST NOT contain values that a contributor could use unchanged. Actual values are
stored in `infra/.env`, which remains ignored by Git.

## Configuration invariants

1. Consumers validate all of their required fields before opening a connection.
2. Consumers construct driver configuration from discrete fields and do not require a
   credential-bearing URL.
3. Missing or invalid fields produce a nonzero command result before any verification
   query runs.
4. Error messages may name the variable, service, operation, and safe error class.
5. Error messages and logs MUST NOT contain either password or a connection string
   with embedded credentials.
6. Go and Python enforce the configured timeout independently.
7. Compose health checks use the same database identities as native clients.

## Maintained commands

The repository root exposes these stable operator commands. Implementations may
delegate to module Makefiles and scripts but MUST preserve the behavior below.

| Command | Success behavior | Failure behavior |
| --- | --- | --- |
| `make persistence` | Starts both stores, applies pending migrations, then verifies Go and Python connectivity in order. | Stops at the first failed step and preserves its diagnostic output. |
| `make persistence-up` | Starts exactly PostgreSQL and Neo4j, waits for authenticated health, then returns zero. | Returns nonzero and names any unhealthy service. |
| `make persistence-health` | Reports both services healthy and returns zero. | Returns nonzero with a service-scoped diagnostic command. |
| `make persistence-migrate` | Applies pending migrations and reports the current version; repeat execution is safe. | Returns nonzero and identifies the failed migration or prerequisite. |
| `make persistence-verify` | Runs Go and Python checks; both stores must pass for both modules. | Returns nonzero and preserves module and service attribution. |
| `make persistence-stop` | Stops the Compose project without deleting named volumes. | Returns nonzero without attempting data deletion. |
| `make persistence-reset CONFIRM=reset-norvii-data` | Validates project ownership, removes exactly the two named data volumes, and reports that local data is unrecoverable. | Refuses absent or incorrect confirmation, missing or unexpected ownership, or an ambiguous target. |

## Module command contract

Both module verifiers:

- accept configuration only through the variables defined above;
- authenticate and execute `SELECT 1` against PostgreSQL;
- authenticate and execute `RETURN 1` against Neo4j;
- close all clients on success and failure;
- create no table, row, node, relationship, corpus, source, document, or artifact;
- return zero only when both checks succeed;
- return nonzero within the configured timeout when configuration, authentication, or
  connectivity fails;
- write concise English diagnostics without secrets or stored content.

Example success output is semantic, not byte-for-byte normative:

```text
PostgreSQL connectivity verified.
Neo4j connectivity verified.
Persistence verification succeeded.
```

## Compatibility

Adding an optional variable is backward-compatible within contract version 1.
Renaming a variable, changing validation semantics, changing reset confirmation, or
removing a maintained command requires a contract version change and coordinated Go,
Python, Compose, CI, and documentation updates.
