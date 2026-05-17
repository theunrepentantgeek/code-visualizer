# Squad Decisions — Archive

Archived entries (older than 30 days).

## Archived Decisions

### Kong struct fields use pointers; defaults in config.New()

**Author:** Dallas  
**Date:** 2026-04-15  
**Status:** Implemented

**Context:** PR review identified that `Labels string \`default:"all"\`` in `RadialCmd` caused Kong to always write `"all"` into `c.Labels`, silently ignoring user-configured defaults.

**Decision:** All Kong CLI struct fields that mirror a `config.*` pointer field must use pointer-compatible semantics:
- Remove `default:` tags from Kong string fields with corresponding `*string` in config
- Add `""` as first value in `enum:` so Kong accepts unset/empty state
- Handle defaults in `config.New()` only — single authoritative source

**Consequence:** `config.New()` sets `Radial.Labels = "all"`. CLI `--labels` only overrides on explicit user flag. `resolveLabels` simplified to single code path.

**Pattern:** Apply to any future CLI flags mapping to config pointer fields.

---

### Render + Layout Fixes — Code Quality

**Author:** Parker  
**Date:** 2026-04-15  
**Status:** Implemented  
**Files:** `internal/render/radialtree.go`, `internal/radialtree/layout.go`

**RenderRadialPNG signature:** Takes `*radialtree.RadialNode` pointer (Dallas updates call site; Lambert uses `&node` in tests).

**External label colour:** All non-root labels use fixed dark constant `#222222` (canvas background is white; disc-fill-based colour always wrong).

**Disc z-order:** Collect-sort-draw in `drawDiscs` (no recursion) guarantees smaller nodes render on top regardless of traversal order.

**Stroke batching:** One `dc.Stroke()` per node level (after all child edges) reduces GPU/CPU round-trips.

**Crowding prevention:** 
- Ring spacing floor: minimum ≥ `n * (2*minFileDisc + 4)` pixels
- Disc shrink: `adjustedDiscFactor()` reduces max when `n > π/maxDiscFactor`
- `dirDiscFactor` halved (`0.12 → 0.06`) for proportionate directory dots

**Docs:** `layout.go` computeLeafCount doc fixed (returns 0; callers guard).

---
