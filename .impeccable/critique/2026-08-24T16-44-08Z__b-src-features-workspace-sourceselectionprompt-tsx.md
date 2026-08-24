---
target: Source tab empty selection state
total_score: 28
max_score: 40
na_heuristics:
p0_count: 0
p1_count: 2
timestamp: 2026-08-24T16-44-08Z
slug: b-src-features-workspace-sourceselectionprompt-tsx
---
## Design Health Score

| # | Heuristic | Score | Key issue |
|---|---|---:|---|
| 1 | Visibility of system status | 3/4 | The Source tab is active, but the panel does not acknowledge the ready source. |
| 2 | Match between system and real world | 3/4 | Source and library fit research work; Source desk is poetic rather than operational. |
| 3 | User control and freedom | 3/4 | Source selection is available, with no direct fast path from the empty state. |
| 4 | Consistency and standards | 4/4 | Typography, paper palette, and document motif fit the workspace. |
| 5 | Error prevention | 2/4 | The ready source can be mistaken for passive metadata. |
| 6 | Recognition rather than recall | 3/4 | The library remains visible, but the panel does not name a ready source. |
| 7 | Flexibility and efficiency | 2/4 | Every Source-tab visit requires scanning and a separate selection action. |
| 8 | Aesthetic and minimalist design | 3/4 | The design is refined, but oversized empty-state elements consume workspace. |
| 9 | Help users recover from errors | 3/4 | The state is recoverable, but has no specific unavailable-source guidance. |
| 10 | Help and documentation | 2/4 | The copy does not explain the value of opening a source. |
| **Total** | | **28/40** | Strong visual identity; weak operational handoff. |

## Design Specificity Verdict

The source-selection panel is distinctly Norvii: editorial, calm, and credible for legal reading. It is not a true empty state, however. A ready source already exists in the library and the large central state only instructs the user to act elsewhere. The result is ceremonial when it needs to be operational.

The deterministic scan found no anti-patterns in `SourceSelectionPrompt.tsx`. Browser inspection was unavailable because no browser automation tool was exposed; the review used the supplied screenshot and source inspection.

## Overall Impression

The first impression is composed and trustworthy, but the task stalls at the moment a researcher should open evidence. The greatest opportunity is to make the ready source itself the primary next action.

## What's Working

- The document motif, grid, and restrained palette establish a credible source-reading environment.
- The Source library remains visible while the central panel explains the state.
- The decorative document is correctly hidden from assistive technology, and the panel has a labelled heading.

## Priority Issues

1. **[P1] Indirect primary action** - The panel says to select a source but does not offer a source action, directional cue, or visible signal that the ready row is interactive. Add a primary `Open <source title>` control for one ready source, or a `Focus source library` control when multiple sources exist. Keep the sidebar as the canonical library.
2. **[P1] Copy does not explain the value** - `Source desk` and the body copy describe a location, not the research outcome. Explain that opening a source exposes preserved legal text and cited provisions.
3. **[P2] Excessive visual mass** - A 54px heading, large illustration, and generous empty floor give a simple pause too much weight. Retain the document motif, reduce the vertical footprint, and bring the action nearer the library seam.
4. **[P2] No keyboard route within the state** - The panel has no focusable action that satisfies its own instruction. Add an accessible source action, preserve focus feedback after selection, and use at least 12px for secondary explanatory copy.
5. **[P3] Misleading ordinal** - The decorative `01` appears to identify a selected document even though none is open. Remove it unless it derives from true collection data.

## Persona Red Flags

- **First-time legal researcher**: may add a redundant URL or PDF because those controls appear more actionable than the ready source row.
- **Repeat researcher**: pays a selection cost on every return to the Source tab despite an available source.
- **Keyboard, screen-reader, or low-vision user**: must leave the explanatory state to discover a source-tree action and contend with very small supporting text.

## Minor Observations

- Use an action label without terminal punctuation when it functions as an instruction.
- The bottom-right hairline preserves the visual language but does not help orientation.
- On mobile, the illustration and heading may push the actual source action away from the user.

## Questions to Consider

- When a corpus has one ready source, should it open automatically?
- Should the Source tab inspect the currently cited or opened source instead of requiring reselection?
- Can the panel emphasize research value rather than merely explain where documents appear?
