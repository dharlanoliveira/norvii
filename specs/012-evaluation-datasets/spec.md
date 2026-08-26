# Feature Specification: Versioned Evaluation Datasets

**Feature Branch**: `012-evaluation-datasets`

**Created**: 2026-08-25

**Status**: Draft

**Input**: User description: "Internalize the curated golden datasets for the LGPD, Brazilian anti-corruption, and U.S. fair housing corpora as the next Norvii feature."

## User Scenarios & Testing *(mandatory)*

### User Story 0 - Start a corpus-grounded chat with relevant questions (Priority: P1)

A researcher opening an empty conversation sees up to five suggested questions that belong to
the selected corpus's curated dataset, rather than suggestions from another corpus. The questions
match the selected interface language and submit through the ordinary corpus-bound chat flow.

**Why this priority**: The first interaction must demonstrate the active corpus's legal domain
and retrieval capability without encouraging a question that its evidence boundary cannot answer.

**Independent Test**: Open the empty chat for each of the three named corpora in English and
Portuguese, verify that every visible suggestion belongs to that corpus's approved dataset case,
then submit one and verify the normal chat request retains the selected corpus.

**Acceptance Scenarios**:

1. **Given** a corpus with a published opening-suggestion set for its active snapshot, **When** a
   researcher opens an empty chat, **Then** the interface displays exactly the selected questions
   from that corpus's starter-question selection and no question from another corpus.
2. **Given** the interface is English or Portuguese, **When** the researcher opens an empty chat,
   **Then** every suggestion is the corresponding English or Portuguese member of its approved
   paired dataset case.
3. **Given** a researcher selects a suggested question, **When** the chat request starts, **Then**
   it follows the ordinary chat path with the active corpus and does not expose evaluation answers,
   expected evidence, review state, or dataset internals.
4. **Given** a corpus has no published starter-question selection compatible with its active
   snapshot, **When** a researcher opens an empty chat, **Then** no suggestion buttons are shown
   and the interface does not substitute suggestions from LGPD or another corpus.

---

### User Story 1 - Run a corpus-grounded evaluation (Priority: P1)

An evaluator selects a published corpus snapshot and its compatible golden dataset, starts an
evaluation, and receives one result for every dataset case. Each result identifies the expected
answer language, expected evidence locations, generated answer, actual citations, outcome, and
safe failure reason where applicable.

**Why this priority**: A reproducible evaluation proves that a corpus works beyond a few
hand-picked chat demonstrations and exposes whether answers remain grounded in the active
evidence boundary.

**Independent Test**: Use a seeded corpus snapshot and one internalized dataset to produce a
complete result set, then inspect the result for each case without consulting another corpus or
snapshot.

**Acceptance Scenarios**:

1. **Given** a published corpus snapshot with a compatible, validated dataset, **When** an
   evaluator starts an evaluation, **Then** every case is run only against that corpus and
   snapshot and produces an inspectable result or an explicit execution failure.
2. **Given** a case that requires a legal unit, **When** the answer cites evidence, **Then** the
   result shows whether the cited evidence resolves to the expected source location in the
   evaluated snapshot.
3. **Given** a case written in English or Portuguese, **When** it is evaluated, **Then** the
   expected answer language and the authoritative source language remain separately visible.

---

### User Story 2 - Trust dataset provenance and compatibility (Priority: P2)

A corpus maintainer can inspect which version of a curated dataset was used, its source
manifest, review status, supported languages, and required corpus sources before running it.
The maintainer cannot accidentally apply a dataset to another jurisdiction, corpus, or snapshot
that does not contain its required evidence.

**Why this priority**: The golden answer is useful only when its legal evidence and the corpus
being evaluated are reproducible and compatible.

**Independent Test**: Attempt to select a dataset for an incompatible corpus or a snapshot
missing a required source and confirm that no evaluation starts; select the intended snapshot and
confirm that the dataset version and source requirements are visible.

**Acceptance Scenarios**:

1. **Given** an internalized dataset, **When** a maintainer inspects it, **Then** its stable
   identity, revision, jurisdiction, snapshot date, source manifest, language fields, and review
   status are available.
2. **Given** a snapshot lacks a source or location required by a dataset case, **When** an
   evaluator requests an evaluation, **Then** the system rejects the request with the missing
   requirements identified and produces no partial quality score.
3. **Given** a dataset source is non-normative guidance, **When** a case is evaluated, **Then**
   the dataset can require the answer to distinguish guidance from statutory or regulatory text.

---

### User Story 3 - Compare repeatable quality results (Priority: P3)

