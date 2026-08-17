# Architecture Risks

This register contains known cross-feature risks. The feature that introduces or materially changes a risk owns its concrete mitigation tasks and verification.

| Risk | Impact | Current mitigation direction |
| --- | --- | --- |
| Incorrect legal-structure extraction | Citations point to the wrong provision | Preserve legal units and validate representative samples for each document |
| Jurisdiction or corpus mixing | Legally misleading answer | Require an active corpus and isolate every retrieval operation |
| Unsupported model claims | Hallucinated legal statements | Require citations, verify claim support, and abstain on weak evidence |
| Expensive graph extraction | POC cost exceeds demonstrated value | Extract only entities and relations justified by evaluation questions |
| Changing source content | Evaluation stops being reproducible | Version source URL, capture date, hash, and corpus snapshot |
| Complex PDF layout | Fragmented or reordered text | Prefer official structured HTML when justified and retain PDF identity for reference |
| External URL changes or disappears | Citation no longer matches current page | Preserve captured text, hash, and capture date as versioned artifacts |
| External URL reaches private networks | SSRF and internal network exposure | Validate protocol, DNS, resolved IP, redirect chain, size, and timeout before capture |
| Large PDF binaries | Database and backup growth | Set a low POC upload limit and track total corpus size |
| Stale or partial graph projection | GraphRAG omits or misrepresents canonical evidence | Publish versioned releases, checkpoint projection, detect lag, and rebuild Neo4j from PostgreSQL |
| Model-derived assertion treated as fact | Users infer unsupported legal certainty | Store attributed statements separately from official metadata and require exact evidence spans and review state |
| Python and Go schema drift | Unreadable artifacts or failed ingestion | Use versioned schemas and provider-consumer contract tests |
| Go stream differs from client semantics | Broken message parts or tool rendering | Keep stream types small and verify them with contract tests |
| Excessive technical presentation | Demonstration value is difficult to understand | Keep the default answer simple and place traces in inspection views |

## Review rule

Review this register during feature planning. Promote a mitigation into acceptance criteria or tasks when the feature can trigger the risk. Remove a risk only when the implementation and tests make it no longer applicable; preserve the rationale in the feature or ADR that resolved it.
