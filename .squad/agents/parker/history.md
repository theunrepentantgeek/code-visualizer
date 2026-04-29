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

### Spiral lint fixes (PR #129, issue #127)

- **nilaway nil-source elimination:** Changed `BuildTimeBuckets` and `Layout` to return empty slices (`[]TimeBucket{}`, `[]SpiralNode{}`) instead of `nil` for zero-length input. This eliminates nilaway nil-flow warnings at the source rather than adding guards at every call site (28+ in layout_test, 18+ in timebucket_test). Updated tests to assert `BeEmpty()` instead of `BeNil()`.

- **Belt-and-suspenders check:** Added `len(buckets) == 0` guard in `buildTimeBuckets` (spiral_cmd.go) returning an eris-wrapped error, so callers always receive non-empty buckets.

- **revive identical-switch-branches:** Removed redundant explicit cases where the branch body matched `default` — `case Hourly` in `SpotsPerLap()`, `bucketDuration()`, `truncateToResolution()`; `case LabelNone` in `computeLabelVisibility()`; `case Daily` in `formatBucketLabel()`.

- **revive confusing-results:** Named return values on `commitTimeRange` (`earliest`, `latest`).

- **revive unused-receiver:** Renamed `c` to `_` on `applyBorder` method.
