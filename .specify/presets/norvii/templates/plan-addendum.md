
## Norvii Implementation Requirements *(mandatory)*

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | [Change or No change] | [Prototype behavior] | [Checks or N/A] |
| `apps/web/` | [Change or No change] | [Client behavior] | [Checks or N/A] |
| `apps/api/` | [Change or No change] | [Online backend behavior] | [Checks or N/A] |
| `apps/ingestion/` | [Change or No change] | [Offline pipeline behavior] | [Checks or N/A] |
| `contracts/` | [Change or No change] | [Cross-language contract] | [Checks or N/A] |
| `infra/` | [Change or No change] | [Local service] | [Checks or N/A] |

### Boundaries and Constraints

- **Cost limits**: [Document, token, model, storage, and runtime limits or N/A]
- **Prototype baseline**: [Verified prototype feature, screenshots, and intentional deviations or N/A]
- **Public contracts**: [Compatibility, ownership, and version impact or N/A]
- **Persistence**: [Schema, migration, binary, retention, and rollback impact or N/A]
- **Ingestion artifacts**: [Version and publication impact or N/A]
- **Streaming**: [Message-part and error compatibility impact or N/A]
- **Corpus boundary and citations**: [Enforcement and verification or N/A]
- **Security and privacy**: [Untrusted input, secrets, logs, SSRF, and upload impact]
- **Observability**: [Metrics, traces, states, and safe diagnostic context]
- **Local environment**: [Compose, migration, health check, and quickstart impact]

### Repository Paths

Use only the modules marked as changed above:

```text
apps/web/                    # React and TypeScript production client
apps/api/                    # Go online backend
apps/ingestion/              # Python offline pipeline
prototypes/web/              # Executable React product prototype
contracts/                   # Stable cross-language schemas
infra/                       # Local backing services
docs/                        # Durable global documentation
```
