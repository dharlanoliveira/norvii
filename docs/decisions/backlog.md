# Decision Backlog

These choices remain open. A feature plan may research one when implementation requires it. Once accepted, replace the row with a link to an ADR and remove it from this backlog.

| Topic | Decision required |
| --- | --- |
| LLM | Hosted model, local model, or both |
| Embeddings | Model that performs adequately for Portuguese and English |
| Reranking | Whether evaluation justifies a dedicated reranker |
| Extraction source | Prefer structured official HTML when available or normalize all content from PDF |
| Ingestion dispatch | Queue, polling, or explicit command |
| Source limits | Maximum PDF size, URL response size, capture duration, and redirects |
| URL policy | Allowed protocols, domains, redirects, DNS results, and resolved IP ranges |
| Artifact contract | Versioned schema connecting Python ingestion and the Go API |
| Chat streaming | Cross-language agent-to-Go-to-React message semantics |
| Deployment | Hosting model for the application and indexes |
| Local models | Whether offline operation is part of the demonstration |
| Cross-jurisdiction comparison | Initial POC capability or later extension |
| Conversation history | Whether conversations persist |
| Evaluation thresholds | Minimum retrieval, citation, abstention, latency, and cost targets |

## Resolved decisions

| Topic | Decision |
| --- | --- |
| Primary database and vector storage | [ADR 0005](0005-postgresql-and-neo4j-persistence.md) selects PostgreSQL with pgvector as canonical persistence |
| Legal graph | [ADR 0005](0005-postgresql-and-neo4j-persistence.md) selects standalone Neo4j Community as a rebuildable projection |

## Decision rule

Do not select an item implicitly during unrelated implementation. The owning feature records decision drivers and research; difficult-to-reverse or cross-feature choices require an ADR.
