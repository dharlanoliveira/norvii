# Feature Specification: Corpus Catalog and Ingestion

**Feature Branch**: `004-corpus-catalog`

**Created**: 2026-08-17

**Status**: Draft

**Input**: User description: "Deliver one real vertical workflow instead of mixing an authoritative corpus catalog with simulated workspace data: persist and manage corpora, accept PDF and official URL sources, ingest and version their content, preserve document structure, expose real data through product contracts, and render real sources and documents in the production workspace for initial English and Portuguese corpora."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Explore Two Initial Real Corpora (Priority: P1)

A researcher opens Norvii after a clean initialization, sees one English data-protection corpus and one Portuguese data-protection corpus, enters either workspace, and browses a real ingested official legal source. The source library, document metadata, document text, and section navigation all come from authoritative product data rather than local demonstration fixtures.

**Why this priority**: This is the smallest end-to-end outcome that proves the catalog, ingestion boundary, durable data, public contracts, and production workspace work together. It keeps the bilingual portfolio demonstration useful without presenting simulated documents as real ones.

**Independent Test**: Initialize a clean environment with source-network access, wait for the two initial sources to reach a terminal state, open each corpus, and verify that its official source, captured provenance, complete normalized content, and addressable document units are visible and isolated from the other corpus.

**Acceptance Scenarios**:

1. **Given** a newly initialized environment, **When** initialization completes, **Then** exactly one enabled English corpus and one enabled Portuguese corpus exist, each with one official URL source queued for ingestion.
2. **Given** the initial official sources are reachable and valid, **When** ingestion completes, **Then** each source becomes ready with an immutable captured revision, a complete normalized document, provenance metadata, and ordered addressable units.
3. **Given** either initial corpus is selected, **When** the researcher opens its workspace, **Then** only that corpus's persisted sources and document content are shown.
4. **Given** an initial source cannot be acquired or extracted, **When** the researcher opens its workspace, **Then** the source remains visible with a safe actionable failure state and no simulated replacement content.
5. **Given** initialization is repeated, **When** the initial corpus and source identities already exist, **Then** no duplicate is created and user-maintained corpus metadata is not overwritten.

---

### User Story 2 - Create and Manage a Corpus (Priority: P2)

A catalog maintainer creates an isolated corpus, corrects its descriptive metadata, disables it when it should not be available for research, and later re-enables it without changing its stable identity.

**Why this priority**: Durable catalog management establishes the ownership boundary for every source and future retrieval operation while preserving the approved catalog experience.

**Independent Test**: Create a third corpus, refresh the catalog, edit it, disable it, verify it cannot be selected, re-enable it, and confirm that every step preserves the same stable identifier.

**Acceptance Scenarios**:

1. **Given** valid name, description, language, and jurisdiction values, **When** the maintainer creates a corpus, **Then** it is stored as enabled with a stable identity and appears in the selectable catalog.
2. **Given** an existing corpus, **When** the maintainer saves valid metadata changes, **Then** the new values persist while its identifier, creation time, sources, and ownership boundary remain unchanged.
3. **Given** an enabled corpus, **When** the maintainer confirms disabling it, **Then** it remains available for management but cannot be selected or opened as an available research workspace.
4. **Given** a disabled corpus, **When** the maintainer re-enables it, **Then** it returns to the selectable catalog with the same identity, metadata, and sources.
5. **Given** invalid metadata or a failed mutation, **When** the operation ends, **Then** no partial catalog change is committed and the maintainer receives an actionable localized explanation.
6. **Given** the maintainer creates or edits a corpus, **When** the management flow opens, **Then** it uses a dedicated, directly addressable form screen while the catalog retains its selection-focused layout and primary open-corpus action.

---

### User Story 3 - Add and Ingest an Official URL (Priority: P3)

A catalog maintainer adds an official HTTPS legal source to a corpus. The product validates and captures the destination, extracts its legal content, preserves the captured revision and provenance, and makes the ready document browsable.

**Why this priority**: Official web sources are common for both initial corpora, and safe capture is required before citations or reproducible retrieval can exist.

**Independent Test**: Add a controlled public HTTPS legal page to an enabled corpus, observe its lifecycle through ready, and verify that the captured URL, capture time, content hash, complete normalized document, and units are available only in that corpus.

**Acceptance Scenarios**:

