# Research: GraphRAG and Hybrid Retrieval

## Decision 1: Keep the graph derived and snapshot-scoped

**Decision**: PostgreSQL records immutable semantic extraction artifacts and graph-release
manifests. Neo4j receives only an idempotent projection for one staged immutable snapshot.

**Rationale**: The graph can be deleted and rebuilt without affecting legal source content or an
active snapshot. This preserves the existing canonical/rebuildable store decision.

**Alternatives considered**:

- Use Neo4j as the authority for semantic artifacts. Rejected because immutable source evidence,
  publication, and transactional source lifecycle already belong to PostgreSQL.
- Project newest ready documents directly. Rejected because it would make candidates visible
  before snapshot and graph validation.

## Decision 2: Separate offline semantic enrichment from online retrieval

**Decision**: Extract a bounded set of typed entities and relationships after immutable document
artifacts exist. The ingestion worker stages an immutable snapshot, builds the graph release, and
activates the snapshot only after validation succeeds. A separate command remains available only
to reproduce an already recorded release.

**Rationale**: Model calls cannot occur on the chat path. Completing the offline release lifecycle
inside ingestion keeps cost and failures observable while ensuring a successful source attempt is
actually ready for every supported retrieval strategy.

**Alternatives considered**:

- Extract relations while answering a question. Rejected because it is slow, unrepeatable, and
  allows transient model output to masquerade as curated evidence.
- Run a graph build on every local bootstrap. Rejected because startup must not incur provider
  cost or make arbitrary network calls.

## Decision 3: Use a minimal evidence-backed vocabulary

**Decision**: The first release projects structural document links and only the entity and
relationship types needed by seeded connected questions: legal concept, actor, right, obligation,
and an explicit directed legal vocabulary. The vocabulary distinguishes scope (`applies_to`),
observance addressee (`must_be_observed_by`), duty (`imposes_duty_on`), responsibility
(`assigns_responsibility_to`), definition (`defines`), grant (`grants`), protection (`protects`),
and condition (`conditions`).

**Rationale**: A small vocabulary is inspectable and fits the two-corpus POC. Every semantic
relationship carries an evidence unit and extraction provenance. Ambiguous `requires` and
`governs` labels are excluded so graph paths state the legal role being represented.

**Alternatives considered**:

- Project every possible legal ontology category. Rejected as an unjustified cost and validation
  burden.
- Treat extracted relationships as official facts. Rejected because they are model-derived
  interpretations, not authoritative source metadata.

## Decision 4: Make strategy selection explicit and fail closed

**Decision**: A request selects `vector`, `graph`, or `hybrid`. Graph and hybrid require a ready
release for the active snapshot. A missing, stale, failed, or weak graph result returns a safe,
localized outcome; it never silently executes vector retrieval under another label.

**Rationale**: The POC must demonstrate meaningful comparison and prevent false claims about the
retrieval path used.

**Alternatives considered**:

- Automatically downgrade graph to vector. Rejected because it invalidates strategy comparison.
- Make graph strategy the only default. Rejected because vector is a useful baseline and graph
  release construction is intentionally explicit.

## Decision 5: Preserve shared citation behavior

**Decision**: Graph paths reference the same immutable source and legal-location identifiers used
by vector evidence. Hybrid deduplicates locations before generation and preserves which strategy
contributed each evidence item in inspection.

**Rationale**: Citation navigation remains one coherent user interaction and all claims remain
grounded in official text.

**Alternatives considered**:

- Return graph nodes without source locations. Rejected because graph structure alone cannot
  support legal claims.
- Produce a separate graph-only answer format. Rejected because it would fragment the existing
  grounded-chat experience.
