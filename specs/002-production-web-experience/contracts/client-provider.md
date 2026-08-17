# Client Research Provider Contract

## Status

This is a feature-local production-client boundary. It is not a public HTTP, streaming, Go, Python, ingestion, or persistence contract.

## Purpose

Presentation code receives validated research domain records without importing demonstration storage or assistant-ui details. A later feature may implement the same user-facing capabilities through versioned public API contracts and a new adapter.

## Operations

### List corpora

- Input: none.
- Output: an ordered read-only collection of corpus summaries.
- Error: a classified unavailable result; no raw provider error reaches presentation.
- Invariant: each summary has a unique identifier and accurate source count.

### Find corpus

- Input: one route-safe corpus identifier.
- Output: one complete corpus or a not-found result.
- Error: invalid or unknown identifiers are indistinguishable to the user and expose the localized recovery state.
- Invariant: returned sources all belong to the returned corpus.

### Resolve prepared response

- Input: active corpus identifier and user question.
- Output: answered, abstained, or failed result with structured parts.
- Error: failures use a stable client-domain classification and preserve recoverable input.
- Invariant: text and citation parts are distinct; every citation resolves inside the active corpus.

### Resolve citation

- Input: active corpus identifier and citation.
- Output: validated source and stable location or an unavailable result.
- Error: cross-corpus, missing-source, and missing-location references never navigate the viewer.
- Invariant: successful resolution never changes the active corpus.

## Compatibility expectation

The provider interface is intentionally narrow but not versioned for cross-language use. The feature that introduces the Go API must define a language-neutral contract under the owning feature or `contracts/`, generate or validate transport types, and adapt them into these client domain concepts.
