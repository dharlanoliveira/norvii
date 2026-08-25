# Feature Specification: Bilingual Corpus Snapshots

**Feature Branch**: `007-corpus-snapshots`

**Created**: 2026-08-24

**Status**: Implemented

**Input**: User description: "Create Feature 007: reproducible Portuguese and English legal corpus snapshots. A new ingestion creates an immutable source revision and its artifacts without changing the active snapshot. A contributor can explicitly publish a validated revision as a new snapshot, preserve prior snapshots, choose the active snapshot for each corpus, and demonstrate that chat retrieval never mixes languages, corpora, or snapshot versions."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Research the curated bilingual corpora (Priority: P1)

A researcher chooses either the curated Portuguese corpus or the curated English corpus and
asks a question. The answer and every cited passage come only from the published snapshot
of the selected corpus.

**Why this priority**: Two small, trustworthy corpora make the bilingual RAG demonstration
repeatable and show that evidence boundaries are enforced rather than implied by the interface.

**Independent Test**: Open each curated corpus, ask a seeded question, and verify that the
answer cites only evidence belonging to that corpus's active snapshot.

**Acceptance Scenarios**:

1. **Given** the POC has been initialized, **When** a researcher opens the catalog,
   **Then** exactly one curated Portuguese corpus and one curated English corpus are available
   with their language and published snapshot identity visible.
2. **Given** a researcher has selected either curated corpus, **When** they ask a question
   supported by its published evidence, **Then** the answer and all citations resolve only to
   that corpus and its active snapshot.
3. **Given** an equivalent question is asked in the other curated corpus, **When** its
   evidence differs, **Then** the answer does not reuse a source, retrieval artifact, or
   citation from the first corpus.

---

### User Story 2 - Reingest without changing current research (Priority: P2)

A corpus maintainer requests a new ingestion of an official source. The current published
snapshot remains available for research while the candidate revision is captured, processed,
and validated.

**Why this priority**: Official legal material changes. A reingestion must refresh evidence
without silently changing answers that a researcher can still inspect or reproduce.

**Independent Test**: Start a reingestion for one source, ask a seeded question before the
candidate is published, and verify that retrieval continues to use the preceding active
snapshot.

**Acceptance Scenarios**:

1. **Given** a corpus has an active snapshot, **When** a maintainer reingests one of its
   sources, **Then** the system preserves the active snapshot unchanged and records a distinct
   candidate revision.
2. **Given** the candidate revision is incomplete or fails validation, **When** processing
   ends, **Then** it is not published and the preceding active snapshot remains usable.
3. **Given** the candidate revision is identical to the revision already represented by the
   active snapshot, **When** processing completes, **Then** the system does not create a
   duplicate published snapshot.

---

### User Story 3 - Publish and reproduce a validated snapshot (Priority: P3)

A corpus maintainer explicitly publishes a validated candidate as a new named snapshot and
can inspect the earlier snapshot that it superseded. An evaluator can rebuild either curated
snapshot and obtain the same set of source revisions and evidence locations.

**Why this priority**: Explicit publication separates acquisition from release and makes a
POC demonstration, regression investigation, and later evaluation reproducible.

**Independent Test**: Publish a changed candidate, compare its identity with the preceding
snapshot, then rebuild both snapshots and verify that each resolves to its original set of
immutable source revisions.

**Acceptance Scenarios**:

1. **Given** a candidate has completed validation, **When** a maintainer publishes it,
   **Then** the system creates a new snapshot, makes it active only after publication succeeds,
   and retains the previously active snapshot.
2. **Given** a snapshot is active, **When** an evaluator inspects it, **Then** they can see its
   identity, creation time, source revisions, official origins, capture times, and content
   identities.
3. **Given** a historical snapshot is selected for reproduction, **When** it is rebuilt in a
   clean environment, **Then** it resolves to the same source revisions and addressable legal
   locations as its recorded manifest.

### Edge Cases

- One curated source is temporarily unreachable during a rebuild.
- The official source changes while it is being captured.
- A candidate has valid text but incomplete retrieval artifacts.
- A maintainer tries to publish a candidate that belongs to another corpus.
- A request refers to a deleted, malformed, or unavailable historical snapshot identity.
- One corpus is reingested while a researcher has an answer or cited source open from its
  preceding snapshot.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide exactly two initial curated legal corpora: one Portuguese
  corpus and one English corpus, each with a documented official origin and an initial
  published snapshot.
