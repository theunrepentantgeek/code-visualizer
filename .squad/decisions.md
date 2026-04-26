# Squad Decisions

## Active Decisions

### Radial Tree — Type Reference and Layout

**Author:** Dallas  
**Status:** Implemented

**RadialNode struct:**
```go
type RadialNode struct {
    X, Y         float64     // pixel position relative to canvas centre
    DiscRadius   float64     // radius in pixels
    Angle        float64     // angle in radians (0 = right/east)
    Label        string      // directory or file name
    ShowLabel    bool        // render this label
    IsDirectory  bool        // directory node flag
    FillColour   color.RGBA  // zero = use default
    BorderColour *color.RGBA // nil = use default
    Children     []RadialNode
}
```

**LabelMode constants:**
- `LabelAll` — show labels on all nodes
- `LabelFoldersOnly` — directories only
- `LabelNone` — hide all labels

**Layout() function:** `func Layout(root *model.Directory, canvasSize int, discMetric metric.Name, labels LabelMode) RadialNode`
- Root at (0, 0); coordinates relative to canvas centre
- Angle stored on every node for label rotation
- FillColour/BorderColour set by renderer, not layout

---

### Radial Tree — CLI Design

**Author:** Kane  
**Status:** Implemented

**Key flags:**
- `-d/--disc-size` (required, metric.Name) — numeric metrics only
- `-f/--fill` (optional, metric) — fill colour mapping
- `-b/--border` (optional, metric) — border colour mapping
- `--labels all|folders|none` (default: all)
- `--width`, `--height` (default: 1920)

**Canvas size:** `min(width, height)` — square layout for radial geometry

**Config struct:** `config.Radial` with Fill, FillPalette, Border, BorderPalette, Labels fields

---

### Radial Tree — Three-Pass Rendering

**Author:** Parker  
**Status:** Implemented

**Rendering order:**
1. Edges pass — all parent→child lines
2. Discs pass — all filled circles and borders
3. Labels pass — all text labels

**Why:** Single-pass recursion creates z-order problems. Separating passes ensures edges < discs < labels visually.

**Radial label rotation:**
- Right half (angle ≤ π/2 or > 3π/2): rotate by angle, anchor left
- Left half (angle > π/2 and ≤ 3π/2): rotate by angle + π, anchor right
- Root: centred, unrotated

This keeps text upright on both canvas halves.

---

### Radial Tree — Test Coverage

**Author:** Lambert  
**Status:** Complete (16 tests, all passing)

**Coverage:**
- Root positioning (at origin)
- Ring placement (by depth)
- Angular spread (no duplicates, full circle)
- Disc scaling (metric-based)
- Label modes (all three variants)
- Edge cases (empty tree, single child, nested depth)

**Test files:** 
- `internal/radialtree/layout_test.go` (12 tests)
- `internal/render/radialtree_test.go` (4 smoke tests, new)

---

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

### Foliage Palette — Issue #46

**Author:** Dallas  
**Date:** 2026-04-18  
**Status:** Implemented  
**Files:** `internal/palette/palette.go`, `internal/palette/palette_test.go`

**Decision:**
Added `Foliage` palette (`"foliage"`) — an 11-step ordered colour ramp from near-black (dead plant) through brown, orange, yellow, to intense green (healthy plant). Intended for "health" visualisations where low values = dead/brown and high values = healthy green.

**Colour Progression:**
Black → dark brown → brown → dark orange → orange → yellow-orange → yellow → yellow-green → medium green → intense green.

**Rationale:**
- Plant-health metaphor is intuitive for code-health metrics (age, churn, coverage).
- 11 steps matches the temperature palette granularity.
- All adjacent steps pass WCAG 2.0 contrast ratio >= 1.0.
- Palette is `Ordered: true` so it works with the existing numeric metric mapper.

---

### Bubble Tree Visualization — Architecture Proposal

**Author:** Ripley  
**Date:** 2026-04-19  
**Status:** Proposed  
**Issue:** #33 — Add Bubble visualization

**Summary:**
Add `codeviz render bubbletree` — a circle-packing visualization where directories are labelled circles containing nested child circles, and files are unlabelled dots. This follows patterns from GitHub Next and repo-visualizer.

