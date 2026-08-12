# Go Guidelines

## Design

- Follow standard Go conventions and prefer the standard library before adding dependencies.
- Organize packages around cohesive capabilities. Avoid packages named `common`, `shared`, `helpers`, or `utils` unless their scope is genuinely narrow.
- Keep package APIs small. Export only what another package must use.
- Define interfaces at the consuming side and keep them minimal. Accept interfaces and return concrete types by default.
- Use structs with explicit constructors when construction must validate invariants or provide required dependencies.
- Prefer composition and plain data flow over inheritance-like embedding or framework-driven indirection.
- Avoid a package-per-layer layout when it separates code that changes together.

## Behavior and concurrency

- Pass `context.Context` as the first parameter for request-scoped cancellation, deadlines, and metadata. Do not store it in a struct.
- Wrap errors with operation context using `%w` when callers need the cause.
- Use sentinel or typed errors only when callers must branch on error identity.
- Handle every error intentionally. Do not panic for expected runtime failures.
- Make goroutine ownership and termination explicit. Avoid starting background work without cancellation and cleanup.
- Protect shared state deliberately and prefer message passing only when it makes ownership clearer.
- Close resources at the layer that acquires them.

## HTTP, persistence, and domain boundaries

- Keep transport DTOs separate from domain types when their validation or evolution differs.
- Decode, validate, and bound request bodies at the HTTP boundary.
- Keep handlers thin: translate transport data, invoke an application capability, and map the result.
- Put transaction ownership around the complete business operation, not inside unrelated repository calls.
- Keep SQL explicit and observable. Prevent unbounded queries and N+1 access patterns.
- Never expose database models as API contracts by accident.

## Testing and tooling

- Prefer table-driven tests when cases share behavior, not as a reflex for every test.
- Use fakes for stable domain ports; avoid mocks that duplicate internal call sequences.
- Test handlers, persistence, and cross-service contracts at their boundaries.
- Run `gofmt` on changed Go files.
- Run `go vet ./...`, relevant `go test` targets, and `go test ./...` when repository size permits.
- Run the race detector for changed concurrent behavior when feasible.
