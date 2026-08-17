<!--
Sync Impact Report
- Version change: 1.1.0 -> 1.2.0
- Modified principles:
  - none
- Added principles:
  - VIII. English as the Engineering Language
- Added sections: none
- Removed sections: none
- Templates: compatible; commands read the constitution at runtime
- Runtime guidance:
  - updated: AGENTS.md
  - updated: CONTRIBUTING.md
  - updated: .agents/skills/norvii-code-quality/SKILL.md
  - updated: docs/README.md
  - updated: docs/development/continuous-integration.md
  - updated: specs/001-product-experience-prototype/spec.md
- Enforcement:
  - added: .github/scripts/validate_repository_language.py
  - added: .github/scripts/tests/test_validate_repository_language.py
  - updated: .github/workflows/ci.yml
- Deferred items: none
-->
# Norvii Constitution

## Core Principles

### I. Specification Before Implementation

Every product capability MUST have a numbered feature directory under `specs/`
before application code is written. Its `spec.md` MUST define user value,
independently testable scenarios, scope boundaries, failure behavior, and measurable
success criteria. Implementation MUST be based on an approved `plan.md` and
traceable `tasks.md`. A task is complete only when its requirement and verification
evidence can be identified.

The initial product experience MUST be validated in the numbered executable prototype feature under `prototypes/web/`. Production application code under `apps/` MUST NOT be implemented until that feature is `Verified` and its approval gate passes. Production modules MUST NOT import prototype code, fixtures, or styles. Later production UI features MUST link to the approved prototype baseline and document intentional differences.

Rationale: Norvii is a reference project. The reasoning that led to the code is part
of the deliverable and must be reviewable.

### II. Vertical Features and Explicit Module Boundaries

Features MUST be planned as vertical increments of demonstrable behavior. A feature
may change the React client, Go API, Python ingestion service, contracts, and local
infrastructure, but each responsibility MUST remain owned by exactly one module.
Modules MUST communicate through explicit public contracts and MUST NOT import one
another's internal implementation. Shared abstractions are allowed only when they
represent an actual stable contract.

Rationale: vertical delivery exposes working value early while clear ownership keeps
the three-language system loosely coupled.

### III. Evidence-Grounded Legal Answers

The active corpus MUST be an enforced retrieval boundary. Every material legal claim
MUST cite evidence from that corpus, and the application MUST abstain when evidence
is missing, weak, or conflicting. Citations MUST resolve to a stable source location
such as a document, article, section, page, or captured excerpt. Normative text and
interpretative guidance MUST remain distinguishable. The product MUST state that it
is a technical demonstration and not legal advice.

Rationale: citation quality and corpus isolation are core product behavior, not
optional presentation details.

### IV. Versioned Cross-Language Contracts

Data exchanged between TypeScript, Go, and Python MUST be defined in versioned,
language-neutral contracts stored under `contracts/` or the owning feature's
`contracts/` directory. Contracts MUST define inputs, outputs, errors, invariants,
and compatibility expectations. Contract changes MUST include consumer and provider
verification. Generated types MUST be derived from the contract; one module MUST NOT
become the implicit schema source for another.

Rationale: explicit contracts prevent runtime drift across independently evolving
language ecosystems.

### V. Idiomatic, Tested, Maintainable Code

All code and configuration MUST follow `AGENTS.md` and the
`norvii-code-quality` skill. Designs MUST favor high cohesion, low coupling, clear
domain vocabulary, dependency injection at useful boundaries, and the smallest
abstraction that protects real behavior. Python MUST favor cohesive classes for
stateful pipeline stages and domain behavior, while pure functions remain valid for
small stateless transformations. Changed behavior, failure paths, and public
boundaries MUST have proportionate automated tests. Tests MUST be deterministic and
MUST fail for the missing behavior they protect.

Rationale: the repository is intended to demonstrate engineering judgment, not only
framework integration.

### VI. Reproducible and Cost-Bounded POC

The local environment MUST be reproducible from versioned instructions and pinned
dependencies. Backing services MUST run through `infra/compose.yaml`, use health
checks, and avoid floating image tags. Corpus inputs MUST record official origin,
capture date, hash, and a versioned manifest. Features that invoke models or process
documents MUST define practical limits for document size, token use, retries, and
evaluation scope. Complexity that does not improve a stated POC outcome MUST be
deferred.