- **FR-002**: The system MUST give every snapshot a stable identity and record the corpus,
  creation time, source revisions, official origins, capture times, and content identities that
  define it.
- **FR-003**: The system MUST enforce an active snapshot boundary for every research request so
  retrieval, generated answers, citations, and inspection data belong to one selected corpus and
  one selected snapshot.
- **FR-004**: The system MUST preserve an answer's evidence references to the immutable snapshot
  used when the answer was generated, even after a later snapshot becomes active.
- **FR-005**: The system MUST create a distinct candidate revision when a maintainer reingests an
  official source and MUST NOT modify the active snapshot during acquisition, processing, or
  validation.
- **FR-006**: The system MUST permit publication only for a candidate whose complete source,
  document, and retrieval evidence are valid and traceable to the same corpus.
- **FR-007**: The system MUST require an explicit maintainer publication action before a valid
  candidate becomes the active snapshot.
- **FR-008**: The system MUST retain prior published snapshots and make their manifest and
  addressable legal locations available for inspection and reproduction.
- **FR-009**: The system MUST prevent duplicate publication when a candidate represents the same
  complete source-revision set and content identities as an existing snapshot for that corpus.
- **FR-010**: The system MUST report a clear publication or validation failure without changing
  the active snapshot.
- **FR-011**: The catalog and corpus workspace MUST show the active snapshot identity without
  obscuring the corpus language, jurisdiction, source library, chat, or cited evidence workflow.
- **FR-012**: The system MUST provide a reproducible initialization path that creates the two
  curated corpora and their initial snapshots without duplicating their identities on repeated
  execution.

### Key Entities *(include if feature involves data)*

- **Corpus snapshot**: A named, immutable release of the complete evidence set available for
  research within one corpus.
- **Snapshot manifest**: The recorded identity and membership of a snapshot, including its source
  revisions and provenance needed to reproduce it.
- **Candidate revision**: A newly captured and processed source revision that is not yet part of
  an active snapshot.
- **Publication**: The explicit maintainer decision that promotes a validated candidate set into
  a new active snapshot while preserving prior snapshots.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A clean local initialization produces exactly two curated corpora and two active
  snapshots, one for each required language, in 100% of five consecutive runs.
- **SC-002**: In the seeded bilingual research scenarios, 100% of answers and citations resolve
  only to the selected corpus and its active snapshot.
- **SC-003**: In deterministic reingestion tests, 100% of failed or incomplete candidate
  revisions leave the preceding active snapshot unchanged and available for research.
- **SC-004**: In deterministic publication tests, 100% of valid changed candidates create one
  new snapshot while unchanged candidates create no duplicate snapshot.
- **SC-005**: An evaluator can inspect and reproduce the source-revision membership and legal
  locations of either initial snapshot using only its recorded identity and the project
  instructions.

## Assumptions

- The two initial corpora remain small, curated legal collections suitable for a cost-bounded
  POC.
- A maintainer is an authenticated or locally authorized contributor; end-user authorization
  design is outside this feature.
- A snapshot represents published retrieval evidence, not a translation of legal text; official
  source language and preserved legal content remain unchanged.
- Only one snapshot is active for each corpus at a time. Historical snapshot exploration may be
  exposed for evaluation without changing the normal researcher workflow.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: Initial Portuguese and English curated corpora; immutable snapshot manifests;
  candidate reingestion; explicit validation and publication; active-snapshot isolation for chat,
  retrieval, citations, and inspection; preservation and reproduction of historical snapshots;
  and the minimal catalog and workspace visibility needed to understand the active release.
- **Out of scope**: Graph-based retrieval, semantic extraction beyond the artifacts required for
  grounded retrieval, legal change summaries or diff views, scheduled refreshes, arbitrary
  end-user snapshot creation, cross-corpus search, and evaluation scoring dashboards.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: Feature 001's corpus catalog and workspace journeys, promoted through
  Feature 002 and subsequently connected to authoritative data by Feature 004.
- **Intentional differences**: The production catalog and workspace gain compact active-snapshot
  identity and maintainer publication feedback. The existing research, source-reading, and chat
  layout remain the baseline.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: Every research request is constrained to exactly one selected
  corpus and its active snapshot. Evidence from the other language, another corpus, or a
  superseded snapshot cannot be included.
- **Evidence behavior**: Every material answer claim remains cited to immutable evidence in the
  selected snapshot. The system abstains when that snapshot lacks sufficient evidence and never
  silently substitutes evidence from another snapshot or corpus.
