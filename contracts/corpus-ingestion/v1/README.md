# Corpus Ingestion Contract v1

Owner: Feature 004. The Go API provides the HTTP contract and creates ingestion work. The Python worker consumes work and publishes artifacts. The React client consumes HTTP responses.

- `openapi.json`: public application API.
- `ingestion-work.schema.json`: language-neutral work and publication payload invariants.

Additive optional fields are compatible within v1. Removing fields, changing enum meaning, ownership, lifecycle preconditions, lease semantics, or hash semantics requires a new major contract directory and coordinated migration. Provider and consumer tests must validate both schemas and committed fixtures before release.
