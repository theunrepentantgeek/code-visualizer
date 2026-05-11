# Parker: Radial Canvas Migration Review

## Summary

Solid migration that faithfully reproduces the old three-pass rendering logic (edges → discs → labels) via the Canvas retained-mode API. Architecture follows treemap/spiral bridge patterns correctly. However, the branch introduces **6 lint failures** (3 in new code, 3 from orphaned symbols in deleted-file dependants) that block CI.

## Issues Found

### Major: Orphaned `makeFile` in renderer_test.go

- **File:** internal/render/renderer_test.go:8
- **Problem:** Deleting `radialtree_test.go` removed all callers of `makeFile`, leaving an unused function that fails the `unused` linter. This was not an issue on main.
- **Fix:** Delete or relocate `makeFile`. If other test files in `internal/render/` still exist and could use it, keep it; otherwise remove.

### Major: Orphaned `writeSVGTextRotated` in svg_helpers.go

- **File:** internal/render/svg_helpers.go:38
- **Problem:** Deleting `svg_radial.go` removed all callers of `writeSVGTextRotated`, leaving an unused function that fails the `unused` linter. Not an issue on main.
- **Fix:** Delete the function if no other SVG renderer uses it. If bubble/treemap SVG renderers are still in-flight, consider whether they'll need it.

### Major: Stale nolint directive in bubbletree_cmd.go

- **File:** cmd/codeviz/bubbletree_cmd.go:397
- **Problem:** The `//nolint:dupl` directive was suppressing a duplicate warning between the bubbletree and radial versions of `applyBorderColours`. With the radial version deleted, `dupl` no longer fires, so the directive is unused and `nolintlint` flags it.
- **Fix:** Remove the `//nolint:dupl` comment from `bubbletree_cmd.go:397`.

### Minor: flag-parameter lint on `radialFillInk`

- **File:** cmd/codeviz/radial_canvas.go:229
- **Problem:** `isDir bool` parameter triggers revive `flag-parameter` — a control flag selects between two code paths.
- **Fix:** Inline the check at the call site in `addRadialDisc` (line 210):
  ```go
  fill := inks.fill
  if e.isDir {
      fill = canvas.FixedInk(radialDefaultDirFill)
  }
  ```
  Then delete `radialFillInk`.

### Minor: flag-parameter lint on `radialMetricValue`

- **File:** cmd/codeviz/radial_canvas.go:239
- **Problem:** `isDir bool` parameter is redundant — directories always have `file == nil`. The bool triggers revive `flag-parameter`.
- **Fix:** Remove the `isDir` parameter. The existing `file == nil` check is sufficient:
  ```go
  func radialMetricValue(file *model.File, ink canvas.Ink) canvas.MetricValue {
      if file == nil {
          return canvas.MetricValue{}
      }
      return metricValueForFile(file, ink)
  }
  ```

### Minor: wsl_v5 blank line violation

- **File:** cmd/codeviz/radial_canvas.go:310
- **Problem:** `var rotation float64` is cuddled with the preceding `var anchor` declaration. `wsl_v5` requires a blank line before a new `var` declaration block.
- **Fix:** Add blank line between the two `var` declarations (matching the pattern in `spiral_canvas.go:231-233`).

## Correctness Notes (No Issues)

- **Colour constants:** All 6 colour values match the old renderer exactly.
- **Disc z-order:** Collect → sort descending by DiscRadius → draw. Faithful port.
- **Label rotation:** Half-plane check (`angle <= π/2 || angle > 3π/2`), anchor flip, and angle+π offset all match the old `drawExternalLabel`.
- **Root label contrast:** `radialEffectiveFill` returns `radialDefaultDirFill` for root (always a directory), then `TextColourFor()` computes contrast. Matches old `effectiveFill` behaviour.
- **Tree walk invariant:** `collectRadialDiscs` uses `fileIdx`/`dirIdx` counters dispatched by `IsDirectory`, identical to the old `applyRadialFillColours` pattern and treemap's `addTreemapRect`.
- **Edge width:** 0.5 matches the old renderer.
- **Border width on discs:** 1.0 matches old `drawSingleDisc`.
- **Empty directories / no children:** Handled correctly — parent disc still added, child loop is a no-op.
- **Nil border metric:** `buildRadialInks` keeps `FixedInk(radialDefaultBorder)` when `borderMetric == ""`. Correct.
- **`buildMetricInk` / `metricValueForFile` reuse:** Correctly shared from `treemap_canvas.go`. Same function, same semantics.
- **Canvas layer assignments:** Background → Structure (edges) → Content (discs) → Overlay (labels). Correct z-ordering via Canvas layer system.

## Legend Note

Legend rendering is deferred with `slog.Warn`. This is expected — Canvas `SetLegend` exists but legend rendering is not yet implemented per the Canvas spec. Not a regression since the old legend path would need a Canvas-native rewrite.

## Approved

**Yes, with fixes** — the 3 Major lint issues (orphaned symbols + stale nolint) and 3 Minor lint issues must be resolved before merge. All are mechanical fixes. The rendering logic itself is correct.
