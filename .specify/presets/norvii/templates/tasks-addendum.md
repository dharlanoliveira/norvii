
## Norvii Task Requirements *(mandatory)*

- Tests are required for changed behavior, failure paths, and public boundaries.
- Every implementation and test task must identify its owning requirement IDs.
- Every task must name exact repository paths.
- Use only modules marked as changed in `plan.md`.
- Keep test tasks with the user story and requirement they protect.
- Commit complete logical groups; do not create partial or mechanically generated checkpoints.

### Path Conventions

- Web client: `apps/web/`
- Go API: `apps/api/`
- Python ingestion: `apps/ingestion/`
- Web prototype: `prototypes/web/`
- Shared contracts: `contracts/`
- Local infrastructure: `infra/`
- Durable documentation: `docs/`
