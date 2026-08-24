---
target: citation list below a legal RAG answer
total_score: 23
max_score: 40
na_heuristics:
p0_count: 0
p1_count: 3
timestamp: 2026-08-24T16-36-11Z
slug: apps-web-src-features-workspace-researchchat-tsx
---
## Design Health Score

| # | Heuristic | Score | Key issue |
|---|---|---:|---|
| 1 | Visibility of system status | 2/4 | Citation activation has no immediate feedback. |
| 2 | Match between system and real world | 2/4 | `article-474` is a storage-shaped locator, not legal research language. |
| 3 | User control and freedom | 2/4 | The action opens a source but leaves no selected-state cue in the citation row. |
| 4 | Consistency and standards | 3/4 | The visual system is coherent, though chips resemble metadata instead of actions. |
| 5 | Error prevention | 3/4 | Native buttons are sound; unavailable targets need an explanatory state. |
| 6 | Recognition rather than recall | 2/4 | The reader must infer the source and action from rank plus locator. |
| 7 | Flexibility and efficiency | 2/4 | Repeated legal locations create too many nearly identical controls. |
| 8 | Aesthetic and minimalist design | 3/4 | The styling is restrained, but a long row of pills has weak information value. |
| 9 | Help users recover from errors | 2/4 | The citation footer has no immediate source-navigation feedback. |
| 10 | Help and documentation | 2/4 | It does not say that a citation opens an exact legal passage. |
| **Total** | | **23/40** | Needs a citation-information hierarchy, not a new visual language. |

## Design Specificity Verdict

Norvii's paper, teal, and wine visual language is clear, and the quiet research-record disclosure is appropriately subordinate. The citation row is less specific: eight identical tags look like filters or metadata, rather than legal authorities that open a precise passage. Repeated `article-474` labels make distinct retrieved chunks look like distinct sources.

The deterministic scan found no anti-patterns in `ResearchChat.tsx`. Browser visualization was unavailable because no browser automation tool was exposed; review used the supplied screenshot and source inspection.

## Overall Impression

The answer first feels well-grounded, then loses clarity at the citation footer. The biggest opportunity is to represent legal locations rather than raw retrieval chunks.

## What's Working

- One-click source navigation establishes direct provenance.
- Citation chips appear before diagnostics, prioritizing evidence over technical metadata.
- The current palette distinguishes citations without making them look like warnings.

## Priority Issues

1. **[P1] Repeated opaque locators** - Four chips for `article-474` suggest four authorities when they are passages from one legal location. Group footer citations by source and legal location, for example `Art. 474 - 4 supporting passages`; retain individual chunk/rank information in the research record.
2. **[P1] Weak citation semantics** - Rounded tags look like filters. Use compact citation links or a small source inventory with legal labels such as `1 - LGPD, Art. 474`; preserve inline answer markers for exact claim mapping.
3. **[P1] No feedback after navigation** - Mark the activated citation, and announce that the source location opened. The source reader's selected passage should remain the durable confirmation.
4. **[P2] Accessibility and action clarity** - Give each control an explicit accessible name such as `Open Article 474 in Official LGPD text`, a visible focus state, a 32-36px target, and an unavailable explanation when navigation cannot occur.
5. **[P2] Citation scale** - For long answers, initially show three unique locations and a `View more sources` control. Never hide evidence permanently.

## Persona Red Flags

- **Legal researcher**: Cannot immediately distinguish one authority with several passages from several authorities.
- **First-time policy analyst**: May not know that the mint controls open the Source tab or what rank labels signify.
- **Keyboard or screen-reader user**: Must navigate many terse, similar controls without adequate action context.

## Minor Observations

- Localize legal labels: `Art.` in Portuguese and `Article` in English.
- Sort visible source groups by source and legal order, not retrieval rank.
- Use a tooltip only to clarify supporting-passage counts; do not expose retrieval jargon in the primary path.

## Questions to Consider

- Is the reader verifying legal locations or individual retrieved chunks?
- Should source authority be more prominent than retrieval rank?
- When an answer has many citations, should the default emphasize the first three locations or the number of sources?
