# Research: Corpus Catalog and Ingestion

## HTTP service and client state

- **Decision**: Use Go's standard `net/http` router and existing pgx pool. Keep React server state behind feature adapters and hooks using `fetch`, `AbortController`, and bounded status polling.
- **Rationale**: Current routing requirements fit standard method-aware patterns. A new web framework or client cache would add policy without solving a demonstrated scale problem.
- **Alternatives considered**: Chi and Gin add routing dependencies; TanStack Query adds useful caching but is unnecessary for a single-user bounded catalog.

## Durable work dispatch

- **Decision**: Use a PostgreSQL leased work queue claimed with `FOR UPDATE SKIP LOCKED`, a unique deterministic ordering, and expiring leases. Do not add a message broker.
- **Rationale**: PostgreSQL documents `SKIP LOCKED` as appropriate for queue-like consumers. The canonical transaction can create the source and work item together, while leases recover crashed workers.
- **Alternatives considered**: Explicit CLI dispatch does not satisfy automatic ingestion; in-process Go jobs violate ingestion ownership; Redis/RabbitMQ adds a fourth local service and dual-write failure modes.
- **Reference**: [PostgreSQL locking clause](https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE)

## Shared persistence ownership

- **Decision**: The Go migrator deploys the schema. Go owns corpus, source, origin, and online command writes. Python owns work leases, attempts, revisions, documents, and units through the versioned ingestion-work contract. Both modules may read contract-defined projections.
- **Rationale**: This preserves one migration mechanism while assigning each mutation to one module and avoiding internal package coupling.
- **Alternatives considered**: Separate migration tools risk ordering drift; routing artifact publication through Go adds a large binary/text service hop and couples offline processing to API availability.

## Safe URL acquisition

- **Decision**: Implement a small synchronous HTTPS transport over Python standard sockets. For each hop it parses and normalizes the URL, resolves all addresses, rejects any non-public result, pins one approved address for the TCP connection, verifies TLS against the original hostname, sends the original Host header, streams at most 10 MB, and manually revalidates at most five redirects. Disable proxy/environment routing.
- **Rationale**: Pre-resolution without connection pinning remains vulnerable to DNS rebinding. Manual redirects make every destination inspectable and testable.
- **Alternatives considered**: Default urllib/HTTPX resolution cannot guarantee the validated address is the connected address; allowlists would prevent user-added official domains; external fetch services exceed POC scope.

## URL content extraction

- **Decision**: Pin Trafilatura 2.2.0 and request XML output with structural formatting from already acquired bytes. Convert headings, paragraphs, lists, and tables into normalized blocks before legal-unit detection.
- **Rationale**: Trafilatura removes navigation noise and retains useful structure without owning acquisition. XML output supports deterministic offsets and unit construction.
- **Alternatives considered**: Raw DOM traversal is fragile on EUR-Lex and government portals; plain-text extraction discards hierarchy; browser automation adds a large runtime and attack surface.
- **Reference**: [Trafilatura repository and extraction options](https://github.com/adbar/trafilatura)

## PDF extraction

- **Decision**: Pin pypdf 6.16.1 and extract each page independently in layout mode. Reject encrypted, invalid, empty, image-only, or over-limit documents. Build page units first, then detect nested English and Portuguese legal markers.
- **Rationale**: pypdf is pure Python, supports Python 3.13, and exposes page-level text without native system packages. Page-first extraction provides a deterministic fallback.
- **Alternatives considered**: PyMuPDF has strong extraction but native bindings and AGPL/commercial licensing considerations; pdfminer.six is lower level; OCR is explicitly out of scope.
- **Reference**: [pypdf text extraction](https://pypdf.readthedocs.io/en/latest/user/extract-text.html)

## Normalization and legal hierarchy

- **Decision**: Normalize Unicode and whitespace without translating or rewriting legal text. Construct one complete document string and offset-addressed units. Detect a deliberately small marker set (`Article`, `Art.`, `Section`, `Chapter`, `Title`, numbered paragraphs and items); fall back to pages or ordered URL blocks.
- **Rationale**: Complete coverage and stable offsets matter more than pretending every jurisdictional structure is perfectly parsed.
- **Alternatives considered**: Site-specific parsers improve individual documents but do not generalize; LLM structure extraction adds cost and non-determinism reserved for later semantic features.

## Idempotent publication

- **Decision**: Hash acquired content, normalized document text, and canonical unit serialization. Publish revision, document, and units in one transaction. Reuse the current active artifact when all hashes and pipeline version match; retain older versions when they differ.
- **Rationale**: Hashes provide cheap reproducibility and prevent duplicate active output while immutable history supports future citation stability.
- **Alternatives considered**: Delete-and-replace breaks references; distributed transactions are unnecessary because Neo4j is not touched.

## Initial data

- **Decision**: Seed stable UUIDs for the Brazilian LGPD corpus/source and English EU GDPR corpus/source. Seed only official HTTPS origin metadata and enqueue acquisition; never bundle source text as runtime fallback.
- **Rationale**: Reviewers get a real bilingual path, repeated initialization is safe, and remote failure remains visible rather than disguised.
- **Alternatives considered**: Bundled snapshots are more reproducible but belong to the later curated-snapshot feature; multiple initial sources expand tokens and ingestion time.

## Contract strategy

- **Decision**: Promote a machine-readable HTTP v1 contract plus a language-neutral ingestion-work v1 contract. Use a stable JSON error envelope and UUID strings, RFC 3339 timestamps, lowercase enum values, and SHA-256 hex hashes.
- **Rationale**: These representations are equally consumable from TypeScript, Go, and Python and support provider/consumer verification.
- **Alternatives considered**: Go structs or database rows as implicit contracts violate the constitution; GraphQL adds tooling without a query-shape need.

## Test strategy

- **Decision**: Apply red-green-refactor per user story. Unit tests cover domain rules and parsers; contract tests cover schemas; PostgreSQL integration tests cover constraints, leases, atomicity, and isolation; controlled HTTPS fixtures cover SSRF/redirect/size behavior; Playwright covers the real vertical journey.
- **Rationale**: External legal sites are not deterministic test dependencies. The clean quickstart separately proves live acquisition.
- **Alternatives considered**: Live-site CI tests are flaky and make third-party availability a merge gate; end-to-end tests alone localize failures poorly.
