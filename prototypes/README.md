# Norvii Prototypes

This directory contains executable, disposable-by-design product experiments. Prototypes validate journeys, interaction states, content, and visual direction before production implementation.

## Current prototypes

- [`web/`](web/): React prototype for the Norvii product experience.

## Boundary rules

- Prototypes use deterministic fixtures and in-memory adapters.
- Prototypes do not call production APIs, databases, LLMs, or ingestion services.
- Production applications under `apps/` do not import from `prototypes/`.
- A production feature may intentionally reimplement an approved pattern; it does not promote the directory wholesale.
- Prototype limitations and approval status are documented by the owning Spec Kit feature.

See the [executable prototyping workflow](../docs/development/prototyping.md).