An evaluator can view the aggregate and case-level results of completed runs for the same dataset
and snapshot, including evidence support, expected-evidence retrieval, citation validity,
abstention correctness, latency, and token use. Runs remain attributable to their retrieval
configuration without presenting them as legal advice.

**Why this priority**: Comparable evidence-backed measurements make it possible to demonstrate
improvement or regression in the Norvii research experience.

**Independent Test**: Complete two runs with distinct recorded retrieval configurations against
the same dataset and snapshot, then compare aggregate metrics and open an individual case from
each run.

**Acceptance Scenarios**:

1. **Given** a completed run, **When** an evaluator opens its summary, **Then** the summary
   states the dataset revision, corpus snapshot, configuration identity, case totals, outcome
   counts, and cost/latency measures.
2. **Given** a completed run, **When** an evaluator opens a case result, **Then** they can see
   the question, reference answer, expected and actual evidence, generated answer, and scoring
   rationale without exposing raw provider prompts or secrets.

### Edge Cases

- A dataset file is malformed, duplicates a case identifier, or contains a broken language pair.
- A dataset refers to a source manifest entry that is absent from the selected snapshot.
- A required legal locator cannot be resolved after source reingestion or source-structure
  changes.
- A model or retrieval operation fails for one case while other cases are still eligible to run.
- A case expects abstention, but the model provides an uncited answer.
- The same dataset revision is imported twice or a later revision changes its expected evidence.
- An evaluator requests a comparison of runs whose datasets or snapshots differ.
- The selected interface language has no matching starter-case member, a starter-case pair is
  incomplete, or the selected corpus has no published starter-question set for its active
  snapshot.
- A researcher changes corpus while an empty chat is visible; the former corpus's suggestions
  must not persist into the new workspace.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST internalize the curated LGPD, Brazilian anti-corruption, and U.S.
  fair-housing dataset manifests and cases as versioned project-owned evaluation assets.
- **FR-002**: The system MUST preserve every dataset case's stable identifier, query language,
  expected answer language, reference answer, expected evidence, category, paired-case identity,
  authoritative source language, and dataset revision.
- **FR-003**: The system MUST validate dataset syntax, unique case identifiers, reciprocal
  language pairs, source-manifest references, and non-empty required evidence before a dataset
  becomes available for evaluation.
- **FR-004**: The system MUST associate every available dataset with its intended jurisdiction,
  corpus identity, source requirements, and snapshot date, and MUST reject incompatible corpus
  snapshots before a run starts.
- **FR-005**: The system MUST run each dataset case using only the selected corpus snapshot and
  record the snapshot, dataset revision, retrieval configuration, model identity, latency, token
  use, answer, citations, and execution state.
- **FR-006**: The system MUST evaluate whether every material answer claim is supported by
  citations in the selected snapshot, whether expected evidence is retrieved or cited as required
  by the case, and whether abstention occurred when the case requires it.
- **FR-007**: The system MUST keep the reference answer and required evidence available for
  case-level inspection and MUST distinguish authoritative normative evidence from guidance.
- **FR-008**: The system MUST expose completed-run aggregate metrics for evidence support,
  expected-evidence retrieval, citation validity, correct abstention, latency, and token use,
  together with case counts and safe execution-failure counts.
- **FR-009**: The system MUST prevent direct comparison claims between runs that use different
  dataset revisions or corpus snapshots, while still making their identities visible.
- **FR-010**: The system MUST ensure that a failed case does not fabricate a score and does not
  prevent unrelated eligible cases in the same run from reaching a terminal state.
- **FR-011**: The system MUST retain evaluation results and their evidence identities after a
  later corpus snapshot or dataset revision is published.
- **FR-012**: The system MUST present the evaluation as a technical measurement of corpus-
  grounded behavior and not as legal advice or a legal conclusion about a real person or entity.
- **FR-013**: The system MUST create and maintain the following three distinct legal corpora,
  each with its own stable identity, sources, snapshots, access boundary, and evaluation dataset:
  `Brazilian Personal Data Protection (LGPD)`, `Brazilian Anti-Corruption and White-Collar
  Crime`, and `United States Fair Housing and Disability Accommodations`. No source, snapshot,
  dataset, result, or retrieval operation from these corpora may be placed in or combined with an
  information-security corpus.
- **FR-014**: The system MUST allow a reviewed, available dataset revision to identify an
  explicit, immutable, rank-ordered selection of no more than five reciprocal English and
  Portuguese case pairs for its one intended corpus. Import MUST reject duplicate, asymmetric,
  unreviewed, unsafe, or out-of-range selected pairs. Each selected case MUST be an exact
  project-owned dataset question and MUST preserve its stable case identifier and checksum.
