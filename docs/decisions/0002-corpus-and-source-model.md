# ADR 0002: Corpus and Source Model

- Status: Accepted
- Date: 2026-08-12
- Feature: corpus catalog and source management
- Deciders: project maintainers

## Context

Norvii must switch between isolated Portuguese and English legal collections and accept both uploaded PDF files and official external links. The collection name must not be confused with a legal proceeding.

## Decision drivers

- Enforce language and jurisdiction isolation during retrieval.
- Preserve the original source independently from derived ingestion artifacts.
- Support reproducible citations for mutable external pages.
- Use domain vocabulary that is clear in a legal product.

## Options considered

### Corpus with PDF or URL sources

A `Corpus` owns many `Source` records. Each source has one origin type, while normalized text and indexes remain derived artifacts.

### Process as the top-level entity

The term can be interpreted as a legal proceeding or a technical background process and is therefore ambiguous.

### One document record containing origin and every derived artifact

This reduces the initial entity count but mixes immutable origin, mutable captures, processing state, and generated indexes.

## Decision

Use `Corpus` for the isolated searchable collection and `Source` for each PDF or URL. A source belongs to exactly one corpus.

Persist a PDF source as binary data with file metadata and hash. Persist a URL source as an external link with capture metadata and extracted-content hash. Store normalized text, legal units, chunks, embeddings, and graph relations as separately versioned ingestion artifacts.

## Consequences

### Positive

- Corpus isolation becomes explicit in the domain model.
- Original source identity survives reprocessing.
- URL captures and derived artifacts can be versioned independently.

### Negative

- Persistence must handle both binary data and structured artifacts.
- URL content requires a safe capture and recapture policy.
- Lifecycle and artifact publication need explicit state transitions.

## Verification

Corpus and source features must test ownership, source-type invariants, lifecycle transitions, and exclusion of non-ready or foreign-corpus sources from retrieval.

## References

- [Legal corpora](../product/corpora.md)
- [Product overview](../product/overview.md)
