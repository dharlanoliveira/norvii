# Python Guidelines

## Design

- Use modern Python with complete type hints on public and internal application code.
- Prefer classes for domain services, ingestion stages, adapters, repositories, and components that own state, dependencies, invariants, configuration, lifecycle, or multiple related behaviors.
- Keep each class cohesive and give it one reason to change. Inject collaborators through the constructor.
- Prefer composition over inheritance. Use inheritance only for a stable substitutable relationship.
- Use `dataclass` or validated models for structured values. Prefer immutable value objects when mutation is unnecessary.
- Use `Protocol` for structural ports when multiple implementations or isolated testing require a boundary.
- Keep pure functions for small stateless transformations, predicates, parsing helpers, and algorithms. Do not wrap a function in a class solely to satisfy a style preference.
- Avoid god objects, static-method containers, service locators, mutable module globals, and generic `utils.py` modules.

## Ingestion pipeline

- Model each ingestion stage with a narrow input and output contract.
- Separate source acquisition, extraction, normalization, chunking, enrichment, embedding, and persistence.
- Make retries idempotent. Persist enough state to resume or diagnose a failed source.
- Preserve source identity, hashes, locations, and extraction metadata through every stage.
- Bound network downloads, PDF size, page count, memory use, and processing time.
- Validate URLs before access and defend against redirects to private or link-local networks.
- Keep provider SDK objects out of domain and artifact contracts.

## Errors and resources

- Raise domain-specific exceptions at boundaries where callers need recovery decisions.
- Preserve causes with exception chaining and add actionable context.
- Do not catch `Exception` unless adding context, recording failure, or translating at a boundary; re-raise appropriately.
- Use context managers for files, database sessions, network clients, and temporary resources.
- Keep asynchronous and synchronous boundaries explicit. Do not hide event-loop management inside reusable code.

## Testing and tooling

- Use pytest-style behavior tests unless the repository establishes another convention.
- Test classes through public methods and observable state, not private implementation calls.
- Use temporary resources and deterministic fakes for I/O boundaries.
- Add contract tests for artifacts consumed by Go.
- Run the configured formatter, linter, type checker, and relevant tests. If the project has not selected tools yet, propose Ruff, mypy or Pyright, and pytest rather than silently adding them.
