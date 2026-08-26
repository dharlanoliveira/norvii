# Normative Assertion Inspection Contract

## Version and compatibility

This contract replaces the direct graph-path relationship projection after the Feature 011 local reset. All active snapshots produced after the reset use assertion paths. A client must treat an empty active corpus as an unavailable retrieval state, not as graph evidence.

## Terminal inspection projection

When graph retrieval contributes evidence, the completed research event includes a bounded `assertionPath` array. Each entry identifies the exact legal statement that supported one cited location:

```json
{
  "assertionPath": [
    {
      "assertionId": "5a67df7b-32a2-4fa1-9d0b-875a6f4f63cb",
      "predicate": "imposes_duty_on",
      "subjectLabel": "data controller",
      "objectLabel": "data controller",
      "establishingLocator": "article-41",
      "evidenceLocator": "article-41-item-2",
      "hierarchyContext": ["chapter-9", "article-41"],
      "qualifier": null
    }
  ]
}
```

## Field rules

| Field | Rule |
| --- | --- |
| `assertionId` | Required stable assertion identifier for one graph result. |
| `predicate` | One allowed normative predicate. |
| `subjectLabel`, `objectLabel` | Required non-empty legal-entity labels. |
| `establishingLocator`, `evidenceLocator` | Required source locations from the active snapshot. |
| `hierarchyContext` | Ordered, bounded root-to-establishing-unit context. It excludes unrelated siblings. |
| `qualifier` | String or null; null means no retained qualification. |

The event may contain at most eight assertion paths. Every path corresponds to one cited active-snapshot evidence location. It contains no prompt, full document content, credential, database query text, or hidden reasoning.

## Safe stage reasons

The existing graph stage may use `no_assertion_evidence` when a ready graph release has no assertion matching the planned predicate, endpoint labels, and requested hierarchy scope. It may use `graph_release_unavailable` when the corpus is empty or has no ready assertion graph release.

When inspection contains an assertion path selected through a hierarchy scope, the stage inspection also includes `scopeLocator`. It is either null or one published locator from the active snapshot, and it never contains a model-invented location.

## Empty corpus state

The catalog and chat boundaries expose an empty corpus state after a completed reset and before fresh activation. The browser must not render stale citations, graph paths, or a successful graph stage for that state.
