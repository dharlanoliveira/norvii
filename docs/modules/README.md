# Module Models

These documents define stable ownership for the four application modules. They are
not implementation plans and do not select packages that remain undecided.

All production modules live under `apps/`. Executable product exploration lives under `prototypes/` and is governed by the [prototyping workflow](../development/prototyping.md), not by the production module models.

- [Web client](web-client.md)
- [Go API](go-api.md)
- [Python LangGraph agent](python-agent.md)
- [Python ingestion](python-ingestion.md)

## Shared rules

- Domain vocabulary is `Corpus`, `Source`, `IngestionArtifact`, `Citation`, and
  `RetrievalTrace` unless a feature specification introduces a better term.
- A module exposes contracts, not its internal data structures.
- Each module validates untrusted data at its boundary.
- Framework and vendor types stop at adapters and do not define domain behavior.
- Time, IDs, I/O, external models, and network access are injectable at useful test
  seams.
- Cross-module changes start with a schema and compatibility decision.
- Each module documents and runs its own tests; end-to-end checks verify the composed
  user journey.
