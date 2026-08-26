# Data Model: MCP Research Tools and Skills

## Tool Invocation

Each corpus-scoped invocation requires a corpus UUID. The service resolves an enabled
corpus's active published snapshot and includes its identifier in completed results.

| Outcome | Meaning |
| --- | --- |
| `completed` | A bounded result from the selected snapshot. |
| `not_found` | The corpus, snapshot resource, or legal location is absent. |
| `unavailable` | A required graph release is not ready. |
| `invalid_input` | The request is empty or fails field validation. |

## Evidence Reference

Evidence contains the source ID, immutable document version ID, legal-unit locator,
offsets, and selected snapshot identity. A returned reference never identifies an
unpublished candidate document or another corpus.
