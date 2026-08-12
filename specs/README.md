# Feature Specifications

Each numbered directory is a complete Spec Kit workspace for one vertical product
capability.

```text
specs/NNN-feature-name/
|-- spec.md                  # What and why
|-- plan.md                  # Technical approach and module impact
|-- research.md              # Evidence for technical decisions
|-- data-model.md            # Entities, states, ownership, invariants
|-- contracts/               # Feature-owned schemas
|-- quickstart.md            # Reproducible acceptance path
|-- tasks.md                 # Ordered, traceable implementation tasks
`-- checklists/              # Optional requirement-quality checks
```

## Naming and scope

- Let `speckit-specify` allocate the number and create the feature workspace.
- Use a short kebab-case name that describes user value.
- Keep one feature independently demonstrable.
- Link to global rules instead of copying them.
- Keep implementation choices out of `spec.md`; place them in `plan.md` and
  `research.md`.
- Do not create a feature directory manually after implementation has started.

## Required traceability

Requirements use stable IDs such as `FR-001`. Tasks reference the owning user story
and requirement IDs. Tests name the behavior they protect. Pull requests or commits
link to the feature directory.

## Lifecycle status

The status in `spec.md` is one of `Draft`, `Specified`, `Planned`, `In Progress`,
`Verified`, or `Deferred`. An implementation is complete only when the feature is
`Verified` and its completion evidence is current.

See the [feature map](../docs/product/feature-map.md) for proposed sequencing and the
[development workflow](../docs/development/spec-driven-development.md) for the full
process.
