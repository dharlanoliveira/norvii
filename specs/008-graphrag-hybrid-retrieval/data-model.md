# Data Model: GraphRAG and Hybrid Retrieval

| Entity | Owner | Purpose |
| --- | --- | --- |
| Semantic extraction run | Ingestion | Records one bounded enrichment attempt for an immutable document version. |
| Semantic entity | Ingestion | A typed legal concept, actor, right, or obligation with evidence support. |
| Semantic relationship | Ingestion | An attributed connection between entities, linked to evidence and extraction provenance. |
| Graph release | Ingestion | An immutable, rebuildable projection manifest for one published corpus snapshot. |
| Graph path | Agent | An ephemeral ordered series of supported relationships selected for one request. |
| Strategy result | Agent | One vector, graph, or hybrid outcome with evidence and measurements. |

## Semantic extraction run

| Field | Meaning |
| --- | --- |
| `id` | Stable immutable extraction identity. |
| `corpus_id`, `source_id`, `document_id` | Ownership boundary for the enriched document version. |
| `pipeline_version`, `prompt_version`, `model` | Reproducibility and operational provenance. |
| `status` | `ready`, `failed`, or `skipped`. |
| `input_tokens`, `output_tokens`, `duration_milliseconds` | Provider-reported or measured operational values. |
| `manifest_sha256` | Deterministic content identity for entity and relationship output. |

## Semantic entity and relationship

Each entity has a typed normalized label and one immutable evidence unit. A relationship has a
typed source and target entity, one immutable evidence unit, extraction run identity, confidence,
and validation state. Relationships cannot cross corpus, source, document, or evidence ownership
boundaries.

## Graph release

One graph release belongs to one published corpus snapshot. It records its immutable manifest hash,
status, build time, selected extraction runs, and projection measurements. A ready graph release
is the only graph state that online graph or hybrid retrieval may query. Historical releases remain
inspectable; a later snapshot does not mutate them.

## Invariants

1. PostgreSQL semantic artifacts are canonical; a graph projection never owns legal text.
2. Every graph entity and relationship points to an immutable document version and legal unit.
3. A graph release includes only extraction artifacts whose documents are members of its snapshot.
4. Graph and hybrid retrieval must receive matching corpus, snapshot, and ready graph-release IDs.
5. A path contains at most three relationships and every relationship has supporting evidence.
6. Rebuilding an unchanged release yields the same manifest and projection identities.
