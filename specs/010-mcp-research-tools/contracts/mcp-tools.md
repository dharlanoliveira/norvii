# MCP Tool Contract v1

The server exposes eight discoverable read-only tools using MCP protocol version 2.
It supports Streamable HTTP for the Docker deployment and stdio for local development.

| Tool | Required inputs | Maximum result |
| --- | --- | --- |
| `list_corpora` | None | 50 enabled corpora |
| `list_documents` | `corpus_id` | 50 snapshot documents |
| `search_documents` | `corpus_id`, `query` | 8 evidence locations |
| `get_article` | `corpus_id`, `document_version_id`, `unit_locator` | 1 legal unit |
| `get_document_metadata` | `corpus_id`, `document_version_id` | 1 document |
| `find_related_articles` | `corpus_id`, `query` | 8 graph relationships |
| `traverse_legal_graph` | `corpus_id`, `query` | 8 graph relationships |
| `compare_provisions` | `corpus_id`, two document version IDs | 2 cited provisions |

`list_corpora` returns `{ "outcome": "completed", "corpora": [...] }`. Every
completed corpus-scoped result includes `snapshot_id`. The server returns only the
stable outcomes in [data-model.md](../data-model.md); it never exposes raw SQL,
Cypher, provider payloads, private prompts, credentials, or out-of-scope corpus
content.

All tools return one of `completed`, `not_found`, `unavailable`, or `invalid_input`.
Malformed UUIDs and blank required values return `invalid_input`; missing snapshot
resources return `not_found`; unavailable persistence or absent ready graph releases
return `unavailable`. A completed graph request can contain an empty relationship
list when its ready graph has no matches.

Evidence-bearing values include `source_id`, `document_version_id`, a legal-unit
locator (`unit_locator` or `evidence_locator`), and `start_offset` and `end_offset`.
The enclosing result's `snapshot_id` completes the immutable evidence reference.
`get_article` resolves this reference only within the requested corpus and snapshot.

The server also exposes three reusable workflow prompts:
`evidence_grounded_research`, `provision_comparison`, and
`citation_support_verification`. Each requires `corpus_id`, resolves its active
snapshot before returning, instructs the client to cite retrieved evidence and
abstain when it is unavailable, and carries the non-legal-advice boundary.

Each tool and workflow invocation produces a content-safe structured log on stderr:
operation name and kind, outcome, corpus and snapshot identifiers when available,
selected strategy, latency in milliseconds, and item count. Request text and legal
content are never logged.