Rationale: a small, repeatable demonstration is more valuable than a broad system
whose cost or results cannot be reproduced.

### VII. Observable and Safe by Design

Retrieval strategy, selected evidence, graph paths, model identity, latency, token
use, tool calls, ingestion state, and actionable failures MUST be inspectable where
they affect product claims. Logs MUST be structured and MUST NOT expose secrets,
complete document content, prompts, or personal data. File and URL ingestion MUST
validate type, size, protocol, redirects, resolved addresses, and timeouts before
processing. Security, privacy, and observability work MUST be included in the feature
that introduces the risk, not postponed to a generic cleanup phase.

Rationale: a trustworthy AI demonstration must make its behavior explainable without
creating avoidable data or network risk.

### VIII. English as the Engineering Language

All project-owned source code, technical documentation, specifications, plans,
tasks, ADRs, runbooks, configuration, schemas, migrations, tests, logs, error
messages, comments, docstrings, commit messages, and review artifacts MUST be written
in English. Identifiers, package and module names, file and directory names,
localization keys, and test descriptions MUST also use English. English is the
project's default product and engineering language.

Portuguese is permitted only as user-facing content in Portuguese localization
resources, as legal source or corpus content, as a quotation or citation that must
preserve its source, or as deterministic fixture data and test literals that verify
such content. The surrounding code, keys, structure, explanations, and assertions
MUST remain in English. Legal source content MUST NOT be translated automatically to
satisfy this principle. Third-party managed artifacts MAY retain their upstream
language, but project-owned integrations and explanations MUST use English.

Rationale: one engineering language makes the multilingual repository consistent,
searchable, and reviewable without conflating interface localization with technical
communication or altering authoritative legal material.

## Authoritative Project Context

`docs/README.md` defines the single source of truth for product scope, architecture,
module ownership, operations, risks, and decisions. Feature specifications and plans
MUST link to those documents instead of copying their rules.

Accepted ADRs are binding until superseded. Open choices in
`docs/decisions/backlog.md` MUST remain undecided until an owning feature records
research and, when required, an ADR. Codex MUST NOT select an open technology during
unrelated implementation work.

If global documents conflict, this constitution governs principles, accepted ADRs
govern durable choices, and the source registered in `docs/README.md` governs its
named subject. The conflict MUST be resolved before feature implementation.

## Feature Delivery Workflow

1. Start from `docs/product/overview.md` and `docs/product/feature-map.md`, then create
   one numbered feature with `speckit-specify`.
2. Complete and approve the executable prototype feature before creating production implementation tasks under `apps/`.
3. Run `speckit-clarify` when ambiguity changes scope, data ownership, contracts,
   safety, or cost.
4. Use `speckit-plan` to record research, affected modules, contracts, data model,
   operational changes, and verification strategy.
5. Use `speckit-tasks` to produce tasks grouped by independently testable user story,
   with exact paths and requirement identifiers.
6. Run `speckit-analyze` before implementation. Critical inconsistencies MUST be
   resolved before code changes begin.
7. Use `speckit-implement` in task order. Each user story MUST end at a runnable,
   testable checkpoint.
8. Update global documentation or create an ADR only when the feature changes a
   durable project rule or architectural decision. Feature-specific detail stays in
   its numbered feature directory.

A feature is done only when its acceptance scenarios pass, required quality checks
pass, contracts and operational instructions match the implementation, and its
quickstart can be followed from a clean environment.

## Governance

This constitution is the highest-priority project guidance. Feature specifications,
plans, tasks, skills, and code reviews MUST verify compliance. Any exception MUST be
listed in the feature plan's Complexity Tracking table with the rejected simpler
alternative and an explicit removal or review condition.

Amendments require a documented rationale, updates to affected templates and runtime
guidance, and a semantic version change. Removing or weakening a principle requires a
major version; adding or materially expanding governance requires a minor version;
clarifications require a patch version. Compliance MUST be checked before research
and again after feature design.

**Version**: 1.2.0 | **Ratified**: 2026-08-12 | **Last Amended**: 2026-08-12