- **FR-015**: The system MUST publish selected starter questions as an immutable, corpus-bound
  projection of one reviewed dataset revision and one compatible corpus snapshot. Each projection
  item MUST retain case ID, case checksum, language, rank, and original question text, while
  excluding reference answers, expected evidence, review notes, and evaluation outcomes.
- **FR-016**: The system MUST return starter questions only when the projection's corpus,
  snapshot, and snapshot-manifest identities match the requested corpus's active release. It MUST
  return no suggestion for another corpus, a draft/unavailable dataset, a disabled or missing
  release, or a stale or incompatible projection.
- **FR-017**: The system MUST expose only the active corpus's rank-ordered starter questions in
  the corpus workspace, choosing the paired case that matches the interface language verbatim. It
  MUST NOT translate, synthesize, randomize, or use a cross-corpus or hard-coded fallback
  question.
- **FR-018**: The system MUST submit a selected starter question through the existing ordinary
  corpus-bound chat request. Runtime chat MUST consume only the versioned suggestion read
  contract; it MUST NOT read evaluation cases or results, invoke evaluation preflight or scoring,
  or alter public chat streaming.

### Key Entities *(include if feature involves data)*

- **Evaluation dataset**: A versioned, jurisdiction-bound collection of curated evaluation cases
  and its source manifest.
- **Evaluation case**: One question, reference answer, language expectations, expected evidence,
  category, and optional paired case.
- **Dataset source requirement**: A source identity and legal locator a compatible corpus snapshot
  must provide for a dataset case.
- **Evaluation run**: An immutable attempt to run a dataset revision against one corpus snapshot
  and one recorded retrieval configuration.
- **Case result**: The case-level answer, retrieved evidence, citations, scores, metrics, and
  terminal execution outcome belonging to an evaluation run.
- **Evaluation metric**: A named aggregate or case-level measure derived from a completed run.
- **Legal corpus**: An isolated, named jurisdictional collection of sources, revisions, snapshots,
  and compatible evaluation datasets.
- **Starter question selection**: The immutable, rank-ordered paired curated dataset cases
  eligible for a one-corpus, one-snapshot opening-suggestion projection.
- **Opening-suggestion projection**: The published, read-only corpus and active-snapshot-bound
  chat view containing only selected case IDs, checksums, ranks, languages, and question text.

### Curated Starter Questions

The following paired dataset cases are the required initial selections. The English question text
is reproduced here for review; the exact Portuguese counterpart is preserved in the referenced
project-owned dataset asset. A selected case remains traceable by its case ID, while the chat
surface exposes only the matching question text and ID.

| Corpus | Portuguese case ID | English case and question |
| --- | --- | --- |
| Brazilian Personal Data Protection (LGPD) | `lgpd-002-pt` | `lgpd-002-en` - What is the difference between a controller and an operator under the LGPD? |
| Brazilian Personal Data Protection (LGPD) | `lgpd-003-pt` | `lgpd-003-en` - What do the purpose and necessity principles require when processing personal data? |
| Brazilian Personal Data Protection (LGPD) | `lgpd-004-pt` | `lgpd-004-en` - Can a data subject request confirmation that processing exists and access to their data? |
| Brazilian Personal Data Protection (LGPD) | `lgpd-005-pt` | `lgpd-005-en` - What is the deadline for providing a clear and complete statement in response to a confirmation or access request? |
| Brazilian Personal Data Protection (LGPD) | `lgpd-007-pt` | `lgpd-007-en` - When must a security incident be reported to the ANPD and the data subject? |
| Brazilian Anti-Corruption and White-Collar Crime | `brac-001-pt` | `brac-001-en` - What type of liability does Brazilian Law 12,846/2013 establish for legal entities? |
| Brazilian Anti-Corruption and White-Collar Crime | `brac-002-pt` | `brac-002-en` - Can a company that indirectly offers an undue advantage to a public official commit a harmful act under the Brazilian Anti-Corruption Law? |
| Brazilian Anti-Corruption and White-Collar Crime | `brac-003-pt` | `brac-003-en` - Which administrative sanctions does Article 6 of Law 12,846/2013 provide for legal entities? |
| Brazilian Anti-Corruption and White-Collar Crime | `brac-005-pt` | `brac-005-en` - How does the Brazilian Penal Code define active corruption? |
| Brazilian Anti-Corruption and White-Collar Crime | `brac-007-pt` | `brac-007-en` - What general anti-money-laundering duties apply to persons subject to Brazilian Law 9,613/1998? |
| United States Fair Housing and Disability Accommodations | `fh-002-pt` | `fh-002-en` - Does the Fair Housing Act address discrimination against a buyer or renter because of disability? |
| United States Fair Housing and Disability Accommodations | `fh-003-pt` | `fh-003-en` - What is a reasonable accommodation under the Fair Housing Act? |
| United States Fair Housing and Disability Accommodations | `fh-004-pt` | `fh-004-en` - How does a reasonable modification differ from a reasonable accommodation in existing housing? |
| United States Fair Housing and Disability Accommodations | `fh-005-pt` | `fh-005-en` - Can an assistance animal be treated as an ordinary pet for a housing accommodation request? |
| United States Fair Housing and Disability Accommodations | `fh-007-pt` | `fh-007-en` - What information does HUD ask a person to provide when reporting housing discrimination? |

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The three curated datasets import deterministically, with 100% of their cases,
  source requirements, and reciprocal language pairs preserved across five clean initializations.
