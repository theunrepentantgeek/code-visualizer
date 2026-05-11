# Bishop: Radial Canvas Migration Review

## Summary

Clean, well-structured migration that correctly follows the Canvas bridge pattern established by treemap and spiral. The data flow is sound, layer usage is correct, and the colour pipeline has been properly shifted from pre-applied node fields to deferred Ink resolution. Three lint warnings need fixing, and one piece of dead code was left behind by the deletions.

## Issues Found

### Minor: Lint — `isDir` control flag coupling
- **File:** `cmd/codeviz/radial_canvas.go:229` and `:239`
- **Problem:** `revive` flags `isDir` parameter as control coupling in `radialFillInk` and `radialMetricValue`. In `radialMetricValue`, the `isDir` guard is redundant — `file` is already `nil` for directory nodes, so `file == nil` alone suffices.
- **Fix:** For `radialMetricValue`, drop the `isDir` parameter and rely on `file == nil`. For `radialFillInk`, consider inlining the decision at the call site in `addRadialDisc` instead of branching inside the function.

### Minor: Lint — missing blank line above declaration
- **File:** `cmd/codeviz/radial_canvas.go:310`
- **Problem:** `wsl_v5` requires whitespace before `var rotation float64` when preceded by another `var` declaration.
- **Fix:** Add a blank line between `var anchor canvas.TextAnchor` and `var rotation float64`, matching the spiral_canvas.go pattern at line 233.

### Minor: Orphaned code from deletions
- **File:** `internal/render/renderer_test.go:8` — `func makeFile` now unused
- **File:** `internal/render/svg_helpers.go:38` — `func writeSVGTextRotated` now unused
- **Problem:** Deleting `radialtree_test.go` and `svg_radial.go` left these helper functions without callers. This causes two `unused` lint errors.
- **Fix:** Delete both orphaned functions in this PR so lint stays green.

### Minor: Missing `FontSize: 0` in TextSpec
- **File:** `cmd/codeviz/radial_canvas.go:266` and `:320`
- **Problem:** The `TextSpec` structs in `addRadialRootLabel` and `addRadialExternalLabel` omit `FontSize: 0`. While Go zero-values handle this correctly, both `treemap_canvas.go` (lines 165, 231) and `spiral_canvas.go` (line 245) explicitly set `FontSize: 0`. Inconsistency across bridges.
- **Fix:** Add `FontSize: 0` to both TextSpec literals for uniformity. This is cosmetic but aids grep-ability and pattern consistency.

### Observation: `radialEffectiveFill` unreachable file path
- **File:** `cmd/codeviz/radial_canvas.go:336-342`
- **Problem:** `radialEffectiveFill` is only called from `addRadialRootLabel`, which is only reached when `dist == 0` (i.e., the root node). The root node is always a directory, so the `!node.IsDirectory` branch (`inks.fill.Dip(canvas.MetricValue{})`) is unreachable. If it were reached, it would dip with an empty MetricValue, returning the first bucket colour rather than the actual file colour. Not a bug today, but a latent defect if the function is ever reused.
- **Fix:** No action required now. If the function scope widens, it would need the file's MetricValue passed in.

## What Looks Good

1. **Layer usage is correct:** Background=0 (white rect), Structure=10 (edges), Content=20 (discs), Overlay=30 (labels). Matches spec and other bridges.

2. **Disc sorting preserved:** Largest-first draw order (`slices.SortFunc` with `cmp.Compare(b, a)`) correctly prevents small discs from being occluded. Faithful port from old renderer.

3. **Ink construction pattern:** `buildRadialInks` → `buildMetricInk` (shared helper) follows the treemap bridge exactly. Properly handles the fill-always / border-optional asymmetry.

4. **Parallel tree walk:** `collectRadialDiscs` correctly walks `RadialNode` and `model.Directory` trees in parallel using the files-first/dirs-second invariant. This is the right approach for deferred colour resolution.

5. **Label geometry:** Radial label positioning (angle normalization, anchor flip, label gap) matches the old renderer's behaviour and is consistent with spiral_canvas.go's approach.

6. **Legend deferral:** `slog.Warn` for unimplemented legend matches the spiral_cmd.go pattern.

7. **Node stripping is clean:** Removing `FillColour` and `BorderColour` from `RadialNode` correctly makes it geometry-only, matching the Canvas spec's principle of separating layout from colour.

8. **Error propagation:** `eris.Wrap(err, "render failed")` is consistent with other command pipelines.

## Testing Gaps to Consider

1. **No bridge-level tests exist** — `radial_canvas.go` has no `_test.go`. The treemap bridge also lacks unit tests, so this isn't a regression, but both would benefit from mock-backend tests (using `RenderTo` with a stub `Backend`).

2. **Edge cases worth testing:**
   - Empty root directory (no files, no subdirs) — does `collectRadialDiscs` return empty?
   - Single file in root — does the parallel tree walk pair correctly?
   - Deeply nested directories — does recursion work without stack issues at ~100 levels?
   - Node with `ShowLabel=false` — confirm no text shapes added
   - Node with `DiscRadius=0` — confirm it's excluded from disc entries but children are still walked

3. **Ink edge case:** `buildRadialInks` with unknown `fillMetric` (provider not found) — should return fixed ink fallback. The shared `buildMetricInk` handles this, but it's worth a test.

## Approved

**Yes, with fixes** — the three lint issues (control flag, whitespace, orphaned code) should be resolved before merge. The FontSize consistency is optional but recommended.
