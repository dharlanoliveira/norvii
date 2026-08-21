# ADR 0001: Application Module Boundaries

- Status: Accepted
- Date: 2026-08-12
- Feature: project foundation
- Deciders: project maintainers

## Context

Norvii needs a portfolio-grade web interface, an online application backend, and a document-processing pipeline. Chat requests and document ingestion have different runtime characteristics and benefit from different language ecosystems.

## Decision drivers

- React ecosystem for a polished chat and document interface.
- Go for the public online API and streaming facade.
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

Use a React and TypeScript SPA built with Vite under `apps/web/`, a Go public API facade under `apps/api/`, a Python LangGraph agent under `apps/agent/`, and a Python ingestion service under `apps/ingestion/`.

Group all four deployable modules under `apps/` to keep their repository paths symmetric. Keep executable product experiments under the separate `prototypes/` root.

Use assistant-ui for chat presentation and AI SDK compatible message semantics at the client boundary. The Go API remains the only public online backend. The Python agent is an internal online runtime behind the Go facade. Streamlit is not part of the primary interface.

## Consequences

### Positive

- Clear workload and language ownership.
- Independent module tests and dependency graphs.
- React control over chat, citation, and inspection presentation.
- Python ingestion libraries remain outside online request latency, while the dedicated Python agent provides an explicit LangGraph boundary for online AI orchestration.

### Negative

- Public and internal schemas and contract tests are required across the runtimes.
- Local development requires Node.js, Go, and Python toolchains.
- Streaming compatibility cannot rely on a JavaScript server implementation.

## Verification

Each cross-module feature plan must identify affected modules and contract tests. The first chat feature must prove that the Python graph produces grounded internal events, the Go facade produces structured public message parts, and React consumes them. The Python agent was added by Feature 005 after confirming that LangGraph's supported ecosystems provide a safer orchestration boundary than hand-managed graph state in Go.

## References

- [Repository structure](../architecture/repository-structure.md)
- [Module models](../modules/README.md)
