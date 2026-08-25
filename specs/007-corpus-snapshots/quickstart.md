# Quickstart: Bilingual Corpus Snapshots

## Prerequisites

- Docker and Docker Compose
- Go, Python/uv, and Node versions pinned by the repository modules
- An OpenAI-compatible embedding configuration for initial ingestion

## Initialize the reproducible POC

1. Run `make bootstrap`. It starts the persistence services and local modules, applies all
   migrations, completes initial ingestion when required, and initializes releases only after
   both initial sources have ready retrieval artifacts.
2. Run `make local-status`. Confirm the report includes `Initial sources: en=ready, pt=ready`
   and `Snapshots: ready`.
3. Run `make persistence-initialize-snapshots` twice. Both calls must report the same active
   snapshot IDs with `created: false`; the command does not create additional releases.

## Verify isolation

1. List the catalog and record the active snapshot IDs for both corpora.
2. Ask an English GDPR question in the English corpus and a Portuguese LGPD question in the
   Portuguese corpus.
3. Confirm every returned citation carries the active snapshot ID for the selected corpus.
4. Confirm no returned document, source revision, or citation identity belongs to the other
   corpus.

## Verify a candidate and publication

1. Reprocess one official source and wait for a new ready document version.
2. Ask the original question before publishing. The answer must still reference the preceding
   active snapshot.
3. Request publication for the new candidate with the observed release version.
4. Confirm a new immutable snapshot becomes active and the previous snapshot remains available
   through the history endpoint.
5. Re-submit the same candidate. Confirm the endpoint returns the existing snapshot without
   creating a duplicate.

## Failure check

Attempt to publish a document with missing units, missing chunks, or a non-ready embedding. The
request must return `publication_failed`, and the catalog's active snapshot ID must remain
unchanged.

## Local verification record

Verified on 2026-08-24 with:

```text
make -C apps/api ci
make -C apps/agent ci
make -C apps/ingestion ci
make -C apps/web ci
python infra/scripts/run-with-environment.py infra/.env go -C apps/api test -tags integration ./tests/integration
python infra/scripts/run-with-environment.py infra/.env make -C apps/agent test-integration
python infra/scripts/run-with-environment.py infra/.env make -C apps/ingestion test-integration
make persistence-verify
make persistence-initialize-snapshots
make persistence-initialize-snapshots
python .github/scripts/validate_repository_language.py
```

All commands passed. The API integration suite verifies a changed candidate creates a second
active release, the preceding manifest retains its original document, and publishing the same
candidate again reuses the active release without creating a duplicate.
