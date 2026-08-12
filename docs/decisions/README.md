# Architecture Decision Records

Use an Architecture Decision Record (ADR) for a consequential choice that affects
multiple features, is expensive to reverse, or changes a durable boundary. Examples
include database selection, chat streaming protocol, ingestion dispatch, model
provider strategy, and whether GraphRAG needs a dedicated graph database.

## Naming

Use `NNNN-short-decision-title.md`, starting at `0001`. Numbers are never reused.

## Status

An ADR has one of these statuses:

- `Proposed`: under evaluation and not an implementation constraint;
- `Accepted`: current project decision;
- `Superseded`: replaced by a linked ADR;
- `Deprecated`: retained for history but no longer recommended.

## Required content

Copy [the ADR template](template.md) and record context, decision drivers, considered
options, decision, consequences, verification, and links to the feature that made the
choice. Do not use an ADR as a substitute for feature requirements or task tracking.

## Decision boundary

- The constitution records non-negotiable project principles.
- An ADR records why a durable technical choice was made.
- A feature plan records how that choice applies to one capability.
- Code and tests are the executable implementation.

## Accepted decisions

| ADR | Decision |
| --- | --- |
| [0001](0001-three-module-architecture.md) | Use React, Go, and Python production modules under `apps/` |
| [0002](0002-corpus-and-source-model.md) | Model isolated corpora with PDF or URL sources |
| [0003](0003-spec-driven-delivery.md) | Deliver product capabilities through Spec Kit features |
| [0004](0004-executable-react-prototype.md) | Approve an executable React prototype before production implementation |