1. **Given** a supported public HTTPS URL and valid source metadata, **When** the maintainer adds it, **Then** one pending source is created under the selected corpus and ingestion begins without blocking catalog use.
2. **Given** a URL redirects, **When** it is acquired, **Then** every destination is revalidated and the final official URL is recorded with the capture.
3. **Given** acquisition and extraction succeed, **When** the source becomes ready, **Then** the original link, final captured link, capture time, response metadata, content hash, normalized document, and document units are retained.
4. **Given** a URL uses an unsupported protocol, resolves to a non-public address, exceeds a POC limit, returns unsupported content, redirects unsafely, or times out, **When** validation ends, **Then** no remote content is published and the source exposes a safe categorized failure.
5. **Given** the exact normalized URL is already registered in the same corpus, **When** it is submitted again as a new source, **Then** the duplicate is rejected and the existing source is identified without exposing another corpus's records.

---

### User Story 4 - Upload and Ingest a PDF (Priority: P4)

A catalog maintainer uploads a text-based legal PDF to a corpus. The original binary is preserved, its text and page structure are extracted, and the resulting document can be browsed beside the original source metadata.

**Why this priority**: PDF is the second required origin type and demonstrates that immutable source material remains separate from derived, reprocessable artifacts.

**Independent Test**: Upload a valid text-based PDF within the documented POC limits, observe it become ready, restart the environment, and verify that the same binary hash, complete normalized text, page units, and detected legal sections remain available.

**Acceptance Scenarios**:

1. **Given** a valid PDF and source metadata, **When** the maintainer uploads it, **Then** one pending source is created under the selected corpus and the original binary, filename, media type, size, and hash are preserved.
2. **Given** a valid text-based PDF, **When** extraction succeeds, **Then** a complete normalized document and ordered page units are published, with recognizable legal sections represented as linked child units when detected.
3. **Given** a file is empty, encrypted, not a valid PDF, lacks extractable text, or exceeds a POC limit, **When** validation or extraction ends, **Then** no document version is published and the source exposes a safe categorized failure.
4. **Given** the same PDF hash already exists in the same corpus, **When** it is uploaded again as a new source, **Then** the duplicate is rejected and the existing source is identified without exposing another corpus's records.

---

### User Story 5 - Inspect, Retry, and Reprocess Sources (Priority: P5)

A maintainer can understand whether a source is pending, processing, ready, or failed; inspect safe failure information; retry a failed source; and deliberately reprocess a ready source without losing prior published revisions.

**Why this priority**: Ingestion depends on files, networks, and parsers that can fail. An observable, retryable lifecycle is necessary for a trustworthy demonstration and for future pipeline evolution.

**Independent Test**: Cause one deterministic ingestion failure, inspect its status, retry after correcting the condition, then reprocess the ready source and verify that publication is idempotent when content is unchanged and versioned when content changes.

**Acceptance Scenarios**:

1. **Given** an accepted source, **When** its processing state changes, **Then** the catalog and workspace expose its current state and safe timestamps without requiring direct storage inspection.
2. **Given** a failed source, **When** the maintainer retries it, **Then** a new bounded ingestion attempt begins while prior diagnostic history remains inspectable.
3. **Given** a ready source, **When** the maintainer requests reprocessing, **Then** prior published revisions remain immutable until a complete replacement revision is atomically published.
4. **Given** reprocessing produces the same source content and artifact hashes, **When** publication completes, **Then** no duplicate active document version is created.
5. **Given** processing fails after a ready revision exists, **When** the failed attempt ends, **Then** the last ready revision remains browsable and the failed attempt is reported separately.

---

### User Story 6 - Browse Real Documents Without Simulated Chat (Priority: P6)

A researcher navigates a corpus-rooted source tree, selects a ready source, reads its complete normalized content, moves through its pages or legal sections, and can open or download the preserved origin. Chat remains visibly unavailable until grounded retrieval is implemented.

**Why this priority**: The production experience must remain honest about which capabilities are real. Removing runtime document and chat fixtures avoids mixing authoritative data with prepared legal answers.

**Independent Test**: Open a corpus containing ready, pending, and failed sources; navigate the tree with keyboard only; inspect a ready URL and PDF document; switch interface language; and verify that no simulated source, answer, citation, or prepared legal content is rendered.

**Acceptance Scenarios**:

1. **Given** a corpus with sources in different states, **When** its workspace opens, **Then** the source tree lists only persisted sources from that corpus and communicates type, state, title, and selection without relying on color alone.
2. **Given** a ready document with hierarchical units, **When** a researcher selects a unit, **Then** the viewer opens the corresponding stable location within the complete document.
3. **Given** a ready URL source, **When** the researcher requests the origin, **Then** the captured destination opens safely in a new browser context.
4. **Given** a ready PDF source, **When** the researcher requests the origin, **Then** the preserved PDF is delivered with its recorded filename and media type.
5. **Given** the researcher switches interface language, **When** the workspace rerenders, **Then** product controls and states change language while preserved legal content and source metadata remain unchanged.
6. **Given** the researcher selects Chat, **When** grounded chat is not implemented, **Then** a localized unavailable state explains the limitation and no simulated answer is produced.

