# Quickstart: Evidence-Backed Normative Assertions

## Purpose

Validate the new assertion model with controlled data before removing local corpus data, then reset and reingest the configured sources from zero.

## Prerequisites

- Docker Engine and Docker Compose are available.
- `infra/.env` contains valid local service credentials and any provider configuration required for real ingestion.
- The Feature 011 migrations and implementation checks have passed.

## 1. Run the reset preflight without provider calls

Run the supported preflight:

```bash
make persistence-assertion-preflight
```

It starts the local persistence services, applies pending migrations, then runs the ingestion and
agent test targets. The controlled fixtures must prove that an article and an item can establish
distinct assertions, hierarchy-scoped retrieval excludes siblings, and incomplete provenance is
rejected.

Expected outcome: the test suite reports no provider call, every assertion has two endpoint entities plus establishing and evidence units, and the stream inspection reports the same locations as the citation.

## 2. Reset local corpus data

Run the documented local reset command. It runs the same preflight again, stops managed local
services, and removes only the verified PostgreSQL and Neo4j persistence volumes.

```bash
make persistence-reset CONFIRM=reset-norvii-data
```

Expected outcome: catalog and retrieval show an empty corpus state, no graph release is ready, and no stale citation or graph assertion remains queryable.

## 3. Recreate the local environment and ingest sources anew

Run the managed bootstrap:

```bash
make bootstrap
```

It reapplies migrations, registers the configured seed sources, and starts the normal ingestion
workflow. To ingest other sources, register them through the catalog after bootstrap. Wait for the
worker to normalize documents, extract assertions, publish a snapshot, build its assertion graph
release, and activate that snapshot.

Expected outcome: the active snapshot has a ready graph release whose every graph result exposes an assertion ID, predicate, subject/object labels, establishing locator, evidence locator, and minimal hierarchy context.

## 4. Validate scoped retrieval

Ask one provision-level question and one chapter-scoped relationship question. Inspect the returned assertion path.

Expected outcome: the provision question cites its direct establishing location. The chapter question returns only matching descendant assertions and does not include unrelated chapters or sibling provisions.
