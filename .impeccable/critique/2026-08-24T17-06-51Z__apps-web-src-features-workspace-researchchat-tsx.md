---
target: empty chat presentation
total_score: 25
max_score: 40
na_heuristics:
p0_count: 0
p1_count: 3
timestamp: 2026-08-24T17-06-51Z
slug: apps-web-src-features-workspace-researchchat-tsx
---
## Design Health

| # | Heuristic | Score | Key issue |
| --- | --- | --- | --- |
| 1 | Visibility of system status | 3/4 | Scope is clear, but source readiness is not actionable. |
| 2 | Match to real world | 3/4 | Question-first language works; the generic chat medallion does not express legal research. |
| 3 | User control and freedom | 3/4 | Composer is available, but the entry state offers no productive starting path. |
| 4 | Consistency and standards | 3/4 | Typography and palette fit; the medallion is disconnected from the source-library language. |
| 5 | Error prevention | 3/4 | Scope note sets expectations, but no starter prompts prevent unfocused questions. |
| 6 | Recognition rather than recall | 2/4 | The user must remember source themes and invent a question. |
| 7 | Flexibility and efficiency | 2/4 | The composer is visually distant and there are no one-click starting questions. |
| 8 | Aesthetic and minimalist design | 2/4 | Empty-state ceremony and repeated scope copy outweigh the task. |
| 9 | Error recognition and recovery | 2/4 | The entry state does not explain evidence insufficiency or citations. |
| 10 | Help and documentation | 2/4 | Grounding is stated but not made concrete at the point of action. |

Total: 25/40.

## Design Specificity Verdict

The editorial legal type, paper palette, and evidence-boundary language are recognizably Norvii. The empty-state medallion with a generic message icon is interchangeable with a generic assistant product and becomes the primary visual cue without explaining the legal research task.

## Priority Issues

1. Remove the generic message medallion rather than replacing it with another decorative icon.
2. Replace the tall centred hero with a compact, reading-width entry state that brings the composer close to the headline.
3. Add two or three corpus-safe starter questions through the existing chat submission path.

## Recommended Direction

Use a ready-to-research state on the same reading rail as answers: a factual source-ready status, a compact title, one sentence that answers cite source locations, the composer, and a small set of starter prompts. Keep a single scope statement under the composer and remove the duplicate kicker.

## Accessibility and Responsiveness

The composer remains the primary keyboard destination. Starter questions must be labelled buttons and use the normal send path. At narrower widths, the prompt list should wrap without creating horizontal overflow.