- **SC-002**: The system rejects 100% of evaluation requests whose selected snapshot lacks a
  required dataset source or legal location before generating any answer.
- **SC-003**: In a completed compatible run, 100% of terminal case results retain the exact
  dataset revision, corpus snapshot, and retrieval-configuration identities used.
- **SC-004**: For 100% of completed case results, an evaluator can inspect the reference answer,
  expected evidence, generated answer, actual citations, scoring rationale, latency, and token
  use from the run record.
- **SC-005**: Equivalent English and Portuguese cases remain linked and their language metadata
  is preserved in 100% of imported cases.
- **SC-006**: A comparison view labels 100% of attempts involving different dataset revisions or
  corpus snapshots as non-comparable rather than presenting a direct quality delta.
- **SC-007**: For every published initial opening-suggestion projection, 100% of the five visible
  starter questions per interface language resolve to the required corpus, active snapshot, and
  reciprocal curated case; 0% of starter questions shown in another corpus originate from LGPD or
  any other dataset.

## Assumptions

- The three draft datasets created under `data/corpora/` are the initial owned inputs for this
  feature and receive legal-domain review before being promoted from draft to an available
  evaluation revision.
- The named legal corpora are separate product entities, not tags or folders under an
  information-security corpus. A future cross-corpus experience requires a separately specified
  feature and must not alter the retrieval boundary of this one.
- Evaluation is maintainer-facing in this feature; end users cannot upload or edit golden cases.
- The existing immutable corpus snapshot boundary and citation locations are available to the
  evaluation runner.
- The feature evaluates the current grounded answer and citation behavior; it does not create a
  new legal-answer model or make law-changing-source updates automatic.
- A small curated dataset is sufficient for the POC; large-scale benchmarks and external
  benchmark services are outside this feature.
- Starter questions are a discovery aid, not an evaluation result or a legal recommendation. They
  remain unavailable until a reviewed selection is published for the selected corpus's active
  snapshot.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: Creation of three named, isolated legal corpora and their official source
  manifests; dataset asset validation and versioning; provenance and compatibility checks;
  snapshot-scoped evaluation execution; case and aggregate results; bilingual case metadata;
  evidence, citation, abstention, latency, and token metrics; and a compact maintainer-facing
  inspection and comparison experience; and a corpus-bound empty-chat starter-question selection
  derived from the approved curated dataset cases.
- **Out of scope**: User-authored datasets, automatic legal review, automatic source updates,
  changes to retrieval algorithms, training or fine-tuning models, external benchmark sharing,
  legal advice, and comparison of incompatible runs as if they were equivalent.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: Feature 001's research workspace and technical inspection journey,
  promoted through the production web feature.
- **Intentional differences**: This feature adds a maintainer-facing evaluation entry point and
  result inspection, plus a small dataset-backed empty-chat suggestion surface. It does not alter
  the ordinary researcher chat request, answer, or source-reading workflow.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: Every evaluation case runs against exactly one explicit corpus
  snapshot; no source, result, or citation from another corpus or snapshot may satisfy the case.
- **Evidence behavior**: Expected evidence is a dataset requirement, not a substitute for
  generated citations. The system records both expected and actual evidence and abstains or
  fails safely where snapshot evidence is insufficient.

### Cost and Operational Limits *(mandatory when models or document processing are involved)*

- **Evaluation limits**: Runs process only the curated cases in one dataset revision, apply the
  existing per-answer retrieval and model budgets, and expose case-level cost and failure data.
- **Dataset ingestion limits**: Import accepts only versioned project-owned assets, validates
  bounded manifests and case files, and performs no network acquisition or model call.
- **Safety**: Logs and result views exclude raw prompts, credentials, complete provider payloads,
  and personal information unrelated to the curated legal sources.
- **Starter-question limits**: The chat surface renders at most five selected questions for the
  active corpus and interface language. It transmits the selected question only when the
  researcher activates it and never ships reference answers or expected evidence to the browser.
