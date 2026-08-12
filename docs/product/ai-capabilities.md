# AI Capabilities

## Retrieval modes

Norvii demonstrates vector RAG, GraphRAG, and hybrid retrieval. Their product behavior and data flow are defined in the [system architecture](../architecture/overview.md). The interface exposes the selected strategy and enough structured trace data to compare results.

## Proposed MCP tools

These names describe intended capabilities, not existing public contracts:

- `list_corpora`: list available corpora.
- `list_documents`: list sources and metadata in the active corpus.
- `search_documents`: perform textual or semantic retrieval.
- `get_article`: return one identified legal unit.
- `get_document_metadata`: return origin, status, and version metadata.
- `find_related_articles`: locate related references and concepts.
- `traverse_legal_graph`: traverse explicit graph relations.
- `compare_provisions`: retrieve provisions for a grounded comparison.

The feature that implements each tool owns its versioned input, output, error, and authorization contract.

## Proposed skills

- Evidence-grounded legal research.
- Provision comparison.
- Structured document summary.
- Security-incident obligation analysis.
- Citation-support verification.
- Evaluation question generation and execution.

Each skill declares when to call tools, how to validate evidence, which corpus is active, and when to abstain.

## Demonstration boundary

RAG chat and navigable citations form the first navigable delivery. GraphRAG, MCP, and skills are later independently specified features and remain required for the final portfolio demonstration.
