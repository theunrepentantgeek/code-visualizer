# Parker — History

## Core Context

- **Project:** A Go CLI tool (`codeviz`) that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Staff Developer — technical quality, debt, maintainability, and long-term viability
- **Joined:** 2026-04-15
- **Requested by:** Bevan Arps

## Project Knowledge

- Language: Go 1.26+
- Key packages: `cmd/codeviz/` (entry), `internal/metric/`, `internal/palette/`, `internal/render/`, `internal/scan/`, `internal/treemap/`
- Build: `task build` → `bin/codeviz`
- Test: `task test` (`go test ./... -count=1`)
- Lint: `task lint` (golangci-lint v2 with nilaway, wsl_v5, revive, wrapcheck, gci)
- Format: `task fmt` (gofumpt)
- Full CI: `task ci` (build + test + lint)
- Error handling: eris wrapping throughout
- Test assertions: Gomega (not testify); golden files via Goldie v2
- Formatting enforced by gofumpt; import ordering by gci

## Learnings

<!-- Append learnings below -->

### RenderRadialPNG (2026-04-15)

- **Signature:** `func RenderRadialPNG(root radialtree.RadialNode, canvasSize int, outputPath string) error`  
  Located in `internal/render/radialtree.go`. Square canvas only; all node positions are offsets from canvas centre.

- **Three-pass rendering:** edges → discs → labels. Each pass is a full recursive traversal of the tree. Required to avoid z-order issues (e.g., parent discs drawn over child edges).

- **Label rotation:** Right half uses `RotateAbout(node.Angle)` + left anchor (ax=0). Left half uses `RotateAbout(node.Angle + π)` + right anchor (ax=1). This flips the baseline direction so characters stay upright. Root node (dist=0) gets an unrotated centred label.

- **Colour defaults:** file fill `#CCCCCC`, directory fill `#444444`, border `#333333`, edge `#999999`. Custom colours applied if `FillColour.A > 0` (fill) or `BorderColour != nil` (border).

- **Dallas's radialtree package** (`internal/radialtree/`) was already in progress when this renderer was written: `node.go` defines `RadialNode`, `layout.go` defines `Layout`. The `render_cmd.go` already references `RadialCmd` (pre-existing lint failure, not mine to fix).

### Render + Layout fixes (2026-04-15)

- **RenderRadialPNG signature:** Updated to take `*radialtree.RadialNode` (pointer). Internal draw functions still take by value via `*root` dereference at the top level.

- **Stroke batching:** `drawEdges` now calls `dc.Stroke()` once per node level (after the loop over children) rather than once per edge. The recursive call stays inside the loop so children are still fully processed. Net effect: one stroke call per tree node instead of one per edge.

- **Z-order for discs:** `drawDiscs` was refactored from a recursive traversal into a two-phase collect-then-draw approach. `collectDiscs` gathers all `(node, sx, sy)` tuples; they are sorted by `DiscRadius` descending and drawn via `drawSingleDisc`. Larger nodes render first, so smaller nodes always appear on top.

- **Label colour bug:** External labels (non-root nodes) were using `TextColourFor(fill)` against the disc fill colour, but the label is positioned on the white canvas — so dark-filled directory nodes produced white (invisible) text. Fixed by adding `radialLabelColour = #222222` used for all non-root labels. Root label (on-disc) still uses `TextColourFor(effectiveFill(node))`.

- **Angle normalisation:** Replaced O(n) `for` loops in `drawLabels` with `math.Mod(angle, 2π)` + a single `if angle < 0` guard.

- **dirDiscFactor reduced:** `0.12` → `0.06` in `layout.go` so directory dots are proportional to small file nodes.

- **Crowding prevention:** `Layout()` now adjusts `ringSpacing` upward when depth-1 has many children (ensures minimum circumference for `n * (2*minFileDisc + 4px)` nodes), and reduces `maxDiscFactor` via `adjustedDiscFactor()` when discs would overlap even after spacing is increased.

- **computeLeafCount doc fix:** The old comment claimed it returned 1 for empty dirs — wrong. The function returns 0; callers guard with `if leafCount == 0 { leafCount = 1 }`. Comment corrected to match reality.
### 2026-04-15: PR #39 Review — Provider Interface Extension

Reviewed PR #39 (issue #38) adding `Scope()` and `Description()` to `provider.Interface` plus 9 new folder-level metrics.