**Architecture highlights:**
- **Package:** `internal/bubbletree/` (mirrors radialtree/treemap pattern)
- **Node type:** `BubbleNode` with `X, Y, Radius` (single radius primitive vs RadialNode's Angle+DiscRadius)
- **Algorithm:** Front-chain circle packing (Wang et al. 2006) + Welzl's enclosing circle — implement in pure Go, no D3 dependency
- **Layout function:** `Layout(root, width, height, sizeMetric, labels) BubbleNode` — takes width/height like treemap (non-square canvas support)
- **CLI:** `BubbletreeCmd` with `--size` flag (not `--disc-size` as in radial); default 1920×1080 (like treemap)
- **Config:** `Bubbletree` struct with Fill/Border/Labels; defaults to `Labels="folders"` (file dots unlabelled)
- **Rendering:** Three-pass PNG (directories→files→labels for z-order); SVG with nested `<g>` groups
- **Reuse:** All existing packages (model, metric, palette, scan, render infrastructure); no new dependencies

**Implementation phases:**
1. Layout engine (front-chain + enclosing circle)
2. PNG rendering (three-pass with fogleman/gg)
3. SVG rendering (nested groups)
4. CLI + Config wiring
5. Optional: curved labels, force-sim polish

**No force-simulation required for v1** — front-chain alone produces good results; polish is a follow-up.

**Open questions:** Minimum file radius, directory transparency alpha, max-depth support, root-level file grouping.

**Risk:** Low (algorithm well-documented, O(n²) is acceptable for typical codebases). Visual quality without force-sim is medium risk — mitigated by front-chain being proven good.

**Decision:** Proceed with this architecture.

---

### Bubble Tree Layout Engine — Algorithm & Constants

**Author:** Dallas  
**Date:** 2026-04-19  
**Status:** Implemented (Phase 1)  
**Issue:** #33

**BubbleNode struct:**
```go
type BubbleNode struct {
    X, Y         float64
    Radius       float64
    Label        string
    ShowLabel    bool
    IsDirectory  bool
    FillColour   color.RGBA
    BorderColour *color.RGBA
    Children     []BubbleNode
}
```

**Layout() signature:** `func Layout(root *model.Directory, width, height int, sizeMetric metric.Name, labels LabelMode) BubbleNode`
- Takes width+height (like treemap, non-square canvas). Returns value type.

**Algorithm choices:**
- **Front-chain packing** without chain pruning. O(n³) per level, acceptable for typical directory sizes (<100 direct children).
- **Welzl's enclosing circle** adapted for circles (not points). Falls back to pairwise enclosing when degenerate.
- **Leaf sizing:** `radius = sqrt(metricValue)` with `minFileRadius = 2px` floor.
- **Padding constants:** `siblingPadding = 3px`, `parentPadding = 6px`.
- **Fallback placement:** Golden angle distribution on outer edge when no valid tangent position found.

**Consequence:** Renderers receive a fully-positioned tree with absolute pixel coordinates. Colours set by renderer/CLI, not layout.

---

### Validation Ordering — Validate() vs validateEffective()

**Author:** Kane  
**Date:** 2026-04-26  
**Status:** Implemented  
**Issue:** #99

**Problem:**
Kong calls `Validate()` on command structs during `ctx.Run()`, BEFORE the command's `Run()` method executes. Config file loading (`TryAutoLoad`) and CLI→config merging (`applyOverrides`) happen inside `Run()`. This means `Validate()` sees empty size fields when `--size`/`--disc-size`/`--bubble-size` wasn't passed on CLI, even though the config file supplies them.

**Decision:**
Split validation into two phases:

1. **`Validate()`** — Only checks CLI-only concerns that don't depend on config file values. Currently: filter glob syntax validation.
2. **`validateEffective()`** — Called from `Run()` after config load + merge + size backfill. Checks size metric existence and kind, fill/border metric-palette validity, and border-palette-requires-border constraint.

**Kong struct tag changes:**
- Removed `required:"true"` from size fields
- Added `default:""` (Kong requires either required or default for enum fields)
- Added leading comma to enum list: `enum:",file-size,file-lines,..."` to accept empty values

**Size backfill pattern in Run():**
After `applyOverrides()`, if the size field is still empty, read it back from the merged config:
```go
if c.Size == "" {
    if s := ptrString(flags.Config.Treemap.Size); s != "" {
        c.Size = metric.Name(s)
    }
}
```

**Applies to:** `TreemapCmd`, `RadialCmd`, `BubbletreeCmd`

**Consequence:** Any future command with config-dependent fields must follow this pattern — keep `Validate()` for CLI-only checks, defer config-dependent validation to `Run()`.

---

### User Directive — PR Review Thread Replies

**Author:** Bevan (via Copilot)  
**Date:** 2026-04-18  
**Status:** Active

**Directive:**
Anytime changes are pushed to address PR review comments, always reply to the review thread confirming the action taken (or explaining why no change was needed).

**Rationale:** User request — reinforces existing team practice of maintaining clear communication on review threads.

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