### Edge Cases

- The catalog is empty: a localized empty state offers corpus creation and renders no fallback fixtures.
- A corpus has no sources: its workspace shows an honest source-empty state and a source-add action.
- A corpus has sources but none are ready: statuses remain visible, the document viewer explains that no content is ready, and chat remains unavailable.
- A corpus is disabled while its workspace is open: the next authoritative refresh makes the workspace unavailable without switching to another corpus.
- A requested corpus or source identifier is unknown or belongs to another corpus: the request is rejected through the existing recovery experience without revealing foreign metadata.
- A process stops during acquisition, extraction, or publication: incomplete artifacts never become active, and the source can be retried safely.
- A document has no recognizable legal hierarchy: the complete document remains browsable through a deterministic fallback of page units for PDF or ordered content blocks for URL content.
- Extracted unit boundaries overlap, contain gaps, or fall outside the normalized document: publication fails rather than exposing an inconsistent hierarchy.
- A live URL changes after a successful capture: the prior revision remains immutable, and explicit reprocessing may publish a new revision linked to the same source.
- Two maintainers act on the same record despite the single-user POC assumption: a stale mutation fails without silently overwriting a newer committed state.
- Initial live sources are unavailable during clean initialization: the corpora and failed source records remain usable for inspection and explicit retry.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST maintain one authoritative catalog shared by the production client and all processing components.
- **FR-002**: Each corpus MUST have a system-assigned stable identifier, name, description, legal-content language, jurisdiction, lifecycle status, creation time, last-update time, and concurrency value.
- **FR-003**: Supported corpus languages for this POC MUST be English and Portuguese, represented independently from the interface language.
- **FR-004**: The system MUST initialize exactly one English European Union data-protection corpus and one Portuguese Brazilian data-protection corpus when their stable initialization identities are absent.
- **FR-005**: Each initial corpus MUST include exactly one stable official URL source identity: the English corpus for the GDPR and the Portuguese corpus for the LGPD.
- **FR-006**: Initial source ingestion MUST be requested automatically after clean initialization and MUST expose failure for explicit retry rather than block the rest of the application indefinitely.
- **FR-007**: Repeating initialization MUST NOT duplicate initial corpora or sources, replace immutable revisions, or overwrite user-maintained corpus metadata.
- **FR-008**: Researchers MUST be able to list and select enabled corpora; maintainers MUST be able to list enabled and disabled corpora with unambiguous lifecycle state.
- **FR-009**: Maintainers MUST be able to create a corpus and edit its name, description, language, and jurisdiction on dedicated create and edit routes without changing stable identity, creation time, source ownership, or prior revisions; opening those routes MUST NOT reflow the catalog cards.
- **FR-010**: Maintainers MUST be able to disable and re-enable a corpus without deleting it; disabled corpora MUST be unavailable for researcher selection and direct workspace access.
- **FR-011**: Corpus mutations MUST reject stale concurrency values and MUST commit all validated changes atomically or none of them.
- **FR-012**: Every source MUST belong to exactly one corpus and have a stable identifier, title, origin type, processing state, creation and update times, latest safe failure category, and latest ready revision when one exists.
- **FR-013**: The only supported source origin types MUST be an uploaded PDF binary or an external HTTPS URL.
- **FR-014**: The system MUST preserve PDF binaries with original filename, declared and detected media type, byte size, and cryptographic content hash separately from derived artifacts.
- **FR-015**: The system MUST preserve URL origins with the submitted URL, final captured URL, capture time, response media type, response size, and cryptographic extracted-content hash; persisting complete raw web responses remains outside scope.
- **FR-016**: PDF uploads and URL captures MUST each be limited to 10 MB, and a corpus MUST accept no more than 20 registered sources in this POC.
- **FR-017**: PDF ingestion MUST accept only valid, unencrypted PDFs with extractable text; OCR and image-only document recognition MUST NOT be presented as supported.
- **FR-018**: URL acquisition MUST allow only HTTPS destinations that resolve exclusively to public network addresses, MUST revalidate every redirect, MUST enforce bounded redirects and timeouts, and MUST reject unsupported response types before extraction.
- **FR-019**: Duplicate PDF hashes and duplicate normalized URLs MUST be rejected within the same corpus, while duplicate checks MUST NOT disclose sources owned by another corpus.
- **FR-020**: Adding a valid source MUST create it in `pending`, and its lifecycle MUST support `pending`, `processing`, `ready`, and `failed` with explicit retry and reprocess actions.
- **FR-021**: Processing attempts MUST be bounded, MUST NOT retry forever automatically, and MUST retain safe attempt status, timestamps, pipeline version, and failure category without logging credentials, complete document content, or private request data.
- **FR-022**: Each successful acquisition MUST create an immutable source revision tied to origin metadata and hashes.
- **FR-023**: Each successfully extracted source revision MUST produce one immutable document version containing the complete normalized document and a cryptographic content hash.
- **FR-024**: Each document version MUST own an ordered hierarchy of addressable document units linked to the complete document by stable locator, parent, order, visible marker, text offsets, content hash, and page range when applicable.
- **FR-025**: Recognizable titles, chapters, sections, articles, paragraphs, items, recitals, and pages MUST be represented as linked units when reliably detected; otherwise, deterministic page or ordered-block units MUST preserve complete coverage.
- **FR-026**: Document publication MUST validate that units are ordered, non-overlapping where they represent peer spans, within document bounds, and traceable to exactly one source revision before atomically making a version ready.
- **FR-027**: A failed or interrupted attempt MUST NOT expose a partial document as ready and MUST NOT replace the last ready document version.
- **FR-028**: Reprocessing unchanged content with the same artifact hashes MUST be idempotent; changed content MUST create a new immutable revision while retaining prior revisions.
- **FR-029**: Versioned public contracts MUST define corpus management, source creation, source lifecycle and retry, source listing, document metadata, document units, origin delivery, and explicit error outcomes across product components.
- **FR-030**: Public errors MUST distinguish invalid input, payload limit, unsafe URL, unsupported content, duplicate source, stale state, unknown record, unavailable record, acquisition failure, extraction failure, publication failure, and unexpected service failure without exposing internals.
- **FR-031**: The production catalog and workspace MUST load authoritative corpus, source, lifecycle, document, and unit data through public contracts and MUST NOT fall back to runtime corpus, source, document, answer, or citation fixtures.
- **FR-032**: The corpus-rooted source tree MUST list only sources owned by the active corpus and MUST communicate source type, processing state, title, and selection accessibly.
- **FR-033**: The document viewer MUST render the complete normalized document, navigate stable document units, expose provenance and current revision metadata, and provide safe access to the preserved PDF or captured external origin.
- **FR-034**: The production client MUST provide localized loading, empty, validation, processing, ready, failed, unavailable, retry, and stale-state experiences in English and Portuguese, with English as the default interface language.
- **FR-035**: Corpus and document language MUST remain preserved domain metadata and content and MUST NOT be automatically translated when interface language changes.
- **FR-036**: Catalog and workspace actions MUST be keyboard operable, provide visible focus and status feedback, and require confirmation before corpus disablement or source reprocessing.
- **FR-037**: Chat MUST present a localized unavailable state and MUST NOT produce simulated or model-generated answers until a grounded-chat feature replaces that state.
- **FR-038**: This feature MUST NOT create embeddings, retrieval chunks, semantic statements, LLM extractions, graph projections, RAG answers, or citations.

