# Data Model: Versioned Evaluation Datasets

## Corpus separation

Each evaluation dataset belongs to exactly one `corpus`. The three corpus records introduced by this feature are independent retrieval roots: `brazil-personal-data-protection`, `brazil-anti-corruption-white-collar-crime`, and `us-fair-housing-disability-accommodations`. Sources, snapshot members, datasets, runs, and actual evidence must have the same corpus ID. Information-security corpus IDs are invalid for every dataset and evaluation relation.

## Immutable dataset catalog

```text
evaluation_dataset_revision
  +-- evaluation_dataset_source
  +-- evaluation_dataset_case
  |     +-- evaluation_case_expected_evidence
  +-- evaluation_dataset_publication
```

### `evaluation_dataset_revision`

One immutable imported manifest and JSONL pair.

- `id` UUID; `dataset_key`; semantic revision; `corpus_id`; jurisdiction.
- Manifest, JSONL, and combined canonical SHA-256 hashes; original project-relative paths.
- Declared snapshot date, query-language set, authoritative-evidence-language set, imported time, importer version, and bounded import diagnostics.
- Cannot be edited after import. Re-import with same combined hash is idempotent; changed content creates a new revision.

### `evaluation_dataset_source`

One manifest source requirement per revision.

- Manifest alias, title, official URL, issuing authority, document type, authority role (`statute`, `regulation`, `guidance`, `procedure`, and future controlled values).
- Explicit `corpus_source_id` binding, created only after reviewer confirmation. The URL is provenance, not a foreign key or runtime matching key.
- Every declared source must belong to the revision's corpus and must be present in a compatible snapshot, including sources not used by the first case set.

### `evaluation_dataset_case`

One immutable JSONL record.

- Dataset revision, ordered position, external case ID, project-normalized query language (`pt` or `en`), original asset language (`pt-BR` or `en`), question, reference answer, category, authoritative source language, reciprocal paired-case external ID, and per-case checksum.
- Add `expected_outcome` with `answer` as the backward-compatible default. `abstain` requires a controlled expected-reason code.
- Paired cases must be different, reciprocal, and use different query languages. They share no generated result and are not independent statistical observations.

### `evaluation_case_expected_evidence`

An ordered required source/locator selector with `required_propositions` retained as review annotations. The v1 policy is `all`: every compound selector is expanded into atomic legal targets before it can be used. Alternative (`any`) targets require a future explicit reviewed schema.

### `evaluation_dataset_publication`

Append-only record of review and availability. It identifies the dataset revision, review decision, reviewer identity, timestamp, bounded note, and publication state (`draft`, `available`, `superseded`, `withdrawn`). Imported data is draft; only `available` data can start a run.

## Locator resolution and compatibility

```text
evaluation run request
  -> dataset publication + source binding
  -> corpus/snapshot membership verification
  -> legal locator alias resolution
  -> immutable evaluation_run_expected_evidence
```

`evaluation_run_expected_evidence` materializes each original selector against the requested snapshot. It stores snapshot ID, source ID, document version ID, source revision ID, legal-unit ID or span, canonical locator, original display locator, and content hash. A missing/ambiguous resolution rejects the run before queueing model work. This prevents source re-extraction or a new active release from rewriting historical meaning.

## Runs and results

```text
evaluation_run
  +-- evaluation_run_case
  |     +-- evaluation_run_actual_evidence  (retrieved | cited)
  |     +-- evaluation_run_metric
  +-- evaluation_run_metric                 (aggregate)
```

### `evaluation_run`

Immutable experiment identity: dataset revision/hashes, corpus ID, snapshot ID/manifest hash, ordered case-set hash, retrieval configuration fingerprint, strategy, agent build, chat/embedding model identity, scoring-policy version, initiator, and timestamps. State transitions only: `queued -> running -> completed | completed_with_failures | failed`.

### `evaluation_run_case`

One ledger row created per dataset case. It has lifecycle `pending -> leased -> completed | abstained | failed | cancelled`, attempt count, lease token/expiry, answer, answer-citation markers, graph grounding state, safe failure code, nullable latency/tokens, verdict, and scoring rationale. Terminal results are immutable. Retrying creates a new attempt record or changes only a non-terminal lease; it never replaces a persisted terminal case.

### `evaluation_run_actual_evidence`

Stores either retrieved or actually cited evidence, with rank/marker position and complete immutable provenance: corpus, snapshot, source, source revision, document version, legal unit, locator, offsets, and evidence hash. It is valid only when all ownership values match the parent run. The database/repository checks snapshot membership as well as the worker check.

### `evaluation_run_metric`

Stores case and aggregate component value, numerator, denominator, eligibility, scorer version, and rationale. Required components are retrieval coverage, citation coverage, citation validity, citation scope validity, expected abstention outcome, execution outcome, latency, and token use. Semantic claim support has value `needs_human_review` until a reviewed claim-to-citation contract exists.

## Invariants

1. Dataset revisions, cases, source requirements, and publications are append-only.
2. A run is created only after all source bindings and locator resolutions succeed for its fixed corpus snapshot; it never resolves the active snapshot during execution.
3. Every actual evidence reference belongs to the run corpus and snapshot; cross-corpus and cross-snapshot references are invalid.
4. Execution failure/cancellation is `not_scored`, never a fabricated zero. Aggregates include `total`, `eligible`, `scored`, `failed`, `cancelled`, and `not_applicable`; telemetry percentiles use only non-null measurements and declare `n`.
5. A direct comparison requires equal dataset revision/hash, snapshot/manifest hash, ordered case-set hash, and scoring-policy version. Configuration differences are visible but allowed.
6. Query language, expected answer language, and authoritative evidence language remain distinct.