**Patterns worth preserving:**
- Scope type uses string constants — extensible without interface changes
- Helper functions in `folder/metrics.go` compose well for aggregation (sum, min, max, mean)
- `model.WalkDirectories()` uses post-order traversal for bottom-up aggregation
- Dependency declarations enable correct scheduling of file→folder metric chains

**Minor debt identified (not blocking):**
- `FolderAuthorCountProvider` doesn't declare dependency on file metrics (queries git directly)
- Binary file handling differs between MeanFileLines (skips) and MeanFileSize (includes) — intentional but undocumented
- Git error logging inconsistent across operations

**Review output:** Orchestration log at `.squad/orchestration-log/2026-04-15T04:50:46Z-parker.md`

### Bubble tree lint fixes (2026-04-19)

- **Front-chain decomposition:** `packCircles` (cognitive complexity 34) split into four helpers: `placeInitialCircles` (initial 0-2 placement), `initFrontChain` (linked list setup), `findBestPlacement` (chain scan loop), `bestTangentPosition` (per-pair overlap check). Each stays well under complexity 10.

- **Quadratic solver extraction:** `enclosingThree` (cyclomatic 11) reduced to 3 by extracting `solveQuadraticForRadius`. The solver also absorbs the `R < minR` post-check for the linear case, keeping behavior identical.

- **Uppercase local variables in geometry code:** Go's `unexported-naming` rule bans uppercase locals in unexported functions. Renamed: `P`→`pts`, `R`→`r`/`boundary`, `A/B/C/D` eliminated by inlining (`fu`, `fv` used directly, offsets renamed to `u0`, `v0`).

- **Flag-parameter pattern:** `collectBubblesByType(node, isDir bool)` flagged by revive. Split into `collectBubbleDirs` and `collectBubbleFiles` — used by both PNG and SVG renderers.

- **Pre-existing lint issues:** `goconst` in `renderer_test.go` and `unparam` in `svg_helpers.go` are known and not ours to fix.

### SVG Legend Support (2026-04-21)

- **Pattern:** SVG renderers use raw `fmt.Fprintf` to `*os.File` — no templates, no SVG library. The SVG legend follows the same approach: `writeSVGLegend` writes a `<g>` group with `<rect>` swatches and `<text>` labels.

- **Signature expansion:** `Render`, `RenderRadial`, `RenderBubble` all gained a `*LegendInfo` parameter. Nil is a no-op (0 extra height, no legend elements). Both SVG and PNG/JPG paths handle legend identically — expand canvas/viewBox, render legend at `(0, vizHeight)`.

- **SVG viewBox strategy:** `totalHeight = vizHeight + ComputeLegendHeight(legend)`. Background `<rect>` also uses `totalHeight` so the legend area has a white backdrop.

- **Reuse of constants:** `svg_legend.go` reuses all legend layout constants from `legend.go` (legendRowHeight, legendSwatchHeight, legendLabelWidth, etc.) and the existing `formatBreakpoint` function. `legendTextColour` is defined as a package-level var in `svg_legend.go` matching the PNG legend's `#222222`.

- **Line-length lint:** Long function signatures (>120 chars) need multi-line formatting. Revive's `line-length-limit` catches these.

- **Key files:** `internal/render/svg_legend.go` (new), `internal/render/legend.go` (shared types+constants), `internal/render/svg_helpers.go` (writeSVGText helper).

### Legend Phase 2+4 Complete (2026-04-19)

- **Phase 2 (Dallas):** Legend builder API in cmd package, config wiring, NoLegend flag. All three viz commands updated. PNG/JPG paths fully integrated.
- **Phase 4 (our branch):** SVG legend rendering via `fmt.Fprintf` (raw elements, no templates). `writeSVGLegend` writes `<g>` groups with `<rect>` swatches and `<text>` labels. Nil legend no-op.
- **Signature expansion:** All three public render functions (`Render`, `RenderRadial`, `RenderBubble`) gained `*LegendInfo` parameter. PNG and SVG paths handle identically — expand canvas/viewBox, render legend at `(0, vizHeight)`.
- **Shared constants:** `legendRowHeight`, `legendSwatchHeight`, `legendLabelWidth`, `formatBreakpoint` used by both PNG and SVG. Reuse avoids duplication.
- **Key insight:** SVG legend is a direct mirror of PNG legend — same layout logic, same LegendInfo struct. Only rendering mechanism differs (fmt.Fprintf vs gg drawing calls).
- **Testing:** Nil legend tests unchanged (golden files unchanged). Full build and lint passes.