### Key Entities

- **Corpus**: An isolated legal research collection with stable identity, mutable descriptive metadata, one legal-content language, one jurisdiction, lifecycle state, concurrency value, timestamps, and exclusively owned sources.
- **Source**: A stable corpus-owned origin record for either one PDF or one external URL. It retains lifecycle and attempt status independently from immutable captured and derived versions.
- **PDF origin**: The preserved original binary and its filename, media types, size, and content hash.
- **URL origin**: The submitted official link and the safe acquisition policy used to create captured revisions.
- **Processing attempt**: One bounded acquisition, extraction, and publication attempt with state, timestamps, pipeline version, and safe outcome details.
- **Source revision**: An immutable captured representation of one source at one point in time, tied to origin provenance and content hashes.
- **Document version**: The immutable complete normalized text derived from one source revision. Only an atomically published version may become the source's latest ready document.
- **Document unit**: An ordered, addressable part of a document version, such as a title, chapter, section, article, paragraph, item, recital, page, or fallback content block. Units form a parent-child hierarchy and point into the complete document rather than duplicating its identity.
- **Initial source identity**: A stable manifest identity for one official source in each initial corpus that makes initialization and ingestion requests repeatable.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A clean environment with source-network access presents exactly two enabled initial corpora and makes one real official source per corpus browsable without manual catalog or source entry.
- **SC-002**: Repeating initialization three times leaves exactly two initial corpus identities, two initial source identities, and one active ready document per unchanged initial source while preserving user edits.
- **SC-003**: Under documented POC conditions, at least 95% of catalog and workspace reads show data, an empty state, or an actionable failure state within 2 seconds.
- **SC-004**: A reviewer can create a corpus, add a valid source, observe processing, and open its real document in under 3 minutes, excluding remote-server delay beyond the documented timeout.
- **SC-005**: Supported initial URL sources and valid text-based PDFs of up to 2 MB complete ingestion within 90 seconds in at least 95% of clean local verification runs.
- **SC-006**: Every ready test document exposes 100% of its normalized text through one complete document and valid addressable units with no out-of-bounds unit spans.
- **SC-007**: Invalid files, unsafe URLs, duplicates, stale mutations, timeouts, extraction failures, and interrupted publication produce the expected safe outcome and no partial active artifact in 100% of automated failure scenarios.
- **SC-008**: Retrying a corrected failed source succeeds without creating duplicate source identity, and reprocessing unchanged content creates no duplicate active version in 100% of idempotency tests.
- **SC-009**: Corpus isolation tests demonstrate zero cross-corpus source, document, unit, duplicate-check, binary, or failure-detail disclosure across all public operations.
- **SC-010**: A reviewer can edit, disable, and re-enable a corpus while observing the same identity and source ownership across all operations.
- **SC-011**: All catalog, source-management, lifecycle, tree-navigation, and document-viewer journeys are keyboard operable with no critical or serious automated accessibility findings.
- **SC-012**: Production runtime verification finds zero local corpus, source, document, prepared-answer, or citation fixture data rendered when authoritative services are unavailable.
- **SC-013**: Another contributor can start the complete local environment, ingest both initial sources, and browse both real documents by following one documented bootstrap workflow without direct database administration.

