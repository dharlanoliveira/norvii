# Decision Backlog

These choices remain open. A feature plan may research one when implementation requires it. Once accepted, replace the row with a link to an ADR and remove it from this backlog.

| Topic | Decision required |
| --- | --- |
| LLM | Hosted model, local model, or both |
| Embeddings | Model that performs adequately for Portuguese and English |
| Vector index | Storage and query technology for the POC |
| Legal graph | Dedicated graph database or graph representation over another store |
| Reranking | Whether evaluation justifies a dedicated reranker |
| Extraction source | Prefer structured official HTML when available or normalize all content from PDF |
| Primary database | Storage for corpus metadata, source binaries, and derived artifacts |
| Ingestion dispatch | Queue, polling, or explicit command |
| Source limits | Maximum PDF size, URL response size, capture duration, and redirects |
| URL policy | Allowed protocols, domains, redirects, DNS results, and resolved IP ranges |
| Artifact contract | Versioned schema connecting Python ingestion and the Go API |
| Chat streaming | Go implementation of message semantics consumed by the AI SDK client |
| Deployment | Hosting model for the application and indexes |
| Local models | Whether offline operation is part of the demonstration |
| Cross-jurisdiction comparison | Initial POC capability or later extension |
| Conversation history | Whether conversations persist |
| Evaluation thresholds | Minimum retrieval, citation, abstention, latency, and cost targets |

## Decision rule

Do not select an item implicitly during unrelated implementation. The owning feature records decision drivers and research; difficult-to-reverse or cross-feature choices require an ADR.
