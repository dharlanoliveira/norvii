# Product Overview

## Vision

Norvii is a proof of concept for legal research assisted by AI. It demonstrates RAG, GraphRAG, LLM, MCP, and skills through a web application in which a user selects a small legal corpus, browses its sources, and asks questions whose answers contain traceable citations.

The project is a technical demonstration and does not provide legal advice.

## Delivery approach

The product experience is validated in an executable React prototype before production implementation. The prototype uses deterministic fixtures to exercise journeys, states, responsive behavior, chat streaming presentation, citations, abstention, and technical inspection without a backend.

Production development begins after the prototype baseline is approved. The prototype does not validate retrieval quality, legal correctness, persistence, security, or performance. See the [executable prototyping workflow](../development/prototyping.md).

## Goals

- Answer questions using only the active corpus.
- Link citations to the source passage, article, section, or page used in the answer.
- Support an isolated Portuguese corpus and an isolated English corpus.
- Present the complete product interface in English and Portuguese, with English as the default.
- Compare vector RAG, GraphRAG, and hybrid retrieval.
- Expose retrieved chunks, entities, relations, latency, and token use for inspection.
- Offer research operations through MCP tools and reusable skills.
- Evaluate answer and citation quality with a reproducible question set.
- Keep corpora small enough for a cost-controlled POC.

## Initial exclusions

- Coverage of an entire national or regional legal system.
- Replacement of professional legal research.
- Continuous automatic updates of legislation.
- Large-scale case law indexing.
- Private documents submitted by external users.
- Collaboration, billing, or law-firm management features.
- Production-level availability, scale, or security commitments.

## Audience and evidence of capability

The primary audience is technical: recruiters, engineers, architects, and people interested in applied AI systems.

| Capability | Product evidence |
| --- | --- |
| RAG | Semantic retrieval, retrieved chunks, and grounded answers |
| GraphRAG | Entities, relations, and paths used by multi-hop questions |
| LLM | Controlled synthesis, corpus-language responses, and abstention |
| MCP | Search, provision lookup, and graph traversal tools |
| Skills | Specialized research, comparison, and verification workflows |
| Engineering | Reproducible ingestion, observability, tests, and evaluation |
| Product | Clear interface, accessible sources, and explicit limitations |

## Product experience

### Interface languages

Norvii supports English and Portuguese as interface languages. English is the default when the user has not selected a language. The interface language applies to product-authored navigation, actions, labels, validation, status, error, empty-state, inspection, and accessibility text.

Interface language and corpus language are independent. Changing the interface language does not translate corpus names that are proper titles, source documents, legal passages, citations, user questions, or generated answers. Each of those retains the language defined by its content and active-corpus behavior.

### Entry page

The first page lists registered corpora. The initial demonstration contains one Portuguese legal corpus and one English legal corpus.

Each corpus card shows its name, description, language, jurisdiction, source count, and ingestion state. Opening a corpus displays its sources on the left and a chat restricted to that corpus on the right.

### Corpus workspace

The workspace contains:

1. A header with corpus selection, retrieval strategy selection, and access to technical inspection.
2. A left panel with ready sources, legal metadata, document navigation, and textual search.
3. A right panel with conversation history, question composer, inline citations, loading, errors, and abstention.
4. An inspection panel with retrieved chunks, scores, graph paths, tool calls, model identity, latency, tokens, and citation verification.

### Corpus management

A user can create and edit a corpus, add a PDF or external URL, inspect processing state, retry a failed source, and disable a source. A corpus becomes searchable after at least one source is ready.

Use `Corpus` for the searchable collection and `Source` for each PDF or URL. Do not use `Process` for this entity because it can be confused with a legal proceeding.

### Main journey

1. The user selects a corpus.
2. The application loads only ready sources from that corpus.
3. The user asks a question.
4. Retrieval searches only the active corpus.
5. The LLM produces an answer limited by retrieved evidence.
6. Citation verification checks whether evidence supports material claims.
7. The user opens a citation and reads the original passage.

Grounding, citation, abstention, and corpus isolation rules are governed by the [project constitution](../../.specify/memory/constitution.md).

## MVP acceptance

The MVP is complete when:

- corpora can be created, edited, listed, selected, and disabled;
- each corpus accepts PDF and URL sources;
- PDF sources persist their binary and URL sources persist their external link;
- each source exposes its processing state;
- the Portuguese and English corpora remain isolated;
- ready sources appear in the corpus workspace;
- chat answers in the expected language with navigable citations;
- the complete interface is available in English and Portuguese, starts in English by default, and contains no untranslated product-authored copy in either language;
- questions without sufficient evidence produce explicit abstention;
- inspection exposes retrieved chunks;
- a minimum automated evaluation set exists;
- another contributor can run the POC from maintained documentation.

GraphRAG, MCP, and skills are required for the final demonstration but do not block the first navigable RAG delivery.

## Related documents

- [Corpus definition](corpora.md)
- [Feature map](feature-map.md)
- [AI capabilities](ai-capabilities.md)
- [Evaluation](evaluation.md)
- [System architecture](../architecture/overview.md)
- [Executable prototyping](../development/prototyping.md)
