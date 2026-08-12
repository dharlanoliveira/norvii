# ADR 0001: Three Application Modules

- Status: Accepted
- Date: 2026-08-12
- Feature: project foundation
- Deciders: project maintainers

## Context

Norvii needs a portfolio-grade web interface, an online application backend, and a document-processing pipeline. Chat requests and document ingestion have different runtime characteristics and benefit from different language ecosystems.

## Decision drivers

- React ecosystem for a polished chat and document interface.
- Go for online APIs, streaming, retrieval orchestration, MCP, and skills.
- Python ecosystem for document extraction, NLP, embeddings, and evaluation.
- Explicit boundaries that demonstrate cross-language engineering without duplicating server responsibilities.
- A small operational footprint appropriate for a POC.

## Options considered

### React SPA, Go API, and Python ingestion

This separates the browser, online backend, and offline pipeline. It requires explicit contracts but gives each workload an appropriate language and runtime.

### Next.js application plus separate backend services

This adds server-side JavaScript responsibilities beside Go. SSR and SEO are not current requirements, so the extra server layer has no demonstrated product value.

### Streamlit as the primary interface

This accelerates internal data applications but provides less control over the portfolio-grade React experience and duplicates Python's role as ingestion rather than presentation.

## Decision

Use a React and TypeScript SPA built with Vite under `apps/web/`, a Go backend under `apps/api/`, and a Python ingestion service under `apps/ingestion/`.

Group all three deployable modules under `apps/` to keep their repository paths symmetric. Keep executable product experiments under the separate `prototypes/` root.

Use assistant-ui for chat presentation and AI SDK compatible message semantics at the client boundary. The Go API remains the only online application backend. Streamlit is not part of the primary interface.

## Consequences

### Positive

- Clear workload and language ownership.
- Independent module tests and dependency graphs.
- React control over chat, citation, and inspection presentation.
- Python libraries remain outside online request latency.

### Negative

- Public schemas and contract tests are required across three languages.
- Local development requires three toolchains.
- Streaming compatibility cannot rely on a JavaScript server implementation.

## Verification

Each cross-module feature plan must identify affected modules and contract tests. The first chat feature must prove that the Go stream produces structured message parts consumed by the React client.

## References

- [Repository structure](../architecture/repository-structure.md)
- [Module models](../modules/README.md)