## Assumptions

- Norvii remains a local proof of concept operated by one trusted user. Authentication, authorization roles, multi-tenancy, and untrusted public uploads are deferred, but stale-state protection still prevents accidental overwrite.
- Disablement is reversible and is not physical deletion. Source deletion is also outside this feature; failed or superseded sources and revisions remain inspectable.
- The initial Portuguese source is the official LGPD page maintained by Brazil's federal government, and the initial English source is the official English GDPR page maintained by EUR-Lex, as proposed in the global corpus document.
- Clean initial ingestion requires access to those official HTTPS sources. Automated tests use controlled deterministic inputs, while production runtime never substitutes those inputs for unavailable official content.
- The POC limits each source origin to 10 MB and each corpus to 20 registered sources. Planning may choose a smaller extracted-text safety bound if the two initial sources remain supported.
- Only digitally generated PDFs with extractable text are supported. OCR, scanned-image recognition, password handling, digital-signature validation, and malware scanning beyond safe file validation are outside this trusted local POC.
- Reliable legal hierarchy detection is best-effort. Complete normalized text and deterministic fallback units are mandatory even when specialized article or section detection is unavailable.
- Feature 002 supplies the approved production layout and interaction baseline. Feature 003 supplies the initialized canonical persistence and component connectivity boundary.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: Durable corpus management; repeatable initial English and Portuguese corpora and official URL sources; PDF and safe HTTPS source registration; bounded acquisition; extraction; immutable source revisions; complete normalized documents; hierarchical document units; lifecycle, retry, and reprocessing; versioned cross-component contracts; real production catalog, source tree, and document viewer data; bilingual product states; and one complete local bootstrap journey.
- **Out of scope**: Authentication and authorization; physical corpus or source deletion; OCR and scanned PDFs; scheduled URL recapture; raw web-response archival; more than two initial sources; embeddings; retrieval fragments; semantic or factual extraction with models; Neo4j publication; RAG; live chat; citations; GraphRAG; MCP research tools; and legal advice.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: The verified catalog, source-tree, viewer, state, responsive, and accessibility journeys in [Feature 001](../001-product-experience-prototype/spec.md), independently implemented for production in [Feature 002](../002-production-web-experience/spec.md).
- **Intentional differences**: Catalogs, sources, lifecycle state, document content, and locations come from authoritative product contracts rather than runtime fixtures. Corpus and source management plus processing states are added. Prepared sources, simulated answers, and simulated citations are removed from the production runtime. The approved Chat surface remains visible only as a localized unavailable state until grounded chat is delivered.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: Every source, revision, document, unit, duplicate check, and origin delivery operation is scoped by the active corpus identity. This feature creates the canonical boundary but performs no retrieval.
- **Evidence behavior**: Source provenance and stable document locations become real and inspectable. No legal answer or citation is generated; the product explicitly reports that grounded chat is unavailable rather than presenting prepared evidence as a live result.
